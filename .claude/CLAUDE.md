# NexSpaces API - Claude Code Project Guidelines

> **Multi-Tenant SaaS Platform Backend (Go + Fiber + PostgreSQL)**
> **Architecture:** Clean Architecture + Schema-per-Tenant Isolation
> **Level:** Senior/Principal Engineer Expectations

---

## 🎯 Project Mission

NexSpaces is an enterprise-grade multi-tenant SaaS platform supporting vertical-specific solutions (LMS, HMS, ERP, E-commerce, CMS) through a unified schema-per-tenant architecture. Every code change must prioritize:

1. **Tenant Isolation** - Zero cross-tenant data leakage tolerance
2. **Clean Architecture** - Strict layer separation and dependency rules
3. **Production Safety** - Every commit must be production-ready
4. **Performance** - P95 <200ms API response time
5. **Security** - SOC 2 Type II & GDPR compliance by design

---

## 🏗️ Architecture Principles

### Clean Architecture Layers (Strict Enforcement)

```
┌─────────────────────────────────────────────────┐
│ Presentation Layer (HTTP Handlers)             │
│ - Fiber handlers only                           │
│ - Input validation (struct tags)                │
│ - Response formatting                           │
│ - NO business logic                             │
└─────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│ Use Case Layer (Business Logic)                │
│ - Service interfaces and implementations        │
│ - Transaction orchestration                     │
│ - Domain logic coordination                     │
│ - NO database details, NO HTTP concerns         │
└─────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│ Domain Layer (Entities & Business Rules)       │
│ - Entity definitions                            │
│ - Domain logic methods                          │
│ - Validation rules                              │
│ - PURE - no external dependencies              │
└─────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│ Infrastructure Layer (Repository)               │
│ - Database operations (PostgreSQL)              │
│ - External service integrations                 │
│ - Cache operations (Redis)                      │
│ - Implements interfaces from use case layer     │
└─────────────────────────────────────────────────┘
```

**Dependency Rule:** Inner layers NEVER depend on outer layers. Use case layer defines interfaces, infrastructure implements them.

---

## 🔒 Multi-Tenant Architecture (CRITICAL)

### Schema-per-Tenant Strategy

Every tenant has a dedicated PostgreSQL schema (`tenant_acme`, `tenant_xyz`). This provides:
- Strong logical isolation
- Security compliance (SOC 2, HIPAA, GDPR)
- Per-tenant data management
- Scalable to database-per-tenant if needed

### Mandatory Tenant Context

```go
// ✅ CORRECT: Every request MUST have tenant context
type RequestContext struct {
    TenantID   string `validate:"required"`
    SchemaName string `validate:"required"`
    UserID     string `validate:"required"`
    Roles      []string
}

// ❌ WRONG: Never write queries without tenant scoping
db.Query("SELECT * FROM users WHERE email = $1", email)

// ✅ CORRECT: Schema scoping via search_path
db.Exec("SET search_path TO tenant_acme, public")
db.Query("SELECT * FROM users WHERE email = $1", email)
```

### Schema Provisioning Workflow

1. **Tenant Creation** → Generate unique `schema_name` via `generate_schema_name()`
2. **Schema Creation** → `CREATE SCHEMA tenant_xyz`
3. **Template Application** → Run plan-specific SQL template (LMS, HMS, etc.)
4. **Metadata Update** → Store `schema_name` in tenants table
5. **Validation** → Verify schema integrity with integration tests

**File References:**
- Migration: `migrations/025_add_tenant_schema_support.up.sql`
- Provisioning: `internal/usecase/tenant/provisioning.go`
- Service: `internal/usecase/tenant/service.go`

---

## 💎 Code Quality Standards

### DRY (Don't Repeat Yourself)

