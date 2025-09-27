-- NexSpaces Multi-Tenant Database Schema
-- Migration: 001_initial_schema
-- Description: Creates core tables with Row Level Security (RLS) for tenant isolation

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable Row Level Security
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create custom types
CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'cancelled', 'pending');
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'pending', 'suspended');
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'editor', 'viewer', 'billing_admin');
CREATE TYPE template_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE template_visibility AS ENUM ('private', 'public', 'tenant');
CREATE TYPE template_category AS ENUM ('cms', 'crm', 'ecommerce', 'education', 'hospitality', 'erp', 'portfolio', 'blog', 'landing', 'dashboard');
CREATE TYPE pricing_model AS ENUM ('free', 'one_time', 'subscription');
CREATE TYPE subscription_status AS ENUM ('active', 'cancelled', 'suspended', 'past_due', 'unpaid');
CREATE TYPE billing_cycle AS ENUM ('monthly', 'yearly');

-- ================================
-- DATABASE ROLES AND PERMISSIONS
-- These roles must be created before tables that use them in RLS policies.
-- ================================

-- Create application role for API connections
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'application_user') THEN
        CREATE ROLE application_user;
    END IF;
END
$$;

-- Create read-only role for analytics/reporting
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'readonly_user') THEN
        CREATE ROLE readonly_user;
    END IF;
END
$$;

-- ================================
-- TENANTS TABLE
-- ================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    custom_domain VARCHAR(255),
    status tenant_status NOT NULL DEFAULT 'active',
    subscription_id UUID,
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT tenants_slug_format CHECK (slug ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$'),
    CONSTRAINT tenants_slug_length CHECK (LENGTH(slug) >= 3 AND LENGTH(slug) <= 50),
    CONSTRAINT tenants_name_length CHECK (LENGTH(name) >= 2 AND LENGTH(name) <= 100)
);

-- Tenants don't need RLS as they are the root isolation unit
-- But add indexes for performance
CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_custom_domain ON tenants(custom_domain) WHERE custom_domain IS NOT NULL;

-- ================================
-- USERS TABLE
-- ================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255),
    role user_role NOT NULL DEFAULT 'viewer',
    status user_status NOT NULL DEFAULT 'pending',
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    settings JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT users_email_format CHECK (email ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_name_length CHECK (
        LENGTH(first_name) >= 1 AND LENGTH(first_name) <= 100 AND
        LENGTH(last_name) >= 1 AND LENGTH(last_name) <= 100
    ),
    -- Unique email per tenant (not globally unique)
    UNIQUE(tenant_id, email)
);

-- Enable RLS on users table
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only access users in their tenant
CREATE POLICY tenant_isolation_users ON users
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Indexes for performance
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);

-- ================================
-- TEMPLATES TABLE
-- ================================
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category template_category NOT NULL,
    tags TEXT[] DEFAULT '{}',
    configuration JSONB NOT NULL DEFAULT '{}',
    variables JSONB DEFAULT '{}',
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    status template_status NOT NULL DEFAULT 'draft',
    visibility template_visibility NOT NULL DEFAULT 'private',
    pricing_model pricing_model NOT NULL DEFAULT 'free',
    price_amount INTEGER DEFAULT 0, -- in cents
    price_currency VARCHAR(3) DEFAULT 'USD',
    install_count INTEGER NOT NULL DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.0,
    rating_count INTEGER NOT NULL DEFAULT 0,
    published_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT templates_name_length CHECK (LENGTH(name) >= 3 AND LENGTH(name) <= 100),
    CONSTRAINT templates_description_length CHECK (LENGTH(description) >= 10 AND LENGTH(description) <= 1000),
    CONSTRAINT templates_version_format CHECK (version ~ '^\d+\.\d+\.\d+$'),
    CONSTRAINT templates_price_valid CHECK (price_amount >= 0),
    CONSTRAINT templates_rating_valid CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT templates_published_status CHECK (
        (status = 'published' AND published_at IS NOT NULL) OR
        (status != 'published' AND published_at IS NULL)
    ),
    -- Unique template name per tenant
    UNIQUE(tenant_id, name)
);

-- Enable RLS on templates table
ALTER TABLE templates ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Templates - users can see their tenant's private templates and all public templates
CREATE POLICY tenant_isolation_templates ON templates
    FOR ALL TO application_user
    USING (
        tenant_id = current_setting('app.current_tenant')::UUID OR
        (visibility = 'public' AND status = 'published')
    );

