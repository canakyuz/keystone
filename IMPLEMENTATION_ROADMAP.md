# NexSpaces API - Complete Implementation Roadmap

> **Multi-Vertical Enterprise SaaS Platform**
> Supporting Blog, E-commerce, LMS, HMS, ERP, and more with tiered multi-tenancy

---

## 📊 Current Status (Updated: January 2025)

### ✅ Completed Phases

**Phase 1-6:** Foundation & Core Modules ✅ **100%**
- Clean Architecture setup
- Multi-tenant infrastructure
- Authentication & Authorization
- Tenant, User, Website management
- **32+ API endpoints**

**Phase 7:** Additional Modules ✅ **100%**
- ✅ **Projects Module** - Portfolio/Project management
- ✅ **Lessons Module** - Student, Lesson, Assignment (LMS foundation)
- ✅ **Booking Module** - Appointments, Availability (HMS/Services foundation)
- ✅ **Services Module** - Service catalog, pricing
- ✅ **Blog/CMS Module** - Posts, Categories

**Phase 8:** API Documentation ✅ **100%**
- ✅ OpenAPI 3.0 specification
- ✅ Swagger UI integration
- ✅ Request/Response validation middleware
- ✅ Auto-generated client code

### 📈 Build & Test Status

- **Build:** ✅ Successful (12MB binary)
- **Endpoints:** 60+ REST endpoints
- **Test Coverage:**
  - Domain layer: 36.8%
  - **Target:** 80%+
- **Database:** PostgreSQL with RLS
- **Docker:** Multi-stage optimized

---

## 🎯 Architecture Evolution: Tiered Multi-Tenancy

### Current Challenge

**Problem:** Single shared database + RLS works for small tenants, but:
- ❌ **Compliance:** HIPAA/PCI-DSS require physical isolation
- ❌ **Performance:** Large ERP tenant slows down small blog tenants
- ❌ **Scalability:** Cannot handle 500GB+ hospital or factory data

### Solution: **Tiered Isolation Strategy**

```
┌─────────────────────────────────────────────────────────────────┐
│                     Tenant Isolation Tiers                       │
├──────────────┬──────────────────┬──────────────────────────────┤
│   SHARED     │   SCHEMA         │   DEDICATED DATABASE          │
│              │                  │                               │
│ Blog         │ E-commerce       │ HMS (Hospital)                │
│ Website      │ LMS (Education)  │ ERP (Factory)                 │
│ Portfolio    │ Medium business  │ Enterprise                    │
│              │                  │                               │
│ Same DB      │ Same DB,         │ Separate DB                   │
│ + RLS        │ Separate Schema  │ Full isolation                │
│              │                  │                               │
│ $29/mo       │ $199/mo          │ $999/mo+                      │
└──────────────┴──────────────────┴──────────────────────────────┘
```

---

## 🏗️ PHASE 9: TIERED MULTI-TENANCY ARCHITECTURE

**Priority:** 🔴 CRITICAL | **Time:** 1-2 weeks

### 9.1 Database Schema Updates

**Migration:** `014_add_tenant_isolation_tiers.up.sql`

```sql
-- Add isolation configuration to tenants table
ALTER TABLE tenants
ADD COLUMN isolation_level VARCHAR(20) DEFAULT 'shared'
    CHECK (isolation_level IN ('shared', 'schema', 'database', 'dedicated')),
ADD COLUMN database_host VARCHAR(255),
ADD COLUMN database_port INT,
ADD COLUMN database_name VARCHAR(255),
ADD COLUMN schema_name VARCHAR(255),
ADD COLUMN region VARCHAR(50) DEFAULT 'us-east-1',
ADD COLUMN data_residency VARCHAR(50);

-- Module-level configuration
CREATE TABLE tenant_modules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_name VARCHAR(50) NOT NULL, -- 'blog', 'erp', 'hms', 'lms'
    enabled BOOLEAN DEFAULT true,
    isolation_level VARCHAR(20), -- Override tenant-level
    compliance_mode VARCHAR(50), -- 'hipaa', 'pci-dss', 'gdpr'
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, module_name)
);

CREATE INDEX idx_modules_tenant ON tenant_modules(tenant_id);
CREATE INDEX idx_modules_compliance ON tenant_modules(compliance_mode);
```

### 9.2 Tenant Connection Manager

