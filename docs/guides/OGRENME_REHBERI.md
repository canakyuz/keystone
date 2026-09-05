# 🎓 Keystone Backend - Quick Learning Guide

> **Goal:** Learn Go backend development and understand the Keystone platform
> **Level:** Beginner to Intermediate
> **Estimated Time:** 2-3 weeks for basics

---

## 📚 Table of Contents

1. [Go Essentials](#1-go-essentials)
2. [Clean Architecture Explained](#2-clean-architecture-explained)
3. [Project Structure Walkthrough](#3-project-structure-walkthrough)
4. [Multi-Tenant Concepts](#4-multi-tenant-concepts)
5. [Development Workflow](#5-development-workflow)
6. [Testing Guide](#6-testing-guide)
7. [Common Tasks](#7-common-tasks)

---

## 1. Go Essentials

### Quick Start

```go
package main

import "fmt"

func main() {
    // Variables
    name := "Keystone"  // Type inference
    var count int = 42   // Explicit type

    // Structs (like classes)
    type Tenant struct {
        ID         string
        Name       string
        SchemaName string // e.g., 'tenant_acme'
    }

    tenant := Tenant{
        ID:   "550e8400",
        Name: "Acme Corp",
        SchemaName: "tenant_acme_corp",
    }

    fmt.Println(tenant.Name) // Acme Corp
}
```

### Error Handling (Critical!)

```go
// Go's signature pattern: return value + error
func GetTenant(id string) (*Tenant, error) {
    if id == "" {
        return nil, errors.New("tenant ID required")
    }

    tenant, err := db.Query("SELECT * FROM tenants WHERE id = ?", id)
    if err != nil {
        return nil, fmt.Errorf("database query failed: %w", err)
    }

    return tenant, nil
}

// ALWAYS check errors
tenant, err := GetTenant("123")
if err != nil {
    log.Error("Failed to get tenant:", err)
    return
}
```

### Key Concepts

| Concept | Explanation | Example |
|---------|-------------|---------|
| **Struct** | Data structure (like class) | `type User struct { Name string }` |
| **Interface** | Contract (defines methods) | `type Repository interface { Save() }` |
| **Pointer** | Memory reference | `*Tenant` vs `Tenant` |
| **Error handling** | Explicit error returns | `func Do() (result, error)` |
| **Defer** | Execute at function end | `defer file.Close()` |
| **Goroutine** | Lightweight thread | `go doWork()` |

### Essential Commands

```bash
go mod init keystone    # Create new project
go get github.com/pkg/name  # Install dependency
go mod tidy                 # Clean dependencies

go run cmd/server/main.go   # Run application
go build -o bin/app ./cmd   # Build binary
go test ./...               # Run all tests
go fmt ./...                # Format code
```

---

## 2. Clean Architecture Explained

### Why Clean Architecture?

✅ **Testable:** Each layer independent
✅ **Maintainable:** Easy to modify
✅ **Scalable:** Clear separation of concerns

### The Four Layers

```
┌─────────────────────────────────────────────┐
│  1. HANDLER (HTTP Layer)                    │
│     What: Receives HTTP requests            │
│     File: internal/handler/*/handler.go     │
│     Job: Parse request → Call service       │
└──────────────────┬──────────────────────────┘
                   ↓
┌──────────────────▼──────────────────────────┐
│  2. SERVICE (Business Logic)                │
│     What: Business rules                    │
│     File: internal/usecase/*/service.go     │
│     Job: Validate → Process → Save          │
└──────────────────┬──────────────────────────┘
                   ↓
┌──────────────────▼──────────────────────────┐
│  3. REPOSITORY (Data Access)                │
│     What: Database operations               │
│     File: internal/repository/*/postgres.go │
│     Job: SQL queries (CRUD)                 │
└──────────────────┬──────────────────────────┘
                   ↓
┌──────────────────▼──────────────────────────┐
│  4. DOMAIN (Core Entities)                  │
│     What: Business models                   │
│     File: internal/domain/*/entity.go       │
│     Job: Define structure + rules           │
└─────────────────────────────────────────────┘
```

### Example Flow: Create Student

```go
// 1. HANDLER: Receive HTTP request
func (h *StudentHandler) Create(c *fiber.Ctx) error {
    var req CreateStudentRequest
    c.BodyParser(&req)  // Parse JSON

    // Call service
    student, err := h.service.Create(c.Context(), req)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.Status(201).JSON(student)
}

// 2. SERVICE: Business logic
func (s *StudentService) Create(ctx context.Context, req CreateStudentRequest) (*Student, error) {
    // Validate
    if req.Email == "" {
        return nil, errors.New("email required")
    }

    // Create entity
    student := &Student{
        ID:    uuid.New().String(),
        Email: req.Email,
    }

    // Save via repository
    err := s.repo.Create(ctx, student)
    return student, err
}

// 3. REPOSITORY: Database operation
func (r *PostgresRepo) Create(ctx context.Context, student *Student) error {
    query := `INSERT INTO students (id, email) VALUES ($1, $2)`
    _, err := r.db.ExecContext(ctx, query, student.ID, student.Email)
    return err
}

// 4. DOMAIN: Entity definition
type Student struct {
    ID        string
    Email     string
    CreatedAt time.Time
}
```

---

## 3. Project Structure Walkthrough

### Key Directories

```
keystone/
│
├── cmd/server/main.go          # ⭐ START HERE - Entry point
│
├── internal/                   # Private code
│   ├── domain/                 # Entities (Student, Tenant)
│   ├── repository/             # Database (PostgreSQL)
│   ├── usecase/                # Business logic
│   ├── handler/                # HTTP endpoints
│   ├── middleware/             # Auth, logging
│   └── app/                    # Bootstrap
│
├── pkg/                        # Shared utilities
│   ├── database/               # DB connection
│   ├── logger/                 # Structured logs
│   └── validator/              # Input validation
│
├── migrations/                 # SQL schema
│   ├── 001_create_tenants.up.sql
│   └── 002_create_users.up.sql
│
└── Makefile                    # Development commands
```

### File Naming Convention

| File | Purpose | Example |
|------|---------|---------|
| `*_entity.go` | Domain model | `student_entity.go` |
| `*_postgres.go` | PostgreSQL repo | `student_postgres.go` |
| `*_service.go` | Business logic | `student_service.go` |
| `*_handler.go` | HTTP endpoints | `student_handler.go` |
| `dto.go` | Request/Response | `dto.go` |
| `errors.go` | Custom errors | `errors.go` |

---

## 4. Multi-Tenant Concepts

### What is Multi-Tenancy?

One application serves multiple customers (tenants), but each tenant's data is logically isolated and invisible to other tenants.

### Our Model: Schema-per-Tenant

Keystone uses a **Schema-per-Tenant** architecture. This means every tenant gets their own dedicated schema within a single PostgreSQL database. This provides strong data isolation without the complexity of managing multiple databases.

```
┌──────────────────────────────────────────┐
│         KEYSTONE PLATFORM (Single DB)    │
├──────────────────┬──────────────────┬────┤
│ Schema: tenant_a │ Schema: tenant_b │ ...│
│ (Tables for A)   │ (Tables for B)   │    │
│ - users          │ - users          │    │
│ - projects       │ - projects       │    │
└──────────────────┴──────────────────┴────┘
```

### How It Works in Code

We **DO NOT** use `WHERE tenant_id = ?` in our queries. Instead, we use PostgreSQL's `search_path` to scope the entire database session to a single tenant's schema.

```go
// 1. Middleware sets the tenant context
func AuthMiddleware(c *fiber.Ctx) error {
    token := c.Get("Authorization")
    claims := parseJWT(token)

    // Get tenant's unique schema name (e.g., "tenant_acme_corp")
    schemaName := claims.SchemaName

    // Store in request context
    c.Locals("schema_name", schemaName)

    return c.Next()
}

// 2. A database manager sets the search_path for the connection
func (m *TenantManager) SetTenantScope(ctx context.Context, schemaName string) error {
    // This line tells PostgreSQL to only look for tables in this schema
    _, err := m.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", pq.QuoteIdentifier(schemaName)))
    return err
}

// 3. Repository queries are simple and clean
func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    // No tenant_id filter is needed!
    // The query automatically runs inside the correct tenant's schema.
    query := `SELECT * FROM users WHERE id = $1`
    return r.db.QueryRow(query, id)
}
```

#### Plan Bazlı İzolasyon Haritası

- `starter` ve `pro` planları şu anda varsayılan olarak schema-per-tenant modelini kullanıyor; `enterprise` planı için database-per-tenant geçişi planlanıyor.
- `internal/usecase/tenant/provisioning.go` içindeki **TODO** notu, plan → izolasyon eşleşmesini tek bir konfigürasyon kaynağına taşıma gereksinimini hatırlatıyor.
- `TenantConnectionManager` katmanı henüz implemente edilmedi; request bazında `search_path` ayarlama sorumluluğu middleware + repository kombinasyonunda manuel ilerliyor. Bu bileşeni tamamlamak, ileride database-per-tenant senaryosuna geçişi kolaylaştıracak.

#### SaaS Paketleri ve Provisioning Akışı

1. Platform admin paneli veya doğrudan API ile `POST /api/v1/tenants` çağrısı yapılır.
2. `Service.Create` ( `internal/usecase/tenant/service.go` ) tenant’ı oluşturup seçilen planı domain entity’sine bağlar.
3. `ProvisioningService.GenerateSchemaName` benzersiz `schema_name` üretir; `ProvisionTenantSchema` plan bazlı şema şablonunu uygular.
4. **Yeni**: `api/openapi.yaml` artık tüm tenant, registry, ödeme ve upload uç noktalarını içeriyor; Swagger UI üzerinden (`/docs`) veya doğrudan YAML dosyasıyla entegrasyon yapılabilir.

#### Geliştirme Ortamı Kontrol Adımları

```bash
# Migration ve seed işlemleri
make migrate-up
make seed-dev

# Postgres içinde şemaları doğrula
make db-shell
\dn tenant_*          -- Oluşan tenant şemaları
SELECT schema_name
FROM tenants
ORDER BY created_at DESC;  -- API üzerinden açılan tenant'ların kaydı
```

`seed-dev` scriptleri ( `scripts/seed/dev_seed.sql` ), idempotent hale getirildiğinden aynı komut tekrar çalıştırıldığında mevcut tenant verisini bozmadan ilerler.

### Why This Model?

- **Strong Isolation:** It's impossible for one tenant's query to see another tenant's data.
- **Clean Code:** Repositories don't need to be aware of multi-tenancy. No more forgetting a `WHERE tenant_id` clause.
- **Flexibility:** Different tenants can (in the future) have slightly different table structures based on their plan.

---

## 5. Development Workflow

### Daily Development

```bash
# 1. Start services
docker-compose up -d postgres redis

# 2. Run migrations
make migrate-up

# 3. Start dev server
make dev

# 4. Test API
curl http://localhost:8080/health
```

> **İpucu:** Swagger arayüzü için `http://localhost:8080/docs`, ham OpenAPI dosyası için proje kökündeki `api/openapi.yaml` yolunu kullanabilirsiniz. Registry, upload ve ödeme uçları dahil tüm API yüzeyi artık bu dosyada tanımlı.

### Making Changes

**Example: Add new field to Student**

```bash
# 1. Update domain entity
vim internal/domain/lesson/student_entity.go
# Add: GradeLevel string

# 2. Create migration
vim migrations/015_add_student_grade.up.sql
# ALTER TABLE students ADD COLUMN grade_level VARCHAR(50);

# 3. Update repository
vim internal/repository/lesson/student_postgres.go
# Add grade_level to queries

# 4. Update DTO
vim internal/usecase/lesson/dto.go
# Add GradeLevel to StudentResponse

# 5. Test
go test ./internal/domain/lesson -v

# 6. Run migration
make migrate-up

# 7. Restart server
make dev
```

---

## 6. Testing Guide

### Test Levels

```
Unit Tests          (Domain layer)
    ↓
Integration Tests   (Repository + Database)
    ↓
E2E Tests           (HTTP → Service → DB)
```

### Example Unit Test

```go
// File: internal/domain/lesson/student_test.go
package lesson

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestNewStudent(t *testing.T) {
    // Arrange
    email := "john@example.com"

    // Act
    student, err := NewStudent("John", "Doe", email)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, student)
    assert.Equal(t, email, student.Email)
    assert.Equal(t, StudentStatusActive, student.Status)
}

func TestStudent_Suspend(t *testing.T) {
    student, _ := NewStudent("John", "Doe", "j@e.com")

    err := student.Suspend("Non-payment")

    assert.NoError(t, err)
    assert.Equal(t, StudentStatusSuspended, student.Status)
}
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test -v ./internal/domain/lesson

# Watch mode (with entr)
find . -name "*.go" | entr go test ./...
```

#### Tenant Provisioning İçin Kontrol Listesi

- `internal/usecase/tenant/service_test.go` içine plan bazlı provisioning için senaryolar ekleyin: aynı slug/email çakışması, idempotent schema creation, provisioning hatasında rollback.
- Geliştirme ortamında yeni tenant açtıktan sonra `make db-shell` ile `SELECT schema_name FROM tenants WHERE slug = '<slug>';` ve `\dn tenant_*` sorgularıyla şemayı doğrulayın.
- Enterprise planı için database-per-tenant tek seferlik smoke testi yapmak amacıyla `ProvisioningService` üzerine yazılacak yeni fonksiyonlar için `scripts/run_migrations.sh`'i parametrik halde çağıran entegrasyon testi hazırlayın.

---

## 7. Common Tasks

### Task 1: Add New Entity

**Example: Add `Course` entity**

```bash
# 1. Create domain entity
touch internal/domain/lesson/course_entity.go

# 2. Create repository interface + implementation
touch internal/domain/lesson/course_repository.go
touch internal/repository/lesson/course_postgres.go

# 3. Create service
touch internal/usecase/lesson/course_service.go

# 4. Create handler
touch internal/handler/lesson/course_handler.go

# 5. Create migration
touch migrations/016_create_courses.up.sql
touch migrations/016_create_courses.down.sql

# 6. Register in app/app.go and app/routes.go
```

### Task 2: Debug API Error

```bash
# 1. Check logs
docker-compose logs -f api

# 2. Check database
docker exec -it postgres psql -U postgres -d keystone_dev
\dt                              # List tables
SELECT * FROM students LIMIT 5;  # Check data

# 3. Test endpoint
curl -X POST http://localhost:8080/api/v1/students \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"first_name":"John","email":"j@e.com"}'
```

### Task 3: Add New Module

**Steps to add "Courses" module:**

1. **Domain:** Create `Course` entity with validation
2. **Migration:** Create `courses` table (no `tenant_id` needed)
3. **Repository:** Implement CRUD operations
4. **Service:** Add business logic
5. **Handler:** Create HTTP endpoints
6. **Routes:** Register endpoints in `app/routes.go`
7. **Tests:** Write unit + integration tests

**Checklist:**
- [ ] Domain entity created
- [ ] Repository interface defined
- [ ] PostgreSQL implementation done
- [ ] Migration files created
- [ ] Service layer implemented
- [ ] Handler endpoints added
- [ ] Routes registered
- [ ] Tests written (>80% coverage)
- [ ] `go build ./...` succeeds

---

## 🎯 Learning Path

### Week 1: Foundations
- [ ] Learn Go basics (variables, functions, structs)
- [ ] Understand Clean Architecture
- [ ] Read existing code (domain → repository → service → handler)
- [ ] Run the project locally

### Week 2: Hands-On
- [ ] Add a new field to existing entity
- [ ] Write unit tests
- [ ] Create a new endpoint
- [ ] Test with Postman/curl

### Week 3: Advanced
- [ ] Add a new module (complete CRUD)
- [ ] Understand Schema-per-Tenant multi-tenancy
- [ ] Write integration tests for tenant isolation
- [ ] Deploy with Docker

---

## 📖 Resources

### Go Language
- **Official Tour:** https://go.dev/tour/
- **Go by Example:** https://gobyexample.com/
- **Effective Go:** https://go.dev/doc/effective_go

### Clean Architecture
- **Uncle Bob's Article:** https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
- **Practical Guide:** https://www.oreilly.com/library/view/clean-architecture/9780134494272/

### PostgreSQL
- **Tutorial:** https://www.postgresqltutorial.com/
- **SQL Practice:** https://www.hackerrank.com/domains/sql

### Docker
- **Get Started:** https://docs.docker.com/get-started/
- **Compose:** https://docs.docker.com/compose/

---

## 🆘 Troubleshooting

### Build Fails

```bash
# Clean and rebuild
go clean -modcache
go mod tidy
go build ./...
```

### Database Connection Error

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Restart database
docker-compose restart postgres

# Check connection
docker exec -it postgres psql -U postgres
```

### Migration Error

```bash
# Check migration status
make migrate-status

# Force to version
migrate -path migrations -database "postgres://..." force 5

# Rollback and retry
make migrate-down
make migrate-up
```

### JWT Token Issues

```bash
# Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123","first_name":"Test","last_name":"User"}'

# Login to get token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"test123"}'

# Use token
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <TOKEN>"
```

---

## ✅ Quick Reference

### Environment Setup

```bash
# Install Go
brew install go              # macOS
sudo apt install golang-go   # Linux

# Install Docker
# Download from docker.com

# Clone project
git clone <repo-url>
cd keystone

# Start development
cp .env.example .env
docker-compose up -d
make migrate-up
make dev
```

### Common File Locations

```
Entity:     internal/domain/{module}/{entity}_entity.go
Repository: internal/repository/{module}/{entity}_postgres.go
Service:    internal/usecase/{module}/{entity}_service.go
Handler:    internal/handler/{module}/{entity}_handler.go
Migration:  migrations/XXX_create_{table}.up.sql
Test:       *_test.go (same directory as code)
```

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/add-courses-module

# Make changes...

# Commit
git add .
git commit -m "feat: add Courses module with CRUD operations"

# Push
git push origin feature/add-courses-module
```

---

## 🎓 Next Steps

After mastering the basics:

1. **Read IMPLEMENTATION_ROADMAP.md** for detailed plan
2. **Implement a full module** (Domain → Repository → Service → Handler)
3. **Write comprehensive tests** (aim for >80% coverage)
4. **Understand the Schema-per-Tenant** architecture
5. **Implement enterprise modules** (HMS, ERP, LMS)

---

**Good luck! 🚀**

**Remember:** Start small, test often, commit frequently.

---

**Last Updated:** January 2025
**Version:** 2.0
**For Questions:** See README.md or IMPLEMENTATION_ROADMAP.md
