<div align="center">

# 🎯 Interview Simulation Backend Server

[![Go Version](https://img.shields.io/badge/Go-1.24.0-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12-316192?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-Latest-DC382D?style=for-the-badge&logo=redis)](https://redis.io/)

**A robust, production-ready backend server for interview simulation and evaluation platform**

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Architecture](#-architecture) • [Contributing](#-contributing)

</div>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Architecture](#-architecture)
- [Prerequisites](#-prerequisites)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [Development](#-development)
- [API Documentation](#-api-documentation)
- [Database](#-database)
- [Testing](#-testing)
- [Deployment](#-deployment)
- [Project Structure](#-project-structure)
- [Contributing](#-contributing)
- [Troubleshooting](#-troubleshooting)
- [License](#-license)
- [Support](#-support)

---

## 🌟 Overview

The Interview Simulation Backend Server is a comprehensive, enterprise-grade solution designed to power interview simulation platforms. Built with Go, it provides a scalable, secure, and feature-rich backend infrastructure for conducting, evaluating, and managing technical interviews.

### 🎯 Key Highlights

- **Real-time Communication**: WebSocket support for live interview sessions
- **Secure Authentication**: JWT-based auth with OAuth2 (Google, Facebook)
- **Cloud-Native**: Docker-ready with multi-environment support
- **Scalable Architecture**: Clean architecture with dependency injection
- **Background Processing**: Async job queue with Redis and Asynq
- **API-First Design**: RESTful API with comprehensive Swagger documentation
- **Data Persistence**: PostgreSQL with automated migrations
- **Resume Management**: AWS S3 integration for file storage

---

## ✨ Features

### 🔐 Authentication & Authorization
- JWT-based authentication with access and refresh tokens
- OAuth2 integration (Google, Facebook)
- Secure session management with Redis
- Cookie-based authentication support
- Token refresh and grace period handling

### 👤 User Management
- User registration and profile management
- Email-based user operations
- Secure password hashing with bcrypt
- User role and permission system

### 🎤 Interview Sessions
- Real-time interview sessions via WebSocket
- Interview state management
- Turn-based conversation tracking
- Session recording and playback
- AI-powered interview agent integration

### 📊 Evaluation System
- Multi-criteria evaluation framework
- Rubric-based scoring system
- Phrase-level evaluation
- Automated scoring and feedback
- Performance analytics

### 📝 Resume Management
- Resume upload and storage (AWS S3)
- Pre-signed URL generation for secure access
- Resume parsing and metadata extraction
- Version control and history tracking

### 🐛 Issue Reporting
- Comprehensive issue tracking system
- Categorized issue management
- User feedback and review comments
- Issue resolution workflow

### 🔄 Background Jobs
- Async task processing with Asynq
- Queue management with Redis
- Job retry and failure handling
- Scheduled tasks and cron jobs

### 📈 Monitoring & Logging
- Structured logging with Zap
- Error tracking and reporting
- Custom app-level error codes
- HTTP request/response logging

---

## 🏗️ Architecture

### Technology Stack

```
┌─────────────────────────────────────────────────────────────┐
│                     Client Applications                      │
│            (Web, Mobile, Desktop Applications)               │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway Layer                       │
│         (Gin Framework, CORS, Middleware, Auth)              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      Handler Layer                           │
│    (User, Interview, Evaluation, Resume, IssueReports)       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│         (Business Logic, Validation, Orchestration)          │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│              (Data Access, SQLC, Transactions)               │
└──────────────────────────┬──────────────────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
┌────────────────┐  ┌────────────┐  ┌──────────────┐
│   PostgreSQL   │  │   Redis    │  │   AWS S3     │
│   (Primary DB) │  │  (Cache)   │  │  (Storage)   │
└────────────────┘  └────────────┘  └──────────────┘
```

### Design Patterns

- **Clean Architecture**: Separation of concerns with clear boundaries
- **Dependency Injection**: Using uber/dig for IoC container
- **Repository Pattern**: Data access abstraction
- **Service Layer Pattern**: Business logic encapsulation
- **Middleware Pattern**: Request/response processing pipeline
- **Factory Pattern**: Container-based object creation

### Core Components

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **HTTP Framework** | Gin | REST API routing and middleware |
| **Database** | PostgreSQL 12+ | Primary data store |
| **Cache** | Redis | Session storage, caching, queue |
| **ORM/Query Builder** | SQLC | Type-safe SQL generation |
| **Job Queue** | Asynq | Background job processing |
| **WebSocket** | Gorilla WebSocket | Real-time communication |
| **Authentication** | JWT, OAuth2 | User authentication |
| **Cloud Storage** | AWS S3 | File storage and retrieval |
| **Logging** | Zap | Structured logging |
| **Configuration** | Viper | Config management |
| **DI Container** | Dig | Dependency injection |

---

## 📦 Prerequisites

Before you begin, ensure you have the following installed:

- **Go**: 1.24.0 or higher ([Download](https://golang.org/dl/))
- **Docker**: 20.10+ ([Download](https://www.docker.com/))
- **Docker Compose**: 2.0+ (included with Docker Desktop)
- **Make**: Build automation tool
- **Git**: Version control

### Optional Tools

- **SQLC**: For SQL code generation ([Install](https://docs.sqlc.dev/en/latest/overview/install.html))
- **Mockery**: For generating mocks ([Install](https://github.com/vektra/mockery))
- **Swagger**: For API documentation ([Install](https://github.com/swaggo/swag))
- **migrate**: For database migrations ([Install](https://github.com/golang-migrate/migrate))

---

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://gitlab.com/interview-simulation/interview-backend-server.git
cd interview-backend-server
```

### 2. Create Docker Network

```bash
docker network create interview-network
```

### 3. Start Development Environment

```bash
# Build and start all services
make run-dev

# Or manually with docker compose
docker compose -f compose.dev.yml up
```

The server will be available at `http://localhost:8080`

### 4. Verify Installation

```bash
# Check if all services are running
make info

# Expected output:
# NAME                       IMAGE                                    STATUS
# interview-backend-server   interview-backend-server                 Up
# interview-db               postgres:12-alpine                       Up
# interview-redis            redis:alpine                             Up
```

### 5. Access API Documentation

Open your browser and navigate to:
- **Swagger UI**: `http://localhost:8080/swagger/index.html`

---

## ⚙️ Configuration

### Environment Configuration

The application supports multiple environments:

- **Development**: `config/config.dev.yaml`
- **Production**: `config/config.prod.yaml`

### Configuration Structure

```yaml
AppConfig:
  APP_PORT: 8080              # Server port
  APP_ENV: dev                # Environment (dev/prod)
  API_HOST: localhost:8080    # API host
  CORS_ORIGINS:               # Allowed CORS origins
    - http://localhost:5173
    - http://localhost:3000

DatabaseConfig:
  POSTGRES_HOST: postgres
  POSTGRES_PORT: 5432
  POSTGRES_USER: user
  POSTGRES_PASSWORD: password
  POSTGRES_DB: interview

RedisConfig:
  REDIS_HOST: redis_server
  REDIS_PORT: 6379
  REDIS_PASSWORD: password
  REDIS_DB_TEMP: 0
  REDIS_DB_QUEUE: 1

AuthConfig:
  ENCRYPTION_SECRET_KEY: your-secret-key
  ACCESS_TOKEN_DURATION: 15m
  REFRESH_TOKEN_DURATION: 7d
  TOKEN_GRACE_WINDOW: 5m
  COOKIE_DOMAIN: ""
  COOKIE_REJECT_HTTP: true

AWSConfig:
  AWS_REGION: ap-southeast-1
  AWS_S3_BUCKET: your-bucket-name
  AWS_S3_ACCESS_KEY: your-access-key
  AWS_S3_SECRET_ACCESS_KEY: your-secret-key
  AWS_S3_PRESIGNED_URL_EXPIRY: 15m

InterviewSessionConfig:
  INTERVIEW_AGENT_URL: http://agent-server:8000
  INTERVIEW_WEBSOCKET_PATH: ws://agent-server:8000/api/v1/ws/connect
  INTERVIEW_SESSION_TOKEN_TTL: 24h
  INTERVIEW_SESSION_DURATION: 1h
  ENCRYPTION_SECRET_KEY: your-encryption-key

OAuthConfig:
  OAUTH_GOOGLE_CLIENT_ID: your-google-client-id
  OAUTH_GOOGLE_CLIENT_SECRET: your-google-secret
  OAUTH_GOOGLE_REDIRECT_URI: http://localhost:5173/auth/google/callback
  OAUTH_FACEBOOK_CLIENT_ID: your-facebook-client-id
  OAUTH_FACEBOOK_CLIENT_SECRET: your-facebook-secret
  OAUTH_FACEBOOK_REDIRECT_URI: http://localhost:5173/auth/facebook/callback
```

### Environment Variables

You can override configuration using environment variables:

```bash
export ENV=prod
export APP_PORT=8080
export DB_SOURCE="postgresql://user:password@localhost:5432/interview?sslmode=disable"
```

---

## 💻 Development

### Project Setup

```bash
# Install dependencies
go mod download

# Generate SQL code
make sqlc

# Generate mocks
make mock-gen

# Run tests
make test

# Generate Swagger documentation
make swagger-gen
```

### Running Locally (Without Docker)

```bash
# 1. Start PostgreSQL and Redis
# (Use your preferred method or Docker)

# 2. Run database migrations
migrate -path ./internal/db/migration -database "postgresql://user:password@localhost:5432/interview?sslmode=disable" up

# 3. Set environment
export ENV=dev

# 4. Run the server
go run cmd/server/main.go
```

### Available Make Commands

```bash
# Development
make run-dev          # Start development environment
make build-dev        # Build development images
make clean-dev        # Clean development environment
make rebuild-dev      # Rebuild and restart development

# Production
make run-prod         # Start production environment
make build-prod       # Build production images
make clean-prod       # Clean production environment
make rebuild-prod     # Rebuild and restart production

# Database
make migrate-up-dev   # Run migrations (dev)
make migrate-down-dev # Rollback migrations (dev)
make sqlc             # Generate SQLC code

# Testing & Tools
make test             # Run all tests
make mock-gen         # Generate mocks
make clean-mock       # Remove generated mocks
make swagger-gen      # Generate Swagger docs

# Info
make info             # Show running containers
```

### Code Generation

#### Generate SQL Code (SQLC)

```bash
# Edit queries in internal/db/query/*.sql
# Then run:
make sqlc
```

#### Generate Mocks (Mockery)

```bash
# Generate mocks for all interfaces
make mock-gen

# Clean existing mocks
make clean-mock
```

#### Generate API Documentation (Swagger)

```bash
# Update annotations in handler files
# Then run:
make swagger-gen
```

---

## 📚 API Documentation

### API Endpoints

#### Authentication

```http
POST   /api/v1/auth/register          # Register new user
POST   /api/v1/auth/login             # Login with credentials
POST   /api/v1/auth/refresh           # Refresh access token
POST   /api/v1/auth/logout            # Logout user
GET    /api/v1/auth/google            # Google OAuth
GET    /api/v1/auth/facebook          # Facebook OAuth
```

#### Users

```http
GET    /api/v1/users/me               # Get current user
PUT    /api/v1/users/me               # Update user profile
DELETE /api/v1/users/me               # Delete user account
```

#### Interview Sessions

```http
POST   /api/v1/interviews             # Create interview session
GET    /api/v1/interviews/:id         # Get interview session
GET    /api/v1/interviews             # List user interviews
PUT    /api/v1/interviews/:id         # Update interview session
DELETE /api/v1/interviews/:id         # Delete interview session
WS     /api/v1/ws/interviews/:id      # WebSocket connection
```

#### Evaluations

```http
POST   /api/v1/evaluations            # Create evaluation
GET    /api/v1/evaluations/:id        # Get evaluation
GET    /api/v1/evaluations            # List evaluations
PUT    /api/v1/evaluations/:id        # Update evaluation
GET    /api/v1/evaluations/:id/scores # Get evaluation scores
```

#### Resumes

```http
POST   /api/v1/resumes                # Upload resume
GET    /api/v1/resumes/:id            # Get resume
GET    /api/v1/resumes                # List user resumes
DELETE /api/v1/resumes/:id            # Delete resume
GET    /api/v1/resumes/:id/download   # Get download URL
```

#### Issue Reports

```http
POST   /api/v1/issues                 # Report issue
GET    /api/v1/issues/:id             # Get issue
GET    /api/v1/issues                 # List issues
PUT    /api/v1/issues/:id             # Update issue status
GET    /api/v1/issues/categories      # Get issue categories
```

### Authentication

All protected endpoints require a JWT token in the Authorization header:

```http
Authorization: Bearer <your_jwt_token>
```

### Example API Usage

#### Register a New User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "full_name": "John Doe"
  }'
```

#### Create Interview Session

```bash
curl -X POST http://localhost:8080/api/v1/interviews \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "resume_id": "uuid-here",
    "session_type": "technical",
    "difficulty": "medium"
  }'
```

---

## 🗄️ Database

### Schema Overview

The database consists of the following main entities:

- **users**: User accounts and profiles
- **auth_sessions**: Active authentication sessions
- **interview_sessions**: Interview session metadata
- **interview_state**: Current state of interviews
- **interview_turns**: Conversation turns in interviews
- **evaluations**: Interview evaluations
- **evaluation_criteria**: Evaluation criteria definitions
- **evaluation_rubrics**: Rubric templates
- **evaluation_scores**: Individual scores
- **phrase_evaluations**: Phrase-level evaluations
- **phrase_rubric_scores**: Phrase-level scores
- **resumes**: Resume metadata
- **issue_reports**: User-reported issues
- **issue_categories**: Issue classification
- **review_comments**: User feedback and reviews
- **user_turn_improvements**: Improvement suggestions

### Migrations

Migrations are located in `internal/db/migration/` and are automatically run on startup.

#### Manual Migration Commands

```bash
# Run all pending migrations
make migrate-up-dev

# Rollback the last migration
make migrate-down-dev

# Create a new migration
migrate create -ext sql -dir internal/db/migration -seq migration_name
```

### Database Queries

SQL queries are managed using SQLC. Query definitions are in `internal/db/query/` and generate type-safe Go code.

#### Add a New Query

1. Create or edit a `.sql` file in `internal/db/query/`
2. Write your SQL query with SQLC annotations
3. Run `make sqlc` to generate Go code

Example query:

```sql
-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: CreateUser :one
INSERT INTO users (
  email, password_hash, full_name
) VALUES (
  $1, $2, $3
)
RETURNING *;
```

---

## 🧪 Testing

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
go test -v -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Tests

```bash
# Test a specific package
go test -v ./internal/services

# Test a specific function
go test -v -run TestUserService_CreateUser ./internal/services
```

### Testing Strategy

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Repository Tests**: Test database operations with mocks
- **Handler Tests**: Test HTTP endpoints with mock services
- **Service Tests**: Test business logic with mock repositories

### Writing Tests

Example test structure:

```go
func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewUserRepository(t)
    service := NewUserService(mockRepo)
    
    // Act
    user, err := service.CreateUser(ctx, req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

---

## 🚢 Deployment

### Production Deployment

#### Using Docker Compose

```bash
# Build production images
make build-prod

# Start production environment
make run-prod
```

#### Using Docker

```bash
# Build image
docker build -f Dockerfile.prod -t interview-server:latest .

# Run container
docker run -d \
  --name interview-server \
  -p 8080:8080 \
  -e ENV=prod \
  -e DB_SOURCE="postgresql://user:password@postgres:5432/interview?sslmode=disable" \
  interview-server:latest
```

### Environment-Specific Configurations

#### Development
- Hot reload enabled
- Verbose logging
- Debug mode
- Local file storage

#### Production
- Optimized builds
- Error logging only
- Production mode
- Cloud storage (S3)
- Rate limiting enabled
- HTTPS only

### Health Checks

```bash
# Check server health
curl http://localhost:8080/health

# Check database connection
curl http://localhost:8080/health/db

# Check Redis connection
curl http://localhost:8080/health/redis
```

### Monitoring

Consider integrating:
- **Prometheus**: Metrics collection
- **Grafana**: Metrics visualization
- **Sentry**: Error tracking
- **New Relic**: APM
- **ELK Stack**: Log aggregation

---

## 📁 Project Structure

```
interview-server/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── config/
│   ├── config.dev.yaml             # Development config
│   └── config.prod.yaml            # Production config
├── docs/
│   ├── swagger.html                # Swagger UI
│   └── swagger.json                # OpenAPI spec
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration loader
│   ├── constants/                  # Application constants
│   ├── containers/                 # DI containers
│   ├── db/
│   │   ├── migration/              # Database migrations
│   │   ├── query/                  # SQL queries
│   │   └── sqlc/                   # Generated SQLC code
│   ├── entities/                   # Domain entities
│   ├── handlers/                   # HTTP handlers
│   ├── infras/
│   │   ├── app_error/              # Error handling
│   │   ├── auth/                   # OAuth clients
│   │   ├── aws/                    # AWS services
│   │   ├── database/               # Database clients
│   │   ├── http/                   # HTTP client
│   │   ├── log/                    # Logger
│   │   ├── queue/                  # Job queue
│   │   ├── routes/                 # Route definitions
│   │   ├── server/                 # HTTP server
│   │   └── websocket/              # WebSocket manager
│   ├── middleware/                 # HTTP middleware
│   ├── mocks/                      # Generated mocks
│   ├── repositories/               # Data access layer
│   ├── services/                   # Business logic
│   └── utils/                      # Utility functions
├── compose.dev.yml                 # Dev Docker Compose
├── compose.prod.yml                # Prod Docker Compose
├── Dockerfile.dev                  # Dev Dockerfile
├── Dockerfile.prod                 # Prod Dockerfile
├── go.mod                          # Go dependencies
├── go.sum                          # Go dependency checksums
├── Makefile                        # Build automation
├── migration.sh                    # Migration script
├── sqlc.yaml                       # SQLC configuration
└── README.md                       # This file
```

### Key Directories

| Directory | Description |
|-----------|-------------|
| `cmd/` | Application entry points |
| `internal/` | Private application code |
| `internal/handlers/` | HTTP request handlers |
| `internal/services/` | Business logic |
| `internal/repositories/` | Data access layer |
| `internal/entities/` | Domain models |
| `internal/infras/` | Infrastructure code |
| `internal/db/` | Database-related code |
| `config/` | Configuration files |
| `docs/` | API documentation |

---

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

### Getting Started

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Write/update tests
5. Ensure all tests pass (`make test`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Coding Standards

- Follow Go best practices and idioms
- Use `gofmt` for code formatting
- Write clear, descriptive commit messages
- Add comments for complex logic
- Update documentation for API changes
- Write unit tests for new features
- Ensure test coverage remains high

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat(auth): add Facebook OAuth integration

Implemented Facebook OAuth2 authentication flow with token
exchange and user profile retrieval.

Closes #123
```

### Pull Request Process

1. Update README.md with details of changes if needed
2. Update API documentation if endpoints change
3. Add tests for new functionality
4. Ensure CI/CD pipeline passes
5. Request review from maintainers
6. Address review comments
7. Squash commits before merging

---

## 🔧 Troubleshooting

### Common Issues

#### Docker Network Error

```bash
# Error: network interview-network not found
# Solution: Create the network
docker network create interview-network
```

#### Port Already in Use

```bash
# Error: port 8080 is already allocated
# Solution: Change port in config or stop the conflicting service
lsof -ti:8080 | xargs kill -9
```

#### Database Connection Failed

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check database logs
docker logs interview-db

# Verify connection string
psql "postgresql://user:password@localhost:5432/interview"
```

#### Migration Errors

```bash
# Force migration version
migrate -path ./internal/db/migration -database "your-db-url" force <version>

# Drop database and recreate (CAUTION: Data loss!)
make clean-dev
make run-dev
```

#### Redis Connection Issues

```bash
# Check if Redis is running
docker ps | grep redis

# Test Redis connection
redis-cli -h localhost -p 6379 ping
```

### Debug Mode

Enable debug logging:

```yaml
# config/config.dev.yaml
AppConfig:
  LOG_LEVEL: debug
```

### Getting Help

- Check [existing issues](https://gitlab.com/interview-simulation/interview-backend-server/-/issues)
- Create a [new issue](https://gitlab.com/interview-simulation/interview-backend-server/-/issues/new)
- Join our [Discord community](#)
- Email: support@interview-simulation.com

---

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

```
Copyright 2025 Interview Simulation Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

---

## 💬 Support

### Contact

- **Email**: support@interview-simulation.com
- **Website**: https://interview-simulation.com
- **GitLab Issues**: [Create an issue](https://gitlab.com/interview-simulation/interview-backend-server/-/issues)
- **Documentation**: [Full docs](https://docs.interview-simulation.com)

### Community

- **Discord**: [Join our community](#)
- **Stack Overflow**: Tag `interview-simulation`
- **Twitter**: [@interview_sim](#)

### Professional Support

For enterprise support, SLA agreements, and custom development:
- Email: enterprise@interview-simulation.com
- Schedule a call: [Calendly link](#)

---

## 🙏 Acknowledgments

Special thanks to:

- The Go community for excellent libraries and tools
- All contributors who have helped improve this project
- Our users for valuable feedback and suggestions

### Built With

- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [SQLC](https://sqlc.dev/) - Type-safe SQL generation
- [Asynq](https://github.com/hibiken/asynq) - Background job processing
- [Zap](https://github.com/uber-go/zap) - Structured logging
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Dig](https://github.com/uber-go/dig) - Dependency injection
- [JWT-Go](https://github.com/dgrijalva/jwt-go) - JWT implementation
- [AWS SDK](https://aws.amazon.com/sdk-for-go/) - AWS integration
- [Redis](https://redis.io/) - Caching and queue
- [PostgreSQL](https://www.postgresql.org/) - Database

---

<div align="center">

**[⬆ Back to Top](#-interview-simulation-backend-server)**

Made with ❤️ by the Interview Simulation Team

[![GitLab](https://img.shields.io/badge/GitLab-330F63?style=for-the-badge&logo=gitlab&logoColor=white)](https://gitlab.com/interview-simulation/interview-backend-server)
[![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)

</div>
