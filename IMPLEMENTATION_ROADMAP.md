# NexSpaces API - Complete Implementation Roadmap

> **Multi-Tenant SaaS Platform** with OAuth, Payment Processing & Advanced Security

---

## 📊 Current Status

**Completed:** Phase 1-6 (Foundation, Domain, Data, Business Logic, HTTP Layer, Bootstrap)

**Build Status:** ✅ Successful (12MB binary)

**Test Coverage:**
- Tenant Entity: 57.9%
- User Entity: 48.1%
- Total Domain: 36.8%

---

## 🎯 Implementation Phases

### ✅ COMPLETED PHASES (1-6)

#### Phase 1: Foundation ✅ 100%
- Logger (zerolog, tenant-aware)
- Error types (50+ application errors)
- Validator (UUID, slug, domain)
- Database (PostgreSQL + multi-tenant transactions)
- Configuration management

#### Phase 2: Domain Layer ✅ 100%
- Tenant entity (subscription plans, feature limits)
- User entity (RBAC, password hashing)
- Website entity (multi-type support)
- Business rules & validation

#### Phase 3: Data Layer ✅ 100%
- Migrations with RLS policies
- Tenant, User, Website repositories
- PostgreSQL implementation
- Multi-tenant isolation at DB level

#### Phase 4: Business Logic ✅ 100%
- Tenant service (CRUD, plan upgrades)
- User service (auth, JWT generation)
- Website service
- DTOs & request/response models

#### Phase 5: HTTP Layer ✅ 100%
- JWT authentication middleware
- Auth handlers (register, login, logout)
- Tenant management handlers (13 endpoints)
- User management handlers (11 endpoints)
- Website handlers (7 endpoints)

#### Phase 6: Application Bootstrap ✅ 100%
- Dependency injection
- Route registration (32 endpoints total)
- Graceful shutdown support

---

## 🧪 PHASE 7: CORE TESTING

**Priority:** HIGH | **Time:** 2-3 days

### 7.1 Repository Integration Tests
- PostgreSQL test database setup
- Tenant repository tests (CRUD + RLS)
- User repository tests (multi-tenant isolation)
- Website repository tests
- Transaction & rollback tests

### 7.2 Service/Usecase Unit Tests
- Tenant service tests
- User service tests (auth flows)
- Website service tests
- Mock repository tests

### 7.3 Handler E2E Tests
- Auth endpoint tests
- Tenant endpoint tests
- User endpoint tests
- Website endpoint tests
- HTTP status validation

### 7.4 Multi-Tenant Security Tests
- Cross-tenant access prevention
- RLS policy enforcement
- JWT tenant isolation
- RBAC tests

**Deliverables:**
- Test coverage >80%
- Integration test suite
- Security test suite

---

## 🔐 PHASE 8: OAUTH & SOCIAL LOGIN

**Priority:** HIGH | **Time:** 3-4 days

### 8.1 OAuth Provider Strategy Pattern

**Files:**
- `internal/domain/auth/oauth_provider.go`
- `internal/oauth/provider.go`

**Features:**
- OAuthProvider interface
- Strategy pattern implementation
- Multi-tenant OAuth config

### 8.2 Google OAuth Implementation

**Files:**
- `internal/oauth/google/provider.go`
- `migrations/004_oauth_connections.sql`

**Endpoints:**
```
GET  /api/v1/auth/google/login
GET  /api/v1/auth/google/callback
```

**Features:**
- Google OAuth 2.0 flow
- User profile fetching
- Token exchange & validation

### 8.3 GitHub OAuth Implementation

**Files:**
- `internal/oauth/github/provider.go`

**Endpoints:**
```
GET  /api/v1/auth/github/login
GET  /api/v1/auth/github/callback
```

### 8.4 Apple OAuth Implementation

**Files:**
- `internal/oauth/apple/provider.go`

**Endpoints:**
```
POST /api/v1/auth/apple/callback
```

**Features:**
- Sign in with Apple (complex JWT validation)
- Apple ID token verification

### 8.5 OAuth Callback Handlers

**Files:**
- `internal/handler/auth/oauth.go`
- `internal/usecase/auth/oauth_service.go`

**Features:**
- Generic OAuth callback handler
- CSRF protection (state parameter)
- JWT generation after OAuth
- Redis state storage (5 min TTL)

