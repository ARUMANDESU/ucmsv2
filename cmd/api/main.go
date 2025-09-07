package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.36.0"

	ucmsv2 "gitlab.com/ucmsv2/ucms-backend"
	"gitlab.com/ucmsv2/ucms-backend/internal/adapters/repos/postgres"
	"gitlab.com/ucmsv2/ucms-backend/internal/adapters/services/s3"
	authapp "gitlab.com/ucmsv2/ucms-backend/internal/application/auth"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/mail"
	"gitlab.com/ucmsv2/ucms-backend/internal/application/registration"
	staffapp "gitlab.com/ucmsv2/ucms-backend/internal/application/staff"
	studentapp "gitlab.com/ucmsv2/ucms-backend/internal/application/student"
	userapp "gitlab.com/ucmsv2/ucms-backend/internal/application/user"
	"gitlab.com/ucmsv2/ucms-backend/internal/domain/user"
	httpport "gitlab.com/ucmsv2/ucms-backend/internal/ports/http"
	watermillport "gitlab.com/ucmsv2/ucms-backend/internal/ports/watermill"
	"gitlab.com/ucmsv2/ucms-backend/pkg/env"
	pgpkg "gitlab.com/ucmsv2/ucms-backend/pkg/postgres"
	"gitlab.com/ucmsv2/ucms-backend/pkg/watermillx"
	"gitlab.com/ucmsv2/ucms-backend/tests/mocks"
)

const (
	traceBatchTimeout      = 1 * time.Second
	metricPeriodicInterval = 3 * time.Second
)

// Application holds all the application dependencies
type Application struct {
	Registration *registration.App
	Mail         *mail.App
	Student      *studentapp.App
	Staff        *staffapp.App
	Auth         *authapp.App
	User         *userapp.App
}

// Config holds all configuration for the application
type Config struct {
	Mode                     env.Mode
	Service                  ServiceConfig
	S3                       S3Config
	Port                     string
	PgDSN                    string
	LogPath                  string
	InitialStaff             *user.CreateInitialStaffArgs
	AccessTokenSecretKey     string
	RefreshTokenSecretKey    string
	StaffInvitationBaseURL   string
	AccestInvitationPageURL  string
	InvitationTokenSecretKey string
}

type ServiceConfig struct {
	Namespace  string
	Name       string
	Version    string
	InstanceId string
}

type S3Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	Region       string // can be anything for MinIO
	BaseURL      string // For building public URLs
	UsePathStyle bool   // true for MinIO
}

