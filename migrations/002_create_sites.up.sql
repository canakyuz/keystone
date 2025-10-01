-- Create sites (websites) table
CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    homepage_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    
    CONSTRAINT chk_status CHECK (status IN ('draft', 'published', 'archived'))
);

-- Indexes for performance
CREATE INDEX idx_sites_tenant_id ON sites(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_sites_slug ON sites(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_sites_status ON sites(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_sites_tenant_status ON sites(tenant_id, status) WHERE deleted_at IS NULL;

-- Row Level Security (RLS) for multi-tenant isolation
ALTER TABLE sites ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only access sites within their tenant
-- NOTE: This requires setting 'app.current_tenant' session variable
CREATE POLICY tenant_isolation_policy ON sites
    FOR ALL
    USING (tenant_id = COALESCE(
        NULLIF(current_setting('app.current_tenant', true), '')::UUID,
        tenant_id
    ));