### 8.6 Account Linking

**Endpoints:**
```
POST /api/v1/auth/link/{provider}
GET  /api/v1/auth/connections
DELETE /api/v1/auth/connections/{provider}
```

**Features:**
- Link OAuth to existing email
- Multiple providers per user
- Primary provider selection

**Database Schema:**
```sql
CREATE TABLE oauth_connections (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider VARCHAR(50) NOT NULL, -- google, github, apple
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
```

---

## 💳 PHASE 9: PAYMENT INFRASTRUCTURE

**Priority:** HIGH | **Time:** 4-5 days

### 9.1 Payment Provider Abstraction

**Files:**
- `internal/domain/payment/provider.go`
- `internal/domain/payment/subscription.go`
- `migrations/005_payments.sql`

**Interface:**
```go
type PaymentProvider interface {
    CreateCustomer(tenantID, email string) (string, error)
    CreateSubscription(customerID, planID string) (*Subscription, error)
    CancelSubscription(subscriptionID string) error
    CreatePaymentIntent(amount int64, currency string) (*PaymentIntent, error)
}
```

**Database Schema:**
```sql
CREATE TABLE payment_customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider VARCHAR(50) NOT NULL, -- stripe, custom
    provider_customer_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    plan_id VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_subscription_id VARCHAR(255),
    status VARCHAR(50), -- active, cancelled, past_due
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    cancel_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    subscription_id UUID REFERENCES subscriptions(id),
    amount_due INTEGER NOT NULL,
    amount_paid INTEGER,
    currency VARCHAR(3) DEFAULT 'USD',
    status VARCHAR(50), -- draft, paid, void
    pdf_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 9.2 Stripe Integration

**Files:**
- `internal/payment/stripe/client.go`
- `internal/payment/stripe/provider.go`

**Dependencies:**
```bash
go get github.com/stripe/stripe-go/v76
```

**Config:**
```env
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

### 9.3 Subscription Lifecycle

**Endpoints:**
```
POST   /api/v1/subscriptions
GET    /api/v1/subscriptions
PATCH  /api/v1/subscriptions/:id/upgrade
POST   /api/v1/subscriptions/:id/cancel
POST   /api/v1/subscriptions/:id/reactivate
```

**Features:**
- Create subscription
- Upgrade/downgrade plans
- Cancel subscription
- Auto-suspend on failed payment

### 9.4 Webhook Handlers

**Endpoint:**
```
POST /api/v1/webhooks/stripe
```

**Events:**
- `invoice.paid` → Activate subscription
- `invoice.payment_failed` → Suspend tenant
- `customer.subscription.deleted` → Cancel subscription
- `customer.subscription.updated` → Sync status

**Database:**
```sql
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY,
    provider VARCHAR(50),
    event_id VARCHAR(255) UNIQUE,
    event_type VARCHAR(100),
    payload JSONB,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 9.5 Invoice Generation

**Endpoints:**
```
GET /api/v1/invoices
GET /api/v1/invoices/:id
GET /api/v1/invoices/:id/pdf
```

**Features:**
- Monthly invoice generation
- PDF generation (go-pdf library)
- Email invoices to tenant admin
- Cron job for automation

### 9.6 Usage-Based Billing

**Files:**
- `internal/metering/usage_tracker.go`
- `migrations/006_usage_metrics.sql`

**Metrics:**
- API calls per tenant
- Storage usage per tenant
- User count per tenant

**Middleware:**
- API call metering middleware
- Report to Stripe for metered billing

---

## 🏢 PHASE 10: MULTI-TENANT PAYMENT

**Priority:** MEDIUM | **Time:** 5-6 days

### 10.1 Tenant Custom Payment Config

**Database:**
```sql
CREATE TABLE tenant_payment_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    provider_type VARCHAR(50), -- stripe, custom
    stripe_account_id VARCHAR(255),
    api_key_encrypted TEXT,
    mode VARCHAR(20), -- test, live
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Endpoint:**
```
POST /api/v1/tenants/:id/payment-config
GET  /api/v1/tenants/:id/payment-config
```

### 10.2 Stripe Connect

