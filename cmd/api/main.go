package main

import (
	"context"
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
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	ucmsv2 "github.com/ARUMANDESU/ucms"
	"github.com/ARUMANDESU/ucms/internal/adapters/repos/postgres"
	authapp "github.com/ARUMANDESU/ucms/internal/application/auth"
	"github.com/ARUMANDESU/ucms/internal/application/mail"
	"github.com/ARUMANDESU/ucms/internal/application/registration"
	studentapp "github.com/ARUMANDESU/ucms/internal/application/student"
	httpport "github.com/ARUMANDESU/ucms/internal/ports/http"
	watermillport "github.com/ARUMANDESU/ucms/internal/ports/watermill"
	"github.com/ARUMANDESU/ucms/pkg/env"
	"github.com/ARUMANDESU/ucms/pkg/logging"
	pgpkg "github.com/ARUMANDESU/ucms/pkg/postgres"
	"github.com/ARUMANDESU/ucms/pkg/watermillx"
	"github.com/ARUMANDESU/ucms/tests/mocks"
)

// Application holds all the application dependencies
type Application struct {
	Registration *registration.App
	Mail         *mail.App
	Student      *studentapp.App
	Auth         *authapp.App
}

// Config holds all configuration for the application
type Config struct {
	Mode    env.Mode
	Port    string
	PgDSN   string
	LogPath string
}

func main() {
	ctx := context.Background()

	// Load configuration from environment
	config := loadConfig()

	// Set up logging
	setupLogging(config.LogPath, config.Mode)

	// Set up tracing (in production you'd configure a proper tracing provider)
	setupTracing()

	slog.InfoContext(ctx, "Starting UCMS API server",
		"mode", config.Mode,
		"port", config.Port,
	)

	// Set up database
	pool, err := setupDatabase(ctx, config)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Set up repositories
	repos := setupRepositories(pool)

	// Set up event processing
	eventRouter, err := setupEventProcessing(ctx, pool)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to setup event processing", "error", err)
		os.Exit(1)
	}

	// Set up applications
	apps := setupApplications(config, repos)

	// Set up event handlers
	wmport, err := watermillport.NewPort(eventRouter, pool, watermill.NewSlogLogger(slog.Default()))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create Watermill port", "error", err)
		os.Exit(1)
	}
	if err := wmport.Run(ctx, watermillport.AppEventHandlers{
		Registration: apps.Registration.Event,
		Mail:         apps.Mail.Event,
		Student:      apps.Student.Event,
	}); err != nil {
		slog.ErrorContext(ctx, "Failed to run Watermill port", "error", err)
		os.Exit(1)
	}

	go func() {
		// Start event router
		if err := eventRouter.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to start event router", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := eventRouter.Close(); err != nil {
				slog.ErrorContext(ctx, "Failed to close event router", "error", err)
			}
		}()
	}()

	// Set up HTTP server
	httpServer := setupHTTPServer(config, apps)

	// Start server
	go func() {
		slog.InfoContext(ctx, "Starting HTTP server", "port", config.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

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
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Server exited")
}

func loadConfig() *Config {
	mode := env.Mode(getEnvOrDefault("MODE", string(env.Dev)))
	port := getEnvOrDefault("PORT", "8080")
	pgdsn := getEnvOrDefault("PG_DSN", "postgres://user:password@localhost:8765/ucms?sslmode=disable")
	logPath := getEnvOrDefault("LOG_PATH", "")

	return &Config{
		Mode:    mode,
		Port:    port,
		PgDSN:   pgdsn,
		LogPath: logPath,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupLogging(_ string, mode env.Mode) {
	// Use the existing logging setup from pkg/logging
	logger, cleanup := logging.Setup(mode)
	slog.SetDefault(logger)

	// Store cleanup function for later use if needed
	_ = cleanup
}

func setupTracing() {
	// In production, you would set up a proper tracing provider (Jaeger, etc.)
	// For now, we use the default no-op tracer
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
	PgxPool      *pgxpool.Pool
	User         *postgres.UserRepo
	Registration *postgres.RegistrationRepo
	Student      *postgres.StudentRepo
	Group        *postgres.GroupRepo
}

func setupRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		PgxPool:      pool,
		User:         postgres.NewUserRepo(pool, nil, nil),
		Registration: postgres.NewRegistrationRepo(pool, nil, nil),
		Student:      postgres.NewStudentRepo(pool, nil, nil),
		Group:        postgres.NewGroupRepo(pool, nil, nil),
	}
}

func setupEventProcessing(ctx context.Context, pool *pgxpool.Pool) (*message.Router, error) {
	wlogger := watermill.NewSlogLogger(slog.Default())

	// Create Watermill router
	router, err := message.NewRouter(message.RouterConfig{}, wlogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermill router: %w", err)
	}

	// Initialize event schema
	if err := watermillx.InitializeEventSchema(ctx, pool, wlogger); err != nil {
		return nil, fmt.Errorf("failed to initialize event schema: %w", err)
	}

	slog.InfoContext(ctx, "Event processing setup completed")
	return router, nil
}

func setupApplications(config *Config, repos *Repositories) *Application {
	// Mock mail sender for development
	mailSender := mocks.NewMockMailSender()

	// Registration application
	regApp := registration.NewApp(registration.Args{
		Mode:        config.Mode,
		Repo:        repos.Registration,
		UserGetter:  repos.User,
		GroupGetter: repos.Group,
	})

	// Mail application
	mailApp := mail.NewApp(mail.Args{
		Mailsender: mailSender,
	})

	// Student application
	studentApp := studentapp.NewApp(studentapp.Args{
		StudentRepo: repos.Student,
		PgxPool:     repos.PgxPool,
	})

	authApp := authapp.NewApp(authapp.Args{
		UserGetter:              repos.User,
		AccessTokenSecretKey:    "secret1",
		RefreshTokenSecretKey:   "secret2",
		AccessTokenlExpDuration: nil,
		RefreshTokenExpDuration: nil,
	})

	return &Application{
		Registration: regApp,
		Mail:         mailApp,
		Student:      studentApp,
		Auth:         authApp,
	}
}

func setupHTTPServer(config *Config, apps *Application) *http.Server {
	// Create main router
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Add CORS for development
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

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"ucms-api"}`))
	})

	// Set up HTTP ports
	httpPort := httpport.NewPort(httpport.Args{
		RegistrationCommand: &apps.Registration.CMD,
		AuthApp:             apps.Auth,
		StudentApp:          apps.Student,
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