**File:** `pkg/database/tenant_router.go`

```go
type IsolationLevel string

const (
    IsolationShared    IsolationLevel = "shared"    // Blog, websites
    IsolationSchema    IsolationLevel = "schema"    // E-commerce, LMS
    IsolationDatabase  IsolationLevel = "database"  // HMS, ERP
    IsolationDedicated IsolationLevel = "dedicated" // On-premise
)

type TenantConnectionManager struct {
    sharedPool      *sql.DB                      // Shared pool
    schemaPools     map[string]*sql.DB           // Schema-isolated
    dedicatedPools  map[string]*sql.DB           // Dedicated DBs
    configCache     sync.Map                     // Tenant configs
}

func (m *TenantConnectionManager) GetConnection(
    ctx context.Context,
    tenantID string,
) (*sql.DB, error) {
    tenant := m.getTenantConfig(tenantID)

    switch tenant.IsolationLevel {
    case IsolationShared:
        return m.sharedPool, nil
    case IsolationSchema:
        return m.getSchemaPool(tenant.SchemaName)
    case IsolationDatabase, IsolationDedicated:
        return m.getDedicatedPool(tenantID, tenant.DatabaseConfig)
    }
}
```

### 9.3 Repository Layer Adaptation

**Update:** All repositories use connection manager

```go
type BaseRepository struct {
    connManager *TenantConnectionManager
}

func (r *BaseRepository) GetDB(ctx context.Context) (*sql.DB, error) {
    tenantID := ctx.Value("tenant_id").(string)

    // Automatically routes to correct database
    db, err := r.connManager.GetConnection(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    // Set search_path for schema isolation
    tenant := r.connManager.getTenantConfig(tenantID)
    if tenant.IsolationLevel == IsolationSchema {
        db.Exec(fmt.Sprintf("SET search_path TO %s", tenant.SchemaName))
    }

    return db, nil
}
```

### 9.4 Automatic Tier Selection

**File:** `internal/usecase/tenant/onboard.go`

```go
func (uc *TenantUsecase) DetermineIsolationLevel(
    modules []string,
    compliance []string,
    estimatedDataGB int,
) IsolationLevel {
    // HIPAA compliance = dedicated database required
    if slices.Contains(compliance, "hipaa") {
        return IsolationDatabase
    }

    // PCI-DSS = minimum schema isolation
    if slices.Contains(compliance, "pci-dss") {
        return IsolationSchema
    }

    // HMS, ERP modules = dedicated database
    if slices.Contains(modules, "hms") || slices.Contains(modules, "erp") {
        return IsolationDatabase
    }

    // LMS with large data = schema isolation
    if slices.Contains(modules, "lms") && estimatedDataGB > 50 {
        return IsolationSchema
    }

    // Default: shared
    return IsolationShared
}
```

### 9.5 Tenant Migration Tools

**Script:** `scripts/upgrade_tenant_tier.go`

```go
// Upgrade tenant from shared to schema isolation
func UpgradeTenantToSchema(tenantID string) error {
    // 1. Create new schema
    schemaName := fmt.Sprintf("tenant_%s", tenantID[:8])
    db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))

    // 2. Copy all tables
    tables := []string{"users", "posts", "projects", ...}
    for _, table := range tables {
        db.Exec(fmt.Sprintf(`
            CREATE TABLE %s.%s AS
            SELECT * FROM public.%s
            WHERE tenant_id = '%s'
        `, schemaName, table, table, tenantID))
    }

    // 3. Update tenant config
    db.Exec(`
        UPDATE tenants
        SET isolation_level = 'schema', schema_name = $1
        WHERE id = $2
    `, schemaName, tenantID)

    // 4. Delete from public schema
    for _, table := range tables {
        db.Exec(fmt.Sprintf(
            "DELETE FROM public.%s WHERE tenant_id = '%s'",
            table, tenantID,
        ))
    }
}
```

### 9.6 Deliverables

- [ ] Migration 014: Tenant isolation configuration
- [ ] TenantConnectionManager implementation
- [ ] BaseRepository update for routing
- [ ] Automatic tier selection logic
- [ ] Tier upgrade scripts (shared → schema → database)
- [ ] Development docker-compose with multiple DB instances
- [ ] Integration tests for cross-tier scenarios

---

## 🏥 PHASE 10: ENTERPRISE MODULES

