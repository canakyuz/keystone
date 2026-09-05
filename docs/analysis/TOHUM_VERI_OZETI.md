# Registry System - Seed Data Summary

**Created:** 2025-10-05
**Week 1 Day 4:** Seed Data Creation
**Status:** ✅ Completed

---

## 📊 Overview

Registry katalog sistemi için seed data oluşturuldu. Tüm mevcut modüller, araçlar ve bağımlılıkları tanımlandı. Test tenant ile birlikte LMS + Payment demo ortamı hazırlandı.

### Created Migrations

**2 yeni seed migration oluşturuldu:**
- ✅ `023_seed_registry_data.up/down.sql` - Catalog data
- ✅ `024_seed_test_tenant.up/down.sql` - Test tenant

**Toplam:** 4 dosya (2 up + 2 down)

---

## 🏗️ Seed Data Architecture

```
┌─────────────────────────────────────────────────────────────┐
│               CATALOG DATA (023_seed_registry_data)         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  MODULES (6):                    TOOLS (3):                 │
│  1. LMS                          1. Payment                 │
│  2. CMS                          2. Refund                  │
│  3. Project Management           3. Webhook                 │
│  4. Booking                                                 │
│  5. Service Catalog                                         │
│  6. Website Builder                                         │
│                                                             │
│  MODULE DEPENDENCIES (4):        TOOL DEPENDENCIES (2):     │
│  - LMS → Payment (optional)      - Refund → Payment (req)   │
│  - Booking → Payment (optional)  - Payment → Webhook (rec)  │
│  - Service → Payment (optional)                             │
│  - Service → Refund (optional)                              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│              TEST TENANT DATA (024_seed_test_tenant)        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  TENANT:                                                    │
│  - Name: Can Akyüz Tech Academy                            │
│  - Slug: canakyuz                                           │
│  - URL: https://canakyuz.keystone.dev                       │
│  - Plan: Pro (Trial - 30 days)                             │
│                                                             │
│  ACTIVATED MODULES (1):          ACTIVATED TOOLS (2):       │
│  - LMS (v1.0.0)                  - Payment (Stripe)         │
│                                  - Webhook                  │
│                                                             │
│  SAMPLE DATA:                                               │
│  - 1 student (Ahmet Yılmaz)                                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 📋 Part 1: Modules Catalog (6 Modules)

### 1.1 LMS (Learning Management System)

**Code:** `LMS`
**Category:** `education`
**Pricing:** Subscription - $29.99/month
**Status:** Active, Public

**Features:**
- One-on-one and group lessons
- Online and in-person sessions
- Assignment management
- Performance tracking
- Attendance tracking
- Material/resource management
- Homework tracking
- Progress reports
- Meeting URL integration (Zoom/Meet)

**Database Tables:** `lessons`, `students`, `assignments`

**Default Limits:**
```json
{
  "max_lessons": 100,
  "max_students": 50,
  "max_assignments": 200
}
```

**Tags:** `education`, `learning`, `tutoring`, `courses`, `lessons`, `students`

---

### 1.2 CMS (Content Management System)

**Code:** `CMS`
**Category:** `content`
**Pricing:** Free
**Status:** Active, Public

**Features:**
- Blog post creation
- Category management
- Draft/Publish workflow
- Featured content
- Tag system
- View count tracking
- Rich text editor
- SEO optimization
- Image support

**Database Tables:** `blog_posts`, `blog_categories`

**Default Limits:**
```json
{
  "max_posts": 1000,
  "max_categories": 50
}
```

**Tags:** `blog`, `content`, `cms`, `publishing`, `articles`, `writing`

---

### 1.3 Project Management

**Code:** `PROJECT`
**Category:** `management`
**Pricing:** Free
**Status:** Active, Public

**Features:**
- Project tracking
- Status management (Planned/In-Progress/Completed)
- Client tracking
- Technology stack display
- Image gallery
- Live URL & GitHub links
- Featured projects
- View count
- Project categories

**Database Tables:** `projects`

**Default Limits:**
```json
{
  "max_projects": 100,
  "max_images_per_project": 10
}
```

**Tags:** `portfolio`, `projects`, `freelance`, `agency`, `showcase`

---

### 1.4 Booking & Appointments

**Code:** `BOOKING`
**Category:** `hospitality`
**Pricing:** Subscription - $19.99/month
**Status:** Active, Public

**Features:**
- Appointment scheduling
- Availability management
- Client information tracking
- Reminder system
- Cancellation management
- Meeting URL support
- Status tracking (Pending/Confirmed/Completed)
- No-show tracking
- Time slot management

**Database Tables:** `appointments`, `availabilities`

**Default Limits:**
```json
{
  "max_appointments": 500,
  "max_availability_slots": 200
}
```

**Tags:** `booking`, `appointments`, `scheduling`, `healthcare`, `consulting`, `reservations`

---

### 1.5 Service Catalog

**Code:** `SERVICE`
**Category:** `commerce`
**Pricing:** Free
**Status:** Active, Public

**Features:**
- Service/Product catalog
- Multiple pricing models
- Feature lists
- Public/Private visibility
- Featured services
- Image support
- Duration tracking
- Max client limits
- Category management

**Database Tables:** `services`

**Default Limits:**
```json
{
  "max_services": 100
}
```

**Pricing Models Supported:**
- `one_time`
- `recurring`
- `subscription`
- `usage_based`

**Tags:** `services`, `products`, `catalog`, `pricing`, `offerings`, `ecommerce`

---

### 1.6 Website Builder (Platform Core)

**Code:** `WEBSITE`
**Category:** `other`
**Pricing:** Free
**Status:** Active, **NOT Public** (Platform Infrastructure)

**Features:**
- Website creation
- Custom domain support
- Template system
- Multi-tenant isolation
- Draft/Published workflow
- Domain verification
- Website types (CMS/CRM/E-commerce/Education/Hospitality/ERP/Blog/Portfolio)

**Database Tables:** `websites`

**Default Limits:**
```json
{
  "max_websites": 10
}
```

**Metadata:**
```json
{
  "is_platform_core": true,
  "auto_enabled": true
}
```

**Tags:** `website`, `builder`, `platform`, `multi-tenant`

---

## 🔧 Part 2: Tools Catalog (3 Tools)

### 2.1 Payment Gateway

**Code:** `PAYMENT`
**Category:** `payment`
**Type:** Global (Cross-module)
**Pricing:** Transaction Fee - 2.5% + $0.30 per transaction
**Status:** Active, Public

**Features:**
- Multi-provider support (Stripe, Iyzico, Checkout)
- 3DS authentication (SCA compliant)
- Installment support (Turkey market)
- Card information (PCI-compliant)
- Customer information tracking
- Billing address
- Payment status tracking
- Webhook integration
- Metadata support

**Providers:** `Stripe`, `Iyzico`, `Checkout.com`
**Integration Type:** Third-party
**Requires API Keys:** ✅ Yes
**Requires Webhook:** ✅ Yes

**Database Tables:** `payments`

**Default Limits:**
```json
{
  "max_transactions_per_month": 1000
}
```

**Rate Limits:**
```json
{
  "requests_per_minute": 60
}
```

**Tags:** `payment`, `stripe`, `iyzico`, `checkout`, `gateway`, `transaction`, `billing`

---

### 2.2 Refund Management

**Code:** `REFUND`
**Category:** `payment`
**Type:** Global (Cross-module)
**Pricing:** Free
**Status:** Active, Public

**Features:**
- Full/Partial refund support
- Multi-provider support
- Refund status tracking
- Refund reasons (Customer request, Duplicate, Fraudulent, Other)
- Provider integration
- Audit trail

**Providers:** `Stripe`, `Iyzico`, `Checkout.com`
**Integration Type:** Third-party
**Requires API Keys:** ❌ No (uses Payment tool API keys)
**Requires Webhook:** ❌ No

**Database Tables:** `refunds`

**Default Limits:**
```json
{
  "max_refunds_per_month": 100
}
```

**Tags:** `refund`, `return`, `payment`, `transaction`, `reversal`

---

### 2.3 Webhook Handler

**Code:** `WEBHOOK`
**Category:** `integration`
**Type:** Global (Cross-module)
**Pricing:** Free
**Status:** Active, Public

**Features:**
- Provider-agnostic events
- Idempotency support
- Signature validation
- Retry mechanism
- Processing status tracking
- Request metadata (IP, headers)
- Event correlation
- Support for payment/refund/3DS/chargeback events

**Event Types:** `payment.*`, `refund.*`, `3ds.*`, `chargeback.*`
**Integration Type:** Native
**Requires Webhook:** ✅ Yes

**Database Tables:** `payment_events`

**Default Limits:**
```json
{
  "max_events_per_day": 10000
}
```

**Rate Limits:**
```json
{
  "events_per_second": 100
}
```

**Tags:** `webhook`, `events`, `integration`, `notification`, `automation`

---

## 🔗 Part 3: Dependencies

### Module Dependencies (4 total)

#### 3.1 LMS → Payment Tool
- **Type:** Optional
- **Scope:** Feature
- **Auto-install:** No
- **Priority:** 5
- **Description:** Enable payment collection for lessons and courses

#### 3.2 Booking → Payment Tool
- **Type:** Optional
- **Scope:** Feature
- **Auto-install:** No
- **Priority:** 5
- **Description:** Enable payment collection for appointments and reservations

#### 3.3 Service → Payment Tool
- **Type:** Optional
- **Scope:** Runtime
- **Auto-install:** No
- **Priority:** 10
- **Description:** Enable service/product purchases and checkout

#### 3.4 Service → Refund Tool
- **Type:** Optional
- **Scope:** Runtime
- **Auto-install:** No
- **Priority:** 5
- **Description:** Enable refund processing for services and products

---

### Tool Dependencies (2 total)

#### 4.1 Refund → Payment Tool
- **Type:** Required (Hard Dependency)
- **Scope:** Runtime
- **Auto-install:** Yes
- **Allow Disable:** No (Payment cannot be disabled if Refund is active)
- **Priority:** 10
- **Install Order:** 1 (Payment must be installed first)
- **Description:** Refund tool requires Payment tool to process refunds

#### 4.2 Payment → Webhook Tool
- **Type:** Recommended
- **Scope:** Runtime
- **Auto-install:** Yes
- **Allow Disable:** Yes (Webhook can be disabled independently)
- **Priority:** 5
- **Install Order:** 2 (Webhook installed after Payment)
- **Description:** Webhook tool recommended for real-time payment event notifications

---

## 👤 Part 4: Test Tenant (Can Akyüz)

### Tenant Details

**ID:** `aaaaaaaa-bbbb-cccc-dddd-000000000001`
**Name:** Can Akyüz Tech Academy
**Slug:** `canakyuz`
**Email:** can@akyuz.tech
**Phone:** +90 555 123 4567
**URL:** https://canakyuz.keystone.dev

**Status:** Trial
**Plan:** Pro
**Trial Period:** 30 days

**Settings:**
```json
{
  "timezone": "Europe/Istanbul",
  "language": "tr",
  "currency": "TRY"
}
```

**Metadata:**
```json
{
  "company": "Can Akyüz Tech",
  "industry": "Education",
  "country": "Turkey"
}
```

---

### Activated Modules (1)

#### LMS Module

**Status:** Active, Enabled
**Version:** 1.0.0
**Subscription:** Trial (30 days), Pro plan, $29.99/month

**Configuration:**
```json
{
  "theme": "professional",
  "language": "tr",
  "timezone": "Europe/Istanbul",
  "email_notifications": true,
  "lesson_reminders": true,
  "auto_attendance": false
}
```

**Features Enabled:**
- `lessons`
- `students`
- `assignments`
- `performance_tracking`
- `materials`

**Limits:**
```json
{
  "max_lessons": 500,
  "max_students": 200,
  "max_assignments": 1000,
  "max_materials_per_lesson": 20
}
```

**Current Usage:**
```json
{
  "lessons_count": 0,
  "students_count": 0,
  "assignments_count": 0,
  "active_lessons": 0
}
```

**Allowed Roles:** `admin`, `teacher`, `student`

---

### Activated Tools (2)

#### Payment Tool (Stripe)

**Status:** Active, Enabled
**Version:** 1.0.0
**Integration:** Connected, Verified
**Health:** Healthy
**Linked to:** LMS Module

**Configuration:**
```json
{
  "default_provider": "stripe",
  "currency": "TRY",
  "allow_installments": false,
  "capture_method": "automatic",
  "statement_descriptor": "CAN AKYUZ TECH"
}
```

**API Keys:** (Encrypted in production)
```json
{
  "stripe_public_key": "pk_test_...",
  "stripe_secret_key": "sk_test_...",
  "iyzico_api_key": null,
  "iyzico_secret_key": null
}
```

**Webhook Config:**
```json
{
  "payment_success_url": "https://canakyuz.keystone.dev/payment/success",
  "payment_cancel_url": "https://canakyuz.keystone.dev/payment/cancel",
  "webhook_url": "https://canakyuz.keystone.dev/api/webhooks/payment",
  "webhook_secret": "whsec_test_..."
}
```

**Limits:**
```json
{
  "max_transactions_per_month": 1000,
  "max_amount_per_transaction": 10000
}
```

**Current Usage:**
```json
{
  "transactions_count": 0,
  "total_amount_collected": 0,
  "last_transaction_at": null
}
```

**Allowed Roles:** `admin`, `teacher`

---

#### Webhook Tool

**Status:** Active, Enabled
**Version:** 1.0.0
**Integration:** Connected
**Health:** Healthy
**Scope:** Global (not module-specific)

**Configuration:**
```json
{
  "retry_attempts": 3,
  "retry_delay_seconds": 60,
  "signature_verification": true,
  "log_all_events": true
}
```

**Webhook Config:**
```json
{
  "endpoint": "https://canakyuz.keystone.dev/api/webhooks/handler",
  "events": ["payment.*", "refund.*", "3ds.*"],
  "headers": {
    "X-Tenant-ID": "aaaaaaaa-bbbb-cccc-dddd-000000000001"
  }
}
```

**Limits:**
```json
{
  "max_events_per_day": 10000
}
```

**Current Usage:**
```json
{
  "events_received": 0,
  "events_processed": 0,
  "events_failed": 0
}
```

**Allowed Roles:** `admin`

---

### Sample Data

#### Student Record

**Name:** Ahmet Yılmaz
**Email:** ahmet.yilmaz@example.com
**Phone:** +90 555 999 8888
**Grade:** 10
**Status:** Active

**Metadata:**
```json
{
  "parent_name": "Mehmet Yılmaz",
  "parent_phone": "+90 555 999 7777",
  "emergency_contact": "+90 555 999 6666"
}
```

---

## 📊 Statistics

### Catalog Summary

| Resource | Count |
|----------|-------|
| Modules | 6 |
| Tools | 3 |
| Module Dependencies | 4 |
| Tool Dependencies | 2 |
| **Total Catalog Items** | **15** |

### Pricing Models

**Free Modules:** 3 (CMS, Project, Website Builder)
**Paid Modules:** 2 (LMS - $29.99/mo, Booking - $19.99/mo)

**Free Tools:** 2 (Refund, Webhook)
**Transaction-based Tools:** 1 (Payment - 2.5% + $0.30)

---

## 🔍 Example Queries

### Get all public modules

```sql
SELECT name, code, category, pricing_model, base_price
FROM modules
WHERE is_public = TRUE
  AND status = 'active'
  AND deleted_at IS NULL