func main() {
	startTime := time.Now()
	ctx := context.Background()

	config := loadConfig()

	env.SetMode(config.Mode)

	shutdownOTel, err := setupOTelSDK(ctx, config)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to set up OpenTelemetry SDK", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to set up OpenTelemetry SDK: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if shutdownOTel != nil {
			if err := shutdownOTel(ctx); err != nil {
				slog.ErrorContext(ctx, "Failed to shutdown OpenTelemetry SDK", "error", err)
			}
		}
	}()

	slog.InfoContext(ctx, "Starting UCMS API server",
		"mode", config.Mode,
		"port", config.Port,
	)

	pool, err := setupDatabase(ctx, config)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup database", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to setup database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	repos := setupRepositories(pool)

	infrastructure := setupInfrastructure(ctx, config)

	wlogger := watermillx.NewOTelFilteredSlogLogger(slog.Default(), env.Current().SlogLevel())

	eventRouter, err := setupEventProcessing(ctx, pool, wlogger)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup event processing", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to setup event processing: %v\n", err)
		os.Exit(1)
	}

	apps := setupApplications(config, repos, infrastructure)

	wmport, err := watermillport.NewPort(eventRouter, pool, wlogger)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create Watermill port", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to create Watermill port: %v\n", err)
		os.Exit(1)
	}
	if err := wmport.Run(ctx, watermillport.AppEventHandlers{
		Registration: apps.Registration.Event,
		Mail:         apps.Mail.Event,
		Student:      apps.Student.Event,
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to run Watermill port", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to run Watermill port: %v\n", err)
		os.Exit(1)
	}

	go func() {
		// Start event router
		if err := eventRouter.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to start event router", "error", err)
			fmt.Fprintf(os.Stderr, "Failed to start event router: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := eventRouter.Close(); err != nil {
				slog.ErrorContext(ctx, "Failed to close event router", "error", err)
			}
		}()
	}()

	// Create initial staff user if configured
	hasStaff, err := repos.Staff.HasAnyStaff(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to check for existing staff users", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to check for existing staff users: %v\n", err)
		os.Exit(1)
	}

	if config.InitialStaff != nil && !hasStaff {
		initStaff, err := user.CreateInitialStaff(*config.InitialStaff)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create initial staff user", "error", err)
			fmt.Fprintf(os.Stderr, "Failed to create initial staff user: %v\n", err)
			os.Exit(1)
		}
		if err := repos.Staff.SaveStaff(ctx, initStaff); err != nil {
			slog.ErrorContext(ctx, "Failed to save initial staff user", "error", err)
			fmt.Fprintf(os.Stderr, "Failed to save initial staff user: %v\n", err)
			os.Exit(1)
		}

		slog.InfoContext(ctx, "Initial staff user created", "email", config.InitialStaff.Email)
	} else {
		slog.InfoContext(ctx, "Skipping initial staff user creation", "hasStaff", hasStaff, "initialStaffConfigured", config.InitialStaff != nil)
	}
	// Set up HTTP server
	httpServer := setupHTTPServer(config, apps)

	// Start server
	go func() {
		slog.InfoContext(ctx, "Starting HTTP server", "port", config.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "HTTP server error", "error", err)
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	slog.InfoContext(ctx, "Server started in", "duration", time.Since(startTime).String())
	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.InfoContext(ctx, "Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(shutdownCtx, "Server forced to shutdown", "error", err)
		fmt.Fprintf(os.Stderr, "Server forced to shutdown: %v\n", err)
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Server exited")
}

func loadConfig() *Config {
	mode := env.Mode(getEnvOrDefault("MODE", string(env.Dev)))
	port := getEnvOrDefault("PORT", "8080")
	pgdsn := getEnvOrDefault("PG_DSN", "postgres://user:password@localhost:8765/ucms?sslmode=disable")
	logPath := getEnvOrDefault("LOG_PATH", "")
	accessTokenSecretKey := getEnvOrDefault("ACCESS_TOKEN_SECRET", "default_access_secret")
	refreshTokenSecretKey := getEnvOrDefault("REFRESH_TOKEN_SECRET", "default_refresh_secret")
	staffInvitationBaseURL := getEnvOrDefault("STAFF_INVITATION_BASE_URL", "http://localhost:3000/invitations/accept")
	acceptInvitationPageURL := getEnvOrDefault("STAFF_INVITATION_PAGE_URL", "http://localhost:3000/invitations/accept")
	invitationTokenSecretKey := getEnvOrDefault("INVITATION_TOKEN_SECRET", "default_invitation_secret")
	var service ServiceConfig
	service.Namespace = getEnvOrDefault("SERVICE_NAMESPACE", "ucms")
	service.Name = getEnvOrDefault("SERVICE_NAME", "ucms-api")
	service.Version = getEnvOrDefault("SERVICE_VERSION", "0.1.0")
	service.InstanceId = getEnvOrDefault("SERVICE_INSTANCE_ID", "instance-1")
	var s3 S3Config
	s3.Endpoint = getEnvOrDefault("S3_ENDPOINT", "http://localhost:9000")
	s3.AccessKey = getEnvOrDefault("S3_ACCESS_KEY", "minioadmin")
	s3.SecretKey = getEnvOrDefault("S3_SECRET_KEY", "minioadmin")
	s3.Bucket = getEnvOrDefault("S3_BUCKET", "ucms-avatars")
	s3.Region = getEnvOrDefault("S3_REGION", "us-east-1")
	s3.BaseURL = getEnvOrDefault("S3_BASE_URL", "http://localhost:9000/ucms-avatars")
	s3.UsePathStyle = getEnvOrDefault("S3_USE_PATH_STYLE", "true") == "true"

	var initialStaff *user.CreateInitialStaffArgs
	if os.Getenv("INITIAL_STAFF_EMAIL") != "" {
		initialStaff = &user.CreateInitialStaffArgs{
			Username:  getEnvOrDefault("INITIAL_STAFF_USERNAME", "admin"),
			Email:     os.Getenv("INITIAL_STAFF_EMAIL"),
			Password:  getEnvOrDefault("INITIAL_STAFF_PASSWORD", "StrongP@ssw0rd"),
			Barcode:   user.Barcode(getEnvOrDefault("INITIAL_STAFF_BARCODE", "000000")),
			FirstName: getEnvOrDefault("INITIAL_STAFF_FIRST_NAME", "Admin"),
			LastName:  getEnvOrDefault("INITIAL_STAFF_LAST_NAME", "User"),
		}
	}

	return &Config{
		Mode:                     mode,
		Service:                  service,
		S3:                       s3,
		Port:                     port,
		PgDSN:                    pgdsn,
		LogPath:                  logPath,
		InitialStaff:             initialStaff,
		AccessTokenSecretKey:     accessTokenSecretKey,
		RefreshTokenSecretKey:    refreshTokenSecretKey,
		StaffInvitationBaseURL:   staffInvitationBaseURL,
		AccestInvitationPageURL:  acceptInvitationPageURL,
		InvitationTokenSecretKey: invitationTokenSecretKey,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupDatabase(ctx context.Context, config *Config) (*pgxpool.Pool, error) {
	// Create connection pool
	pool, err := pgpkg.NewPgxPool(ctx, config.PgDSN, config.Mode)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	migrateDSN := strings.Replace(config.PgDSN, "postgres://", "pgx://", 1)

	// Run migrations
	if err := pgpkg.Migrate(migrateDSN, &ucmsv2.Migrations); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return pool, nil
}

type Repositories struct {
	PgxPool         *pgxpool.Pool
	User            *postgres.UserRepo
	Registration    *postgres.RegistrationRepo
	Student         *postgres.StudentRepo
	Staff           *postgres.StaffRepo
	StaffInvitation *postgres.StaffInvitationRepo
	Group           *postgres.GroupRepo
}

func setupRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		PgxPool:         pool,
		User:            postgres.NewUserRepo(pool, nil, nil),
		Registration:    postgres.NewRegistrationRepo(pool, nil, nil),
		Student:         postgres.NewStudentRepo(pool, nil, nil),
		Staff:           postgres.NewStaffRepo(pool, nil, nil),
		StaffInvitation: postgres.NewStaffInvitationRepo(pool, nil, nil),
		Group:           postgres.NewGroupRepo(pool, nil, nil),
	}
}

type Infrastructure struct {
	S3Client *s3.Client
}

func setupInfrastructure(ctx context.Context, config *Config) *Infrastructure {
	s3Storage, err := s3.NewClient(ctx, config.S3.Endpoint, config.S3.AccessKey, config.S3.SecretKey, config.S3.Bucket, config.S3.Region)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to set up S3 storage", "error", err)
		fmt.Fprintf(os.Stderr, "Failed to set up S3 storage: %v\n", err)
		os.Exit(1)
	}

	return &Infrastructure{
		S3Client: s3Storage,
	}
}

