# UCMS v2

University Club Management System - Go backend with DDD architecture.

# Stack

- Go 1.24.0
- PostgreSQL 16.8
- Watermill (CQRS)
- Chi + OpenAPI

# Setup

## Prerequisites

- Go 1.24.0+
- Docker
- Task runner: `go install github.com/go-task/task/v3/cmd/task@latest`

## 1. Clone the repository

```bash
git clone git@gitlab.com:ucmsv2/ucms-backend.git
cd ucms-backend
```

```bash
git clone https://gitlab.com/ucmsv2/ucms-backend.git
cd ucms-backend
```

## 2. Create environment file

Create `.env.local` in the project root:

```bash
# .env.local
MODE=local
PORT=8080
PG_DSN=postgres://user:password@localhost:8765/ucms?sslmode=disable

# Optional: Create initial admin user
INITIAL_STAFF_EMAIL=admin@example.com
INITIAL_STAFF_USERNAME=admin
INITIAL_STAFF_PASSWORD=StrongP@ssw0rd
INITIAL_STAFF_BARCODE=000000
INITIAL_STAFF_FIRST_NAME=Admin
INITIAL_STAFF_LAST_NAME=User

# Staff invitation frontend base url 
STAFF_INVITATION_BASE_URL=

# JWT Configuration
ACCESS_TOKEN_SECRET=secret
REFRESH_TOKEN_SECRET=secret2
INVITATION_TOKEN_SECRET=invitation_secret

# OpenTelemetry Configuration
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE=delta
OTEL_SERVICE_NAME=ucms-api
OTEL_SERVICE_VERSION=0.1.0
OTEL_SERVICE_NAMESPACE=ucms
OTEL_SERVICE_INSTANCE_ID=instance-1

# Service Configuration (for OpenTelemetry resource attributes)
SERVICE_NAME=ucms-api
SERVICE_VERSION=0.1.0
SERVICE_NAMESPACE=ucms
SERVICE_INSTANCE_ID=instance-1

# S3/MinIO Configuration (for avatar storage for now)
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=ucmsadmin
S3_SECRET_KEY=ucmsadminpass
S3_BUCKET=ucms
S3_REGION=us-east-1
S3_BASE_URL=http://localhost:9000/ucms-avatars
S3_USE_PATH_STYLE=true
```

## 3. Run docker compose file

```bash
task infra:up
```

This starts all needed infrastructure (postgres, minio, otel-collector, grafana, loki, tempo, prometheus)

## 4. Run the application

```bash
task http:local
```

The API server will start on `http://localhost:8080`.

## 5. Test it works

```bash
# Health check
curl http://localhost:8080/health
```

If you see a response (not an error), everything is working! :tada:


## Documentation

📚 **[Full Documentation](docs/README.md)**

**Quick Links:**
- [Getting Started](docs/quick-start.md) - Setup in 5 minutes
- [First API Call](docs/first-api-call.md) - Complete registration flow
- [Architecture](docs/architecture.md) - System design and patterns
- [API Reference](docs/api.md) - Detailed endpoint docs

**Common commands:**
```bash
task http              # Start API server
task test              # Run all tests  
task lint              # Code quality checks
task openapi:http      # Generate API types
```

**Project Structure:**
```
├── cmd/api/           # Application entry point
├── internal/
│   ├── domain/        # Business logic and aggregates
│   ├── application/   # Command/query handlers
│   ├── adapters/      # Database repositories
│   └── ports/         # HTTP handlers
├── api/openapi/       # OpenAPI specifications
├── migrations/        # Database migrations
└── docs/             # Documentation
```