ORDER BY category, name;
```

**Expected Result:**
- LMS (education, subscription, $29.99)
- CMS (content, free)
- Project (management, free)
- Booking (hospitality, subscription, $19.99)
- Service (commerce, free)

---

### Get Can Akyüz tenant's active modules

```sql
SET app.current_tenant = 'aaaaaaaa-bbbb-cccc-dddd-000000000001';

SELECT m.name, m.code, tm.status, tm.pricing_plan
FROM tenant_modules tm
JOIN modules m ON m.id = tm.module_id
WHERE tm.is_enabled = TRUE
  AND tm.deleted_at IS NULL;
```

**Expected Result:**
- LMS (active, pro)

---

### Get dependencies for LMS module

```sql
SELECT
  m.name as module_name,
  t.name as dependency_name,
  md.dependency_type,
  md.auto_install
FROM modules m
JOIN module_dependencies md ON md.module_id = m.id
JOIN tools t ON t.id = md.depends_on_tool_id
WHERE m.code = 'LMS';
```

**Expected Result:**
- LMS → Payment Gateway (optional, manual install)

---

### Check if Payment tool dependencies are met

```sql
SELECT
  t1.name as tool_name,
  t2.name as required_tool,
  td.dependency_type,
  CASE
    WHEN tt.id IS NOT NULL THEN 'Installed'
    ELSE 'Not Installed'
  END as status
