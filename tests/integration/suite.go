package integration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Import the stdlib driver for pgx
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	ucmsv2 "github.com/ARUMANDESU/ucms"
	"github.com/ARUMANDESU/ucms/internal/domain/event"
	"github.com/ARUMANDESU/ucms/internal/domain/registration"
	postgrespkg "github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
)

type TestSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	pgPool      *pgxpool.Pool
	app         *App // Your application instance
}

func (s *TestSuite) SetupSuite() {
	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:17-alpine"),
		postgres.WithDatabase("ucms_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	s.Require().NoError(err)
	s.pgContainer = pgContainer

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	// Setup database pool
	s.pgPool, err = pgxpool.New(ctx, connStr)
	s.Require().NoError(err)

	// Run migrations
	s.T().Logf("Running migrations on database: %s", connStr)
	connStr = strings.Replace(connStr, "postgres://", "pgx://", 1)
	err = postgrespkg.Migrate(connStr, &ucmsv2.Migrations)
	s.Require().NoError(err)

	wlogger := watermill.NewStdLogger(true, true)
	watermillx.InitializeEventSchema(s.T().Context(), s.pgPool, wlogger)

	s.app, err = NewApp(s.pgPool)
	s.Require().NoError(err)
}

func (s *TestSuite) TearDownSuite() {
	if s.pgPool != nil {
		s.pgPool.Close()
	}

	if s.pgContainer != nil {
		ctx := context.Background()
		err := s.pgContainer.Terminate(ctx)
		s.Require().NoError(err)
	}
}

func (s *TestSuite) BeforeTest(suiteName, testName string) {
	// mock data
	_, err := s.pgPool.Exec(context.Background(), `
		INSERT INTO users (id, email, role_id, first_name, last_name, avatar_url, pass_hash, created_at, updated_at)
		VALUES ('210427', '210427@astanait.edu.kz', 2, 'Armand', 'Zhunissov', '', 'hashed_password', NOW(), NOW());
		INSERT INTO registrations (id, email, status, verification_code, code_attempts, code_expires_at, resend_timeout, created_at, updated_at)
		VALUES ('2f4c9b86-1d3e-4a72-a5c1-9b8f0d7e6c3b', '210427@astanait.edu.kz', 'completed', '123456', 0, NOW() + INTERVAL '10 minutes', NOW() + INTERVAL '1 hour', NOW(), NOW());
		`)
	s.Require().NoError(err)
	s.T().Log("Test data inserted")
	s.T().Logf("Running test: %s in suite: %s", testName, suiteName)
}

func (s *TestSuite) AfterTest(suiteName, testName string) {
	_, err := s.pgPool.Exec(context.Background(), "TRUNCATE TABLE users, registrations RESTART IDENTITY CASCADE")
	s.Require().NoError(err)
	s.T().Logf("Test data truncated after test: %s in suite: %s", testName, suiteName)
	if s.app.MockMailSender != nil {
		s.app.MockMailSender.Reset() // Reset mock mail sender after each test
		s.T().Log("Mock mail sender reset after test")
	}
}

func (s *TestSuite) App() *App {
	return s.app
}

func (s *TestSuite) AssertRegistration(expected registration.Registration) {
	actual, err := s.app.RegistrationRepo.GetRegistrationByID(s.T().Context(), expected.ID())
	s.Require().NoError(err, "Failed to get registration by ID")
	s.Require().NotNil(actual, "Expected registration to be found")

	s.Equal(expected.ID(), actual.ID(), "Registration ID mismatch")
	s.Equal(expected.Email(), actual.Email(), "Registration email mismatch")
	s.Equal(expected.Status(), actual.Status(), "Registration status mismatch")
	s.Equal(expected.VerificationCode(), actual.VerificationCode(), "Registration verification code mismatch")
	s.Equal(expected.CodeAttempts(), actual.CodeAttempts(), "Registration code attempts mismatch")
	s.WithinDuration(expected.CodeExpiresAt(), actual.CodeExpiresAt(), 1*time.Second, "Registration code expiration time mismatch")
	s.WithinDuration(expected.ResendTimeout(), actual.ResendTimeout(), 1*time.Second, "Registration resend timeout mismatch")
	s.WithinDuration(expected.CreatedAt(), actual.CreatedAt(), 1*time.Second, "Registration creation time mismatch")
	s.WithinDuration(expected.UpdatedAt(), actual.UpdatedAt(), 1*time.Second, "Registration update time mismatch")

	s.Equal(expected, actual)
}

func (s *TestSuite) AssertRegistrationExists(email string) {
	_, err := s.app.RegistrationRepo.GetRegistrationByEmail(s.T().Context(), email)
	s.Require().NoError(err, "Expected registration to exist for email: %s", email)
}

func (s *TestSuite) AssertRegistrationStartedEvent(email string) {
	AssertEvent(s, func(event *registration.RegistrationStarted) {
		s.Equal(email, event.Email, "RegistrationStarted event email mismatch")
		s.NotEmpty(event.VerificationCode, "Verification code should not be empty in RegistrationStarted event")
	})
}

func AssertDataInDB[T any](s *TestSuite, query string, args []any, fn func(row pgx.Row) (T, error), assertFn func(data T)) {
	row := s.pgPool.QueryRow(context.Background(), query, args...)
	data, err := fn(row)
	s.Require().NoError(err)
	assertFn(data)
}

func AssertEvent[T event.Event](s *TestSuite, fn func(event T)) {
	typeName := fmt.Sprintf("%T", new(T))
	s.T().Logf("Event type: %s", typeName)
	// remove * from typeName if it exists
	if strings.HasPrefix(typeName, "*") {
		typeName = typeName[2:]
	}
	e := new(T)

	s.T().Logf("Waiting for event of type: %s", typeName)
	query := fmt.Sprintf(
		`SELECT payload  FROM %s WHERE metadata->>'name' = $1 ORDER BY "offset" DESC LIMIT 1`,
		"watermill_"+T.GetStreamName(*e),
	)

	row := s.pgPool.QueryRow(context.Background(), query, typeName)
	err := row.Scan(&e)
	s.Require().NoError(err, "Failed to get event from database")
	s.T().Logf("Event %s received: %v", typeName, e)
	s.Require().NotNil(e, "Event should not be nil")
	fn(*e)
}
