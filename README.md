# NexSpaces API

**Production-Ready Multi-Tenant SaaS Platform Backend**

A secure, scalable, and maintainable Go backend for the NexSpaces template marketplace platform. Built with Clean Architecture principles, designed for enterprise multi-tenancy.

---

## 🎯 Project Overview

NexSpaces is a multi-tenant SaaS platform that provides customizable templates (CMS, CRM, E-commerce, Education, Hospitality, ERP) for different industries. This repository contains the backend API service.

### Key Features

- ✅ **Multi-Tenant Architecture**: Complete tenant isolation at database and application level
- ✅ **Clean Architecture**: Domain-driven design with clear separation of concerns
- ✅ **Security First**: JWT authentication, RBAC/ABAC, Row Level Security (RLS)
- ✅ **Production Ready**: Docker, migrations, structured logging, health checks
- ✅ **API Documentation**: Auto-generated Swagger/OpenAPI docs
- ✅ **Type Safety**: Comprehensive input validation and error handling
- ✅ **Performance**: PostgreSQL with optimized queries, Redis caching
- ✅ **Observability**: Structured logging, metrics, distributed tracing ready

---

## 📐 Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Handlers                         │
│              (Fiber, REST API, Swagger)                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                    Use Cases                             │
│         (Business Logic, Orchestration)                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                   Repositories                           │
│        (Data Access, PostgreSQL, Redis)                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Domain Entities                         │
│          (Business Models, Value Objects)                │
└─────────────────────────────────────────────────────────┘
```

### Directory Structure

```
nexpaces-api/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
│
├── internal/
│   ├── domain/                  # Business entities & rules
│   │   ├── website/
│   │   │   ├── entity.go        # Website domain model
│   │   │   └── errors.go        # Domain-specific errors
│   │   ├── tenant/
│   │   ├── user/
│   │   └── template/
│   │
│   ├── repository/              # Data access layer
│   │   └── website/
│   │       ├── repository.go    # Repository interface
│   │       └── postgres.go      # PostgreSQL implementation
│   │
│   ├── usecase/                 # Business logic
│   │   └── website/
│   │       ├── service.go       # Website service
│   │       ├── dto.go           # Data transfer objects
│   │       └── validation.go    # Business validation
│   │
│   ├── handler/                 # HTTP layer
│   │   └── website/
│   │       ├── handler.go       # HTTP handlers
│   │       └── routes.go        # Route definitions
│   │
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go              # JWT validation
│   │   ├── tenant.go            # Tenant context
│   │   └── logger.go            # Request logging
│   │
│   ├── config/                  # Configuration
│   │   └── config.go            # App configuration
│   │
│   └── app/                     # Application setup
│       ├── app.go               # Dependency injection
│       └── routes.go            # Route registration
│
├── pkg/                         # Shared packages
│   ├── database/
│   │   ├── postgres.go          # PostgreSQL connection
│   │   └── migrations.go        # Migration runner
│   ├── logger/
│   │   └── logger.go            # Structured logging
│   ├── validator/
│   │   └── validator.go         # Input validation
│   └── errors/
│       └── errors.go            # Error handling utilities
│
├── migrations/                  # Database migrations
│   ├── 001_create_tenants.up.sql
│   ├── 001_create_tenants.down.sql
│   ├── 002_create_users.up.sql
│   └── 003_create_sites.up.sql
│
├── tests/                       # Test files
│   ├── integration/
│   └── e2e/
│
├── docker/
│   ├── Dockerfile               # Multi-stage production build
│   └── Dockerfile.dev           # Development build
│
├── .env.example                 # Environment variables template
├── docker-compose.yml           # Local development stack
├── Makefile                     # Development commands
└── README.md                    # This file
```

---

## 🚀 Getting Started

### Prerequisites

- **Go**: 1.21 or higher
- **PostgreSQL**: 14 or higher
- **Redis**: 7 or higher (optional, for caching)
- **Docker**: 24 or higher (recommended)
- **Make**: For development commands

### Quick Start (Docker)

```bash
# Clone the repository
git clone <repo-url>
cd nexpaces-api

# Copy environment variables
cp .env.example .env

# Start all services (API + PostgreSQL + Redis)
docker-compose up -d

# Check API health
curl http://localhost:8080/health

# View logs
docker-compose logs -f api
```

### Local Development (Without Docker)

```bash
# Install dependencies
go mod download

# Setup PostgreSQL
createdb nexspaces_dev

# Run migrations
make migrate-up

# Start development server
make dev

# API will be available at http://localhost:8080
```

---

## 🔧 Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

### Critical Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=nexspaces_dev
DB_SSLMODE=disable

# Server
PORT=8080
ENVIRONMENT=development

# Security
JWT_SECRET=your-secret-key-min-32-chars
ALLOWED_ORIGINS=http://localhost:3000

# Multi-Tenant
DEFAULT_TENANT_ID=550e8400-e29b-41d4-a716-446655440000  # For development
```

---

## 🛠️ Development

### Available Make Commands

```bash
make help           # Show all available commands

# Development
make dev            # Run development server with hot-reload
make build          # Build production binary
make test           # Run all tests
make test-unit      # Run unit tests only
make test-int       # Run integration tests

# Database
make migrate-up     # Run all migrations
make migrate-down   # Rollback last migration
make migrate-create # Create new migration file

# Code Quality
make lint           # Run golangci-lint
make fmt            # Format code
make vet            # Run go vet

# Docker
make docker-build   # Build Docker image
make docker-up      # Start docker-compose stack
make docker-down    # Stop docker-compose stack
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./internal/usecase/website/...

# Run integration tests
make test-int
```

