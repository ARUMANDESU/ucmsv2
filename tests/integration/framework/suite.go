package framework

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	ucmsv2 "gitlab.com/ucmsv2/ucms-backend"
	postgresrepo "gitlab.com/ucmsv2/ucms-backend/internal/adapters/repos/postgres"
	"gitlab.com/ucmsv2/ucms-backend/internal/adapters/services/s3"
	authapp "gitlab.com/ucmsv2/ucms-backend/internal/application/auth"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/mail"
	registrationapp "gitlab.com/ucmsv2/ucms-backend/internal/application/registration"
	staffapp "gitlab.com/ucmsv2/ucms-backend/internal/application/staff"
	studentapp "gitlab.com/ucmsv2/ucms-backend/internal/application/student"
	userapp "gitlab.com/ucmsv2/ucms-backend/internal/application/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/group"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	httpport "gitlab.com/ucmsv2/ucms-backend/internal/ports/http"
	watermillport "gitlab.com/ucmsv2/ucms-backend/internal/ports/watermill"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	postgrespkg "gitlab.com/ucmsv2/ucms-backend/pkg/postgres"
	"gitlab.com/ucmsv2/ucms-backend/pkg/watermillx"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/builders"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/fixtures"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/db"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/event"
	"gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/http"
	s3helper "gitlab.com/ucmsv2/ucms-backend/tests/integration/framework/s3"
	"gitlab.com/ucmsv2/ucms-backend/tests/mocks"
)

const (
	MinIOUsername = "test-minio"
	MinIOPassword = "test-minio-password"
	MinIOBucket   = "ucms-test-bucket"
)

type IntegrationTestSuite struct {
	suite.Suite

	HTTPPort *httpport.Port

	// Infrastructure
	pgContainer    *postgres.PostgresContainer
	pgPool         *pgxpool.Pool
	minioContainer *minio.MinioContainer

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
	S3      *s3helper.Helper

	MockMailSender *mocks.MockMailSender
	S3Client       *s3.Client
}

type Application struct {
	Registration *registrationapp.App
	Mail         *mail.App
	Student      *studentapp.App
	Staff        *staffapp.App
	Auth         *authapp.App
	User         *userapp.App
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	s.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s.traceRecorder = tracetest.NewSpanRecorder()
	s.traceProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(s.traceRecorder))
	otel.SetTracerProvider(s.traceProvider)

	s.startPostgreSQL(ctx)
	s.runMigrations()
	s.startMinIO()
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

func (s *IntegrationTestSuite) startMinIO() {
	minioContainer, err := minio.Run(s.Context(), "minio/minio:latest",
		minio.WithUsername(MinIOUsername),
		minio.WithPassword(MinIOPassword),
	)
	s.Require().NoError(err)
	s.minioContainer = minioContainer
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
	endpoint, err := s.minioContainer.Endpoint(s.Context(), "")
	s.Require().NoError(err)

	// Ensure endpoint has http:// prefix for S3 client
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	s3Client, err := s3.NewClient(s.Context(),
		endpoint,
		MinIOUsername,
		MinIOPassword,
		MinIOBucket,
		"us-east-1",
	)
	s.Require().NoError(err)

	// Create the bucket for tests
	err = s3Client.CreateBucket(s.Context())
	s.Require().NoError(err)

	s.S3Client = s3Client

	registrationRepo := postgresrepo.NewRegistrationRepo(s.pgPool, nil, nil)
	userRepo := postgresrepo.NewUserRepo(s.pgPool, nil, nil)
	studentRepo := postgresrepo.NewStudentRepo(s.pgPool, nil, nil)
	staffInvitationRepo := postgresrepo.NewStaffInvitationRepo(s.pgPool, nil, nil)
	staffRepo := postgresrepo.NewStaffRepo(s.pgPool, nil, nil)
	groupRepo := postgresrepo.NewGroupRepo(s.pgPool, nil, nil)

	s.MockMailSender = mocks.NewMockMailSender()
	s.Require().NotNil(s.MockMailSender, "MockMailSender should be initialized")

	regApp := registrationapp.NewApp(registrationapp.Args{
		Mode:         env.Test,
		Repo:         registrationRepo,
		UserGetter:   userRepo,
		GroupGetter:  groupRepo,
		StudentSaver: studentRepo,
		PgxPool:      s.pgPool,
	})
	mailApp := mail.NewApp(mail.Args{
		Mailsender:              s.MockMailSender,
		StaffInvitationBaseURL:  "http://localhost:3000/invitations/staff",
		InvitationCreatorGetter: staffRepo,
	})

	studentApp := studentapp.NewApp(studentapp.Args{
		Tracer:  nil,
		Logger:  s.logger,
		PgxPool: s.pgPool,
	})

	staffApp := staffapp.NewApp(staffapp.Args{
		StaffInvitationRepo: staffInvitationRepo,
		StaffRepo:           staffRepo,
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

	userApp := userapp.NewApp(userapp.Args{
		S3BaseURL:     fixtures.ValidS3BaseURL,
		AvatarStorage: s3Client,
		UserRepo:      userRepo,
	})

	s.app = &Application{
		Registration: regApp,
		Mail:         mailApp,
		Student:      studentApp,
		Staff:        staffApp,
		Auth:         authApp,
		User:         userApp,
	}

	s.httpHandler = chi.NewRouter()
	s.HTTPPort = httpport.NewPort(httpport.Args{
		RegistrationApp:         regApp,
		AuthApp:                 authApp,
		StudentApp:              studentApp,
		StaffApp:                staffApp,
		CookieDomain:            "localhost",
		Secret:                  []byte(fixtures.AccessTokenSecretKey),
		AcceptInvitationPageURL: fixtures.StaffInvitationAcceptPageURL,
		InvitationTokenAlg:      fixtures.InvitationTokenAlg,
		InvitationTokenKey:      fixtures.InvitationTokenKey,
		InvitationTokenExp:      fixtures.InvitationTokenExp,
		ServiceName:             fixtures.ServiceName,
		UserApp:                 userApp,
	})
	s.HTTPPort.Route(s.httpHandler)
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
	s.DB = db.NewHelper(db.Args{Pool: s.pgPool})
	s.Event = event.NewHelper(s.pgPool)
	s.Builder = builders.NewFactory()
	s.S3 = s3helper.NewHelper(s.S3Client)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.pgPool != nil {
		s.pgPool.Close()
	}

	if s.minioContainer != nil {
		_ = s.minioContainer.Terminate(s.Context())
	}
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.Context())
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

func (s *IntegrationTestSuite) SeedStaff(t *testing.T, email string) *user.Staff {
	t.Helper()
	staffUser := s.Builder.User.Staff(email)
	s.DB.SeedStaff(t, staffUser)
	return staffUser
}

func (s *IntegrationTestSuite) SeedGroup(t *testing.T) group.ID {
	t.Helper()
	groupID := group.NewID()
	s.DB.SeedGroup(t, groupID, fixtures.SEGroup.Name, fixtures.SEGroup.Year, fixtures.SEGroup.Major)
	return groupID
}

func (s *IntegrationTestSuite) SeedStudent(t *testing.T, email string, groupID group.ID) *user.Student {
	t.Helper()
	studentUser := builders.NewStudentBuilder().WithEmail(email).WithGroupID(groupID).Build()
	s.DB.SeedStudent(t, studentUser)
	return studentUser
}
