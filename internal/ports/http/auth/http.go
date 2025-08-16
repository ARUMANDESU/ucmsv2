package authhttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
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
	CookieDomain string
}

func NewHTTP(args Args) *HTTP {
	if args.Tracer == nil {
		args.Tracer = tracer
	}
	if args.Logger == nil {
		args.Logger = logger
	}

	return &HTTP{
		tracer:     args.Tracer,
		logger:     args.Logger,
		app:        args.App,
		errhandler: httpx.NewErrorHandler(),
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

	var req LoginRequest
	// TODO: sanitize
	// TODO: validate

	res, err := h.app.LoginHandle(ctx, authapp.Login{
		EmailOrBarcode: req.EmailOrBarcode,
		Password:       req.Password,
	})
	if err != nil {
		h.errhandler.HandleError(w, r, err)
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

func (h *HTTP) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "Refresh")
	defer span.End()

	refreshCookie, err := r.Cookie(RefreshJWTCookie)
	if err != nil {
		h.errhandler.HandleError(w, r, fmt.Errorf("failed to get cookie from request: %w", err))
		return
	}

	res, err := h.app.RefreshHandle(ctx, authapp.Refresh{RefreshToken: refreshCookie.Value})
	if err != nil {
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

	http.SetCookie(w, &http.Cookie{
		Name:   AccessJWTCookie,
		MaxAge: -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   RefreshJWTCookie,
		MaxAge: -1,
	})
	span.AddEvent("User logged out", trace.WithAttributes(
		attribute.String("cookie_domain", h.cookiedomain),
	))

	httpx.Success(w, r, http.StatusOK, nil)
}
