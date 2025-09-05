# UCMS v2

**University Clubs Management System** - A modern Go backend for university clubs management.

[![Go Version](https://img.shields.io/badge/go-1.24.0-blue.svg)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/postgresql-16.8-blue.svg)](https://postgresql.org)

## Features

- 🎓 **Student Registration** - Email-verified workflow with group assignment
- 👥 **Staff Management** - Invitation-based onboarding system  
- 🔐 **JWT Authentication** - Secure token-based auth with refresh tokens
- 🌐 **Multi-language** - English, Kazakh, Russian support
- ⚡ **Event-Driven** - CQRS architecture with async processing
- 📧 **Email Integration** - Automated notifications and verification

## Quick Start

```bash
# Clone repository
git clone <repo-url>
cd ucms-v2

# Create environment file
cp .env.example .env.local
# Edit .env.local with your settings

# Start PostgreSQL
task docker:local

# Run the API
task http:local
```

**Test it works:**
```bash
curl http://localhost:8080/health
```

🎉 **API running at http://localhost:8080**

## Tech Stack

- **Language:** Go 1.24.0
- **Database:** PostgreSQL 16.8  
- **Architecture:** DDD + CQRS + Event Sourcing
- **Message Broker:** Watermill
- **HTTP Router:** Chi with OpenAPI
- **Testing:** Testcontainers + Integration tests

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/registrations/students/start` | POST | Start student registration |
| `/v1/registrations/verify` | POST | Verify email with code |
| `/v1/registrations/students/complete` | POST | Complete registration |
| `/v1/auth/login` | POST | User authentication |
| `/v1/students/me` | GET | Get student information |

## Documentation

📚 **[Full Documentation](docs/README.md)**

**Quick Links:**
- [Getting Started](docs/quick-start.md) - Setup in 5 minutes
- [First API Call](docs/first-api-call.md) - Complete registration flow
- [Architecture](docs/architecture.md) - System design and patterns
- [API Reference](docs/api.md) - Detailed endpoint docs

## Development

**Prerequisites:**
- Go 1.24.0+
- Docker
- Task runner: `go install github.com/go-task/task/v3/cmd/task@latest`

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

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Make changes and add tests
4. Run tests: `task test`
5. Commit changes: `git commit -m 'Add amazing feature'`
6. Push branch: `git push origin feature/amazing-feature`
7. Create Pull Request

**Development workflow:**
- Read [Development Guide](docs/development.md)
- Follow [Architecture patterns](docs/architecture.md)
- Write tests for new features

## Environment Variables

Create `.env.local` from `.env.example`:

```bash
MODE=local
PORT=8080
PG_DSN=postgres://user:password@localhost:8765/ucms?sslmode=disable

# Optional: Initial admin user
INITIAL_STAFF_EMAIL=admin@example.com
INITIAL_STAFF_PASSWORD=StrongP@ssw0rd
```

See [Configuration Guide](docs/configuration.md) for all options.
