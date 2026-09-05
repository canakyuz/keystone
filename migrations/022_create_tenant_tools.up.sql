-- Migration: Create tenant_tools table
-- Description: Tracks which tools are active for each tenant (multi-tenant tool activation)
-- Author: Keystone Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tenant_tools table (tenant-scoped tool activations)
CREATE TABLE IF NOT EXISTS tenant_tools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tenant and tool relationship
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    module_id UUID REFERENCES modules(id) ON DELETE SET NULL, -- Optional: if tool is activated via a module

    -- Activation status
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'inactive', 'suspended', 'pending_setup', 'pending_verification'
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- Installation info
    installed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE, -- Last time this tool was accessed

    -- Version tracking
    installed_version VARCHAR(20) NOT NULL,
    latest_compatible_version VARCHAR(20),

    -- Configuration (tenant-specific overrides)
    configuration JSONB DEFAULT '{}'::JSONB, -- Tenant-specific tool config
    api_keys JSONB DEFAULT '{}'::JSONB, -- Encrypted API keys (if required)
    webhook_config JSONB DEFAULT '{}'::JSONB, -- Webhook configuration

    -- Integration settings (for external tools)
    integration_enabled BOOLEAN DEFAULT TRUE,
    integration_status VARCHAR(50), -- 'connected', 'disconnected', 'error', 'pending'
    integration_verified BOOLEAN DEFAULT FALSE,
    integration_verified_at TIMESTAMP WITH TIME ZONE,
    provider_account_id VARCHAR(255), -- External provider account ID (e.g., Stripe account)

    -- Limits and quotas (tenant-specific overrides)
    limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"max_messages": 10000, "max_storage_gb": 50}
    current_usage JSONB DEFAULT '{}'::JSONB, -- e.g., {"messages_sent": 245, "storage_used_gb": 3.2}
    rate_limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"requests_per_minute": 100}

    -- Billing and subscription (for paid tools)
    subscription_status VARCHAR(20), -- 'trial', 'active', 'canceled', 'past_due'
    subscription_start TIMESTAMP WITH TIME ZONE,
    subscription_end TIMESTAMP WITH TIME ZONE,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    next_billing_date TIMESTAMP WITH TIME ZONE,

    -- Payment info (if tool is paid)
    pricing_plan VARCHAR(50), -- e.g., 'free', 'basic', 'pro', 'enterprise'
    billing_cycle VARCHAR(20), -- 'monthly', 'yearly', 'per_transaction'
    amount_paid DECIMAL(10,2), -- Last payment amount
    currency VARCHAR(3) DEFAULT 'USD',
    transaction_fees_collected DECIMAL(10,2) DEFAULT 0.00, -- For transaction-based tools

    -- Setup and onboarding
    setup_completed BOOLEAN DEFAULT FALSE,
    setup_steps_completed JSONB DEFAULT '[]'::JSONB, -- Array of completed setup steps
    onboarding_completed BOOLEAN DEFAULT FALSE,

    -- Permissions (which roles can access this tool)
    allowed_roles TEXT[], -- Array of role names
    restricted_features JSONB DEFAULT '[]'::JSONB, -- Features restricted for this tenant

    -- Health and monitoring
    health_status VARCHAR(20) DEFAULT 'healthy', -- 'healthy', 'degraded', 'unavailable'
    last_health_check TIMESTAMP WITH TIME ZONE,
    error_count INTEGER DEFAULT 0, -- Recent error count
    last_error TEXT, -- Last error message
    last_error_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    notes TEXT, -- Admin notes about this tenant's tool installation
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,
    activated_by UUID, -- User who activated this tool
    deactivated_by UUID, -- User who deactivated this tool

    -- Constraints
    CONSTRAINT tenant_tools_status_check CHECK (status IN ('active', 'inactive', 'suspended', 'pending_setup', 'pending_verification')),
    CONSTRAINT tenant_tools_integration_status_check CHECK (integration_status IS NULL OR integration_status IN ('connected', 'disconnected', 'error', 'pending')),
    CONSTRAINT tenant_tools_health_status_check CHECK (health_status IN ('healthy', 'degraded', 'unavailable')),
    CONSTRAINT tenant_tools_subscription_status_check CHECK (subscription_status IS NULL OR subscription_status IN ('trial', 'active', 'canceled', 'past_due', 'suspended')),
    CONSTRAINT tenant_tools_pricing_plan_check CHECK (pricing_plan IS NULL OR pricing_plan IN ('free', 'basic', 'pro', 'enterprise', 'custom')),
    CONSTRAINT tenant_tools_billing_cycle_check CHECK (billing_cycle IS NULL OR billing_cycle IN ('monthly', 'yearly', 'per_transaction', 'one_time')),
    CONSTRAINT tenant_tools_error_count_positive CHECK (error_count >= 0),
    CONSTRAINT tenant_tools_unique_tenant_tool UNIQUE (tenant_id, tool_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_tenant_tools_tenant_id ON tenant_tools(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_tool_id ON tenant_tools(tool_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_module_id ON tenant_tools(module_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_status ON tenant_tools(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_enabled ON tenant_tools(is_enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_integration_status ON tenant_tools(integration_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_health_status ON tenant_tools(health_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_subscription_status ON tenant_tools(subscription_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_installed_at ON tenant_tools(installed_at DESC);
CREATE INDEX idx_tenant_tools_last_used ON tenant_tools(last_used_at DESC);
CREATE INDEX idx_tenant_tools_next_billing ON tenant_tools(next_billing_date) WHERE subscription_status = 'active';
CREATE INDEX idx_tenant_tools_trial_ends ON tenant_tools(trial_ends_at) WHERE subscription_status = 'trial';
CREATE INDEX idx_tenant_tools_provider_account ON tenant_tools(provider_account_id) WHERE provider_account_id IS NOT NULL;
CREATE INDEX idx_tenant_tools_configuration ON tenant_tools USING GIN (configuration);
CREATE INDEX idx_tenant_tools_limits ON tenant_tools USING GIN (limits);
CREATE INDEX idx_tenant_tools_usage ON tenant_tools USING GIN (current_usage);
CREATE INDEX idx_tenant_tools_webhook_config ON tenant_tools USING GIN (webhook_config);

-- Composite indexes for common queries
CREATE INDEX idx_tenant_tools_tenant_status ON tenant_tools(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_tenant_enabled ON tenant_tools(tenant_id, is_enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_tool_status ON tenant_tools(tool_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_tools_tenant_health ON tenant_tools(tenant_id, health_status) WHERE deleted_at IS NULL;

-- Row Level Security (RLS)
ALTER TABLE tenant_tools ENABLE ROW LEVEL SECURITY;

-- Policy: Tenants can only see their own tool activations
CREATE POLICY tenant_isolation_tools ON tenant_tools
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- Create trigger for updated_at
CREATE TRIGGER tenant_tools_updated_at
    BEFORE UPDATE ON tenant_tools
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to update tool install_count when tenant activates/deactivates
CREATE OR REPLACE FUNCTION update_tool_install_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'active' THEN
        UPDATE tools SET install_count = install_count + 1 WHERE id = NEW.tool_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status != 'active' AND NEW.status = 'active' THEN
        UPDATE tools SET install_count = install_count + 1 WHERE id = NEW.tool_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'active' AND NEW.status != 'active' THEN
        UPDATE tools SET install_count = GREATEST(install_count - 1, 0) WHERE id = NEW.tool_id;
    ELSIF TG_OP = 'DELETE' AND OLD.status = 'active' THEN
        UPDATE tools SET install_count = GREATEST(install_count - 1, 0) WHERE id = OLD.tool_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update tool install_count
CREATE TRIGGER tenant_tools_install_count
    AFTER INSERT OR UPDATE OR DELETE ON tenant_tools
    FOR EACH ROW
    EXECUTE FUNCTION update_tool_install_count();

-- Add comments for documentation
COMMENT ON TABLE tenant_tools IS 'Tracks which tools are active for each tenant (multi-tenant tool activation)';
COMMENT ON COLUMN tenant_tools.tenant_id IS 'Tenant who has this tool installed';
COMMENT ON COLUMN tenant_tools.tool_id IS 'Tool that is installed';
COMMENT ON COLUMN tenant_tools.module_id IS 'Optional: Module that activated this tool (if any)';
COMMENT ON COLUMN tenant_tools.status IS 'Tool activation status: active, inactive, suspended, pending_setup, pending_verification';
COMMENT ON COLUMN tenant_tools.is_enabled IS 'Whether tool is currently enabled (can be toggled)';
COMMENT ON COLUMN tenant_tools.configuration IS 'Tenant-specific tool configuration (JSONB)';
COMMENT ON COLUMN tenant_tools.api_keys IS 'Encrypted external API keys (JSONB)';
COMMENT ON COLUMN tenant_tools.webhook_config IS 'Webhook endpoints and configuration (JSONB)';
COMMENT ON COLUMN tenant_tools.integration_status IS 'External integration status: connected, disconnected, error, pending';
COMMENT ON COLUMN tenant_tools.provider_account_id IS 'External provider account identifier (e.g., Stripe account ID)';
COMMENT ON COLUMN tenant_tools.limits IS 'Tenant-specific usage limits (JSONB)';
COMMENT ON COLUMN tenant_tools.current_usage IS 'Current usage metrics for this tenant (JSONB)';
COMMENT ON COLUMN tenant_tools.rate_limits IS 'API rate limits for this tenant (JSONB)';
COMMENT ON COLUMN tenant_tools.subscription_status IS 'Billing subscription status';
COMMENT ON COLUMN tenant_tools.pricing_plan IS 'Pricing tier for this tenant';
COMMENT ON COLUMN tenant_tools.transaction_fees_collected IS 'Total transaction fees collected (for payment tools)';
COMMENT ON COLUMN tenant_tools.health_status IS 'Tool health: healthy, degraded, unavailable';
COMMENT ON COLUMN tenant_tools.error_count IS 'Recent error count for monitoring';
COMMENT ON COLUMN tenant_tools.setup_completed IS 'Whether initial setup wizard is completed';
COMMENT ON COLUMN tenant_tools.allowed_roles IS 'Array of roles allowed to access this tool';
COMMENT ON COLUMN tenant_tools.last_used_at IS 'Last time this tool was accessed by tenant';