**Priority:** 🔴 HIGH | **Time:** 4-6 weeks

### Module Matrix

| Module | Complexity | Isolation | Compliance | Time |
|--------|-----------|-----------|------------|------|
| **HMS** (Hospital) | Very High | Database | HIPAA | 2 weeks |
| **ERP** (Manufacturing) | Very High | Database | SOC 2 | 2 weeks |
| **LMS** (Education) | High | Schema | FERPA | 1 week |
| **OMS** (Orders) | Medium | Schema | PCI-DSS | 1 week |

### 10.1 HMS (Hospital Management System)

**Tables:**
```sql
-- Patient records (HIPAA-sensitive)
CREATE TABLE patients (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
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
    tenant_id UUID NOT NULL,
    patient_id UUID NOT NULL REFERENCES patients(id),
    visit_date TIMESTAMP NOT NULL,
    diagnosis TEXT ENCRYPTED,
    prescription TEXT ENCRYPTED,
    doctor_id UUID REFERENCES users(id),
    notes TEXT ENCRYPTED,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE appointments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    patient_id UUID NOT NULL REFERENCES patients(id),
    doctor_id UUID NOT NULL REFERENCES users(id),
    appointment_time TIMESTAMP NOT NULL,
    status VARCHAR(20), -- scheduled, completed, cancelled
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Key Features:**
- PHI (Protected Health Information) encryption
- Audit logging for all access
- HIPAA-compliant retention policies
- Access control (doctor can only see own patients)
- Integration with lab systems (HL7/FHIR)

### 10.2 ERP (Enterprise Resource Planning)

**Tables:**
```sql
-- Inventory management
CREATE TABLE inventory_items (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sku VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    quantity_on_hand INT DEFAULT 0,
    reorder_level INT,
    unit_cost DECIMAL(12,2),
    location VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Production orders
CREATE TABLE production_orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    product_id UUID REFERENCES inventory_items(id),
    quantity INT NOT NULL,
    start_date DATE,
    due_date DATE,
    status VARCHAR(20), -- pending, in_progress, completed
    cost DECIMAL(12,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Purchase orders
CREATE TABLE purchase_orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    po_number VARCHAR(50) UNIQUE NOT NULL,
    supplier_id UUID REFERENCES suppliers(id),
    total_amount DECIMAL(12,2),
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Key Features:**
- Real-time inventory tracking
- Bill of Materials (BOM) management
- MES (Manufacturing Execution System) integration
- Supply chain optimization
- Cost accounting

### 10.3 LMS (Learning Management System)

**Enhanced from existing Lessons module:**

```sql
-- Courses (extends existing lessons)
CREATE TABLE courses (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    instructor_id UUID REFERENCES users(id),
    duration_weeks INT,
    price DECIMAL(10,2),
    enrollment_limit INT,
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Course enrollments
CREATE TABLE enrollments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    student_id UUID REFERENCES students(id),
    course_id UUID REFERENCES courses(id),
    enrolled_at TIMESTAMP DEFAULT NOW(),
    status VARCHAR(20), -- active, completed, dropped
    progress_percentage INT DEFAULT 0,
    final_grade DECIMAL(5,2)
);

-- Video content
CREATE TABLE course_videos (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    course_id UUID REFERENCES courses(id),
    title VARCHAR(255) NOT NULL,
    video_url TEXT NOT NULL,
    duration_seconds INT,
    order_index INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Key Features:**
- FERPA compliance (student data protection)
- Video hosting integration (Vimeo/YouTube)
- Quiz & assessment engine
- Certificate generation
- Progress tracking
- Discussion forums

### 10.4 OMS (Order Management System)

**Tables:**
```sql
-- Orders
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id UUID REFERENCES customers(id),
    status VARCHAR(20), -- pending, processing, shipped, delivered
    subtotal DECIMAL(12,2),
    tax DECIMAL(12,2),
    shipping_cost DECIMAL(12,2),
    total DECIMAL(12,2),
    payment_method VARCHAR(50),
    shipping_address JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Order items
CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    order_id UUID REFERENCES orders(id),
    product_id UUID REFERENCES products(id),
    quantity INT NOT NULL,
    unit_price DECIMAL(12,2),
    total_price DECIMAL(12,2),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Shipping
CREATE TABLE shipments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    order_id UUID REFERENCES orders(id),
    tracking_number VARCHAR(100),
    carrier VARCHAR(50),
    shipped_at TIMESTAMP,
    delivered_at TIMESTAMP,
    status VARCHAR(20)
);
```

**Key Features:**
- PCI-DSS compliance (payment data)
- Multi-warehouse support
- Shipping integration (UPS, FedEx, DHL)
- Return management
- Real-time order tracking

---

## 🧪 PHASE 11: COMPREHENSIVE TESTING

**Priority:** 🔴 CRITICAL | **Time:** 2-3 weeks

### 11.1 Multi-Tenant Security Tests

**File:** `test/security/tenant_isolation_test.go`

```go
func TestCrossTenantAccessPrevention(t *testing.T) {
    // Setup: Create two tenants
    tenantA := createTestTenant("Tenant A", IsolationShared)
    tenantB := createTestTenant("Tenant B", IsolationShared)

    // Tenant A creates a patient
    patientA := createPatient(tenantA.ID, "John Doe")

    // Tenant B tries to access Tenant A's patient
    _, err := getPatient(tenantB.ID, patientA.ID)

    // MUST fail
    assert.Error(t, err)
    assert.Equal(t, ErrPatientNotFound, err)
}

func TestRLSPolicyEnforcement(t *testing.T) {
    // Direct SQL bypass attempt
    db.Exec("SET app.current_tenant = ''")

    rows, err := db.Query("SELECT * FROM patients")
    assert.NoError(t, err)

    // Should return 0 rows (RLS blocks access)
    count := 0
    for rows.Next() {
        count++
    }
    assert.Equal(t, 0, count, "RLS failed - unauthorized access!")
}
```

### 11.2 Compliance Tests

**HIPAA Audit Logging:**
```go
func TestHIPAAAuditLogging(t *testing.T) {
    patient := createPatient(tenantID, "Jane Doe")

    // Access patient record
    getPatient(tenantID, patient.ID)

    // Verify audit log
    logs := getAuditLogs(tenantID, patient.ID)
    assert.NotEmpty(t, logs)
    assert.Equal(t, "patient.view", logs[0].Action)
    assert.Equal(t, userID, logs[0].UserID)
    assert.NotEmpty(t, logs[0].IPAddress)
}
```

### 11.3 Performance & Load Tests

**File:** `test/performance/load_test.go`

```go
func TestNoisyNeighborIsolation(t *testing.T) {
    // Tenant A: Heavy ERP queries (10,000 req/s)
    tenantA := "erp-factory"

    // Tenant B: Light blog queries (100 req/s)
    tenantB := "simple-blog"

    // Run concurrent load
    go runHeavyLoad(tenantA)

    // Measure Tenant B latency
    latency := measureLatency(tenantB)

    // Tenant B should NOT be affected (dedicated pool)
    assert.Less(t, latency, 100*time.Millisecond)
}
```

### 11.4 Integration Tests

- [ ] Repository integration tests (all modules)
- [ ] Service layer unit tests
- [ ] Handler E2E tests
- [ ] Cross-tier tenant tests (shared ↔ schema ↔ database)
- [ ] Migration rollback tests
- [ ] Disaster recovery tests

### 11.5 Test Coverage Targets

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
        run: docker build -t nexpaces-api:${{ github.sha }} .

      - name: Push to registry
        run: docker push nexpaces-api:${{ github.sha }}

  deploy:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        run: kubectl set image deployment/api api=nexpaces-api:${{ github.sha }}
```

### 14.2 Kubernetes Manifests

**File:** `k8s/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nexpaces-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nexpaces-api
  template:
    metadata:
      labels:
        app: nexpaces-api
    spec:
      containers:
      - name: api
        image: nexpaces-api:latest
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
| **9. Tiered Multi-Tenancy** | 🔴 CRITICAL | 1-2 weeks | None |
| **10. Enterprise Modules** | 🔴 HIGH | 4-6 weeks | Phase 9 |
| **11. Comprehensive Testing** | 🔴 CRITICAL | 2-3 weeks | Phase 9-10 |
| **12. OAuth & Advanced Auth** | 🟡 MEDIUM | 1 week | Phase 11 |
| **13. Payment Infrastructure** | 🟡 MEDIUM | 1-2 weeks | Phase 11 |
| **14. Deployment & DevOps** | 🟡 MEDIUM | 1 week | Phase 11-13 |

**Total Estimated Time:** 10-15 weeks (2.5-3.5 months)

### Week-by-Week Plan

**Week 1-2: Tiered Multi-Tenancy**
- Migration 014: Isolation configuration
- TenantConnectionManager
- Repository routing updates
- Integration tests

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
- Security tests (cross-tenant, RLS)
- Compliance tests (HIPAA, PCI-DSS, FERPA)
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

### Phase 9: Tiered Multi-Tenancy
- [ ] Migration 014 applied successfully
- [ ] TenantConnectionManager routes to correct DB
- [ ] Shared, schema, database tiers all working
- [ ] Tier upgrade scripts tested
- [ ] No cross-tier data leakage

### Phase 10: Enterprise Modules
- [ ] HMS: Patient CRUD with HIPAA compliance
- [ ] ERP: Inventory tracking functional
- [ ] LMS: Course enrollment working
- [ ] OMS: Order processing end-to-end
- [ ] All modules tenant-isolated

### Phase 11: Testing
- [ ] Test coverage >75%
- [ ] All security tests passing
- [ ] Compliance tests (HIPAA, PCI-DSS) green
- [ ] Load tests: 10,000 req/s sustained
- [ ] Zero cross-tenant access bugs

### Phase 12: OAuth
- [ ] Google login working
- [ ] GitHub login working
- [ ] Apple Sign In working
- [ ] Account linking functional

### Phase 13: Payment
- [ ] Stripe integration complete
- [ ] Subscription creation/cancellation working
- [ ] Webhook handling tested
- [ ] Usage-based billing accurate

### Phase 14: DevOps
- [ ] CI/CD pipeline passing
- [ ] Production deployment successful
- [ ] Monitoring dashboards active
- [ ] Alerts configured

---

## 🔗 External Dependencies

### Go Packages
```bash
# OAuth
go get golang.org/x/oauth2
go get google.golang.org/api/oauth2/v2

# Payments
go get github.com/stripe/stripe-go/v76

# Encryption (HIPAA)
go get golang.org/x/crypto/nacl/secretbox

# Monitoring
go get github.com/prometheus/client_golang
```

### External Services
- **Stripe:** Payment processing
- **AWS RDS:** PostgreSQL hosting (multi-region)
- **Redis:** Session storage, rate limiting
- **SendGrid/AWS SES:** Email delivery
- **Cloudflare:** CDN, WAF, DDoS protection
- **MaxMind GeoIP:** Geolocation (compliance)

---

## 📝 Environment Variables (Full List)

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ENVIRONMENT=production

# Database (Shared)
DATABASE_HOST=shared-db.amazonaws.com
DATABASE_PORT=5432
DATABASE_NAME=nexpaces_shared
DATABASE_USER=postgres
DATABASE_PASSWORD=***
DATABASE_SSL_MODE=require

# Database (Enterprise - Example)
DATABASE_ENTERPRISE_HOST=enterprise-db.amazonaws.com
DATABASE_ENTERPRISE_PORT=5432
DATABASE_ENTERPRISE_USER=postgres
DATABASE_ENTERPRISE_PASSWORD=***

# Redis
REDIS_URL=redis://redis-cluster.amazonaws.com:6379

# JWT
JWT_SECRET=***-min-32-chars

# OAuth - Google
GOOGLE_CLIENT_ID=***.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=***
GOOGLE_REDIRECT_URL=https://api.nexpaces.com/auth/google/callback

# OAuth - GitHub
GITHUB_CLIENT_ID=***
GITHUB_CLIENT_SECRET=***
GITHUB_REDIRECT_URL=https://api.nexpaces.com/auth/github/callback

# OAuth - Apple
APPLE_CLIENT_ID=com.nexpaces.signin
APPLE_TEAM_ID=***
APPLE_KEY_ID=***
APPLE_PRIVATE_KEY=/keys/apple-key.p8

# Stripe
STRIPE_SECRET_KEY=sk_live_***
STRIPE_PUBLISHABLE_KEY=pk_live_***
STRIPE_WEBHOOK_SECRET=whsec_***

# Email
SENDGRID_API_KEY=SG.***
FROM_EMAIL=noreply@nexpaces.com

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
**Maintained By:** NexSpaces Development Team
