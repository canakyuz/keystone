# Keystone API - Complete Implementation Roadmap

> **Multi-Vertical Enterprise SaaS Platform**
> Supporting Blog, E-commerce, LMS, and more with a unified Schema-per-Tenant architecture.

---

## 📊 Current Status (Updated: January 2025)

### ✅ Completed Phases

**Phase 1-8:** Foundation, Core Modules & API Documentation ✅ **100%**
- Clean Architecture setup
- Multi-tenant infrastructure foundation
- Authentication & Authorization
- Core business modules (Projects, Lessons, Blog, etc.)
- OpenAPI 3.0 specification with Swagger UI

### 📈 Build & Test Status

- **Build:** ✅ Successful (12MB binary)
- **Endpoints:** 60+ REST endpoints
- **Test Coverage:**
  - Domain layer: 36.8%
  - **Target:** 80%+
- **Database:** PostgreSQL (Schema-per-Tenant Model)
- **Docker:** Multi-stage optimized

---

## 🎯 Architectural Foundation: Schema-per-Tenant

### The Challenge

While a single shared database is simple, it presents significant challenges for a true multi-vertical SaaS platform regarding data isolation, security, compliance, and performance. A more robust model is required.

### The Solution: **Unified Schema-per-Tenant Architecture**

The entire platform is built on a **Schema-per-Tenant** model. This strategy provides strong logical data isolation for all tenants, ensuring security and compliance while maintaining a manageable and scalable infrastructure.

**How It Works:**
1.  **Provisioning:** When a new tenant is created, the system provisions a dedicated schema for them within the main PostgreSQL database (e.g., `CREATE SCHEMA tenant_acme;`).
2.  **Templating:** Based on the subscribed plan (e.g., LMS), a corresponding SQL template is executed to create all necessary tables (`courses`, `students`, etc.) inside the new schema.
3.  **Connection Scoping:** The API backend identifies the tenant from the incoming request and sets the database connection's `search_path` to the tenant's schema. All subsequent queries are automatically and safely scoped to that tenant's data.

This model eliminates the need for complex, tiered logic and provides a consistent, secure, and scalable foundation for all verticals.

---

## 🏗️ PHASE 9: SCHEMA-PER-TENANT ARCHITECTURE

**Priority:** 🔴 CRITICAL | **Time:** 1-2 weeks

### 9.1 Database Schema Updates

**Migration:** `014_add_tenant_schema_support.up.sql`

```sql
-- Add schema name to tenants table for easy lookup
ALTER TABLE tenants
ADD COLUMN schema_name VARCHAR(63) UNIQUE;

-- Create a function to generate a unique schema name
CREATE OR REPLACE FUNCTION generate_schema_name(name TEXT) RETURNS TEXT AS $$
DECLARE
  slug TEXT;
  schema_name TEXT;
  counter INT := 0;
BEGIN
  slug := lower(regexp_replace(name, '[^a-zA-Z0-9_]+', '', 'g'));
  schema_name := 'tenant_' || slug;
  -- Check for uniqueness and append a number if needed
  WHILE EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = schema_name) LOOP
    counter := counter + 1;
    schema_name := 'tenant_' || slug || '_' || counter;
  END LOOP;
  RETURN schema_name;
END;
$$ LANGUAGE plpgsql;

-- Remove tenant_id from tables that will now be inside tenant schemas
-- Example for the 'projects' table:
-- ALTER TABLE projects DROP COLUMN tenant_id;
-- Note: This will be done within the schema templates themselves.
```

### 9.2 Tenant Connection & Session Scoping

**File:** `pkg/database/tenant_manager.go`

```go
// TenantManager is responsible for setting the session's search_path.
type TenantManager struct {
    db *sql.DB
}

// SetTenantScope configures the database connection for a specific tenant.
func (m *TenantManager) SetTenantScope(ctx context.Context, schemaName string) error {
    conn, err := m.db.Conn(ctx)
    if err != nil {
        return err
    }
    defer conn.Close()

    // Set the search_path for the duration of the request.
    // This is the core of our multi-tenancy isolation.
    _, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", pq.QuoteIdentifier(schemaName)))
    return err
}
```

### 9.3 Tenant Provisioning Service

**File:** `internal/usecase/tenant/provision.go`

