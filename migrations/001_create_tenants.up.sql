-- Migration: Create tenants table
-- Description: Multi-tenant isolation foundation
-- Author: Keystone Team

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- The application role the RLS policies are written against.
--
-- Attempt the create and catch the duplicate, rather than checking pg_roles first.
--
-- WHY: roles are cluster-wide, not per-database, while this migration runs once per
-- database. The test suite creates an isolated database per test and runs the whole
-- migration chain in each, in parallel, against one cluster. Several of them reached the
-- IF NOT EXISTS at the same moment, all saw the role missing, and all issued CREATE ROLE;
-- one won and the rest failed on pg_authid_rolname_index.
--
-- It is the same check-then-write race this repository documents for idempotency keys in
-- migration 031, and the fix is the same shape: let the database enforce uniqueness and
-- handle the violation. A guard that reads before it writes is not a guard.
--
-- It went unnoticed locally because the role already existed from an earlier run, so the
-- branch was never taken. CI starts from an empty cluster every time, which is where the
-- race became visible.
--
-- Both exception classes are caught, and that is not belt and braces. PostgreSQL raises
-- duplicate_object when its own check finds the role, but two sessions that race past
-- that check are stopped by the unique index on pg_authid, which raises unique_violation
-- instead. Catching only duplicate_object still failed one session in sixteen; catching
-- both survived sixteen out of sixteen.
DO $$
BEGIN
    CREATE ROLE application_user;
EXCEPTION
    WHEN duplicate_object OR unique_violation THEN NULL;
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
-- The development seed now lives in scripts/seed/dev_seed.sql
