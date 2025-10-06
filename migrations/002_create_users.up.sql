-- Migration: Create users table
-- Description: Multi-tenant user management with RLS
-- Author: NexSpaces Team

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    -- Authentication
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMP WITH TIME ZONE,
    last_login_at TIMESTAMP WITH TIME ZONE,

    -- Profile
    avatar VARCHAR(500),
    phone VARCHAR(20),
    timezone VARCHAR(50) DEFAULT 'UTC',
    locale VARCHAR(10) DEFAULT 'en',

    -- Security
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    password_changed_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    preferences JSONB DEFAULT '{}'::JSONB,
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT users_role_check CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'inactive', 'suspended', 'pending')),
    CONSTRAINT users_email_tenant_unique UNIQUE (email, tenant_id),
    CONSTRAINT users_first_name_length CHECK (LENGTH(first_name) >= 1 AND LENGTH(first_name) <= 100),
    CONSTRAINT users_last_name_length CHECK (LENGTH(last_name) >= 1 AND LENGTH(last_name) <= 100)
);

-- ⚠️ CRITICAL: Multi-tenant indexes (tenant_id first for isolation)
CREATE INDEX idx_users_tenant_id ON users(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant_role ON users(tenant_id, role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_tenant_status ON users(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Create trigger for updated_at
CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ⚠️ CRITICAL: Row Level Security (RLS) for tenant isolation
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only access their own tenant's data
CREATE POLICY tenant_isolation_policy ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- Policy: Allow SELECT without tenant context for authentication
CREATE POLICY auth_policy ON users
    FOR SELECT
    USING (TRUE);

-- Add comments for documentation
COMMENT ON TABLE users IS 'Multi-tenant user management with RLS enforcement';
COMMENT ON COLUMN users.tenant_id IS 'CRITICAL: Tenant isolation - every user belongs to a tenant';
COMMENT ON COLUMN users.role IS 'User role: owner, admin, editor, viewer';
COMMENT ON COLUMN users.status IS 'User status: active, inactive, suspended, pending';
COMMENT ON COLUMN users.email_verified IS 'Email verification status';
COMMENT ON COLUMN users.two_factor_enabled IS '2FA enabled flag';

-- Development kullanıcı seed'leri scripts/seed/dev_seed.sql üzerinden uygulanır
