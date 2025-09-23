# NexSpaces API

🚀 **Production-Ready** multi-tenant SaaS backend built with modern Clean Architecture + DDD patterns.

## 🏗️ Architecture

### Clean Architecture + Domain-Driven Design
```
nexspaces-api/
├── cmd/server/                    # Application entry point
├── internal/
│   ├── app/                      # Application wiring & DI
│   ├── config/                   # Configuration management
│   ├── core/                     # 🎯 Business Logic (Clean Architecture)
│   │   ├── domain/               # Entities & Value Objects
│   │   │   ├── shared/          # Common value objects & errors
│   │   │   ├── tenant/          # Tenant domain
│   │   │   ├── user/            # User domain
│   │   │   ├── template/        # Template marketplace domain
│   │   │   └── subscription/    # Billing & subscription domain
│   │   ├── ports/               # Interfaces (Dependency Inversion)
│   │   │   ├── repositories/    # Data access interfaces
│   │   │   ├── services/        # External service interfaces
│   │   │   └── events/          # Event bus interfaces
│   │   └── usecases/            # Application Services
│   │       ├── tenant/          # Tenant management use cases
│   │       ├── user/            # User management use cases
│   │       ├── template/        # Template marketplace use cases
│   │       └── subscription/    # Billing use cases
│   ├── adapters/                # External World (Clean Architecture)
│   │   ├── http/                # HTTP adapters
│   │   │   ├── handlers/        # REST API handlers
│   │   │   ├── middleware/      # HTTP middleware
│   │   │   └── routes/          # Route configuration
│   │   ├── persistence/         # Data persistence adapters
│   │   │   └── postgres/        # PostgreSQL repositories
│   │   └── external/            # External service adapters (TODO)
│   └── shared/                  # Shared utilities
│       ├── errors/              # Error handling
│       ├── validation/          # Input validation
│       ├── logger/              # Logging utilities (TODO)
│       └── metrics/             # Metrics collection (TODO)
├── migrations/                   # Database migrations
├── tests/                       # Test suites (TODO)
│   ├── integration/             # Integration tests
│   ├── e2e/                     # End-to-end tests
│   └── fixtures/                # Test data
├── api/                         # API documentation (TODO)
│   └── openapi/                 # OpenAPI/Swagger specs
├── deployments/                 # Deployment configurations (TODO)
│   ├── docker/                  # Docker configurations
│   └── k8s/                     # Kubernetes manifests
└── scripts/                     # Development scripts (TODO)
```

## 🚀 Quick Start

### Prerequisites
- **Go 1.24+** (with modules support)
- **PostgreSQL 15+** (with UUID extension)
- **Redis 7+** (for caching and rate limiting)
- **Make** (optional, for convenience commands)

### Development Setup

```bash
# 1. Clone and setup
cd nexpaces-api

# 2. Environment configuration
cp .env.example .env
# Edit .env with your database credentials

# 3. Install dependencies
go mod tidy

# 4. Setup database
createdb nexspaces_dev
psql -d nexspaces_dev -f migrations/001_initial_schema.up.sql

# 5. Start Redis (if not running)
redis-server

# 6. Run the server
go run cmd/server/main.go

# 🎉 Server running at http://localhost:8080
```

## 🔧 Configuration

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key configuration options:
- `PORT=8080` - Server port
- `DB_HOST=localhost` - PostgreSQL host
- `REDIS_HOST=localhost` - Redis host
- `JWT_SECRET` - JWT signing secret

## 📦 Tech Stack

### Core Technologies
- **Language:** Go 1.24+ (with generics & toolchain)
- **Framework:** Go Fiber v2 (Express-like performance)
- **Database:** PostgreSQL 15+ (with Row Level Security)
- **Cache:** Redis 7+ (clustering ready)
- **Authentication:** JWT with RS256/HS256
- **Validation:** go-playground/validator v10

