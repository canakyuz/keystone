-- Migration: Rollback tenant_modules table
-- Description: Drop tenant_modules table and related objects
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Drop triggers
DROP TRIGGER IF EXISTS tenant_modules_install_count ON tenant_modules;
DROP TRIGGER IF EXISTS tenant_modules_updated_at ON tenant_modules;

-- Drop functions
DROP FUNCTION IF EXISTS update_module_install_count();

-- Drop policies
DROP POLICY IF EXISTS tenant_isolation_modules ON tenant_modules;

-- Drop indexes
DROP INDEX IF EXISTS idx_tenant_modules_tenant_id;
DROP INDEX IF EXISTS idx_tenant_modules_module_id;
DROP INDEX IF EXISTS idx_tenant_modules_status;
DROP INDEX IF EXISTS idx_tenant_modules_enabled;
DROP INDEX IF EXISTS idx_tenant_modules_subscription_status;
DROP INDEX IF EXISTS idx_tenant_modules_installed_at;
DROP INDEX IF EXISTS idx_tenant_modules_last_used;
DROP INDEX IF EXISTS idx_tenant_modules_next_billing;
DROP INDEX IF EXISTS idx_tenant_modules_trial_ends;
DROP INDEX IF EXISTS idx_tenant_modules_configuration;
DROP INDEX IF EXISTS idx_tenant_modules_limits;
DROP INDEX IF EXISTS idx_tenant_modules_usage;
DROP INDEX IF EXISTS idx_tenant_modules_tenant_status;
DROP INDEX IF EXISTS idx_tenant_modules_tenant_enabled;
DROP INDEX IF EXISTS idx_tenant_modules_module_status;

-- Drop table
DROP TABLE IF EXISTS tenant_modules CASCADE;