FROM tools t1
JOIN tool_dependencies td ON td.tool_id = t1.id
JOIN tools t2 ON t2.id = td.depends_on_tool_id
LEFT JOIN tenant_tools tt ON tt.tool_id = t2.id
  AND tt.tenant_id = 'aaaaaaaa-bbbb-cccc-dddd-000000000001'::UUID
WHERE t1.code = 'REFUND';
```

**Expected Result:**
- Refund → Payment Gateway (required, Not Installed)

---

## 🎯 Next Steps (Week 1 Day 5)

**Repository & Service Layer Implementation:**

1. **Create Domain Entities (Go):**
   - `internal/domain/registry/module_entity.go`
   - `internal/domain/registry/tool_entity.go`
   - `internal/domain/registry/tenant_module_entity.go`
   - `internal/domain/registry/tenant_tool_entity.go`

2. **Create Repositories:**
   - `ModuleRepository` (CRUD + catalog queries)
   - `ToolRepository` (CRUD + catalog queries)
   - `TenantModuleRepository` (activation/deactivation)
   - `TenantToolRepository` (activation/deactivation)

3. **Create Service Layer:**
   - `ModuleCatalogService` (marketplace queries)
   - `ToolCatalogService` (marketplace queries)
   - `TenantModuleService` (activation logic)
   - `TenantToolService` (activation logic)
   - `DependencyCheckerService` (validate dependencies)

4. **Testing:**
   - Unit tests for repositories
   - Integration tests for services
   - Dependency validation tests

---

## ✅ Self-Review Checklist

**Seed Data Quality:**
- ✅ All 6 modules defined with comprehensive features
- ✅ All 3 tools defined with integration details
- ✅ Dependencies correctly defined (4 module deps, 2 tool deps)
- ✅ Test tenant with realistic configuration
- ✅ Sample data for demo purposes

**Data Integrity:**
- ✅ UUIDs properly generated
- ✅ Foreign key references valid
- ✅ JSONB fields properly formatted
- ✅ Default values sensible
- ✅ Constraints respected

**Business Logic:**
- ✅ Pricing models realistic
- ✅ Limits appropriate for each tier
- ✅ Dependencies logical (Refund → Payment)
- ✅ Auto-install flags correct
- ✅ Trial periods set (30 days)

**Tenant Isolation:**
- ✅ Test tenant ID fixed for reproducibility
- ✅ RLS will enforce tenant isolation
- ✅ Cross-tenant access prevented

**Documentation:**
- ✅ Each module/tool well documented
- ✅ Features clearly listed
- ✅ Example queries provided
- ✅ Next steps outlined

---

## 📈 Migration Statistics

**Total SQL Lines:** ~700 lines
**Total Catalog Records:** 15
- Modules: 6
- Tools: 3
- Dependencies: 6

**Total Tenant Records:** 4
- Tenants: 1
- Tenant Modules: 1
- Tenant Tools: 2
- Sample Students: 1

**Time Taken:** Week 1 Day 4
**Status:** ✅ Completed

---

**End of Seed Data Summary**
**Next:** Week 1 Day 5 - Repository & Service Layer