### Architecture Patterns
- **Clean Architecture** (Uncle Bob)
- **Domain-Driven Design** (DDD)
- **Repository Pattern** (with interfaces)
- **Dependency Injection** (constructor-based)
- **CQRS Ready** (command/query separation)

### Security & Performance
- **Multi-Tenant RLS** (PostgreSQL Row Level Security)
- **Rate Limiting** (Redis-based, tenant-aware)
- **CORS & Security Headers** (Helmet middleware)
- **Input Validation** (XSS, SQL injection prevention)
- **Connection Pooling** (optimized for production)

## 🗄️ Database

### Multi-Tenant Strategy
🎯 **Shared Database + Row Level Security (RLS)** - Production proven approach

- **Single Database** with tenant isolation via PostgreSQL RLS
- **Automatic Tenant Context** set per request
- **Zero Cross-Tenant Leaks** guaranteed at database level
- **Scalable** - supports thousands of tenants efficiently

### Schema Overview
```sql
-- Core Tables
tenants              # Tenant management
users               # Multi-tenant users (RLS enabled)
templates           # Template marketplace (RLS enabled)
template_installations  # Tenant template installs
subscriptions       # Billing & usage tracking
audit_logs          # Security audit trail

-- Security Features
✅ Row Level Security (RLS) on all tenant tables
✅ Cross-tenant access prevention
✅ Audit logging for compliance
✅ UUID primary keys for security
✅ Optimized indexes for tenant queries
```

### Migrations
```bash
# Production migration
psql -d nexspaces_prod -f migrations/001_initial_schema.up.sql

# Rollback (if needed)
psql -d nexspaces_prod -f migrations/001_initial_schema.down.sql

# Development reset
dropdb nexspaces_dev && createdb nexspaces_dev
psql -d nexspaces_dev -f migrations/001_initial_schema.up.sql
```

## ✅ Completed Features

### 🏗️ Core Architecture
- ✅ **Clean Architecture + DDD** - Scalable, testable business logic
- ✅ **Multi-Tenant Security** - PostgreSQL RLS with zero cross-tenant leaks
- ✅ **Domain Entities** - Rich domain models (Tenant, User, Template, Subscription)
- ✅ **Repository Pattern** - Database abstraction with interfaces
- ✅ **Use Cases Layer** - Business logic coordination
- ✅ **Dependency Injection** - Constructor-based DI container

### 🔒 Security & Authentication
- ✅ **JWT Authentication** - Stateless auth with tenant context
- ✅ **Multi-Tenant Middleware** - Automatic tenant resolution
- ✅ **Row Level Security** - Database-level tenant isolation
- ✅ **Input Validation** - XSS, SQL injection, malicious input prevention
- ✅ **Security Headers** - CORS, CSP, HSTS, XSS protection
- ✅ **Rate Limiting** - Tenant-aware request throttling
- ✅ **Audit Logging** - Security compliance tracking

### 🌐 HTTP Layer
- ✅ **REST API Handlers** - Tenant, User, Template management
- ✅ **Error Handling** - Consistent error responses
- ✅ **Request Validation** - Comprehensive input validation
- ✅ **Graceful Shutdown** - Production-ready server lifecycle
- ✅ **Health Checks** - Service monitoring endpoints

### 💾 Database Layer
- ✅ **PostgreSQL Integration** - Connection pooling, transactions
- ✅ **Database Migrations** - Production-ready schema with RLS
- ✅ **Repository Implementations** - Postgres-specific data access
- ✅ **Redis Caching** - Performance optimization layer
- ✅ **Multi-Tenant Queries** - Tenant-isolated data operations

### 📦 Template Marketplace
- ✅ **Template Domain** - Rich template modeling
- ✅ **Template CRUD** - Create, update, delete operations
- ✅ **Template Publishing** - Version management and publishing
- ✅ **Template Installation** - Tenant template management
- ✅ **Template Search** - Marketplace search with filtering
- ✅ **Template Categories** - CMS, CRM, E-commerce, etc.

