# NexSpaces API

**Production-Ready Multi-Tenant SaaS Platform Backend**

A secure, scalable, and maintainable Go backend for the NexSpaces template marketplace platform. Built with Clean Architecture principles, designed for enterprise multi-tenancy.

---

## 🎯 Project Status

**Current Phase:** Phase 6 Complete ✅
**Build Status:** ✅ Successful (12MB binary)
**Test Coverage:** Domain layer 36.8% (Target: 80%)
**API Endpoints:** 32 endpoints implemented
**Last Updated:** October 1, 2025

### Completed Phases (1-6)

- ✅ **Phase 1**: Foundation (Logger, Errors, Validator, Database)
- ✅ **Phase 2**: Domain Layer (Tenant, User, Website entities)
- ✅ **Phase 3**: Data Layer (Migrations, Repositories with RLS)
- ✅ **Phase 4**: Business Logic (Services, DTOs, JWT auth)
- ✅ **Phase 5**: HTTP Layer (32 endpoints, Middleware)
- ✅ **Phase 6**: Bootstrap (Dependency injection, Routes)

### In Progress

See [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md) for detailed implementation plan.

---

## 🎯 Project Overview

NexSpaces is a multi-tenant SaaS platform that provides customizable templates (CMS, CRM, E-commerce, Education, Hospitality, ERP) for different industries. This repository contains the backend API service.

### Key Features

- ✅ **Multi-Tenant Architecture**: Complete tenant isolation at database and application level
- ✅ **Clean Architecture**: Domain-driven design with clear separation of concerns
- ✅ **Security First**: JWT authentication, RBAC, Row Level Security (RLS)
- ✅ **Production Ready**: Docker, migrations, structured logging, health checks
- ✅ **Type Safety**: Comprehensive input validation and error handling
- ✅ **Performance**: PostgreSQL with optimized queries, connection pooling
- ✅ **Observability**: Structured logging with tenant context and correlation IDs

---

## 📐 Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Handlers                         │
│         (Fiber, REST API, Middleware)                    │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                    Use Cases                             │
│         (Business Logic, Orchestration)                  │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                   Repositories                           │
│        (Data Access, PostgreSQL)                         │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Domain Entities                         │
│          (Business Models, Rules)                        │
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
│   │   ├── tenant/              # Tenant entity (subscriptions, plans)
│   │   ├── user/                # User entity (RBAC, auth)
│   │   └── website/             # Website entity
│   │
│   ├── repository/              # Data access layer
│   │   ├── tenant/              # Tenant repository (PostgreSQL)
│   │   ├── user/                # User repository (PostgreSQL)
│   │   └── website/             # Website repository (PostgreSQL)
│   │
│   ├── usecase/                 # Business logic
│   │   ├── tenant/              # Tenant service + DTOs
│   │   ├── user/                # User service + JWT generation
│   │   └── website/             # Website service + DTOs
│   │
│   ├── handler/                 # HTTP layer
│   │   ├── auth/                # Authentication endpoints
│   │   ├── tenant/              # Tenant management (13 endpoints)
│   │   ├── user/                # User management (11 endpoints)
│   │   └── website/             # Website CRUD (7 endpoints)
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
│   ├── database/                # PostgreSQL connection + RLS helpers
│   ├── logger/                  # Zerolog structured logging
│   ├── validator/               # Custom validators (UUID, slug, domain)
│   └── errors/                  # Application error types (50+)
│
├── migrations/                  # Database migrations
│   ├── 001_create_tenants.*     # Tenants table with RLS
│   ├── 002_create_users.*       # Users table with RLS
│   └── 003_create_websites.*    # Websites table with RLS
│
├── docker/
│   └── Dockerfile               # Multi-stage production build
│
├── .env.example                 # Environment variables template
├── docker-compose.yml           # Local development stack
├── Makefile                     # Development commands
├── IMPLEMENTATION_ROADMAP.md    # Detailed implementation plan
└── README.md                    # This file
```

---

## 🚀 Getting Started

### Prerequisites

- **Go**: 1.21 or higher
- **PostgreSQL**: 14 or higher
- **Docker**: 24 or higher (recommended)
- **Make**: For development commands

### Quick Start (Docker)

```bash
# Clone the repository
git clone <repo-url>
cd nexpaces-api

# Copy environment variables
cp .env.example .env

# Start all services (API + PostgreSQL + Adminer)
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
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=nexspaces_dev
DATABASE_SSLMODE=disable

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ENVIRONMENT=development

# Security
AUTH_JWT_SECRET=your-secret-key-min-32-chars-change-in-production
SECURITY_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
SECURITY_ALLOW_CREDENTIALS=true

# Rate Limiting
SECURITY_RATE_LIMIT_REQUESTS=100
SECURITY_RATE_LIMIT_DURATION=1m
```

---

## 🛠️ Development

### Available Make Commands

```bash
make help           # Show all available commands

# Development
make dev            # Run development server
make build          # Build production binary
make test           # Run all tests
make test-coverage  # Run tests with coverage

# Database
make migrate-up     # Run all migrations
make migrate-down   # Rollback last migration

# Code Quality
make lint           # Run linters
make fmt            # Format code