**Endpoints:**
```
GET /api/v1/stripe/connect/authorize
GET /api/v1/stripe/connect/callback
```

**Features:**
- OAuth flow for Stripe Connect
- Connected account management
- Platform fee configuration
- Use tenant's Stripe for their customers

### 10.3 Revenue Sharing

**Features:**
- Commission calculation (e.g., 2.5% + $0.30)
- Split payments (Stripe Connect)
- Platform earnings tracking

**Database:**
```sql
CREATE TABLE platform_transactions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    amount INTEGER,
    platform_fee INTEGER,
    tenant_revenue INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 10.4 Billing Dashboard

**Endpoints:**
```
GET /api/v1/billing/dashboard
GET /api/v1/billing/transactions
GET /api/v1/billing/revenue-chart
```

**Metrics:**
- Monthly revenue
- Active subscriptions
- Failed payments
- Revenue charts

---

## 🔒 PHASE 11: ADVANCED AUTH & SECURITY

**Priority:** HIGH | **Time:** 3-4 days

### 11.1 2FA/MFA (TOTP)

**Endpoints:**
```
POST /api/v1/auth/2fa/enable
POST /api/v1/auth/2fa/verify
POST /api/v1/auth/2fa/disable
GET  /api/v1/auth/2fa/backup-codes
```

**Dependencies:**
```bash
go get github.com/pquerna/otp
```

**Database:**
```sql
CREATE TABLE user_2fa_secrets (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    secret TEXT NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE backup_codes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    code VARCHAR(20) NOT NULL,
    used BOOLEAN DEFAULT FALSE
);
```

### 11.2 Magic Link Authentication

**Endpoints:**
```
POST /api/v1/auth/magic-link/send
GET  /api/v1/auth/magic-link/verify/:token
```

**Features:**
- Generate one-time link
- 15 min expiration
- Redis token storage

### 11.3 Password Reset

**Endpoints:**
```
POST /api/v1/auth/forgot-password
POST /api/v1/auth/reset-password
```

**Database:**
```sql
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 11.4 Session Management

**Endpoints:**
```
POST /api/v1/auth/refresh
POST /api/v1/auth/logout-all
GET  /api/v1/auth/sessions
```

**Features:**
- Access token (15 min)
- Refresh token (7 days)
- Token rotation
- Redis session storage

### 11.5 Device Tracking

**Endpoints:**
```
GET    /api/v1/auth/devices
DELETE /api/v1/auth/devices/:id
```

**Features:**
- Device fingerprinting
- Geo-location (MaxMind GeoIP)
- New device email notification

**Database:**
```sql
CREATE TABLE user_devices (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    ip_address INET,
    location VARCHAR(255),
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🚀 PHASE 12: DEPLOYMENT & DEVOPS

**Priority:** MEDIUM | **Time:** 2-3 days

### 12.1 Docker Compose (Dev)

**File:** `docker-compose.yml`

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: nexspaces
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/nexspaces
      REDIS_URL: redis://redis:6379
    depends_on:
      - postgres
      - redis

  adminer:
    image: adminer
    ports:
      - "8081:8080"
```

### 12.2 Kubernetes Manifests

**Files:**
- `k8s/deployment.yaml`
- `k8s/service.yaml`
- `k8s/configmap.yaml`
- `k8s/secret.yaml`
- `k8s/ingress.yaml`
- `k8s/hpa.yaml`

### 12.3 CI/CD Pipeline (Bitbucket Pipelines)

**File:** `bitbucket-pipelines.yml`

```yaml
image: golang:1.21

pipelines:
  default:
    - step:
        name: Test
        caches:
          - go
        script:
          - go test -v ./...
          - go test -coverprofile=coverage.out ./...
          - go tool cover -func=coverage.out

  branches:
    develop:
      - step:
          name: Build & Deploy to Staging
          deployment: staging
          script:
            - docker build -t nexspaces-api:staging .
            - docker push nexspaces-api:staging
            - kubectl set image deployment/api api=nexspaces-api:staging

    main:
      - step:
          name: Build & Deploy to Production
          deployment: production
          script:
            - docker build -t nexspaces-api:$BITBUCKET_COMMIT .
            - docker push nexspaces-api:$BITBUCKET_COMMIT
            - kubectl set image deployment/api api=nexspaces-api:$BITBUCKET_COMMIT
```