```go
// ❌ BAD: Code duplication
func GetUserByEmail(email string) (*User, error) {
    if email == "" {
        return nil, errors.New("email cannot be empty")
    }
    // ... database logic
}

func GetUserByID(id string) (*User, error) {
    if id == "" {
        return nil, errors.New("id cannot be empty")
    }
    // ... database logic
}

// ✅ GOOD: Extract common validation
func validateNonEmpty(value, fieldName string) error {
    if value == "" {
        return fmt.Errorf("%s cannot be empty", fieldName)
    }
    return nil
}

// Rule: Extract to function after ≥3 repetitions
```

### KISS (Keep It Simple, Stupid)

```go
// ❌ BAD: Over-engineered
type UserRepositoryFactoryProvider interface {
    GetFactory() UserRepositoryFactory
}

// ✅ GOOD: Simple and direct
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
}

// Rule: Simplest solution that works. No premature abstraction.
```

### YAGNI (You Aren't Gonna Need It)

```go
// ❌ BAD: Future-proofing that's never used
type User struct {
    ID       string
    Email    string
    // Future features nobody asked for:
    SocialProfiles map[string]string `json:"social_profiles,omitempty"`
    Preferences    map[string]any    `json:"preferences,omitempty"`
    Metadata       map[string]any    `json:"metadata,omitempty"`
}

// ✅ GOOD: Only what's needed NOW
type User struct {
    ID       string `json:"id"`
    Email    string `json:"email" validate:"required,email"`
    TenantID string `json:"tenant_id" validate:"required"`
    Name     string `json:"name" validate:"required"`
}

// Rule: Don't add fields/features until there's a concrete requirement
```

### Single Responsibility Principle

```go
// ❌ BAD: God service with too many responsibilities
type UserService struct {
    db    *sql.DB
    cache *redis.Client
    email EmailService
}

func (s *UserService) CreateUser(req CreateUserRequest) error {
    // 1. Validation
    // 2. Database insert
    // 3. Cache update
    // 4. Send welcome email
    // 5. Analytics tracking
    // 6. Audit logging
    // TOO MANY RESPONSIBILITIES!
}

// ✅ GOOD: Separated concerns
type UserService struct {
    repo      UserRepository      // Database operations
    cache     CacheService        // Cache management
    events    EventBus            // Event publishing
    validator UserValidator       // Validation
}

func (s *UserService) CreateUser(req CreateUserRequest) error {
    if err := s.validator.Validate(req); err != nil {
        return err
    }

    user, err := s.repo.Create(req)
    if err != nil {
        return err
    }

    // Publish event for async operations (email, analytics)
    s.events.Publish("user.created", user)

    return nil
}

// Rule: Each service/function should have ONE reason to change
```

---

## 🧪 Testing Standards

### Test Coverage Requirements

| Layer | Minimum Coverage | Target |
|-------|-----------------|--------|
| Domain | 70% | 85% |
| Use Case (Service) | 75% | 90% |
| Repository | 60% | 80% |
| Handlers | 50% | 70% |
| **Overall** | **65%** | **80%+** |

### Critical Test Scenarios (MANDATORY)

```go
// 1. Tenant Isolation Tests
func TestCrossSchemaAccessPrevention(t *testing.T) {
    tenantA := createTestTenant("Tenant A")
    tenantB := createTestTenant("Tenant B")

    // Tenant A creates data in its schema
    dataA := createDataInSchema(tenantA.SchemaName, "sensitive-data")

    // Tenant B tries to access Tenant A's data
    _, err := getDataFromSchema(tenantB.SchemaName, dataA.ID)

    // MUST fail - this is our security guarantee
    assert.Error(t, err)
    assert.Equal(t, ErrNotFound, err)
}

// 2. Schema Provisioning Tests
func TestTenantProvisioningRollback(t *testing.T) {
    tenant := &Tenant{Name: "Test Corp"}

    // Simulate provisioning failure
    err := provisioningService.ProvisionTenant(tenant)

    // Schema should be cleaned up on failure
    schemaExists := checkSchemaExists(tenant.SchemaName)
    assert.False(t, schemaExists, "Failed provisioning left orphan schema")
}

// 3. Idempotency Tests
func TestCreateTenantIdempotency(t *testing.T) {
    req := CreateTenantRequest{Name: "Same Corp", Slug: "same"}

    tenant1, err1 := service.CreateTenant(req)
    assert.NoError(t, err1)

    // Second call should not create duplicate
    tenant2, err2 := service.CreateTenant(req)

    // Should either return same tenant or return conflict error
    if err2 == nil {
        assert.Equal(t, tenant1.ID, tenant2.ID)
    } else {
        assert.Equal(t, ErrConflict, err2)
    }
}
```