```go
// TenantProvisioningService handles the creation of new tenant schemas.
type TenantProvisioningService struct {
    db *sql.DB
    templateRepo TemplateRepository // To fetch schema templates
}

// ProvisionNewTenant creates a new schema and runs the appropriate template.
func (s *TenantProvisioningService) ProvisionNewTenant(ctx context.Context, tenantID, tenantName, plan string) (string, error) {
    // 1. Generate a unique schema name
    var schemaName string
    err := s.db.QueryRowContext(ctx, "SELECT generate_schema_name($1)", tenantName).Scan(&schemaName)
    if err != nil {
        return "", err
    }

    // 2. Create the schema
    _, err = s.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA %s", pq.QuoteIdentifier(schemaName)))
    if err != nil {
        return "", err
    }

    // 3. Get the schema template SQL for the given plan
    templateSQL, err := s.templateRepo.GetTemplateByPlan(ctx, plan)
    if err != nil {
        return "", err
    }

    // 4. Execute the template SQL within the new schema
    // The SQL script must be written to assume it's running in the new schema.
    _, err = s.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s; %s", pq.QuoteIdentifier(schemaName), templateSQL))
    if err != nil {
        // Rollback: DROP SCHEMA
        return "", err
    }

    // 5. Update the tenant record with the new schema name
    _, err = s.db.ExecContext(ctx, "UPDATE tenants SET schema_name = $1 WHERE id = $2", schemaName, tenantID)
    return schemaName, err
}
```

### 9.4 Deliverables

- [ ] Migration 014: Add `schema_name` to tenants table and create helper function.
- [ ] `TenantManager` implementation to set `search_path`.
- [ ] Middleware to call `SetTenantScope` for every authenticated request.
- [ ] `TenantProvisioningService` for creating new tenant schemas.
- [ ] A repository for storing and retrieving schema templates (`.sql` files).
- [ ] Update `CreateTenant` use case to call the provisioning service.
- [ ] Integration tests to verify tenant data isolation between schemas.

---

## 🏥 PHASE 10: ENTERPRISE MODULES

**Priority:** 🔴 HIGH | **Time:** 4-6 weeks

### Module Matrix

| Module | Complexity | Isolation Model | Compliance | Time |
|--------|-----------|-----------------|------------|------|
| **HMS** (Hospital) | Very High | Schema-per-Tenant | HIPAA | 2 weeks |
| **ERP** (Manufacturing) | Very High | Schema-per-Tenant | SOC 2 | 2 weeks |
| **LMS** (Education) | High | Schema-per-Tenant | FERPA | 1 week |
| **OMS** (Orders) | Medium | Schema-per-Tenant | PCI-DSS | 1 week |

### 10.1 HMS (Hospital Management System)

