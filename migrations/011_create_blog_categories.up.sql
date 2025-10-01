CREATE TABLE blog_categories (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL,
    description TEXT,
    post_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_blog_categories_tenant ON blog_categories(tenant_id);
CREATE INDEX idx_blog_categories_slug ON blog_categories(slug);

ALTER TABLE blog_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON blog_categories
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