func setupEventProcessing(ctx context.Context, pool *pgxpool.Pool, wlogger watermill.LoggerAdapter) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, wlogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermill router: %w", err)
	}

	if err := watermillx.InitializeEventSchema(ctx, pool, wlogger); err != nil {
		return nil, fmt.Errorf("failed to initialize event schema: %w", err)
	}

	slog.InfoContext(ctx, "Event processing setup completed")
	return router, nil
}

func setupApplications(config *Config, repos *Repositories, infrastructure *Infrastructure) *Application {
	mailSender := mocks.NewMockMailSender()

	regApp := registration.NewApp(registration.Args{
		Mode:         config.Mode,
		Repo:         repos.Registration,
		UserGetter:   repos.User,
		GroupGetter:  repos.Group,
		StudentSaver: repos.Student,
		PgxPool:      repos.PgxPool,
	})

	mailApp := mail.NewApp(mail.Args{
		Mailsender:              mailSender,
		StaffInvitationBaseURL:  config.StaffInvitationBaseURL,
		InvitationCreatorGetter: repos.Staff,
	})

	studentApp := studentapp.NewApp(studentapp.Args{
		PgxPool: repos.PgxPool,
	})

	staffApp := staffapp.NewApp(staffapp.Args{
		StaffInvitationRepo: repos.StaffInvitation,
		StaffRepo:           repos.Staff,
	})

	authApp := authapp.NewApp(authapp.Args{
		UserGetter:              repos.User,
		AccessTokenSecretKey:    config.AccessTokenSecretKey,
		RefreshTokenSecretKey:   config.RefreshTokenSecretKey,
		AccessTokenlExpDuration: nil,
		RefreshTokenExpDuration: nil,
	})

	userApp := userapp.NewApp(userapp.Args{
		S3BaseURL:     config.S3.BaseURL,
		AvatarStorage: infrastructure.S3Client,
		UserRepo:      repos.User,
	})

	return &Application{
		Registration: regApp,
		Mail:         mailApp,
		Student:      studentApp,
		Staff:        staffApp,
		Auth:         authApp,
		User:         userApp,
	}
}