**Tables:**
```sql
-- Patient records (HIPAA-sensitive)
CREATE TABLE patients (
    id UUID PRIMARY KEY,
    medical_record_number VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL ENCRYPTED, -- PHI
    last_name VARCHAR(100) NOT NULL ENCRYPTED,  -- PHI
    date_of_birth DATE ENCRYPTED,               -- PHI
    blood_type VARCHAR(5),
    allergies TEXT,
    emergency_contact JSONB ENCRYPTED,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE medical_records (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL REFERENCES patients(id),
    visit_date TIMESTAMP NOT NULL,
    diagnosis TEXT ENCRYPTED,
    prescription TEXT ENCRYPTED,
    doctor_id UUID REFERENCES users(id),
    notes TEXT ENCRYPTED,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 10.2 ERP (Enterprise Resource Planning)

**Tables:**
```sql
-- Inventory management
CREATE TABLE inventory_items (
    id UUID PRIMARY KEY,
    sku VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    quantity_on_hand INT DEFAULT 0,
    reorder_level INT,
    unit_cost DECIMAL(12,2),
    location VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 10.3 LMS (Learning Management System)

**Enhanced from existing Lessons module:**

```sql
-- Courses (extends existing lessons)
CREATE TABLE courses (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    instructor_id UUID REFERENCES users(id),
    price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Course enrollments
CREATE TABLE enrollments (
    id UUID PRIMARY KEY,
    student_id UUID REFERENCES students(id),
    course_id UUID REFERENCES courses(id),
    enrolled_at TIMESTAMP DEFAULT NOW(),
    progress_percentage INT DEFAULT 0
);
```

### 10.4 OMS (Order Management System)

**Tables:**
```sql
-- Orders
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID REFERENCES customers(id),
    status VARCHAR(20), -- pending, processing, shipped, delivered
    total DECIMAL(12,2),
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🧪 PHASE 11: COMPREHENSIVE TESTING

**Priority:** 🔴 CRITICAL | **Time:** 2-3 weeks

### 11.1 Multi-Tenant Security Tests

**File:** `test/security/tenant_isolation_test.go`

```go
func TestCrossSchemaAccessPrevention(t *testing.T) {
    // Setup: Create two tenants, which creates two schemas (e.g., tenant_a, tenant_b)
    tenantA := createTestTenant("Tenant A", "LMS")
    tenantB := createTestTenant("Tenant B", "LMS")

    // Tenant A creates a course in its own schema (tenant_a.courses)
    courseA := createCourseInSchema(tenantA.SchemaName, "History 101")

    // Tenant B tries to access Tenant A's course
    // This query will be executed with search_path = 'tenant_b, public'
    // It should fail because tenant_a.courses is not in the search path.
    _, err := getCourseFromSchema(tenantB.SchemaName, courseA.ID)

    // MUST fail
    assert.Error(t, err, "Cross-schema access was possible, which is a major security flaw!")
    assert.Equal(t, ErrCourseNotFound, err)
}
```

### 11.2 Compliance Tests

**HIPAA Audit Logging:**
```go
func TestHIPAAAuditLogging(t *testing.T) {
    // This test remains relevant. Ensure that when a user in a HIPAA-compliant
    // tenant accesses a patient record, an audit log is created.
    patient := createPatient(tenantID, "Jane Doe")

    // Access patient record
    getPatient(tenantID, patient.ID)

    // Verify audit log
    logs := getAuditLogs(tenantID, patient.ID)
    assert.NotEmpty(t, logs)
    assert.Equal(t, "patient.view", logs[0].Action)
}
```

### 11.3 Integration Tests

- [ ] Repository integration tests (all modules)
- [ ] Service layer unit tests
- [ ] Handler E2E tests
- [ ] Tenant Provisioning Service tests
- [ ] Schema migration script tests
- [ ] Disaster recovery tests (backup/restore of a single schema)

### 11.4 Test Coverage Targets

| Layer | Current | Target |
|-------|---------|--------|
| Domain | 36.8% | **80%+** |
| Repository | 0% | **70%+** |
| Service | 0% | **80%+** |
| Handler | 0% | **60%+** |
| **Total** | **~10%** | **75%+** |

---

## 🔐 PHASE 12: OAUTH & ADVANCED AUTH

**Priority:** 🟡 MEDIUM | **Time:** 1 week

### 12.1 OAuth Provider Strategy

**Interface:** `internal/domain/auth/oauth_provider.go`

```go
type OAuthProvider interface {
    GetAuthURL(state string) string
    ExchangeCode(code string) (*OAuthToken, error)
    GetUserInfo(token string) (*OAuthUser, error)
}

type OAuthUser struct {
    ProviderUserID string
    Email          string
    FirstName      string
    LastName       string
    AvatarURL      string
}
```

### 12.2 Implementations

**Google OAuth:**
```go
type GoogleProvider struct {
    config *oauth2.Config
}

func (p *GoogleProvider) GetAuthURL(state string) string {
    return p.config.AuthCodeURL(state)
}
```

**GitHub OAuth:**
```go
type GitHubProvider struct {
    config *oauth2.Config
}
```

**Apple Sign In:**
```go
type AppleProvider struct {
    clientID   string
    teamID     string
    keyID      string
    privateKey *ecdsa.PrivateKey
}
```

### 12.3 Endpoints

```bash
GET  /api/v1/auth/google/login
GET  /api/v1/auth/google/callback
GET  /api/v1/auth/github/login
GET  /api/v1/auth/github/callback
POST /api/v1/auth/apple/callback

# Account linking
POST   /api/v1/auth/link/{provider}
GET    /api/v1/auth/connections
DELETE /api/v1/auth/connections/{provider}
```

---

## 💳 PHASE 13: PAYMENT INFRASTRUCTURE

**Priority:** 🟡 MEDIUM | **Time:** 1-2 weeks

### 13.1 Stripe Integration

**Tables:**
```sql
CREATE TABLE payment_customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stripe_customer_id VARCHAR(255) UNIQUE,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    plan_id VARCHAR(50),
    status VARCHAR(50), -- active, cancelled, past_due
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    stripe_invoice_id VARCHAR(255) UNIQUE,
    amount_due INTEGER,
    amount_paid INTEGER,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50),
    pdf_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 13.2 Webhook Handlers