---

## 📡 API Documentation

### Swagger UI

Once the server is running, visit:

```
http://localhost:8080/swagger/index.html
```

### Key Endpoints

#### Health Check
```bash
GET /health
```

#### Authentication
```bash
POST /api/v1/auth/login
POST /api/v1/auth/register
POST /api/v1/auth/logout
```

#### Websites (Multi-Tenant)
```bash
GET    /api/v1/websites              # List websites (tenant-scoped)
POST   /api/v1/websites              # Create website
GET    /api/v1/websites/:id          # Get website by ID
PATCH  /api/v1/websites/:id          # Update website
DELETE /api/v1/websites/:id          # Delete website
POST   /api/v1/websites/:id/publish  # Publish website
POST   /api/v1/websites/:id/archive  # Archive website

# Public endpoint (no auth)
GET    /api/v1/websites/slug/:slug   # Get website by slug
```

#### Tenants
```bash
GET    /api/v1/tenants/current       # Get current tenant info
GET    /api/v1/tenants               # List tenants (admin only)
POST   /api/v1/tenants               # Create tenant (admin only)
```

---

## 🔒 Security

### Multi-Tenant Isolation

Every request must include tenant context:

1. **JWT Token**: Contains `tenant_id` claim
2. **Database RLS**: Row Level Security policies enforce isolation
3. **Application Layer**: All queries include tenant_id filter

### Security Headers

All responses include:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security` (production)

### Input Validation

- ✅ Request body validation using struct tags
- ✅ SQL injection prevention (prepared statements)
- ✅ XSS protection (output encoding)
- ✅ CSRF tokens (for state-changing operations)

---

## 🗄️ Database

### Multi-Tenant Strategy

**Shared Database + Row Level Security (RLS)**

```sql
-- Every table includes tenant_id
CREATE TABLE sites (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    ...
);

-- RLS policy enforces isolation
CREATE POLICY tenant_isolation ON sites
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);
```

### Migrations

Migrations use [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
# Create new migration
make migrate-create name=add_templates_table

# Apply migrations
make migrate-up

# Rollback
make migrate-down
```

---

## 🐳 Docker Deployment

### Build Production Image

```bash
docker build -f docker/Dockerfile -t nexspaces-api:latest .
```

### Multi-Stage Build Benefits

- ✅ **Small image size**: ~20MB (alpine-based)
- ✅ **Security**: Runs as non-root user
- ✅ **Fast builds**: Layer caching optimized
- ✅ **No source code**: Only compiled binary

### Environment-Specific Deployment

```bash
# Development
docker-compose up

# Staging
docker-compose -f docker-compose.staging.yml up

# Production (use Kubernetes/ECS)
kubectl apply -f k8s/
```

---

## 📊 Monitoring & Observability

### Structured Logging

JSON-formatted logs with correlation IDs:

```json
{
  "level": "info",
  "timestamp": "2024-01-01T12:00:00Z",
  "correlation_id": "abc-123-def",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-123",
  "method": "POST",
  "path": "/api/v1/websites",
  "status": 201,
  "latency_ms": 45,
  "message": "Website created successfully"
}
```

### Health Checks

```bash
# Liveness probe
GET /health

# Readiness probe
GET /ready
```

### Metrics (Future)

- Request latency (P50, P95, P99)
- Error rates per endpoint
- Database query performance
- Tenant-specific metrics

---

## 🧪 Testing Strategy

### Test Pyramid

```
        E2E (5%)
       /        \
    Integration (15%)
   /                \
  Unit Tests (80%)
```

### Test Categories

1. **Unit Tests**: Domain logic, validators, utilities
2. **Integration Tests**: Repository layer, database operations
3. **E2E Tests**: Full API flows with real database

### Test Coverage Goals

- **Overall**: >80%
- **Critical paths**: >90%
- **Domain layer**: >95%

---

## 🤝 Contributing

### Development Workflow

1. Create feature branch: `git checkout -b feature/add-templates`
2. Make changes with tests
3. Run quality checks: `make lint && make test`
4. Commit with conventional commits: `feat: add template marketplace`
5. Push and create PR

### Code Quality Standards

- ✅ **gofmt**: Code must be formatted
- ✅ **golangci-lint**: No linter errors
- ✅ **Tests**: New code requires tests
- ✅ **Coverage**: Maintain >80% coverage
- ✅ **Documentation**: Public APIs must have godoc comments

---

## 📋 Roadmap

### Phase 1: Core Infrastructure (Current)
- [x] Clean Architecture setup
- [x] Multi-tenant database schema
- [x] Authentication & authorization
- [x] Website CRUD operations
- [ ] Template marketplace

### Phase 2: Advanced Features
- [ ] RBAC/ABAC with OPA
- [ ] Real-time notifications (WebSockets)
- [ ] File storage (S3/R2)
- [ ] Email service integration
- [ ] Billing & subscriptions (Stripe)

### Phase 3: Scale & Performance
- [ ] Redis caching layer
- [ ] Database read replicas
- [ ] Rate limiting per tenant
- [ ] CDN integration (Cloudflare)
- [ ] Horizontal scaling

### Phase 4: Observability
- [ ] OpenTelemetry integration
- [ ] Distributed tracing
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Alert management

---

## 📝 License

Copyright © 2024 NexSpaces. All rights reserved.

---

## 📧 Support

- **Documentation**: [docs.nexspaces.com](https://docs.nexspaces.com)
- **Issues**: [GitHub Issues](https://github.com/nexspaces/api/issues)
- **Email**: support@nexspaces.com

---

**Built with ❤️ using Go, PostgreSQL, and Clean Architecture principles.**
