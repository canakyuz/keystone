-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Basic info
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    content TEXT,

    -- Classification
    category VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'planned',
    client VARCHAR(200),

    -- Technical details
    technologies TEXT[] DEFAULT '{}',
    images TEXT[] DEFAULT '{}',
    cover_image VARCHAR(500),
    live_url VARCHAR(500),
    github_url VARCHAR(500),

    -- Timeline
    start_date TIMESTAMP,
    end_date TIMESTAMP,

    -- Display settings
    featured BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    view_count BIGINT DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP,

    -- Constraints
    UNIQUE(tenant_id, slug)
);

-- Indexes for performance
CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX idx_projects_slug ON projects(slug);
CREATE INDEX idx_projects_tenant_slug ON projects(tenant_id, slug);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_category ON projects(category);
CREATE INDEX idx_projects_featured ON projects(featured) WHERE featured = TRUE;
CREATE INDEX idx_projects_created_at ON projects(created_at DESC);
CREATE INDEX idx_projects_view_count ON projects(view_count DESC);

-- Full-text search index
CREATE INDEX idx_projects_search ON projects USING gin(to_tsvector('english',
    coalesce(title, '') || ' ' ||
    coalesce(description, '') || ' ' ||
    coalesce(content, '') || ' ' ||
    coalesce(client, '')
));

-- Enable Row Level Security
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Tenant isolation
CREATE POLICY tenant_isolation_policy ON projects
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

-- RLS Policy: Public read for completed featured projects
CREATE POLICY public_projects_policy ON projects
    FOR SELECT
    USING (status = 'completed' AND featured = TRUE AND deleted_at IS NULL);

-- Comments
COMMENT ON TABLE projects IS 'Portfolio projects with multi-tenant isolation';
COMMENT ON COLUMN projects.tenant_id IS 'Owner tenant ID - enforces data isolation';
COMMENT ON COLUMN projects.technologies IS 'Array of technologies used (e.g., ["React", "Go", "PostgreSQL"])';
COMMENT ON COLUMN projects.featured IS 'Whether project is featured on portfolio homepage';
COMMENT ON COLUMN projects.view_count IS 'Number of times project has been viewed';