```go
func HandleStripeWebhook(c *fiber.Ctx) error {
    event := stripe.ParseWebhook(c.Body())

    switch event.Type {
    case "invoice.paid":
        activateSubscription(event.Data)
    case "invoice.payment_failed":
        suspendTenant(event.Data)
    case "customer.subscription.deleted":
        cancelSubscription(event.Data)
    }
}
```

### 13.3 Usage-Based Billing

**Metering Middleware:**
```go
func MeteringMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        tenantID := c.Locals("tenant_id").(string)

        // Increment API call counter
        redis.Incr(fmt.Sprintf("usage:%s:api_calls", tenantID))

        // Report to Stripe (daily batch)
        stripeClient.ReportUsage(tenantID, "api_calls", count)

        return c.Next()
    }
}
```

---

## 🚀 PHASE 14: DEPLOYMENT & DEVOPS

**Priority:** 🟡 MEDIUM | **Time:** 1 week

### 14.1 CI/CD Pipeline

**File:** `.github/workflows/main.yml` (or Bitbucket Pipelines)

```yaml
name: CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 75" | bc -l) )); then
            echo "Coverage $coverage% is below 75%"
            exit 1
          fi

  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Build Docker image
        run: docker build -t keystone:${{ github.sha }} .

      - name: Push to registry
        run: docker push keystone:${{ github.sha }}

  deploy:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        run: kubectl set image deployment/api api=keystone:${{ github.sha }}
```

### 14.2 Kubernetes Manifests

**File:** `k8s/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keystone
spec:
  replicas: 3
  selector:
    matchLabels:
      app: keystone
  template:
    metadata:
      labels:
        app: keystone
    spec:
      containers:
      - name: api
        image: keystone:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_HOST
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: host
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
          requests:
            memory: "256Mi"
            cpu: "250m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
```

### 14.3 Monitoring Stack

**Prometheus + Grafana:**
```go
// Metrics endpoint
app.Get("/metrics", func(c *fiber.Ctx) error {
    return c.SendString(prometheus.Handler())
})

// Custom metrics
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status", "tenant_id"},
    )

    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "db_query_duration_seconds",
            Help: "Database query duration",
        },
        []string{"query_type", "table", "tenant_id"},
    )
)
```

---

## 📈 Implementation Timeline

### Critical Path (Next 3 Months)

| Phase | Priority | Duration | Dependencies |
|-------|----------|----------|--------------|
| **9. Schema-per-Tenant Arch.** | 🔴 CRITICAL | 1-2 weeks | None |
| **10. Enterprise Modules** | 🔴 HIGH | 4-6 weeks | Phase 9 |
| **11. Comprehensive Testing** | 🔴 CRITICAL | 2-3 weeks | Phase 9-10 |
| **12. OAuth & Advanced Auth** | 🟡 MEDIUM | 1 week | Phase 11 |
| **13. Payment Infrastructure** | 🟡 MEDIUM | 1-2 weeks | Phase 11 |
| **14. Deployment & DevOps** | 🟡 MEDIUM | 1 week | Phase 11-13 |

**Total Estimated Time:** 10-15 weeks (2.5-3.5 months)

### Week-by-Week Plan

**Week 1-2: Schema-per-Tenant Architecture**
- Migration 014: `schema_name` support
- Tenant Provisioning Service
- Tenant session scoping middleware
- Integration tests for schema isolation

**Week 3-4: HMS Module**
- Patient, MedicalRecord, Appointment entities
- HIPAA compliance features
- Encryption layer
- Audit logging

**Week 5-6: ERP Module**
- Inventory, ProductionOrder, PurchaseOrder entities
- Real-time tracking
- Cost accounting
- MES integration prep

**Week 7: LMS Enhancement**
- Course, Enrollment, CourseVideo entities
- FERPA compliance
- Video integration
- Certificate generation

**Week 8: OMS Module**
- Order, OrderItem, Shipment entities
- PCI-DSS compliance
- Shipping integration
- Payment processing

**Week 9-11: Comprehensive Testing**
- Security tests (cross-schema access)
- Compliance tests (HIPAA, PCI-DSS)
- Performance tests (load, stress)
- Integration tests (E2E)
- Target: 75%+ coverage

