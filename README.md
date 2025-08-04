# NexSpaces API

Modern, scalable Go backend for NexSpaces multi-tenant SaaS platform.

## 🏗️ Architecture

```
nexspaces-api/
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── config/         # Configuration management
│   ├── handlers/       # HTTP handlers
│   ├── models/         # Data models
│   ├── services/       # Business logic
│   └── middleware/     # HTTP middleware
├── pkg/                # Public libraries
│   ├── database/       # Database connections
│   └── utils/          # Utility functions
├── migrations/         # Database migrations
└── docker-compose.yml  # Development environment
```

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Docker & Docker Compose

### Development Setup

1. **Start database services:**
```bash
docker-compose up -d postgres redis
```

2. **Run the server:**
```bash
go run cmd/server/main.go
```

3. **Health check:**
```bash
curl http://localhost:8080/health
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

- **Framework:** Go Fiber v2
- **Database:** PostgreSQL 15
- **Cache:** Redis 7
- **Authentication:** JWT
- **Architecture:** Clean Architecture

## 🗄️ Database

### Multi-Tenant Strategy
- **Shared Database, Isolated Schemas** approach
- Each tenant gets dedicated schema
- Shared plans and platform metadata

### Migrations
```bash
# Database will auto-migrate on first run
# Manual migration available in migrations/001_initial_schema.sql
```

## 🔐 Features

- ✅ Multi-tenant architecture
- ✅ PostgreSQL + Redis integration
- ✅ JWT authentication ready
- ✅ CORS middleware
- ✅ Graceful shutdown
- ✅ Health checks
- ✅ Configuration management
- ⏳ Tenant management API
- ⏳ User authentication
- ⏳ Panel management

## 🧪 Development

### Available Services
- **API Server:** http://localhost:8080
- **PostgreSQL:** localhost:5432
- **Redis:** localhost:6379
- **pgAdmin:** http://localhost:8082 (admin@nexspaces.dev / admin)
- **Redis Commander:** http://localhost:8081

### Docker Services
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📡 API Endpoints

### Health & Status
- `GET /health` - Service health check

### Authentication (Planned)
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration

### Tenants (Planned)
- `GET /api/v1/tenants` - List tenants
- `POST /api/v1/tenants` - Create tenant

## 🎯 Next Steps

1. Implement tenant management endpoints
2. Add JWT authentication middleware
3. Create user registration/login handlers
4. Add panel configuration API
5. Implement multi-tenant data isolation

## 🤝 Integration with Frontend

This backend is designed to work with the Next.js frontend in `../nexpaces-web/`.

API will be consumed by:
- Next.js API routes (Phase 1)
- Direct frontend calls (Phase 2)
- Mobile apps (Phase 3)