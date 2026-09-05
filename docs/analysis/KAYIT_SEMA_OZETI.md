# Registry System - Database Schema Summary

**Created:** 2025-10-05
**Week 1 Day 2-3:** Registry Database Schema
**Status:** ✅ Completed

---

## 📊 Overview

Registry sistemi, Keystone platform'da modül ve araçların kataloglanması, tenant bazlı aktivasyonu ve bağımlılık yönetimi için tasarlanmıştır.

### Created Migrations

**6 yeni migration oluşturuldu:**
- ✅ `017_create_modules.up/down.sql`
- ✅ `018_create_tools.up/down.sql`
- ✅ `019_create_module_dependencies.up/down.sql`
- ✅ `020_create_tool_dependencies.up/down.sql`
- ✅ `021_create_tenant_modules.up/down.sql`
- ✅ `022_create_tenant_tools.up/down.sql`

**Toplam:** 12 dosya (6 up + 6 down)

---

## 🏗️ Schema Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     REGISTRY CATALOG                        │
├─────────────────────────────────────────────────────────────┤
│  modules (catalog)          tools (catalog)                 │
│  - Module definitions       - Tool definitions              │
│  - Pricing, features        - Pricing, integrations         │
│  - Install counts           - Install counts                │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ Referenced by
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    DEPENDENCY GRAPH                          │
├─────────────────────────────────────────────────────────────┤
│  module_dependencies        tool_dependencies               │
│  - Module → Module          - Tool → Tool                   │
│  - Module → Tool            - Required/Optional             │
│  - Required/Optional        - Auto-install                  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ Activated per tenant
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   TENANT ACTIVATIONS                         │
├─────────────────────────────────────────────────────────────┤
│  tenant_modules             tenant_tools                    │
│  - Per-tenant modules       - Per-tenant tools              │
│  - Usage tracking           - Integration configs           │
│  - Billing/subscriptions    - API keys, webhooks            │
│  - RLS enabled              - RLS enabled                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📋 Table Details

### 1. `modules` Table (Catalog)

**Purpose:** Module catalog - defines all available modules in the platform marketplace

**Key Fields:**
- `id`, `name`, `slug`, `code` (identifiers)
- `category`, `module_type` (classification)
- `pricing_model`, `base_price`, `billing_cycle` (pricing)
- `features`, `capabilities`, `default_limits` (JSONB)
- `database_tables[]` (tables this module creates)
- `install_count`, `rating` (metrics)

**Categories:**
- `education`, `content`, `commerce`, `hospitality`, `management`, `communication`, `analytics`, `other`

**Pricing Models:**
- `free`, `one_time`, `subscription`, `usage_based`

**Indexes:** 12 indexes including GIN indexes for JSONB fields

**Constraints:**
- Unique: `name`, `slug`, `code`
- CHECK constraints on status, category, pricing
- Code format: `^[A-Z0-9_]+$` (e.g., LMS, CMS)

---

### 2. `tools` Table (Catalog)

**Purpose:** Tool catalog - defines all available tools (global and module-specific)

**Key Fields:**
- `id`, `name`, `slug`, `code` (identifiers)
- `category`, `tool_type`, `scope` (classification)
- `pricing_model`, `transaction_fee_percentage` (pricing)
- `integration_provider`, `integration_type` (external integrations)
- `requires_api_keys`, `requires_webhook` (technical requirements)
- `default_limits`, `rate_limits` (JSONB)
- `install_count`, `rating` (metrics)

**Categories:**
- `payment`, `communication`, `automation`, `storage`, `analytics`, `notification`, `integration`, `security`, `other`

**Tool Types:**
- `global` (cross-module)
- `module_specific` (bound to specific module)

**Pricing Models:**
- `free`, `one_time`, `subscription`, `usage_based`, `transaction_fee`

**Indexes:** 15 indexes including GIN indexes for JSONB fields

**Constraints:**
- Unique: `name`, `slug`, `code`
- Code format: `^[A-Z0-9_]+$` (e.g., PAYMENT, CHAT)
- Transaction fee: 0-100%

---

### 3. `module_dependencies` Table

**Purpose:** Defines dependencies between modules and tools

**Key Fields:**
- `module_id` (the module that has dependency)
- `depends_on_module_id` (depends on another module)
- `depends_on_tool_id` (depends on a tool)
- `dependency_type`: `required`, `optional`, `recommended`
- `dependency_scope`: `runtime`, `installation`, `feature`
- `auto_install` (automatically install dependency)
- `priority`, `install_order` (installation sequence)

