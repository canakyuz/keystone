CREATE TABLE availabilities (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'available',
    recurrence VARCHAR(50) NOT NULL DEFAULT 'none',
    recurrence_end TIMESTAMP,
    days_of_week INTEGER[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    CONSTRAINT valid_status CHECK (status IN ('available', 'booked', 'unavailable')),
    CONSTRAINT valid_recurrence CHECK (recurrence IN ('none', 'daily', 'weekly', 'monthly'))
);

-- Indexes for performance
CREATE INDEX idx_availabilities_tenant_id ON availabilities(tenant_id);
CREATE INDEX idx_availabilities_user_id ON availabilities(user_id);
CREATE INDEX idx_availabilities_start_time ON availabilities(start_time);
CREATE INDEX idx_availabilities_status ON availabilities(status);
CREATE INDEX idx_availabilities_deleted_at ON availabilities(deleted_at);

-- Composite index for common queries
CREATE INDEX idx_availabilities_tenant_user_time ON availabilities(tenant_id, user_id, start_time, end_time)
    WHERE deleted_at IS NULL;

-- Row Level Security
ALTER TABLE availabilities ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON availabilities
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
