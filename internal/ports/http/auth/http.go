package authhttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ARUMANDESU/validation"
	"github.com/ARUMANDESU/validation/is"
	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/pkg/env"
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
	httpOnly     bool
	secure       bool
	sameSite     http.SameSite
}

type Args struct {
	Tracer       trace.Tracer
	Logger       *slog.Logger
	App          *authapp.App
	Errhandler   *httpx.ErrorHandler
	CookieDomain string
}

func NewHTTP(args Args) *HTTP {
	h := &HTTP{
		tracer:       args.Tracer,
		logger:       args.Logger,
		app:          args.App,
		errhandler:   args.Errhandler,
		cookiedomain: args.CookieDomain,
		httpOnly:     true,
		secure:       true,
		sameSite:     http.SameSiteStrictMode,
	}

	if h.tracer == nil {
		h.tracer = tracer
	}
	if h.logger == nil {
		h.logger = logger
	}
	if h.errhandler == nil {
		h.errhandler = httpx.NewErrorHandler()
	}
	if env.Current() == env.Local {
		h.cookiedomain = "localhost"
		h.secure = false // for local development with http
	}

	return h
}

func (h *HTTP) Route(r chi.Router) {
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/refresh", h.Refresh)
	r.Post("/v1/auth/logout", h.Logout)
}

type LoginRequest struct {
	EmailOrBarcode     string `json:"email_barcode"`
	Password           string `json:"password"`
	isEmail, isBarcode bool   `json:"-"`
}

func (r *LoginRequest) Sanitized() {
	r.EmailOrBarcode = sanitizex.CleanSingleLine(r.EmailOrBarcode)
	r.Password = strings.TrimSpace(r.Password)
	r.isEmail, r.isBarcode = validationx.IsEmailOrBarcode(r.EmailOrBarcode)
}

func (r *LoginRequest) SetSpanAttrs(span trace.Span) {
	if r.isEmail {
		span.SetAttributes(attribute.String("email", r.EmailOrBarcode))
	} else if r.isBarcode {
		span.SetAttributes(attribute.String("barcode", r.EmailOrBarcode))
	}
}

func (r *LoginRequest) Validate() error {
	var validationRules []validation.Rule
	if r.isEmail {
		copy(validationRules, validationx.EmailRules)
	} else if r.isBarcode {
		validationRules = append(validationRules, validation.Length(0, 80), is.Alphanumeric)
	}

	return validation.ValidateStruct(r,
		validation.Field(&r.EmailOrBarcode, validationRules...),
		validation.Field(&r.Password, validation.Required, validation.Length(0, 100)),
	)
}

func (h *HTTP) Login(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "Login")
	defer span.End()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<12) // 4KB cap

	var req LoginRequest
	if err := httpx.ReadJSON(w, r, &req); err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to read json")
		return
	}

	req.Sanitized()
	req.SetSpanAttrs(span)
	if !req.isEmail && !req.isBarcode {
		h.errhandler.HandleError(w, r, span, authapp.ErrWrongEmailOrBarcodeOrPassword, "email or barcode is not valid")
		return
	}
	err := req.Validate()
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to validate request")
		return
	}

	res, err := h.app.LoginHandle(ctx, authapp.Login{
		EmailOrBarcode: req.EmailOrBarcode,
		IsEmail:        req.isEmail,
		Password:       req.Password,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    res.AccessToken,
		Path:     "/",
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.AccessTokenExp).UTC(),
		MaxAge:   int(res.AccessTokenExp.Seconds()),
		Secure:   h.secure,
		HttpOnly: h.httpOnly,
		SameSite: h.sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    res.RefreshToken,
		Path:     RefreshCookiePath,
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.RefreshTokenExp).UTC(),
		MaxAge:   int(res.RefreshTokenExp.Seconds()),
		Secure:   h.secure,
		HttpOnly: h.httpOnly,
		SameSite: h.sameSite,
	})

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) Refresh(w http.ResponseWriter, r *http.Request) {
	const op = "http.auth.Refresh"
	ctx, span := h.tracer.Start(r.Context(), "Refresh")
	defer span.End()

	refreshCookie, err := r.Cookie(RefreshJWTCookie)
	if err != nil {
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, span, err, "failed to get refresh token from cookie")
		return
	}

	err = validation.Validate(refreshCookie.Value, validation.Required, validation.Length(1, 1000))
	if err != nil {
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, span, errorx.NewInvalidCredentials().WithCause(err, op), "invalid refresh token in cookie")
		return
	}

	res, err := h.app.RefreshHandle(ctx, authapp.Refresh{RefreshToken: refreshCookie.Value})
	if err != nil {
		h.resetCookies(w)
		h.errhandler.HandleError(w, r, span, err, "failed to refresh token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    res.AccessToken,
		Path:     "/",
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.AccessTokenExp).UTC(),
		MaxAge:   int(res.AccessTokenExp.Seconds()),
		Secure:   h.secure,
		HttpOnly: h.httpOnly,
		SameSite: h.sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    res.RefreshToken,
		Path:     RefreshCookiePath,
		Domain:   h.cookiedomain,
		Expires:  time.Now().Add(res.RefreshTokenExp).UTC(),
		MaxAge:   int(res.RefreshTokenExp.Seconds()),
		Secure:   h.secure,
		HttpOnly: h.httpOnly,
		SameSite: h.sameSite,
	})

	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) Logout(w http.ResponseWriter, r *http.Request) {
	const op = "http.auth.Logout"
	_, span := h.tracer.Start(r.Context(), "Logout")
	defer span.End()
	defer h.resetCookies(w)

	accessCookie, err := r.Cookie(AccessJWTCookie)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get access token from cookie")
		return
	}
	if accessCookie == nil || accessCookie.Value == "" {
		err = errorx.NewInvalidCredentials().WithCause(fmt.Errorf("no access token found in cookie"), op)
		h.errhandler.HandleError(w, r, span, err, "no access token found in cookie")
		return
	}

	refreshCookie, err := r.Cookie(RefreshJWTCookie)
	if err != nil {
		h.errhandler.HandleError(w, r, span, err, "failed to get refresh token from cookie")
		return
	}
	if refreshCookie == nil || refreshCookie.Value == "" {
		err = errorx.NewInvalidCredentials().WithCause(fmt.Errorf("no refresh token found in cookie"), op)
		h.errhandler.HandleError(w, r, span, err, "no refresh token found in cookie")
		return
	}

	span.AddEvent("User logged out", trace.WithAttributes(attribute.String("cookie_domain", h.cookiedomain)))

	h.resetCookies(w)
	httpx.Success(w, r, http.StatusOK, nil)
}

func (h *HTTP) resetCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessJWTCookie,
		Value:    "",
		Path:     "/",
		Domain:   h.cookiedomain,
		MaxAge:   -1,
		HttpOnly: h.httpOnly,
		Secure:   h.secure,
		SameSite: h.sameSite,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshJWTCookie,
		Value:    "",
		Path:     RefreshCookiePath,
		Domain:   h.cookiedomain,
		MaxAge:   -1,
		HttpOnly: h.httpOnly,
		Secure:   h.secure,
		SameSite: h.sameSite,
	})
}
