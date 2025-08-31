package authhttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
	"github.com/ARUMANDESU/ucms/pkg/sanitizex"
	"github.com/ARUMANDESU/ucms/pkg/validationx"
)

const (
	AccessJWTCookie   = "ucmsv2_access"
	RefreshJWTCookie  = "ucmsv2_refresh"
	RefreshCookiePath = "/v1/auth/refresh"
)

var (
	tracer = otel.Tracer("ucms/internal/ports/http/auth")
	logger = otelslog.NewLogger("ucms/internal/ports/http/auth")
)

type HTTP struct {
	tracer       trace.Tracer
	logger       *slog.Logger
	app          *authapp.App
	errhandler   *httpx.ErrorHandler
	cookiedomain string
}

type Args struct {
	Tracer       trace.Tracer
	Logger       *slog.Logger
	App          *authapp.App
	Errhandler   *httpx.ErrorHandler
	CookieDomain string
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}
	if args.CookieDomain == "" {
		args.CookieDomain = "localhost"
	}

	return &HTTP{
		tracer:       args.Tracer,
		logger:       args.Logger,
		app:          args.App,
		errhandler:   args.Errhandler,
		cookiedomain: args.CookieDomain,
	}
}

func (h *HTTP) Route(r chi.Router) {
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/refresh", h.Refresh)
	r.Post("/v1/auth/logout", h.Logout)
}

type LoginRequest struct {
	EmailOrBarcode string `json:"email_barcode"`
	Password       string `json:"password"`
}

func (h *HTTP) Login(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "Login")
	defer span.End()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<12) // 4KB cap

	var req LoginRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read json")
		httpx.BadRequest(w, r, err.Error())
		return
	}

	req.EmailOrBarcode = sanitizex.CleanSingleLine(req.EmailOrBarcode)

	isEmail, isBarcode := validationx.IsEmailOrBarcode(req.EmailOrBarcode)
	if !isEmail && !isBarcode {
		span.RecordError(authapp.ErrWrongEmailOrBarcodeOrPassword)
		span.SetStatus(codes.Error, "invalid email or barcode format")
		h.errhandler.HandleError(w, r, authapp.ErrWrongEmailOrBarcodeOrPassword)
		return
	}
	var validationRules []validation.Rule
	if isEmail {
		copy(validationRules, validationx.EmailRules)
	} else if isBarcode {
		validationRules = append(validationRules, validation.Length(0, 80), is.Alphanumeric)
	}

	err := validation.ValidateStruct(&req,
		validation.Field(&req.EmailOrBarcode, validationRules...),
		validation.Field(&req.Password, validation.Required, validation.Length(0, 100)),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate request body")
		h.errhandler.HandleError(w, r, err)
		return
	}

	res, err := h.app.LoginHandle(ctx, authapp.Login{
		EmailOrBarcode: req.EmailOrBarcode,
		IsEmail:        isEmail,
		Password:       req.Password,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to login user")
		h.errhandler.HandleError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    res.AccessToken,
		Path:     "/",
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.AccessTokenExp).UTC(),
		MaxAge:   int(res.AccessTokenExp.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    res.RefreshToken,
		Path:     RefreshCookiePath,
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.RefreshTokenExp).UTC(),
		MaxAge:   int(res.RefreshTokenExp.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "Refresh")
	defer span.End()

	refreshCookie, err := r.Cookie(RefreshJWTCookie)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get refresh token from cookie")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, fmt.Errorf("failed to get cookie from request: %w", err))
		return
	}

	err = validation.Validate(refreshCookie.Value, validation.Required, validation.Length(1, 1000))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to validate refresh token")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, fmt.Errorf("failed to validate refresh token from cookie: %w", errorx.NewInvalidCredentials().WithCause(err)))
		return
	}

	res, err := h.app.RefreshHandle(ctx, authapp.Refresh{RefreshToken: refreshCookie.Value})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to refresh access token")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, fmt.Errorf("failed refresh token: %w", err))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    res.AccessToken,
		Path:     "/",
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.AccessTokenExp).UTC(),
		MaxAge:   int(res.AccessTokenExp),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    res.RefreshToken,
		Path:     RefreshCookiePath,
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.RefreshTokenExp).UTC(),
		MaxAge:   int(res.RefreshTokenExp),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) Logout(w http.ResponseWriter, r *http.Request) {
	_, span := h.tracer.Start(r.Context(), "Logout")
	defer span.End()

	accessCookie, err := r.Cookie(AccessJWTCookie)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get access token from cookie")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, fmt.Errorf("failed to get cookie from request: %w", err))
		return
	}
	if accessCookie == nil || accessCookie.Value == "" {
		err = errorx.NewInvalidCredentials().WithCause(fmt.Errorf("no access token found in cookie"))
		span.RecordError(err)
		span.SetStatus(codes.Error, "no access token found in cookie")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, err)
		return
	}

	refreshCookie, err := r.Cookie(RefreshJWTCookie)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get refresh token from cookie")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, fmt.Errorf("failed to get cookie from request: %w", err))
		return
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		err = errorx.NewInvalidCredentials().WithCause(fmt.Errorf("no refresh token found in cookie"))
		span.RecordError(err)
		span.SetStatus(codes.Error, "no refresh token found in cookie")
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, err)
		return
	}

	h.resetCookies(w)
	span.AddEvent("User logged out", trace.WithAttributes(
		attribute.String("cookie_domain", h.cookiedomain),
	))

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) resetCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}