### Table-Driven Tests (Preferred)

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name        string
        user        User
        expectError bool
        errorCode   string
    }{
        {
            name: "valid user",
            user: User{Email: "test@example.com", TenantID: "tenant-1"},
            expectError: false,
        },
        {
            name: "missing email",
            user: User{Email: "", TenantID: "tenant-1"},
            expectError: true,
            errorCode: "MISSING_EMAIL",
        },
        {
            name: "missing tenant_id",
            user: User{Email: "test@example.com", TenantID: ""},
            expectError: true,
            errorCode: "MISSING_TENANT_ID",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateUser(tt.user)
            if tt.expectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorCode)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// Rule: Use table-driven tests for multiple scenarios
```

---

## 🔐 Security Guidelines

### Input Validation

```go
// ✅ MANDATORY: Validate ALL inputs
type CreateTenantRequest struct {
    Name     string `json:"name" validate:"required,min=3,max=100"`
    Slug     string `json:"slug" validate:"required,lowercase,alphanum"`
    Email    string `json:"email" validate:"required,email"`
    PlanID   string `json:"plan_id" validate:"required,uuid"`
}

// Use validator in handlers
func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
    var req CreateTenantRequest

    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
    }

    if err := h.validator.Struct(&req); err != nil {
        return formatValidationErrors(err)
    }

    // ... proceed with business logic
}
```

### SQL Injection Prevention

```go
// ❌ WRONG: String concatenation (SQL injection risk)
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
db.Query(query)

// ✅ CORRECT: Always use parameterized queries
db.Query("SELECT * FROM users WHERE email = $1", email)

// ✅ CORRECT: Use query builder if complex
sq.Select("*").From("users").Where(sq.Eq{"email": email})
```

### Sensitive Data Handling

```go
// ❌ WRONG: Logging sensitive data
log.Printf("User created: %+v", user) // Contains passwords, tokens

// ✅ CORRECT: Redact sensitive fields
log.Printf("User created: id=%s, email=%s", user.ID, maskEmail(user.Email))

// ✅ CORRECT: Use structured logging with PII masking
logger.Info("user_created",
    zap.String("user_id", user.ID),
    zap.String("email", maskEmail(user.Email)),
    zap.String("tenant_id", user.TenantID),
)
```

### Error Messages

```go
// ❌ BAD: Leaks implementation details
return fmt.Errorf("database error: %v", err) // Exposes DB structure

// ✅ GOOD: Generic user-facing message, log details internally
logger.Error("database_error", zap.Error(err))
return errors.New("failed to process request")

// ✅ GOOD: Use error codes for client handling
return &AppError{
    Code:    "TENANT_NOT_FOUND",
    Message: "The requested tenant does not exist",
    Status:  404,
}
```

---

## 📊 Performance Standards

### Database Query Optimization

```go
// ❌ BAD: N+1 query problem
func GetUsersWithRoles(tenantID string) ([]UserWithRoles, error) {
    users, _ := repo.GetUsersByTenant(tenantID)

    for _, user := range users {
        // N+1 - Queries in loop!
        roles, _ := repo.GetUserRoles(user.ID)
        result = append(result, UserWithRoles{User: user, Roles: roles})
    }
    return result, nil
}