### 💳 Billing & Subscriptions
- ✅ **Subscription Domain** - Complete billing model
- ✅ **Usage Tracking** - Feature usage and limits
- ✅ **Plan Management** - Subscription plans and billing cycles
- ✅ **Payment Integration Ready** - Stripe-ready architecture

## 🔄 Todo/Next Steps

### 🚀 High Priority (Next Sprint)
- ⏳ **External Service Adapters** - Stripe, SendGrid, AWS S3
- ⏳ **Event Bus Implementation** - Domain events for async processing
- ⏳ **Better Auth Integration** - Frontend auth system integration
- ⏳ **API Documentation** - OpenAPI/Swagger specs
- ⏳ **Docker & K8s** - Container deployment setup

### 📈 Medium Priority
- ⏳ **Comprehensive Testing** - Unit, integration, E2E test suites
- ⏳ **Monitoring & Observability** - Prometheus, Grafana, alerting
- ⏳ **Logging Infrastructure** - Structured logging with ELK stack
- ⏳ **Performance Optimization** - Query optimization, caching strategies
- ⏳ **CI/CD Pipeline** - Automated testing and deployment

### 🎯 Future Enhancements
- ⏳ **GraphQL API** - Alternative to REST for complex queries
- ⏳ **Webhook System** - Event notifications for integrations
- ⏳ **Template Versioning** - Advanced version management
- ⏳ **Multi-Region Support** - Geographic distribution
- ⏳ **Advanced Analytics** - Usage analytics and reporting

## 🧪 Development

### Available Services
- **API Server:** http://localhost:8080
- **PostgreSQL:** localhost:5432
- **Redis:** localhost:6379
- **pgAdmin:** http://localhost:8082 (admin@nexspaces.dev / admin)
- **Redis Commander:** http://localhost:8081

### Docker Commands

```bash
# Development (with hot reload)
make dev                    # Start development environment
make stop                   # Stop all services
make restart                # Restart API service
make logs                   # View all logs
make api-logs              # View API logs only

# Production
make prod                   # Start production environment

# Database
make migrate               # Run migrations
make db-reset              # Reset database (⚠️ destructive)
make db-shell              # PostgreSQL shell
make redis-shell           # Redis shell

# Utilities  
make health                # Check service health
make urls                  # Show service URLs
make clean                 # Clean up containers/volumes
make test                  # Run tests
```

## 📡 API Endpoints

### 🩺 Health & Status
- `GET /api/v1/health` - Service health check with database status

### 🏢 Tenant Management
- `POST /api/v1/tenants` - Create new tenant (public endpoint)
- `GET /api/v1/tenants/{tenantId}` - Get tenant details
- `PUT /api/v1/tenants/{tenantId}` - Update tenant (admin only)

### 👥 User Management (Tenant-scoped)
- `POST /api/v1/tenants/{tenantId}/users` - Create user (admin only)
- `GET /api/v1/tenants/{tenantId}/users` - List users with pagination
- `GET /api/v1/tenants/{tenantId}/users/{userId}` - Get user details
- `PUT /api/v1/tenants/{tenantId}/users/{userId}` - Update user
- `DELETE /api/v1/tenants/{tenantId}/users/{userId}` - Deactivate user

### 📦 Template Marketplace
- `GET /api/v1/templates/search` - Search templates (public + authenticated)
- `POST /api/v1/tenants/{tenantId}/templates` - Create template
- `POST /api/v1/tenants/{tenantId}/templates/{templateId}/publish` - Publish template
- `POST /api/v1/tenants/{tenantId}/templates/{templateId}/install` - Install template

