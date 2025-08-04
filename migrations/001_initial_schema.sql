-- Initial schema for NexSpaces multi-tenant SaaS platform

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Plans table (shared across all tenants)
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    modules TEXT[], -- Available modules for this plan
    limits JSONB, -- Storage, users, API calls, etc.
    pricing JSONB, -- Pricing information
    created_at TIMESTAMP DEFAULT NOW()
);

-- Tenants table (master tenant registry)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    custom_domain VARCHAR(255) UNIQUE,
    plan_id UUID REFERENCES plans(id),
    status VARCHAR(20) DEFAULT 'trial' CHECK (status IN ('active', 'suspended', 'trial', 'cancelled')),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tenant users table
CREATE TABLE tenant_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('owner', 'admin', 'user')),
    permissions JSONB DEFAULT '{}',
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- Panel configurations per tenant
CREATE TABLE tenant_panels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    panel_type VARCHAR(50) NOT NULL, -- ecommerce, blog, education, etc.
    name VARCHAR(255) NOT NULL,
    config JSONB DEFAULT '{}', -- Panel-specific configuration
    active_modules TEXT[], -- Currently active modules
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- API keys for tenant integrations
CREATE TABLE tenant_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    permissions TEXT[], -- API permissions
    last_used TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by UUID REFERENCES tenant_users(id)
);

-- Indexes for performance
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_custom_domain ON tenants(custom_domain) WHERE custom_domain IS NOT NULL;
CREATE INDEX idx_tenant_users_tenant_id ON tenant_users(tenant_id);
CREATE INDEX idx_tenant_users_email ON tenant_users(tenant_id, email);
CREATE INDEX idx_tenant_panels_tenant_id ON tenant_panels(tenant_id);
CREATE INDEX idx_tenant_api_keys_tenant_id ON tenant_api_keys(tenant_id);
CREATE INDEX idx_tenant_api_keys_hash ON tenant_api_keys(key_hash);

-- Insert default plans
INSERT INTO plans (name, slug, modules, limits, pricing) VALUES 
(
    'Starter', 
    'starter',
    ARRAY['dashboard', 'basic-cms', 'file-manager'],
    '{"users": 5, "storage": "1GB", "api_calls": 1000}',
    '{"monthly": 29, "yearly": 290, "currency": "USD"}'
),
(
    'Professional', 
    'professional',
    ARRAY['dashboard', 'advanced-cms', 'file-manager', 'api-keys', 'chat', 'calendar'],
    '{"users": 25, "storage": "10GB", "api_calls": 10000}',
    '{"monthly": 99, "yearly": 990, "currency": "USD"}'
),
(
    'Enterprise', 
    'enterprise',
    ARRAY['dashboard', 'advanced-cms', 'file-manager', 'api-keys', 'chat', 'calendar', 'mail', 'pos-system', 'kanban'],
    '{"users": -1, "storage": "unlimited", "api_calls": -1}',
    '{"monthly": 299, "yearly": 2990, "currency": "USD"}'
);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_users_updated_at BEFORE UPDATE ON tenant_users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_panels_updated_at BEFORE UPDATE ON tenant_panels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();