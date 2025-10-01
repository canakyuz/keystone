CREATE TABLE blog_posts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id UUID REFERENCES blog_categories(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL,
    content TEXT,
    excerpt TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    featured BOOLEAN DEFAULT FALSE,
    view_count BIGINT DEFAULT 0,
    image VARCHAR(500),
    tags TEXT[],
    published_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP,
    UNIQUE(tenant_id, slug),
    CONSTRAINT valid_status CHECK (status IN ('draft', 'published', 'archived'))
);

CREATE INDEX idx_blog_posts_tenant ON blog_posts(tenant_id);
CREATE INDEX idx_blog_posts_slug ON blog_posts(slug);
CREATE INDEX idx_blog_posts_category ON blog_posts(category_id);
CREATE INDEX idx_blog_posts_status ON blog_posts(status);
CREATE INDEX idx_blog_posts_featured ON blog_posts(featured);
CREATE INDEX idx_blog_posts_tags ON blog_posts USING GIN(tags);

ALTER TABLE blog_posts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON blog_posts
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