### 12.4 Monitoring & Observability

**Tools:**
- Prometheus (metrics)
- Grafana (dashboards)
- Loki (log aggregation)
- Alertmanager (alerts)

**Metrics Endpoint:**
```
GET /metrics
```

---

## 📈 Implementation Timeline

| Phase | Priority | Duration | Dependencies |
|-------|----------|----------|--------------|
| 7. Testing | HIGH | 2-3 days | Phase 1-6 ✅ |
| 8. OAuth | HIGH | 3-4 days | Phase 7 |
| 9. Payments | HIGH | 4-5 days | Phase 7 |
| 10. Multi-Tenant Payment | MEDIUM | 5-6 days | Phase 9 |
| 11. Advanced Auth | HIGH | 3-4 days | Phase 8 |
| 12. DevOps | MEDIUM | 2-3 days | All phases |

**Total Estimated Time:** 19-25 days

---

## 🎯 Recommended Order

### Week 1: Testing + OAuth
- **Day 1-2:** Phase 7 (Core Testing)
- **Day 3-5:** Phase 8 (OAuth & Social Login)

### Week 2: Payments
- **Day 1-5:** Phase 9 (Payment Infrastructure)

### Week 3: Advanced Features
- **Day 1-3:** Phase 11 (Advanced Auth)
- **Day 4-5:** Phase 12.1-12.3 (Docker + CI/CD)

### Week 4: Multi-Tenant Payments (Optional for MVP)
- **Day 1-5:** Phase 10 (Tenant Custom Payments)

---

## 🔗 External Dependencies

### Go Packages
```bash
# OAuth
go get golang.org/x/oauth2
go get google.golang.org/api/oauth2/v2

# Payments
go get github.com/stripe/stripe-go/v76

# 2FA
go get github.com/pquerna/otp

# QR Code
go get github.com/skip2/go-qrcode

# PDF Generation
go get github.com/jung-kurt/gofpdf
```

### External Services
- **Stripe:** Payment processing
- **MaxMind GeoIP:** Geo-location
- **SendGrid/AWS SES:** Email delivery

---

## 📝 Environment Variables

```env
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ENVIRONMENT=production

# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=nexspaces
DATABASE_USER=postgres
DATABASE_PASSWORD=secure_password
DATABASE_SSL_MODE=require

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-super-secret-key-change-in-production

# OAuth - Google
GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_REDIRECT_URL=https://api.nexpaces.com/auth/google/callback

# OAuth - GitHub
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
GITHUB_REDIRECT_URL=https://api.nexpaces.com/auth/github/callback

# OAuth - Apple
APPLE_CLIENT_ID=com.nexpaces.signin
APPLE_TEAM_ID=xxx
APPLE_KEY_ID=xxx
APPLE_PRIVATE_KEY=path/to/key.p8

# Stripe
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_PUBLISHABLE_KEY=pk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Email
SENDGRID_API_KEY=SG.xxx
FROM_EMAIL=noreply@nexpaces.com

# Monitoring
SENTRY_DSN=https://xxx@sentry.io/xxx
```

---

## ✅ Success Criteria

### Phase 7 (Testing)
- [ ] Test coverage >80%
- [ ] All integration tests passing
- [ ] Security tests passing

### Phase 8 (OAuth)
- [ ] Google login working
- [ ] GitHub login working
- [ ] Apple login working
- [ ] Account linking functional

### Phase 9 (Payments)
- [ ] Stripe integration complete
- [ ] Subscription creation working
- [ ] Webhook handling tested
- [ ] Invoice generation working

### Phase 10 (Multi-Tenant Payment)
- [ ] Stripe Connect integrated
- [ ] Tenant custom payment config
- [ ] Revenue sharing working

### Phase 11 (Advanced Auth)
- [ ] 2FA/MFA working
- [ ] Magic links functional
- [ ] Password reset flow complete
- [ ] Device tracking active

### Phase 12 (DevOps)
- [ ] Docker Compose working locally
- [ ] Bitbucket CI/CD pipeline passing
- [ ] Production deployment successful
- [ ] Monitoring dashboards active

---

**Last Updated:** 2025-01-01
**Version:** 1.0
**Maintained By:** NexSpaces Development Team
