-- Migration: Create tenant_modules table
-- Description: Tracks which modules are active for each tenant (multi-tenant activation)
-- Author: Keystone Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tenant_modules table (tenant-scoped module activations)
CREATE TABLE IF NOT EXISTS tenant_modules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tenant and module relationship
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,

    -- Activation status
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'inactive', 'suspended', 'pending_setup'
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,

    -- Installation info
    installed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE, -- Last time this module was accessed

    -- Version tracking
    installed_version VARCHAR(20) NOT NULL,
    latest_compatible_version VARCHAR(20),

    -- Configuration (tenant-specific overrides)
    configuration JSONB DEFAULT '{}'::JSONB, -- Tenant-specific module config
    features_enabled JSONB DEFAULT '[]'::JSONB, -- Which features are enabled for this tenant

    -- Limits and quotas (tenant-specific overrides)
    limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"max_lessons": 500, "max_students": 200}
    current_usage JSONB DEFAULT '{}'::JSONB, -- e.g., {"lessons_count": 45, "students_count": 120}

    -- Billing and subscription
    subscription_status VARCHAR(20), -- 'trial', 'active', 'canceled', 'past_due'
    subscription_start TIMESTAMP WITH TIME ZONE,
    subscription_end TIMESTAMP WITH TIME ZONE,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    next_billing_date TIMESTAMP WITH TIME ZONE,

    -- Payment info (if module is paid)
    pricing_plan VARCHAR(50), -- e.g., 'free', 'basic', 'pro', 'enterprise'
    billing_cycle VARCHAR(20), -- 'monthly', 'yearly'
    amount_paid DECIMAL(10,2), -- Last payment amount
    currency VARCHAR(3) DEFAULT 'USD',

    -- Setup and onboarding
    setup_completed BOOLEAN DEFAULT FALSE,
    setup_steps_completed JSONB DEFAULT '[]'::JSONB, -- Array of completed setup steps
    onboarding_completed BOOLEAN DEFAULT FALSE,

    -- Permissions (which roles can access this module)
    allowed_roles TEXT[], -- Array of role names
    restricted_features JSONB DEFAULT '[]'::JSONB, -- Features restricted for this tenant

    -- Metadata
    notes TEXT, -- Admin notes about this tenant's module installation
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,
    activated_by UUID, -- User who activated this module
    deactivated_by UUID, -- User who deactivated this module

    -- Constraints
    CONSTRAINT tenant_modules_status_check CHECK (status IN ('active', 'inactive', 'suspended', 'pending_setup')),
    CONSTRAINT tenant_modules_subscription_status_check CHECK (subscription_status IS NULL OR subscription_status IN ('trial', 'active', 'canceled', 'past_due', 'suspended')),
    CONSTRAINT tenant_modules_pricing_plan_check CHECK (pricing_plan IS NULL OR pricing_plan IN ('free', 'basic', 'pro', 'enterprise', 'custom')),
    CONSTRAINT tenant_modules_billing_cycle_check CHECK (billing_cycle IS NULL OR billing_cycle IN ('monthly', 'yearly', 'one_time')),
    CONSTRAINT tenant_modules_unique_tenant_module UNIQUE (tenant_id, module_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_tenant_modules_tenant_id ON tenant_modules(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_module_id ON tenant_modules(module_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_status ON tenant_modules(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_enabled ON tenant_modules(is_enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_subscription_status ON tenant_modules(subscription_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_installed_at ON tenant_modules(installed_at DESC);
CREATE INDEX idx_tenant_modules_last_used ON tenant_modules(last_used_at DESC);
CREATE INDEX idx_tenant_modules_next_billing ON tenant_modules(next_billing_date) WHERE subscription_status = 'active';
CREATE INDEX idx_tenant_modules_trial_ends ON tenant_modules(trial_ends_at) WHERE subscription_status = 'trial';
CREATE INDEX idx_tenant_modules_configuration ON tenant_modules USING GIN (configuration);
CREATE INDEX idx_tenant_modules_limits ON tenant_modules USING GIN (limits);
CREATE INDEX idx_tenant_modules_usage ON tenant_modules USING GIN (current_usage);

-- Composite indexes for common queries
CREATE INDEX idx_tenant_modules_tenant_status ON tenant_modules(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_tenant_enabled ON tenant_modules(tenant_id, is_enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenant_modules_module_status ON tenant_modules(module_id, status) WHERE deleted_at IS NULL;

-- Row Level Security (RLS)
ALTER TABLE tenant_modules ENABLE ROW LEVEL SECURITY;

-- Policy: Tenants can only see their own module activations
CREATE POLICY tenant_isolation_modules ON tenant_modules
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- Create trigger for updated_at
CREATE TRIGGER tenant_modules_updated_at
    BEFORE UPDATE ON tenant_modules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to update module install_count when tenant activates/deactivates
CREATE OR REPLACE FUNCTION update_module_install_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.status = 'active' THEN
        UPDATE modules SET install_count = install_count + 1 WHERE id = NEW.module_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status != 'active' AND NEW.status = 'active' THEN
        UPDATE modules SET install_count = install_count + 1 WHERE id = NEW.module_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'active' AND NEW.status != 'active' THEN
        UPDATE modules SET install_count = GREATEST(install_count - 1, 0) WHERE id = NEW.module_id;
    ELSIF TG_OP = 'DELETE' AND OLD.status = 'active' THEN
        UPDATE modules SET install_count = GREATEST(install_count - 1, 0) WHERE id = OLD.module_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update module install_count
CREATE TRIGGER tenant_modules_install_count
    AFTER INSERT OR UPDATE OR DELETE ON tenant_modules
    FOR EACH ROW
    EXECUTE FUNCTION update_module_install_count();

-- Add comments for documentation
COMMENT ON TABLE tenant_modules IS 'Tracks which modules are active for each tenant (multi-tenant module activation)';
COMMENT ON COLUMN tenant_modules.tenant_id IS 'Tenant who has this module installed';
COMMENT ON COLUMN tenant_modules.module_id IS 'Module that is installed';
COMMENT ON COLUMN tenant_modules.status IS 'Module activation status: active, inactive, suspended, pending_setup';
COMMENT ON COLUMN tenant_modules.is_enabled IS 'Whether module is currently enabled (can be toggled)';
COMMENT ON COLUMN tenant_modules.configuration IS 'Tenant-specific module configuration (JSONB)';
COMMENT ON COLUMN tenant_modules.limits IS 'Tenant-specific usage limits (JSONB)';
COMMENT ON COLUMN tenant_modules.current_usage IS 'Current usage metrics for this tenant (JSONB)';
COMMENT ON COLUMN tenant_modules.subscription_status IS 'Billing subscription status';
COMMENT ON COLUMN tenant_modules.pricing_plan IS 'Pricing tier for this tenant';
COMMENT ON COLUMN tenant_modules.setup_completed IS 'Whether initial setup wizard is completed';
COMMENT ON COLUMN tenant_modules.onboarding_completed IS 'Whether onboarding process is completed';
COMMENT ON COLUMN tenant_modules.allowed_roles IS 'Array of roles allowed to access this module';
COMMENT ON COLUMN tenant_modules.last_used_at IS 'Last time this module was accessed by tenant';
