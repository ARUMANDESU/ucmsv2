package integration

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/internal/adapters/repos/postgres"
	registrationapp "github.com/ARUMANDESU/ucms/internal/application/registration"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type App struct {
	HTTPHandler      http.Handler
	MockMailSender   *mocks.MockMailSender
	RegistrationRepo *postgres.RegistrationRepo
	UserRepo         *postgres.UserRepo
}

func NewApp(pool *pgxpool.Pool) (*App, error) {
	mux := chi.NewRouter()

	registrationRepo := postgres.NewRegistrationRepo(pool)
	userRepo := postgres.NewUserRepo(pool)

	mockMailSender := &mocks.MockMailSender{}

	regapp := registrationapp.NewApp(registrationapp.Args{
		Mode:       env.Test,
		Repo:       registrationRepo,
		UserGetter: userRepo,
		Mailsender: mockMailSender,
	})

	registrationHTTP := registrationhttp.NewHTTP(registrationhttp.Args{Command: regapp.CMD})

	registrationHTTP.Route(mux)

	return &App{
		HTTPHandler:      mux,
		MockMailSender:   mockMailSender,
		RegistrationRepo: registrationRepo,
		UserRepo:         userRepo,
	}, nil
}