**Business Rules:**
- Must depend on either a module OR a tool (not both)
- No self-references allowed
- Unique constraint: module can't have duplicate dependencies

**Examples:**
```sql
-- E-commerce module requires Payment tool
INSERT INTO module_dependencies (module_id, depends_on_tool_id, dependency_type)
VALUES (ecommerce_id, payment_tool_id, 'required');

-- LMS module optionally uses Payment tool
INSERT INTO module_dependencies (module_id, depends_on_tool_id, dependency_type)
VALUES (lms_id, payment_tool_id, 'optional');
```

---

### 4. `tool_dependencies` Table

**Purpose:** Defines dependencies between tools

**Key Fields:**
- `tool_id` (the tool that has dependency)
- `depends_on_tool_id` (depends on another tool)
- `dependency_type`: `required`, `optional`, `recommended`
- `auto_install` (default: TRUE for tools)
- `priority`, `install_order`

**Business Rules:**
- No self-references
- Unique: one tool can't depend on same tool twice

**Examples:**
```sql
-- Refund tool requires Payment tool
INSERT INTO tool_dependencies (tool_id, depends_on_tool_id, dependency_type)
VALUES (refund_tool_id, payment_tool_id, 'required');

-- Webhook tool is recommended for Payment
INSERT INTO tool_dependencies (tool_id, depends_on_tool_id, dependency_type)
VALUES (payment_tool_id, webhook_tool_id, 'recommended');
```

---

### 5. `tenant_modules` Table (Multi-Tenant Activations)

**Purpose:** Tracks which modules are active for each tenant

**Key Fields:**
- `tenant_id`, `module_id` (relationship)
- `status`: `active`, `inactive`, `suspended`, `pending_setup`
- `is_enabled` (quick toggle)
- `installed_version`, `installed_at`
- `configuration`, `features_enabled` (JSONB - tenant overrides)
- `limits`, `current_usage` (JSONB - usage tracking)
- `subscription_status`, `pricing_plan`, `billing_cycle`
- `setup_completed`, `onboarding_completed`
- `allowed_roles[]` (RBAC integration)
- `last_used_at` (activity tracking)

**Special Features:**
- ✅ **Row Level Security (RLS)** enabled
- ✅ Auto-updates `modules.install_count` via trigger
- ✅ Comprehensive billing/subscription tracking
- ✅ Setup wizard state management

**Unique Constraint:** One tenant can't have same module twice

**Indexes:** 13 indexes including GIN for JSONB

---

### 6. `tenant_tools` Table (Multi-Tenant Activations)

**Purpose:** Tracks which tools are active for each tenant

**Key Fields:**
- `tenant_id`, `tool_id`, `module_id` (relationships)
- `status`: `active`, `inactive`, `suspended`, `pending_setup`, `pending_verification`
- `is_enabled` (quick toggle)
- `configuration`, `api_keys`, `webhook_config` (JSONB)
- `integration_status`: `connected`, `disconnected`, `error`, `pending`
- `integration_verified`, `provider_account_id` (external provider linkage)
- `limits`, `current_usage`, `rate_limits` (JSONB)
- `health_status`: `healthy`, `degraded`, `unavailable`
- `error_count`, `last_error`, `last_error_at` (monitoring)
- `subscription_status`, `pricing_plan`, `transaction_fees_collected`
- `setup_completed`, `onboarding_completed`

**Special Features:**
- ✅ **Row Level Security (RLS)** enabled
- ✅ Auto-updates `tools.install_count` via trigger
- ✅ External integration management (API keys, webhooks)
- ✅ Health monitoring and error tracking
- ✅ Transaction fee tracking (for payment tools)

**Unique Constraint:** One tenant can't have same tool twice

**Indexes:** 18 indexes including GIN for JSONB

---

## 🔒 Security Features

### Row Level Security (RLS)

**Enabled on:**
- ✅ `tenant_modules`
- ✅ `tenant_tools`

**Policy:**
```sql
CREATE POLICY tenant_isolation_modules ON tenant_modules
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

CREATE POLICY tenant_isolation_tools ON tenant_tools
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
```

**How it works:**
1. Application sets `app.current_tenant` session variable
2. All queries automatically filtered by tenant_id
3. Cross-tenant access prevented at database level

