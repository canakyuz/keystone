# Database Migrations

## Overview
This directory contains PostgreSQL migration files for the NexSpaces multi-tenant SaaS platform.

## Migration Naming Convention
```
XXX_description.up.sql    # Forward migration
XXX_description.down.sql  # Rollback migration
```

## Running Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration status
make migrate-status

# Force to specific version
make migrate-force VERSION=XXX
```

## Migration Categories

### Schema Migrations (001-016)
Core database schema for multi-tenant platform:
- **001-005:** Core entities (tenants, users, websites, projects, students)
- **006-010:** Business entities (lessons, assignments, appointments, services)
- **011-013:** Content management (blog, sites)
- **014-016:** Payment processing (payments, refunds, events)

### Registry Migrations (017-022)
Template marketplace and module system:
- **017-018:** Module and tool definitions
- **019-020:** Dependency management
- **021-022:** Tenant activations (modules, tools)

### Seed Data Migrations (023-024)

#### 023_seed_registry_data
- **Purpose:** Populate core modules and tools registry
- **Environment:** All (development, staging, production)
- **Content:**
  - Default modules (LMS, CMS, CRM, etc.)
  - Default tools (Payment, Webhook, etc.)
  - Module/tool dependency mappings

#### Development Seed Script
- **Dosya:** `scripts/seed/dev_seed.sql`
- **Çalıştırma:** `make seed-dev`
- **Environment:** Sadece local/dev; production pipeline'da kullanılmamalı
- **İçerik:**
  - Kurucu tenant + owner (`owner@dev.keystone.local / DevPass123!`)
  - `canakyuz-dev` test tenantı ve sektör bazlı kullanıcılar (parola `DevPass123!`)
  - LMS modül aktivasyonu ve örnek öğrenci kaydı
- **Not:** Migration 024 kaldırıldı; migration zinciri artık seed veri içermiyor.

**Required Setup:**
```sql
-- Set environment before running (in development)
SET app.environment = 'development';

-- Replace placeholder credentials in tenant_tools configuration:
-- 1. Stripe API key: REPLACE_WITH_STRIPE_TEST_KEY
-- 2. Webhook secret: REPLACE_WITH_WEBHOOK_SECRET
```

### Schema Enhancement Migrations (025+)
- **025_add_tenant_schema_support:** Multi-tenant schema isolation support

## Development Workflow

### Creating a New Migration

```bash
# 1. Create migration files
touch migrations/026_your_migration_name.up.sql
touch migrations/026_your_migration_name.down.sql

# 2. Write migration SQL (see templates below)

# 3. Test migration
make migrate-up
make migrate-down  # Test rollback

# 4. Commit both files
git add migrations/026_*
git commit -m "feat(db): add your_migration_name"
```

### Migration Templates

#### Standard Table Migration
```sql
-- migrations/XXX_create_table.up.sql
CREATE TABLE IF NOT EXISTS table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_table_name_tenant ON table_name(tenant_id);

-- migrations/XXX_create_table.down.sql
DROP TABLE IF EXISTS table_name CASCADE;
```

#### Seed Data Migration (Development Only)
```sql
-- migrations/XXX_seed_data.up.sql
DO $$
DECLARE
    current_env TEXT;
BEGIN
    -- Production guard
    current_env := COALESCE(current_setting('app.environment', true), 'production');

    IF current_env = 'production' THEN
        RAISE EXCEPTION 'BLOCKED: Seed data cannot run in production. Environment: %', current_env
            USING HINT = 'Set "app.environment" to "development" or "staging"';
    END IF;

    -- Idempotency check
    IF EXISTS (SELECT 1 FROM table WHERE condition) THEN
        RAISE NOTICE 'Seed data already exists. Skipping.';
        RETURN;
    END IF;

    -- Insert seed data
    INSERT INTO table (...) VALUES (...) ON CONFLICT DO NOTHING;

    RAISE NOTICE 'Seed data inserted successfully';
END $$;

-- migrations/XXX_seed_data.down.sql
DELETE FROM table WHERE condition;
```

## Multi-Tenant Checklist ⚠️

**Every migration MUST ensure tenant isolation:**

- [ ] All tenant-scoped tables include `tenant_id UUID NOT NULL`
- [ ] Foreign key to tenants table: `REFERENCES tenants(id) ON DELETE CASCADE`
- [ ] Index on `tenant_id` (preferably as first column in composite indexes)
- [ ] Row Level Security (RLS) policies defined where applicable
- [ ] Cross-tenant access prevented at database level
- [ ] Seed data includes proper `tenant_id` values

## Security Best Practices

1. **Never commit sensitive data:**
   - Use placeholders: `REPLACE_WITH_*`
   - Store actual credentials in vault/secrets manager
   - Document required credentials in migration comments

2. **Production guards for seed data:**
   - Check `app.environment` setting
   - Raise exception in production
   - Use idempotent operations (`ON CONFLICT DO NOTHING`)

3. **Rollback strategy:**
   - Always provide `.down.sql` migration
   - Test rollback before merging
   - Use `CASCADE` carefully (document side effects)

4. **Testing:**
   - Test on local database first
   - Verify tenant isolation
   - Check performance on large datasets
   - Review execution plan for complex queries

## Troubleshooting

### Migration Already Applied
```bash
# Check current version
make migrate-status

# Force re-run (dangerous!)
make migrate-force VERSION=XXX
```

### Migration Failed Mid-Way
```bash
# Check schema_migrations table
make db-shell
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 10;

# Manually fix and re-run
make migrate-up
```

### Production Guard Triggered
```sql
-- Error: "BLOCKED: Test tenant seed cannot run in production"
-- Solution: This is working as intended. Seed migrations are development-only.
-- If you need similar data in production, create a separate deployment script.
```

## CI/CD Integration

Migrations run automatically in CI/CD pipeline:

```yaml
# .github/workflows/deploy.yml
- name: Run Database Migrations
  run: make migrate-up
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
    APP_ENVIRONMENT: ${{ env.ENVIRONMENT }}
```

**Environment-specific behavior:**
- **Development:** All migrations including seeds
- **Staging:** All migrations including seeds
- **Production:** Schema migrations only (seeds blocked by guard)

## Support

For migration issues:
1. Check this README
2. Review migration file comments
3. Check `schema_migrations` table
4. Contact DevOps team

---

**Last Updated:** 2024-10-06
**Maintainer:** NexSpaces DevOps Team