# Docker
make docker-up      # Start docker-compose stack
make docker-down    # Stop docker-compose stack
```

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run domain tests
go test -v ./internal/domain/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Current Test Coverage:**
- Tenant Entity: 57.9%
- User Entity: 48.1%
- Total Domain: 36.8%
- **Target: 80%+**

---

## 📡 API Documentation

### Key Endpoints (32 Total)

#### Health Check
```bash
GET /health              # Health check endpoint
```

#### Authentication (4 endpoints)
```bash
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me     # Protected
```

#### Tenants (13 endpoints)
```bash
POST   /api/v1/tenants
GET    /api/v1/tenants/current
GET    /api/v1/tenants/stats
GET    /api/v1/tenants/:id
PATCH  /api/v1/tenants/:id
POST   /api/v1/tenants/:id/upgrade
POST   /api/v1/tenants/:id/suspend
POST   /api/v1/tenants/:id/activate
POST   /api/v1/tenants/:id/domain
DELETE /api/v1/tenants/:id
# ... and more (see routes.go)
```

#### Users (11 endpoints)
```bash
POST   /api/v1/users
GET    /api/v1/users/stats
GET    /api/v1/users/:id
PATCH  /api/v1/users/:id
POST   /api/v1/users/:id/password
POST   /api/v1/users/:id/role
POST   /api/v1/users/:id/suspend
DELETE /api/v1/users/:id
# ... and more (see routes.go)
```

#### Websites (7 endpoints)
```bash
GET    /api/v1/websites
POST   /api/v1/websites
GET    /api/v1/websites/:id
PATCH  /api/v1/websites/:id
DELETE /api/v1/websites/:id
POST   /api/v1/websites/:id/publish
POST   /api/v1/websites/:id/archive

# Public endpoint (no auth)
GET    /api/v1/public/websites/slug/:slug
```

---

## 🔒 Security

### Multi-Tenant Isolation

Every request enforces tenant context:

1. **JWT Token**: Contains `tenant_id` claim
2. **Database RLS**: Row Level Security policies at PostgreSQL level
3. **Application Layer**: All queries include tenant_id filter

### Security Features

- ✅ JWT authentication with bcrypt password hashing (cost 12)
- ✅ Role-based access control (Owner, Admin, Editor, Viewer)
- ✅ Row-level security on all tables
- ✅ Input validation with struct tags
- ✅ SQL injection prevention (prepared statements)
- ✅ CORS configuration
- ✅ Rate limiting per IP
- ✅ Helmet middleware (security headers)

---

## 🗄️ Database

### Multi-Tenant Strategy

**Shared Database + Row Level Security (RLS)**

```sql
-- Example: Users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    ...
);

-- RLS policy enforces isolation
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant')::UUID);
```

### Subscription Plans

- **Free**: 1 user, 1 website, 100MB storage
- **Starter**: 5 users, 3 websites, 1GB storage
- **Pro**: 20 users, 10 websites, 10GB storage, custom domain
- **Enterprise**: Unlimited users, unlimited websites, unlimited storage

---

## 🐳 Docker Deployment

### Build Production Image

```bash
docker build -f docker/Dockerfile -t nexspaces-api:latest .
```

### Multi-Stage Build Benefits

- ✅ **Small image size**: ~12-15MB (alpine-based)
- ✅ **Security**: Runs as non-root user
- ✅ **Fast builds**: Layer caching optimized
- ✅ **No source code**: Only compiled binary

---

## 📊 Monitoring & Observability

### Structured Logging

JSON-formatted logs with correlation IDs:

```json
{
  "level": "info",
  "timestamp": "2025-10-01T16:48:00Z",
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

### Audit Logging

All critical actions are logged:
- User creation/deletion
- Tenant plan changes
- Permission changes
- Failed authentication attempts

---

## 📋 Roadmap

For detailed implementation plan, see [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md)

### ✅ Completed (Phase 1-6)
- [x] Clean Architecture setup
- [x] Multi-tenant database schema with RLS
- [x] JWT authentication & RBAC
- [x] Tenant management (subscriptions, plans)
- [x] User management (CRUD, roles)
- [x] Website CRUD operations
- [x] 32 API endpoints
- [x] Domain layer unit tests

### 🚧 In Progress

**Phase 7: Core Testing** (2-3 days)
- [ ] Repository integration tests
- [ ] Service/usecase unit tests
- [ ] Handler E2E tests
- [ ] Multi-tenant security tests
- [ ] Coverage target: 80%+

**Phase 8: OAuth & Social Login** (3-4 days)
- [ ] OAuth provider strategy pattern
- [ ] Google OAuth integration
- [ ] GitHub OAuth integration
- [ ] Apple OAuth integration
- [ ] Account linking

**Phase 9: Payment Infrastructure** (4-5 days)
- [ ] Stripe integration
- [ ] Subscription lifecycle management
- [ ] Webhook handlers
- [ ] Invoice generation
- [ ] Usage-based billing

**Phase 10: Multi-Tenant Payment** (5-6 days)
- [ ] Tenant custom payment config
- [ ] Stripe Connect integration
- [ ] Revenue sharing system
- [ ] Tenant billing dashboard

**Phase 11: Advanced Auth** (3-4 days)
- [ ] 2FA/MFA (TOTP)
- [ ] Magic link authentication
- [ ] Password reset flow
- [ ] Session management (refresh tokens)
- [ ] Device tracking

**Phase 12: Deployment & DevOps** (2-3 days)
- [ ] Bitbucket CI/CD pipeline
- [ ] Kubernetes manifests
- [ ] Monitoring (Prometheus/Grafana)
- [ ] Log aggregation

**Estimated Total Time:** 19-25 days for Phase 7-12

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
- ✅ **Tests**: New code requires tests
- ✅ **Coverage**: Maintain >80% coverage
- ✅ **Documentation**: Public APIs must have comments

---

## 📝 License

Copyright © 2025 NexSpaces. All rights reserved.

---

## 📧 Support

- **Documentation**: [IMPLEMENTATION_ROADMAP.md](./IMPLEMENTATION_ROADMAP.md)
- **Issues**: Create issues in repository
- **Email**: support@nexspaces.com

---

**Built with ❤️ using Go 1.21, PostgreSQL 14, and Clean Architecture principles.**
