package framework

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
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
	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/application/mail"
	registrationapp "github.com/ARUMANDESU/ucms/internal/application/registration"
	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	httpport "github.com/ARUMANDESU/ucms/internal/ports/http"
	watermillport "github.com/ARUMANDESU/ucms/internal/ports/watermill"
	"github.com/ARUMANDESU/ucms/pkg/env"
	postgrespkg "github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
	"github.com/ARUMANDESU/ucms/tests/integration/builders"
	"github.com/ARUMANDESU/ucms/tests/integration/fixtures"
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
	traceRecorder   *tracetest.SpanRecorder
	logger          *slog.Logger

	routerRunning atomic.Bool
	testStartTime time.Time

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
}

type Application struct {
	Registration *registrationapp.App
	Mail         *mail.App
	Student      *studentapp.App
	Auth         *authapp.App
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s.traceRecorder = tracetest.NewSpanRecorder()
	s.traceProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(s.traceRecorder))
	otel.SetTracerProvider(s.traceProvider)

	s.startPostgreSQL(ctx)
	s.runMigrations()
	s.initializeWatermill()
	s.createApplication()
	s.createWatermillPort()
	s.initializeHelpers()

	s.startWatermillRouter()

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
	studentRepo := postgresrepo.NewStudentRepo(s.pgPool, nil, nil)
	groupRepo := postgresrepo.NewGroupRepo(s.pgPool, nil, nil)

	s.MockMailSender = mocks.NewMockMailSender()
	s.Require().NotNil(s.MockMailSender, "MockMailSender should be initialized")

	regApp := registrationapp.NewApp(registrationapp.Args{
		Mode:         env.Test,
		Repo:         registrationRepo,
		UserGetter:   userRepo,
		GroupGetter:  groupRepo,
		StudentSaver: studentRepo,
	})
	mailApp := mail.NewApp(mail.Args{
		Tracer:     nil,
		Logger:     s.logger,
		Mailsender: s.MockMailSender,
	})

	studentApp := studentapp.NewApp(studentapp.Args{
		Tracer:  nil,
		Logger:  s.logger,
		PgxPool: s.pgPool,
	})

	authApp := authapp.NewApp(authapp.Args{
		Tracer:                  nil,
		Logger:                  s.logger,
		UserGetter:              userRepo,
		AccessTokenSecretKey:    fixtures.AccessTokenSecretKey,
		RefreshTokenSecretKey:   fixtures.RefreshTokenSecretKey,
		AccessTokenlExpDuration: nil,
		RefreshTokenExpDuration: nil,
	})

	s.app = &Application{
		Registration: regApp,
		Mail:         mailApp,
		Student:      studentApp,
		Auth:         authApp,
	}

	s.httpHandler = chi.NewRouter()
	httpPort := httpport.NewPort(httpport.Args{
		RegistrationCommand: &regApp.CMD,
		AuthApp:             authApp,
		StudentApp:          studentApp,
		CookieDomain:        "localhost",
	})
	httpPort.Route(s.httpHandler)
}

func (s *IntegrationTestSuite) createWatermillPort() {
	logger := watermill.NewStdLogger(true, false)

	port, err := watermillport.NewPortForTest(s.watermillRouter, s.pgPool, logger)
	s.Require().NoError(err)

	s.watermillPort = port

	handlers := watermillport.AppEventHandlers{
		Registration: s.app.Registration.Event,
		Mail:         s.app.Mail.Event,
		Student:      s.app.Student.Event,
	}

	err = s.watermillPort.Run(context.Background(), handlers)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) startWatermillRouter() {
	routerStarted := make(chan struct{})

	go func() {
		s.T().Log("Starting Watermill router")
		s.routerRunning.Store(true)
		close(routerStarted)

		if err := s.watermillRouter.Run(context.Background()); err != nil {
			s.T().Logf("Watermill router failed: %v", err)
		}
		s.T().Log("Watermill router stopped")
	}()

	select {
	case <-routerStarted:
		s.T().Log("Router started, waiting for handlers to be ready...")
	case <-time.After(5 * time.Second):
		s.T().Fatal("Router failed to start within timeout")
	}

	s.Require().True(s.routerRunning.Load(), "Router should be running")

	s.T().Log("Watermill router and handlers are ready")
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

	if !s.routerRunning.Load() {
		s.T().Fatal("Router is not running, cannot proceed with test")
	}
}

func (s *IntegrationTestSuite) BeforeTest(suiteName, testName string) {
	s.testStartTime = time.Now()
}

func (s *IntegrationTestSuite) AfterTest(suiteName, testName string) {
	duration := time.Since(s.testStartTime)
	if duration > 2*time.Second {
		s.T().Logf("SLOW TEST: %s took %v", testName, duration)
	}
	// s.T().Logf("Cleaning up after test: %s.%s", suiteName, testName)
	// spans := s.traceRecorder.Ended()
	// s.T().Logf("Total spans recorded: %d", len(spans))
	// for _, span := range spans {
	// 	s.T().Logf("Span: %s, Name: %s, Status: %s", span.SpanContext().TraceID(), span.Name(), span.Status().Code.String())
	// 	attrs := span.Attributes()
	// 	attrjsonbytes, _ := json.MarshalIndent(attrs, "", "  ")
	// 	s.T().Logf("Attributes: %s", string(attrjsonbytes))
	// 	events := span.Events()
	// 	eventjsonbytes, _ := json.MarshalIndent(events, "", "  ")
	// 	s.T().Logf("Events: %s", string(eventjsonbytes))
	// 	fmt.Println("")
	// }
	s.T().Logf("Cleaning up after test: %s.%s", suiteName, testName)
	s.Event.ClearAllEvents(s.T())
	s.traceRecorder.Reset()
	s.DB.TruncateAll(s.T())
	s.MockMailSender.Reset()
}

// Context returns a test context with timeout
func (s *IntegrationTestSuite) Context() context.Context {
	return s.T().Context()
}
