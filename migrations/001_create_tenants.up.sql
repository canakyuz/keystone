-- Migration: Create tenants table
-- Description: Multi-tenant isolation foundation
-- Author: Keystone Team

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create application user role for RLS policies
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'application_user') THEN
        CREATE ROLE application_user;
    END IF;
END
$$;

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),

    -- Status and subscription
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    plan VARCHAR(20) NOT NULL DEFAULT 'free',

    -- Subscription dates
    subscription_start TIMESTAMP WITH TIME ZONE,
    subscription_end TIMESTAMP WITH TIME ZONE,
    trial_ends_at TIMESTAMP WITH TIME ZONE,

    -- Custom domain
    custom_domain VARCHAR(255),
    custom_domain_verified BOOLEAN NOT NULL DEFAULT FALSE,
    custom_domain_verified_at TIMESTAMP WITH TIME ZONE,

    -- Settings and metadata (JSONB for flexibility)
    settings JSONB DEFAULT '{}'::JSONB,
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'inactive', 'trial')),
    CONSTRAINT tenants_plan_check CHECK (plan IN ('free', 'starter', 'pro', 'enterprise')),
    CONSTRAINT tenants_name_length CHECK (LENGTH(name) >= 2 AND LENGTH(name) <= 100),
    CONSTRAINT tenants_slug_length CHECK (LENGTH(slug) >= 2 AND LENGTH(slug) <= 50),
    CONSTRAINT tenants_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

-- Create indexes for better query performance
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_plan ON tenants(plan) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_email ON tenants(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_slug ON tenants(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_custom_domain ON tenants(custom_domain) WHERE custom_domain IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC);

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE tenants IS 'Multi-tenant isolation table - root of tenant hierarchy';
COMMENT ON COLUMN tenants.id IS 'Unique tenant identifier';
COMMENT ON COLUMN tenants.slug IS 'URL-friendly tenant identifier';
COMMENT ON COLUMN tenants.status IS 'Tenant status: active, suspended, inactive, trial';
COMMENT ON COLUMN tenants.plan IS 'Subscription plan: free, starter, pro, enterprise';

-- Insert default tenant for development
-- Development seed artık scripts/seed/dev_seed.sql içinde yönetiliyor
