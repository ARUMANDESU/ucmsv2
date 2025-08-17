package http

import (
	"github.com/go-chi/chi"

	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/application/registration"
	authhttp "github.com/ARUMANDESU/ucms/internal/ports/http/auth"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
)

type Port struct {
	reg  *registrationhttp.HTTP
	auth *authhttp.HTTP
}

type Args struct {
	RegistrationCommand *registration.Command
	AuthApp             *authapp.App
	CookieDomain        string
}

func NewPort(args Args) *Port {
	return &Port{
		reg: registrationhttp.NewHTTP(registrationhttp.Args{
			Command: args.RegistrationCommand,
		}),
		auth: authhttp.NewHTTP(authhttp.Args{
			App:          args.AuthApp,
			CookieDomain: args.CookieDomain,
		}),
	}
}

func (p *Port) Route(r chi.Router) chi.Router {
	if r == nil {
		r = chi.NewRouter()
	}

	p.reg.Route(r)
	p.auth.Route(r)

	return r
}
