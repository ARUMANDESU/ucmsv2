package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/application/registration"
	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	"github.com/ARUMANDESU/ucms/internal/ports/http/middlewares"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
	studenthttp "github.com/ARUMANDESU/ucms/internal/ports/http/student"
	"github.com/ARUMANDESU/ucms/pkg/httpx"
)

type Port struct {
	reg     *registrationhttp.HTTP
	auth    *authhttp.HTTP
	student *studenthttp.HTTP
}

type Args struct {
	RegistrationApp *registration.App
	AuthApp         *authapp.App
	StudentApp      *studentapp.App
	CookieDomain    string
	Secret          []byte
}

func NewPort(args Args) *Port {
	errorHandler := httpx.NewErrorHandler()
	m := middlewares.NewMiddleware(middlewares.Args{
		Secret:     args.Secret,
		Exp:        authapp.AccessTokenExpDuration,
		Errhandler: errorHandler,
	})
	return &Port{
		reg: registrationhttp.NewHTTP(registrationhttp.Args{
			App:        args.RegistrationApp,
			Errhandler: errorHandler,
		}),
		auth: authhttp.NewHTTP(authhttp.Args{
			App:          args.AuthApp,
			CookieDomain: args.CookieDomain,
			Errhandler:   errorHandler,
		}),
		student: studenthttp.NewHTTP(studenthttp.Args{
			App:        args.StudentApp,
			Errhandler: errorHandler,
			Middleware: m,
		}),
	}
}

func (p *Port) Route(r chi.Router) chi.Router {
	if r == nil {
		r = chi.NewRouter()
	}

	r.Use(otelhttp.NewMiddleware("ucmsv2-api"))
	r.Use(middleware.CleanPath)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			h.ServeHTTP(w, r)
		})
	})

	p.reg.Route(r)
	p.auth.Route(r)
	p.student.Route(r)

	return r
}