**Week 12: OAuth & Auth**
- Google, GitHub, Apple OAuth
- Account linking
- 2FA/MFA (optional)

**Week 13-14: Payment & Billing**
- Stripe integration
- Subscription lifecycle
- Webhook handling
- Usage-based billing

**Week 15: DevOps**
- CI/CD pipeline
- Kubernetes deployment
- Monitoring setup

---

## ✅ Success Criteria

### Phase 9: Schema-per-Tenant Architecture
- [ ] Migration 014 applied successfully.
- [ ] New tenants are provisioned with their own schema.
- [ ] API requests are correctly scoped to the tenant's schema.
- [ ] Integration tests prove zero cross-schema data leakage.
- [ ] Schema templates are created for each vertical.

### Phase 10: Enterprise Modules
- [ ] HMS: Patient CRUD with HIPAA compliance
- [ ] ERP: Inventory tracking functional
- [ ] LMS: Course enrollment working
- [ ] OMS: Order processing end-to-end
- [ ] All modules are fully isolated within their tenant schema.

### Phase 11: Testing
- [ ] Test coverage >75%
- [ ] All security tests passing
- [ ] Compliance tests (HIPAA, PCI-DSS) green
- [ ] Load tests: 10,000 req/s sustained
- [ ] Zero cross-tenant access bugs found.

### Phase 12: OAuth
- [ ] Google, GitHub, Apple login working
- [ ] Account linking functional

### Phase 13: Payment
- [ ] Stripe integration complete
- [ ] Subscription creation/cancellation working
- [ ] Webhook handling tested

### Phase 14: DevOps
- [ ] CI/CD pipeline passing
- [ ] Production deployment successful
- [ ] Monitoring dashboards active

---

## 🔗 External Dependencies

### Go Packages
```bash
# OAuth
go get golang.org/x/oauth2

# Payments
go get github.com/stripe/stripe-go/v76

# Encryption (HIPAA)
go get golang.org/x/crypto/nacl/secretbox

# Monitoring
go get github.com/prometheus/client_golang
```

### External Services
- **Stripe:** Payment processing
- **AWS RDS:** PostgreSQL hosting
- **Redis:** Caching, session storage
- **SendGrid/AWS SES:** Email delivery
- **Cloudflare:** CDN, WAF, DDoS protection

---

## 📝 Environment Variables (Full List)

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ENVIRONMENT=production

# Database
DATABASE_HOST=main-db.amazonaws.com
DATABASE_PORT=5432
DATABASE_NAME=keystone_main
DATABASE_USER=postgres
DATABASE_PASSWORD=***
DATABASE_SSL_MODE=require

# Redis
REDIS_URL=redis://redis-cluster.amazonaws.com:6379

# JWT
JWT_SECRET=***-min-32-chars

# OAuth - Google
GOOGLE_CLIENT_ID=***.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=***
GOOGLE_REDIRECT_URL=https://api.keystone.dev/auth/google/callback

# OAuth - GitHub
GITHUB_CLIENT_ID=***
GITHUB_CLIENT_SECRET=***
GITHUB_REDIRECT_URL=https://api.keystone.dev/auth/github/callback

# Stripe
STRIPE_SECRET_KEY=sk_live_***
STRIPE_WEBHOOK_SECRET=whsec_***

# Encryption (HIPAA)
ENCRYPTION_KEY_MASTER=***-32-bytes

# Monitoring
SENTRY_DSN=https://***@sentry.io/***
PROMETHEUS_PUSHGATEWAY=http://prometheus:9091
```

---

## 🎯 Long-Term Roadmap (6-12 Months)

### Q2 2025: Advanced Features
- [ ] AI-powered template recommendations
- [ ] Multi-language support (i18n)
- [ ] Mobile apps (React Native)
- [ ] Advanced analytics dashboard
- [ ] Template marketplace

### Q3 2025: Enterprise Features
- [ ] SSO (SAML, LDAP)
- [ ] Advanced RBAC with custom roles
- [ ] Audit log exports
- [ ] White-label branding
- [ ] API rate limiting per tenant

### Q4 2025: Scale & Performance
- [ ] Database sharding (tenant-based)
- [ ] Read replicas per region
- [ ] GraphQL API
- [ ] Real-time features (WebSockets)
- [ ] Edge computing (Cloudflare Workers)

---

**Last Updated:** January 2025
**Version:** 2.0
**Maintained By:** Keystone Development Team
