-- Migration: Rollback tenant_tools table
-- Description: Drop tenant_tools table and related objects
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Drop triggers
DROP TRIGGER IF EXISTS tenant_tools_install_count ON tenant_tools;
DROP TRIGGER IF EXISTS tenant_tools_updated_at ON tenant_tools;

-- Drop functions
DROP FUNCTION IF EXISTS update_tool_install_count();

-- Drop policies
DROP POLICY IF EXISTS tenant_isolation_tools ON tenant_tools;

-- Drop indexes
DROP INDEX IF EXISTS idx_tenant_tools_tenant_id;
DROP INDEX IF EXISTS idx_tenant_tools_tool_id;
DROP INDEX IF EXISTS idx_tenant_tools_module_id;
DROP INDEX IF EXISTS idx_tenant_tools_status;
DROP INDEX IF EXISTS idx_tenant_tools_enabled;
DROP INDEX IF EXISTS idx_tenant_tools_integration_status;
DROP INDEX IF EXISTS idx_tenant_tools_health_status;
DROP INDEX IF EXISTS idx_tenant_tools_subscription_status;
DROP INDEX IF EXISTS idx_tenant_tools_installed_at;
DROP INDEX IF EXISTS idx_tenant_tools_last_used;
DROP INDEX IF EXISTS idx_tenant_tools_next_billing;
DROP INDEX IF EXISTS idx_tenant_tools_trial_ends;
DROP INDEX IF EXISTS idx_tenant_tools_provider_account;
DROP INDEX IF EXISTS idx_tenant_tools_configuration;
DROP INDEX IF EXISTS idx_tenant_tools_limits;
DROP INDEX IF EXISTS idx_tenant_tools_usage;
DROP INDEX IF EXISTS idx_tenant_tools_webhook_config;
DROP INDEX IF EXISTS idx_tenant_tools_tenant_status;
DROP INDEX IF EXISTS idx_tenant_tools_tenant_enabled;
DROP INDEX IF EXISTS idx_tenant_tools_tool_status;
DROP INDEX IF EXISTS idx_tenant_tools_tenant_health;

-- Drop table
DROP TABLE IF EXISTS tenant_tools CASCADE;
