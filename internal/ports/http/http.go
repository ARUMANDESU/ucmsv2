package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/otel"

	authapp "gitlab.com/ucmsv2/ucms-backend/internal/application/auth"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration"
	staffapp "gitlab.com/ucmsv2/ucms-backend/internal/application/staff"
	studentapp "gitlab.com/ucmsv2/ucms-backend/internal/application/student"
	userapp "gitlab.com/ucmsv2/ucms-backend/internal/application/user"
	authhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/auth"
	"gitlab.com/ucmsv2/ucms-backend/internal/ports/http/middlewares"
	registrationhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/registration"
	staffhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/staff"
	studenthttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/student"
	userhttp "gitlab.com/ucmsv2/ucms-backend/internal/ports/http/user"
	"gitlab.com/ucmsv2/ucms-backend/pkg/httpx"
)

type Port struct {
	serviceName string
	reg         *registrationhttp.HTTP
	auth        *authhttp.HTTP
	student     *studenthttp.HTTP
	staff       *staffhttp.HTTP
	user        *userhttp.HTTP
}

type Args struct {
	ServiceName             string
	RegistrationApp         *registration.App
	AuthApp                 *authapp.App
	StudentApp              *studentapp.App
	StaffApp                *staffapp.App
	UserApp                 *userapp.App
	CookieDomain            string
	Secret                  []byte
	AcceptInvitationPageURL string
	InvitationTokenAlg      jwt.SigningMethod
	InvitationTokenKey      string
	InvitationTokenExp      time.Duration
}

func NewPort(args Args) *Port {
	errorHandler := httpx.NewErrorHandler()
	m := middlewares.NewMiddleware(middlewares.Args{
		Secret:     args.Secret,
		Exp:        authapp.AccessTokenExpDuration,
		Errhandler: errorHandler,
	})
	return &Port{
		serviceName: args.ServiceName,
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
		staff: staffhttp.NewHTTP(staffhttp.Args{
			App:                     args.StaffApp,
			Errhandler:              errorHandler,
			Middleware:              m,
			AcceptInvitationPageURL: args.AcceptInvitationPageURL,
			InvitationTokenAlg:      args.InvitationTokenAlg,
			InvitationTokenKey:      args.InvitationTokenKey,
			InvitationTokenExp:      args.InvitationTokenExp,
		}),
		user: userhttp.NewHTTP(userhttp.Args{
			UserApp:    args.UserApp,
			Middleware: m,
			Errhandler: errorHandler,
		}),
	}
}

func (p *Port) Route(r chi.Router) chi.Router {
	if r == nil {
		r = chi.NewRouter()
	}
	baseCfg := otelchimetric.NewBaseConfig(p.serviceName, otelchimetric.WithMeterProvider(otel.GetMeterProvider()))
	r.Use(
		middlewares.OTel,
		otelchimetric.NewRequestDurationMillis(baseCfg),
		otelchimetric.NewRequestInFlight(baseCfg),
		otelchimetric.NewResponseSizeBytes(baseCfg),
	)
	r.Use(middleware.CleanPath)
	r.Use(middleware.RealIP)
	r.Use(middlewares.OTel)
	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("application/json", "multipart/form-data"))
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
	p.staff.Route(r)
	p.user.Route(r)

	return r
}