func setupHTTPServer(config *Config, apps *Application) *http.Server {
	router := chi.NewRouter()

	if config.Mode == env.Dev {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				origin := r.Header.Get("Origin")

				allowedOrigins := map[string]bool{
					"http://localhost:3000": true,
					"http://localhost:5173": true, // Vite default
					"http://127.0.0.1:3000": true,
					"http://127.0.0.1:5173": true,
					"*":                     true, // Allow all origins in development
					"null":                  true, // Handle null origin for file:// requests
				}

				if allowedOrigins[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if origin == "" {
					// For same-origin requests or when opening HTML file directly
					w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
				}

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, Accept-Language")
				w.Header().Set("Access-Control-Allow-Credentials", "true") // ← THIS IS CRUCIAL!

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}

				next.ServeHTTP(w, r)
			})
		})
	}

	// Set up HTTP ports
	httpPort := httpport.NewPort(httpport.Args{
		ServiceName:             config.Service.Name,
		RegistrationApp:         apps.Registration,
		AuthApp:                 apps.Auth,
		StudentApp:              apps.Student,
		StaffApp:                apps.Staff,
		UserApp:                 apps.User,
		Secret:                  []byte(config.AccessTokenSecretKey),
		CookieDomain:            "",
		AcceptInvitationPageURL: config.AccestInvitationPageURL,
		InvitationTokenAlg:      jwt.SigningMethodHS256,
		InvitationTokenKey:      config.InvitationTokenSecretKey,
		InvitationTokenExp:      15 * time.Minute,
	})

	httpPort.Route(router)

	return &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOTelSDK(ctx context.Context, config *Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	appResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.DeploymentEnvironmentName(config.Mode.String()),
		semconv.ServiceNamespaceKey.String(config.Service.Namespace),
		semconv.ServiceNameKey.String(config.Service.Name),
		semconv.ServiceVersionKey.String(config.Service.Version),
		semconv.ServiceInstanceIDKey.String(config.Service.InstanceId),
	)

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	tracerProvider, err := NewTracerProvider(ctx, appResource)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(ctx, appResource)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	loggerProvider, err := newLoggerProvider(ctx, appResource)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	h := otelslog.NewHandler(
		config.Service.Name,
		otelslog.WithLoggerProvider(loggerProvider),
		otelslog.WithSource(true),
	)
	logger := slog.New(h)
	slog.SetDefault(logger)

	slog.Debug("OpenTelemetry SDK setup completed")

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func NewTracerProvider(ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(traceBatchTimeout),
		),
	)
	return traceProvider, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(metricExporter,
				// Default is 1m. Set to 3s for demonstrative purposes.
				metric.WithInterval(metricPeriodicInterval),
			),
		),
	)
	return meterProvider, nil
}

func newLoggerProvider(ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)

	return loggerProvider, nil
}