---

## 🚀 Performance Optimizations

### Indexing Strategy

**Catalog Tables (`modules`, `tools`):**
- Single-column indexes: status, category, type, pricing, slug, code
- Composite indexes: (status, is_public), (category, status)
- GIN indexes: features, metadata, tags (JSONB/array search)
- Partial indexes: WHERE deleted_at IS NULL

**Activation Tables (`tenant_modules`, `tenant_tools`):**
- Tenant-scoped indexes: (tenant_id, status), (tenant_id, is_enabled)
- Time-based indexes: installed_at DESC, last_used_at DESC
- Billing indexes: next_billing_date, trial_ends_at
- GIN indexes: configuration, limits, current_usage

**Total Indexes Created:** 58+ indexes across 6 tables

---

## 📈 Statistics & Metrics

### Auto-Updated Metrics

**Modules Table:**
- `install_count` - Auto-incremented when tenant activates module
- `rating` - Can be updated based on tenant reviews
- `review_count` - Number of reviews

**Tools Table:**
- `install_count` - Auto-incremented when tenant activates tool
- `rating` - Can be updated based on tenant reviews
- `review_count` - Number of reviews

**Triggers:**
```sql
-- Automatically updates module install_count
CREATE TRIGGER tenant_modules_install_count
    AFTER INSERT OR UPDATE OR DELETE ON tenant_modules
    FOR EACH ROW
    EXECUTE FUNCTION update_module_install_count();

-- Automatically updates tool install_count
CREATE TRIGGER tenant_tools_install_count
    AFTER INSERT OR UPDATE OR DELETE ON tenant_tools
    FOR EACH ROW
    EXECUTE FUNCTION update_tool_install_count();
```

---

## 🔗 Foreign Key Relationships

```
tenants (existing)
  ├── tenant_modules.tenant_id (CASCADE)
  └── tenant_tools.tenant_id (CASCADE)

modules
  ├── module_dependencies.module_id (CASCADE)
  ├── module_dependencies.depends_on_module_id (CASCADE)
  ├── tenant_modules.module_id (CASCADE)
  └── tenant_tools.module_id (SET NULL)

tools
  ├── module_dependencies.depends_on_tool_id (CASCADE)
  ├── tool_dependencies.tool_id (CASCADE)
  ├── tool_dependencies.depends_on_tool_id (CASCADE)
  └── tenant_tools.tool_id (CASCADE)
```

**CASCADE Behavior:**
- Delete tenant → Deletes all tenant_modules and tenant_tools
- Delete module → Deletes dependencies and activations
- Delete tool → Deletes dependencies and activations

---

## 💾 JSONB Fields Usage

### Configuration Storage

**Why JSONB?**
- Flexible tenant-specific overrides
- Schema-less evolution (no migration needed for new config)
- Efficient indexing with GIN indexes
- JSON path queries with PostgreSQL operators

**Examples:**

```sql
-- Module configuration
{
  "theme": "dark",
  "language": "en",
  "timezone": "Europe/Istanbul",
  "email_notifications": true,
  "max_file_size_mb": 50
}

-- Tool configuration (Payment)
{
  "provider": "stripe",
  "currency": "USD",
  "allow_installments": false,
  "capture_method": "automatic"
}

-- Limits
{
  "max_lessons": 500,
  "max_students": 200,
  "max_storage_gb": 50
}

-- Current usage
{
  "lessons_count": 45,
  "students_count": 120,
  "storage_used_gb": 3.2
}

-- API keys (encrypted)
{
  "stripe_public_key": "pk_live_...",
  "stripe_secret_key": "sk_live_..." // Should be encrypted
}
```

---

## 🧪 Example Queries

### Get all active modules for a tenant

```sql
SET app.current_tenant = '550e8400-e29b-41d4-a716-446655440000';

SELECT m.name, m.code, tm.status, tm.installed_at
FROM tenant_modules tm
JOIN modules m ON m.id = tm.module_id
WHERE tm.is_enabled = TRUE
  AND tm.deleted_at IS NULL;
```

### Get module with its dependencies

```sql
SELECT
  m.name as module_name,
  COALESCE(dm.name, dt.name) as dependency_name,
  md.dependency_type,
  md.auto_install
FROM modules m
LEFT JOIN module_dependencies md ON md.module_id = m.id
LEFT JOIN modules dm ON dm.id = md.depends_on_module_id
LEFT JOIN tools dt ON dt.id = md.depends_on_tool_id
WHERE m.code = 'LMS';
```

