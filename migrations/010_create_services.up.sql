CREATE TABLE services (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    features TEXT[],
    base_price DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    pricing_model VARCHAR(50) NOT NULL,
    billing_cycle VARCHAR(50),
    duration INTEGER,
    max_clients INTEGER,
    is_public BOOLEAN DEFAULT TRUE,
    featured BOOLEAN DEFAULT FALSE,
    image VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP,
    UNIQUE(tenant_id, slug),
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'archived')),
    CONSTRAINT valid_category CHECK (category IN ('consulting', 'training', 'support', 'other')),
    CONSTRAINT valid_pricing_model CHECK (pricing_model IN ('one-time', 'recurring', 'subscription', 'usage-based')),
    CONSTRAINT valid_price CHECK (base_price >= 0)
);

-- Indexes
CREATE INDEX idx_services_tenant_id ON services(tenant_id);
CREATE INDEX idx_services_slug ON services(slug);
CREATE INDEX idx_services_category ON services(category);
CREATE INDEX idx_services_status ON services(status);
CREATE INDEX idx_services_featured ON services(featured);
CREATE INDEX idx_services_deleted_at ON services(deleted_at);

-- Composite indexes
CREATE INDEX idx_services_tenant_status ON services(tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_services_featured_public ON services(featured, is_public, status)
    WHERE deleted_at IS NULL AND featured = TRUE;

-- Row Level Security
ALTER TABLE services ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON services
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
