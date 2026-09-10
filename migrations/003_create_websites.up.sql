-- Migration: Create websites table
-- Description: Multi-tenant website management with RLS
-- Author: Keystone Team

-- Create websites table (renamed from sites for clarity)
CREATE TABLE IF NOT EXISTS websites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'custom',
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    homepage_id UUID,

    -- Content
    title VARCHAR(200) NOT NULL,
    logo VARCHAR(500),
    favicon VARCHAR(500),

    -- Domain & URL
    custom_domain VARCHAR(255),
    custom_domain_verified BOOLEAN NOT NULL DEFAULT FALSE,
    primary_url VARCHAR(255) NOT NULL,

    -- Template
    template_id UUID,
    template_name VARCHAR(100),
    template_version VARCHAR(20),

    -- Publishing
    published_at TIMESTAMP WITH TIME ZONE,
    unpublished_at TIMESTAMP WITH TIME ZONE,
    archived_at TIMESTAMP WITH TIME ZONE,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),

    -- Constraints
    CONSTRAINT websites_type_check CHECK (type IN ('cms', 'crm', 'ecommerce', 'education', 'hospitality', 'erp', 'blog', 'portfolio', 'custom')),
    CONSTRAINT websites_status_check CHECK (status IN ('draft', 'published', 'archived', 'unpublished')),
    CONSTRAINT websites_slug_tenant_unique UNIQUE (tenant_id, slug),
    CONSTRAINT websites_name_length CHECK (LENGTH(name) >= 2 AND LENGTH(name) <= 100),
    CONSTRAINT websites_slug_length CHECK (LENGTH(slug) >= 2 AND LENGTH(slug) <= 50),
    CONSTRAINT websites_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$')
);

-- CRITICAL: multi-tenant indexes, tenant_id first for isolation.
CREATE INDEX idx_websites_tenant_id ON websites(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_tenant_slug ON websites(tenant_id, slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_tenant_status ON websites(tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_tenant_type ON websites(tenant_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_slug ON websites(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_status ON websites(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_websites_custom_domain ON websites(custom_domain) WHERE custom_domain IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_websites_created_at ON websites(created_at DESC);
CREATE INDEX idx_websites_published_at ON websites(published_at DESC) WHERE published_at IS NOT NULL;

-- Create trigger for updated_at
CREATE TRIGGER websites_updated_at
    BEFORE UPDATE ON websites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- CRITICAL: Row Level Security for tenant isolation.
ALTER TABLE websites ENABLE ROW LEVEL SECURITY;

-- Policy: Websites can only be accessed within their tenant
CREATE POLICY tenant_isolation_policy ON websites
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- Policy: Allow public SELECT for published websites (for public website access)
CREATE POLICY public_websites_policy ON websites
    FOR SELECT
    USING (status = 'published' AND deleted_at IS NULL);

-- Add comments for documentation
COMMENT ON TABLE websites IS 'Multi-tenant website management with RLS enforcement';
COMMENT ON COLUMN websites.tenant_id IS 'CRITICAL: Tenant isolation - every website belongs to a tenant';
COMMENT ON COLUMN websites.type IS 'Website type: cms, crm, ecommerce, education, hospitality, erp, blog, portfolio, custom';
COMMENT ON COLUMN websites.status IS 'Website status: draft, published, archived, unpublished';
COMMENT ON COLUMN websites.custom_domain IS 'Custom domain for the website';
COMMENT ON COLUMN websites.primary_url IS 'Primary URL for accessing the website';