// ✅ GOOD: Single batch query
func GetUsersWithRoles(tenantID string) ([]UserWithRoles, error) {
    users, _ := repo.GetUsersByTenant(tenantID)

    userIDs := extractIDs(users)
    rolesByUserID, _ := repo.GetRolesByUserIDs(userIDs) // Single query

    for _, user := range users {
        result = append(result, UserWithRoles{
            User:  user,
            Roles: rolesByUserID[user.ID],
        })
    }
    return result, nil
}
```

### Index Strategy

```sql
-- ❌ BAD: Index without tenant_id
CREATE INDEX idx_users_email ON users(email);

-- ✅ GOOD: Tenant-scoped index (tenant_id first)
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);

-- ✅ GOOD: Composite index for common queries
CREATE INDEX idx_lessons_tenant_status_date
    ON lessons(tenant_id, status, start_time DESC);
```

### Caching Strategy

```go
// ✅ Multi-level caching with tenant isolation
func GetTenantSettings(tenantID string) (*Settings, error) {
    // L1: Application-level cache (5 min TTL)
    cacheKey := fmt.Sprintf("tenant:%s:settings", tenantID)

    if cached, found := appCache.Get(cacheKey); found {
        return cached.(*Settings), nil
    }

    // L2: Redis cache (15 min TTL)
    if cached, err := redis.Get(ctx, cacheKey).Result(); err == nil {
        settings := &Settings{}
        json.Unmarshal([]byte(cached), settings)
        appCache.Set(cacheKey, settings, 5*time.Minute)
        return settings, nil
    }

    // L3: Database
    settings, err := repo.GetSettings(tenantID)
    if err != nil {
        return nil, err
    }

    // Cache for future requests
    json, _ := json.Marshal(settings)
    redis.Set(ctx, cacheKey, json, 15*time.Minute)
    appCache.Set(cacheKey, settings, 5*time.Minute)

    return settings, nil
}

// Rule: Cache keys MUST include tenant_id for isolation
```

---

## 🚀 Development Workflow

### Git Commit Standards

```bash
# Use conventional commits (no Co-Authored-By footer)

# Format: <type>(<scope>): <subject>

# Types:
feat:     New feature
fix:      Bug fix
refactor: Code change that neither fixes bug nor adds feature
perf:     Performance improvement
test:     Adding or updating tests
docs:     Documentation changes
chore:    Build process, dependencies, tooling

# Examples:
feat(tenant): add schema-per-tenant provisioning service
fix(auth): prevent cross-tenant session leakage
refactor(user): extract validation logic to separate service
perf(db): optimize tenant-scoped queries with composite indexes
test(tenant): add integration tests for schema isolation
docs(api): update OpenAPI spec with new endpoints
```

### Pre-Commit Checklist

Before every commit, verify:

- [ ] **Tests pass:** `make test`
- [ ] **Linting clean:** `make lint`
- [ ] **Build successful:** `make build`
- [ ] **No sensitive data:** Check for secrets, tokens, passwords
- [ ] **Tenant isolation:** New code respects schema boundaries
- [ ] **Error handling:** All errors are properly wrapped and logged
- [ ] **Documentation:** Update docs if API/behavior changes

### Migration Safety

```sql
-- ✅ SAFE: Backward-compatible changes
ALTER TABLE tenants ADD COLUMN new_field TEXT; -- Nullable

-- ✅ SAFE: Create index concurrently (no locks)
CREATE INDEX CONCURRENTLY idx_users_created_at ON users(created_at);

-- ❌ UNSAFE: Breaking change without migration path
ALTER TABLE tenants DROP COLUMN important_field;

-- ✅ SAFE: Expand-Contract pattern
-- Step 1 (Deploy 1): Add new column
ALTER TABLE tenants ADD COLUMN new_field TEXT;

-- Step 2 (Deploy 2): Migrate data, update app to use new field
-- Step 3 (Deploy 3): Drop old column
ALTER TABLE tenants DROP COLUMN old_field;
```

### Seed Data Protection

```go
// ✅ MANDATORY: Prevent seed data in production
func SeedDatabase(db *sql.DB) error {
    env := os.Getenv("ENVIRONMENT")
    if env == "production" || env == "prod" {
        return errors.New("FATAL: seed data cannot run in production")
    }

    // Safe to seed in dev/staging
    return runSeedScripts(db)
}
```

---

## 📝 Documentation Standards

### Code Comments (WHY, not WHAT)

```go
// ❌ BAD: Comments state the obvious
// Increment counter by 1
counter++