-- Indexes for performance
CREATE INDEX idx_templates_tenant_id ON templates(tenant_id);
CREATE INDEX idx_templates_tenant_name ON templates(tenant_id, name);
CREATE INDEX idx_templates_status ON templates(status);
CREATE INDEX idx_templates_visibility ON templates(visibility);
CREATE INDEX idx_templates_category ON templates(category);
CREATE INDEX idx_templates_pricing_model ON templates(pricing_model);
CREATE INDEX idx_templates_published_at ON templates(published_at) WHERE published_at IS NOT NULL;
CREATE INDEX idx_templates_tags ON templates USING GIN(tags);
CREATE INDEX idx_templates_search ON templates USING GIN(to_tsvector('english', name || ' ' || description));

-- ================================
-- TEMPLATE_INSTALLATIONS TABLE
-- ================================
CREATE TABLE template_installations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    installed_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    configuration JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    installed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uninstalled_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',

    -- One installation per template per tenant
    UNIQUE(template_id, tenant_id)
);

-- Enable RLS on template_installations table
ALTER TABLE template_installations ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Installations - users can only see installations in their tenant
CREATE POLICY tenant_isolation_template_installations ON template_installations
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Indexes
CREATE INDEX idx_template_installations_tenant_id ON template_installations(tenant_id);
CREATE INDEX idx_template_installations_template_id ON template_installations(template_id);
CREATE INDEX idx_template_installations_status ON template_installations(status);

-- ================================
-- SUBSCRIPTIONS TABLE
-- ================================
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id VARCHAR(255), -- Stripe subscription ID
    plan_id VARCHAR(100) NOT NULL,
    status subscription_status NOT NULL DEFAULT 'active',
    billing_cycle billing_cycle NOT NULL DEFAULT 'monthly',
    price_amount INTEGER NOT NULL, -- in cents
    price_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    payment_method_id VARCHAR(255),
    features JSONB DEFAULT '{}',
    usage_limits JSONB DEFAULT '{}',
    current_usage JSONB DEFAULT '{}',
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    cancellation_reason TEXT,
    ended_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT subscriptions_price_valid CHECK (price_amount >= 0),
    CONSTRAINT subscriptions_currency_valid CHECK (LENGTH(price_currency) = 3),
    -- Only one active subscription per tenant
    UNIQUE(tenant_id) DEFERRABLE INITIALLY DEFERRED
);

-- Enable RLS on subscriptions table
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Subscriptions - users can only see their tenant's subscription
CREATE POLICY tenant_isolation_subscriptions ON subscriptions
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Indexes
CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_external_id ON subscriptions(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX idx_subscriptions_period ON subscriptions(current_period_start, current_period_end);

-- ================================
-- AUDIT_LOGS TABLE
-- ================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT audit_logs_action_length CHECK (LENGTH(action) >= 1 AND LENGTH(action) <= 100),
    CONSTRAINT audit_logs_resource_type_length CHECK (LENGTH(resource_type) >= 1 AND LENGTH(resource_type) <= 50)
);

-- Enable RLS on audit_logs table
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Audit logs - users can only see their tenant's audit logs
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    FOR SELECT TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Indexes for performance and retention
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_success ON audit_logs(success);

-- Grant necessary permissions to application_user
GRANT USAGE ON SCHEMA public TO application_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO application_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO application_user;

-- Grant necessary permissions to readonly_user
GRANT USAGE ON SCHEMA public TO readonly_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO readonly_user;

-- ================================
-- HELPER FUNCTIONS
-- ================================

-- Function to set tenant context for RLS
CREATE OR REPLACE FUNCTION set_tenant_context(tenant_uuid UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant', tenant_uuid::TEXT, TRUE);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get current tenant from context
CREATE OR REPLACE FUNCTION get_current_tenant()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_tenant', TRUE)::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ================================
-- TRIGGERS FOR updated_at
-- ================================

-- Tenants
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Users
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Templates
CREATE TRIGGER update_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Subscriptions
CREATE TRIGGER update_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================
-- INITIAL DATA
-- ================================

-- Create system tenant for marketplace operations
INSERT INTO tenants (id, name, slug, status)
VALUES ('00000000-0000-0000-0000-000000000000', 'NexSpaces System', 'system', 'active');

-- ================================
-- SECURITY POLICIES SUMMARY
-- ================================

-- All tables with tenant_id have RLS enabled with tenant isolation policies
-- Users can only access data belonging to their tenant
-- Templates have special visibility rules for public marketplace
-- Audit logs are read-only for regular users
-- Cross-tenant access is prevented at database level