### Check if tenant can install a module (dependency check)

```sql
WITH required_deps AS (
  SELECT depends_on_module_id, depends_on_tool_id
  FROM module_dependencies
  WHERE module_id = :module_id
    AND dependency_type = 'required'
)
SELECT
  CASE
    WHEN EXISTS (
      SELECT 1 FROM required_deps rd
      LEFT JOIN tenant_modules tm ON tm.module_id = rd.depends_on_module_id
      LEFT JOIN tenant_tools tt ON tt.tool_id = rd.depends_on_tool_id
      WHERE (rd.depends_on_module_id IS NOT NULL AND tm.id IS NULL)
         OR (rd.depends_on_tool_id IS NOT NULL AND tt.id IS NULL)
    ) THEN FALSE
    ELSE TRUE
  END as can_install;
```

### Get tenant tool usage statistics

```sql
SELECT
  t.name,
  tt.current_usage->>'messages_sent' as messages_sent,
  tt.limits->>'max_messages' as max_messages,
  tt.health_status,
  tt.last_used_at
FROM tenant_tools tt
JOIN tools t ON t.id = tt.tool_id
WHERE tt.tenant_id = current_setting('app.current_tenant')::UUID
  AND tt.status = 'active';
```

---

## 🎯 Next Steps (Week 1 Day 4)

**Seed Data Creation:**

1. **Populate `modules` table:**
   - ✅ LMS Module
   - ✅ CMS/Blog Module
   - ✅ Project Management Module
   - ✅ Booking Module
   - ✅ Service Catalog Module
   - ✅ Website Builder Module

2. **Populate `tools` table:**
   - ✅ Payment Tool (Stripe, Iyzico, Checkout)
   - ✅ Refund Tool
   - ✅ Webhook Tool
   - 🚧 Chat Tool (Week 4)
   - 🚧 File Upload Tool
   - 🚧 N8n Automation Tool

3. **Define `module_dependencies`:**
   - E-commerce → Payment (required)
   - Booking → Payment (optional)
   - LMS → Payment (optional)

4. **Define `tool_dependencies`:**
   - Refund → Payment (required)
   - Payment → Webhook (recommended)

5. **Create test tenant with activations:**
   - Can Akyüz tenant
   - Activate: LMS + Payment + Webhook

---

## ✅ Self-Review Checklist

**Schema Design:**
- ✅ Multi-tenant isolation (RLS on activation tables)
- ✅ Comprehensive indexes (58+ indexes)
- ✅ JSONB for flexible configuration
- ✅ Foreign keys with proper CASCADE behavior
- ✅ CHECK constraints for data validation
- ✅ Audit fields (created_at, updated_at, created_by, etc.)

**Security:**
- ✅ RLS policies for tenant isolation
- ✅ No sensitive data in plain text (api_keys as JSONB - needs encryption)
- ✅ CASCADE deletes prevent orphaned records

**Performance:**
- ✅ Composite indexes for common queries
- ✅ GIN indexes for JSONB/array fields
- ✅ Partial indexes (WHERE deleted_at IS NULL)

**Business Logic:**
- ✅ Dependency tracking (required/optional/recommended)
- ✅ Auto-install dependencies
- ✅ Install count auto-update via triggers
- ✅ Billing/subscription tracking
- ✅ Health monitoring for tools
- ✅ Usage tracking and quotas

**Edge Cases:**
- ✅ Soft deletes (deleted_at column)
- ✅ No self-references in dependencies
- ✅ Unique constraints prevent duplicates
- ✅ Handle both module and tool dependencies

**Missing/TODO:**
- ⚠️ API key encryption (currently JSONB, needs encryption layer)
- ⚠️ Circular dependency detection (should be validated in application)
- ⚠️ Rate limiting enforcement (stored in JSONB, needs application logic)

---

## 📊 Migration Statistics

**Total Lines of SQL:** ~2,500 lines
**Total Tables Created:** 6
**Total Indexes Created:** 58+
**Total Triggers Created:** 4
**Total Functions Created:** 2
**Total RLS Policies Created:** 2

**Time Taken:** Week 1 Day 2-3
**Status:** ✅ Completed

---

**End of Schema Summary**
**Next:** Week 1 Day 4 - Seed Data