// ❌ BAD: Parroting the code
// CreateUser creates a new user
func CreateUser(req CreateUserRequest) error {

// ✅ GOOD: Explains WHY and business context
// We use exponential backoff to avoid overwhelming payment provider APIs
// during high traffic. Max 3 retries with 1s → 2s → 4s delays.
func callPaymentAPI(data APIRequest) error {
    return retry.Do(
        func() error { return paymentAPI.Charge(data) },
        retry.Attempts(3),
        retry.Delay(1*time.Second),
        retry.DelayType(retry.BackOffDelay),
    )
}
```

### Function Documentation

```go
// CreateTenant provisions a new tenant with dedicated schema isolation.
//
// This function performs the following operations:
// 1. Generates a unique schema name based on tenant name
// 2. Creates a PostgreSQL schema for tenant data isolation
// 3. Applies plan-specific SQL template (LMS, HMS, etc.)
// 4. Updates tenant record with schema_name
//
// Schema Isolation:
// Each tenant's data resides in a separate PostgreSQL schema (e.g., tenant_acme).
// This ensures zero cross-tenant data leakage and supports compliance requirements
// (SOC 2, HIPAA, GDPR).
//
// Parameters:
//   - ctx: Request context with timeout and cancellation
//   - req: Tenant creation request (validated)
//
// Returns:
//   - *Tenant: Created tenant with schema_name populated
//   - error: ErrConflict if tenant exists, ErrProvisioningFailed if schema creation fails
//
// Example:
//   tenant, err := service.CreateTenant(ctx, CreateTenantRequest{
//       Name: "Acme Corp",
//       Slug: "acme",
//       PlanID: "lms-basic",
//   })
func (s *TenantService) CreateTenant(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
    // Implementation...
}
```

### OpenAPI Updates

When adding/modifying endpoints, update `api/openapi.yaml`:

```yaml
paths:
  /api/v1/tenants:
    post:
      summary: Create new tenant
      description: |
        Provisions a new tenant with dedicated schema isolation.
        Creates PostgreSQL schema and applies plan-specific template.
      operationId: createTenant
      tags:
        - Tenants
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTenantRequest'
      responses:
        '201':
          description: Tenant created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Tenant'
        '400':
          $ref: '#/components/responses/BadRequest'
        '409':
          $ref: '#/components/responses/Conflict'
```

---

## 🎯 Quick Reference: Common Patterns

### Error Handling

```go
// Wrap errors with context
if err := repo.Create(user); err != nil {
    return fmt.Errorf("failed to create user in tenant %s: %w", tenantID, err)
}

// Sentinel errors for business logic
var (
    ErrNotFound      = errors.New("resource not found")
    ErrConflict      = errors.New("resource already exists")
    ErrUnauthorized  = errors.New("unauthorized access")
)
```

### Context Propagation

```go
// Always pass context through the stack
func (s *Service) Method(ctx context.Context, params) error {
    // Extract tenant from context
    tenantID := ctx.Value("tenant_id").(string)

    // Pass context to repository
    return s.repo.Operation(ctx, tenantID, params)
}
```

### Resource Cleanup

```go
// Always use defer for cleanup
func ProcessFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close() // ✅ Guaranteed cleanup

    conn, err := db.Connect()
    if err != nil {
        return err
    }
    defer conn.Close() // ✅ Guaranteed cleanup

    // ... processing logic
}
```

---

## 🚨 Anti-Patterns (NEVER DO THIS)

```go
// ❌ Hardcoded credentials
const dbPassword = "my-secret-password"

// ❌ Panic in library code (handlers can panic, libraries should return errors)
func ProcessData(data string) {
    if data == "" {
        panic("data cannot be empty") // WRONG
    }
}

