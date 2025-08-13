package http

import (
	"github.com/go-chi/chi"

	"github.com/ARUMANDESU/ucms/internal/application/registration"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
)

type Port struct {
	reg *registrationhttp.HTTP
}

type Args struct {
	RegistrationCommand *registration.Command
}

func NewPort(args Args) *Port {
	return &Port{
		reg: registrationhttp.NewHTTP(registrationhttp.Args{
			Command: args.RegistrationCommand,
		}),
	}
}

func (p *Port) Route(r chi.Router) chi.Router {
	if r == nil {
		r = chi.NewRouter()
	}

	p.reg.Route(r)

	return r
}
