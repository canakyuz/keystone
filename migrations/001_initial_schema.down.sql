-- NexSpaces Migration Rollback
-- Migration: 001_initial_schema
-- Description: Drops all tables and schema created in initial migration

-- Drop triggers first
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_templates_updated_at ON templates;
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS get_current_tenant();
DROP FUNCTION IF EXISTS set_tenant_context(UUID);

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS template_installations CASCADE;
DROP TABLE IF EXISTS templates CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- Drop custom types
DROP TYPE IF EXISTS billing_cycle;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS pricing_model;
DROP TYPE IF EXISTS template_category;
DROP TYPE IF EXISTS template_visibility;
DROP TYPE IF EXISTS template_status;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS tenant_status;

-- Drop roles
DROP ROLE IF EXISTS readonly_user;
DROP ROLE IF EXISTS application_user;