// ❌ Ignoring errors
db.Exec("UPDATE users SET status = 'active'") // No error check

// ❌ Global mutable state
var currentTenantID string // Race condition nightmare

// ❌ Database logic in handlers
func (h *Handler) GetUser(c *fiber.Ctx) error {
    // WRONG: Database query in handler
    row := h.db.QueryRow("SELECT * FROM users WHERE id = $1", id)
    // Should call service layer instead
}

// ❌ Business logic in entities
type User struct {
    ID    string
    Email string
}

func (u *User) SendWelcomeEmail() error {
    // WRONG: Entities should be pure data structures
    // Email sending is infrastructure concern
}
```

---

## 📦 Project Structure Reference

```
nexpaces-api/
├── cmd/
│   └── server/              # Application entry point
│       └── main.go
├── internal/
│   ├── domain/              # Entities (pure, no dependencies)
│   │   ├── tenant/
│   │   ├── user/
│   │   └── lesson/
│   ├── usecase/             # Business logic (services)
│   │   ├── tenant/
│   │   │   ├── service.go
│   │   │   ├── provisioning.go
│   │   │   └── dto.go
│   │   └── user/
│   ├── repository/          # Data access (implements interfaces)
│   │   ├── postgres/
│   │   └── redis/
│   └── handlers/            # HTTP handlers (presentation)
│       ├── tenant_handler.go
│       └── user_handler.go
├── pkg/                     # Shared utilities (reusable)
│   ├── database/
│   ├── validator/
│   └── logger/
├── migrations/              # Database migrations (sequential)
│   ├── 001_initial.up.sql
│   └── 025_add_tenant_schema_support.up.sql
├── api/
│   └── openapi.yaml         # API specification
├── docs/                    # Project documentation
│   ├── analysis/
│   ├── guides/
│   └── planning/
└── scripts/
    └── seed/                # Development seed data only
```

---

## 🎓 Learning Resources

### Internal Documentation
- **Schema Architecture:** `migrations/025_add_tenant_schema_support.up.sql`
- **Provisioning Logic:** `internal/usecase/tenant/provisioning.go`
- **API Specification:** `api/openapi.yaml`
- **Comprehensive Analysis:** `docs/analysis/KAPSAMLI_ANALIZ.md`
- **Implementation Roadmap:** `docs/planning/UYGULAMA_YOL_HARITASI.md`

### External References
- **Clean Architecture:** Robert C. Martin (Uncle Bob)
- **Go Best Practices:** Effective Go, Go Code Review Comments
- **Multi-Tenancy:** "Multi-Tenant Data Architecture" by Microsoft
- **PostgreSQL Performance:** PostgreSQL Performance Tuning Guide

---

## ✅ Definition of Done

A task is complete when:

1. **Code Quality**
   - [ ] Follows clean architecture principles
   - [ ] DRY, KISS, YAGNI applied
   - [ ] No linting errors
   - [ ] No code duplication >3 lines

2. **Testing**
   - [ ] Unit tests written (coverage meets targets)
   - [ ] Integration tests for multi-tenant scenarios
   - [ ] All tests passing
   - [ ] Edge cases covered

3. **Security**
   - [ ] Input validation implemented
   - [ ] Tenant isolation verified
   - [ ] No sensitive data in logs
   - [ ] SQL injection protected

4. **Documentation**
   - [ ] Code comments explain WHY
   - [ ] OpenAPI spec updated
   - [ ] README updated if needed
   - [ ] Migration documented

5. **Performance**
   - [ ] No N+1 queries
   - [ ] Database indexes optimized
   - [ ] Cache strategy implemented
   - [ ] P95 <200ms verified

6. **Production Ready**
   - [ ] Error handling complete
   - [ ] Logging structured
   - [ ] Migrations backward-compatible
   - [ ] No hardcoded values

---

**Last Updated:** October 2025
**Version:** 1.0
**Maintained By:** NexSpaces Engineering Team

*This document is the single source of truth for development standards. All code must comply with these guidelines.*