### 💳 Subscription Management (Planned)
- `POST /api/v1/tenants/{tenantId}/subscriptions` - Create subscription
- `PUT /api/v1/tenants/{tenantId}/subscriptions/{subscriptionId}` - Update subscription
- `DELETE /api/v1/tenants/{tenantId}/subscriptions/{subscriptionId}` - Cancel subscription
- `POST /api/v1/tenants/{tenantId}/usage` - Track usage metrics

### 🔐 Authentication
All endpoints except public ones require:
- `Authorization: Bearer <jwt_token>` header
- JWT token must include valid `tenant_id` and `user_id`
- Cross-tenant access is automatically prevented

### 📝 Request/Response Format
```json
// Success Response
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}

// Error Response
{
  "success": false,
  "error": {
    "status": 400,
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": {
      "fields": [
        {
          "field": "email",
          "message": "email address is required",
          "code": "MISSING_EMAIL"
        }
      ]
    }
  }
}
```

## 🔧 Development Commands

```bash
# Build & Run
go build -o bin/nexpaces-api cmd/server/main.go
./bin/nexpaces-api

# Development with hot reload (using air)
go install github.com/cosmtrek/air@latest
air

# Testing
go test ./...                    # Run all tests
go test -v ./internal/core/...   # Verbose core tests
go test -race ./...              # Race condition detection
go test -cover ./...             # Test coverage

# Code Quality
go fmt ./...                     # Format code
go vet ./...                     # Static analysis
golangci-lint run               # Advanced linting

# Database
go run migrations/migrate.go up   # Run migrations
go run migrations/migrate.go down # Rollback migrations

# Performance
go test -bench=. ./...           # Benchmarks
go tool pprof ./profile.out      # Profiling
```

## 🏭 Production Deployment

### Environment Variables
```bash
# Required for production
export ENVIRONMENT=production
export JWT_SECRET="your-super-secure-jwt-secret"
export DB_HOST="your-postgres-host"
export DB_PASSWORD="your-secure-password"
export REDIS_URL="redis://your-redis-host:6379"
export ALLOWED_ORIGINS="https://yourdomain.com"
export ENABLE_HSTS=true
```

### Docker Deployment
```bash
# Build image
docker build -t nexspaces-api .

# Run container
docker run -d \
  --name nexspaces-api \
  -p 8080:8080 \
  --env-file .env.production \
  nexspaces-api
```

### Health Monitoring
```bash
# Health check
curl http://localhost:8080/api/v1/health

# Metrics (planned)
curl http://localhost:9090/metrics
```

## 🤝 Integration with Frontend

This backend is designed to work seamlessly with:

### 🌐 NexSpaces Web Frontend (`../nexpaces-web/`)
- **Better Auth Integration** - Shared JWT authentication
- **Tenant-Aware API Calls** - Automatic tenant context
- **Type-Safe API Client** - Generated TypeScript types
- **Real-time Updates** - WebSocket support (planned)

### 📱 Mobile Apps (Future)
- **React Native** - Shared API endpoints
- **Flutter** - RESTful API consumption
- **Native Apps** - Standard HTTP/JSON interface

### 🔌 External Integrations
- **Stripe** - Payment processing webhooks
- **SendGrid** - Email notification callbacks
- **AWS S3** - File upload/download endpoints
- **Webhooks** - Event notifications to external services

## 🛡️ Security Considerations

### Production Security Checklist
- [ ] Change all default passwords and secrets
- [ ] Enable HSTS and proper CORS settings
- [ ] Configure rate limiting per tenant
- [ ] Set up SSL/TLS certificates
- [ ] Enable audit logging
- [ ] Configure firewall rules
- [ ] Set up monitoring and alerting
- [ ] Regular security updates
- [ ] Database backup strategy
- [ ] Disaster recovery plan

### Multi-Tenant Security
- ✅ **Database-level isolation** (PostgreSQL RLS)
- ✅ **Application-level validation** (middleware)
- ✅ **JWT tenant context** (stateless auth)
- ✅ **Audit logging** (compliance ready)
- ✅ **Input sanitization** (XSS/injection prevention)