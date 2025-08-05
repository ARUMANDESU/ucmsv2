package framework

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	ucmsv2 "github.com/ARUMANDESU/ucms"
	postgresrepo "github.com/ARUMANDESU/ucms/internal/adapters/repos/postgres"
	registrationapp "github.com/ARUMANDESU/ucms/internal/application/registration"
	registrationhttp "github.com/ARUMANDESU/ucms/internal/ports/http/registration"
	watermillport "github.com/ARUMANDESU/ucms/internal/ports/watermill"
	"github.com/ARUMANDESU/ucms/pkg/env"
	postgrespkg "github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/db"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/event"
	"github.com/ARUMANDESU/ucms/tests/integration/framework/http"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

type IntegrationTestSuite struct {
	suite.Suite

	// Infrastructure
	pgContainer     *postgres.PostgresContainer
	pgPool          *pgxpool.Pool
	watermillRouter *message.Router
	traceProvider   trace.TracerProvider

	// Application
	app           *Application
	watermillPort *watermillport.Port
	httpHandler   chi.Router

	// Helpers
	HTTP    *http.Helper
	DB      *db.Helper
	Event   *event.Helper
	Builder *builders.Factory

	// Mocks
	MockMailSender *mocks.MockMailSender

	mu sync.RWMutex
}

type Application struct {
	RegistrationApp *registrationapp.App
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	s.traceProvider = tp

	s.startPostgreSQL(ctx)

	s.runMigrations()

	s.initializeWatermill()

	s.createApplication()

	s.createWatermillPort()

	s.initializeHelpers()

	s.T().Log("Test suite setup completed")
}

func (s *IntegrationTestSuite) startPostgreSQL(ctx context.Context) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("ucms_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	s.pgPool, err = pgxpool.New(ctx, connStr)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) runMigrations() {
	connStr, _ := s.pgContainer.ConnectionString(context.Background(), "sslmode=disable")
	connStr = strings.Replace(connStr, "postgres://", "pgx://", 1)

	err := postgrespkg.Migrate(connStr, &ucmsv2.Migrations)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) initializeWatermill() {
	logger := watermill.NewStdLogger(false, false)
	s.watermillRouter, _ = message.NewRouter(message.RouterConfig{}, logger)
	s.watermillRouter.AddMiddleware(
		func(h message.HandlerFunc) message.HandlerFunc {
			return func(msg *message.Message) ([]*message.Message, error) {
				s.T().Logf("Received message: %s; metadata: %v", msg.UUID, msg.Metadata)

				return h(msg)
			}
		},
	)

	err := watermillx.InitializeEventSchema(context.Background(), s.pgPool, logger)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) createApplication() {
	registrationRepo := postgresrepo.NewRegistrationRepo(s.pgPool, nil, nil)
	userRepo := postgresrepo.NewUserRepo(s.pgPool, nil, nil)

	s.MockMailSender = mocks.NewMockMailSender()

	regApp := registrationapp.NewApp(registrationapp.Args{
		Mode:       env.Test,
		Repo:       registrationRepo,
		UserGetter: userRepo,
		Mailsender: s.MockMailSender,
	})

	s.app = &Application{
		RegistrationApp: regApp,
	}

	s.httpHandler = chi.NewRouter()
	registrationHTTP := registrationhttp.NewHTTP(registrationhttp.Args{
		Command: regApp.CMD,
	})
	registrationHTTP.Route(s.httpHandler)
}

func (s *IntegrationTestSuite) createWatermillPort() {
	logger := watermill.NewStdLogger(false, false)

	port, err := watermillport.NewPort(s.watermillRouter, s.pgPool, logger)
	s.Require().NoError(err)

	s.watermillPort = port

	handlers := watermillport.AppEventHandlers{
		Registration: s.app.RegistrationApp.Event,
	}

	err = s.watermillPort.Run(s.Context(), handlers)
	s.Require().NoError(err)

	go func() {
		s.T().Log("Starting Watermill router")
		if err := s.watermillRouter.Run(s.Context()); err != nil {
			s.T().Logf("Watermill router failed: %v", err)
		}
		s.T().Log("Watermill router stopped")
	}()
}

func (s *IntegrationTestSuite) initializeHelpers() {
	s.HTTP = http.NewHelper(s.httpHandler)
	s.DB = db.NewHelper(s.pgPool)
	s.Event = event.NewHelper(s.pgPool)
	s.Builder = builders.NewFactory()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.pgPool != nil {
		s.pgPool.Close()
	}

	if s.pgContainer != nil {
		ctx := context.Background()
		_ = s.pgContainer.Terminate(ctx)
	}

	if s.watermillRouter != nil {
		err := s.watermillRouter.Close()
		if err != nil {
			s.T().Logf("Failed to close Watermill router: %v", err)
		}
	}
}

// SetupTest prepares each test
func (s *IntegrationTestSuite) SetupTest() {
	s.T().Log("Setting up test environment")
}

func (s *IntegrationTestSuite) BeforeTest(suiteName, testName string) {
	s.T().Logf("Starting test: %s.%s", suiteName, testName)
}

func (s *IntegrationTestSuite) AfterTest(suiteName, testName string) {
	s.T().Logf("Completed test: %s.%s", suiteName, testName)
	s.DB.TruncateAll(s.T())
	s.MockMailSender.Reset()
	// s.Event.ClearAllEvents(s.T())
}

// Context returns a test context with timeout
func (s *IntegrationTestSuite) Context() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.T().Cleanup(cancel)
	return ctx
}
