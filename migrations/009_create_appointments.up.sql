CREATE TABLE appointments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    client_name VARCHAR(200) NOT NULL,
    client_email VARCHAR(255) NOT NULL,
    client_phone VARCHAR(50),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    duration INTEGER NOT NULL,
    location VARCHAR(500),
    meeting_url VARCHAR(500),
    notes TEXT,
    cancellation_reason TEXT,
    cancelled_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    completed_at TIMESTAMP,
    reminder_sent BOOLEAN DEFAULT FALSE,
    reminder_sent_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed', 'no-show')),
    CONSTRAINT valid_type CHECK (type IN ('consultation', 'meeting', 'lesson', 'interview', 'other')),
    CONSTRAINT valid_duration CHECK (duration > 0)
);

-- Indexes for performance
CREATE INDEX idx_appointments_tenant_id ON appointments(tenant_id);
CREATE INDEX idx_appointments_user_id ON appointments(user_id);
CREATE INDEX idx_appointments_client_email ON appointments(client_email);
CREATE INDEX idx_appointments_start_time ON appointments(start_time);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_appointments_type ON appointments(type);
CREATE INDEX idx_appointments_deleted_at ON appointments(deleted_at);

-- Composite indexes for common queries
CREATE INDEX idx_appointments_tenant_user_time ON appointments(tenant_id, user_id, start_time, end_time)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_appointments_upcoming ON appointments(tenant_id, start_time, status)
    WHERE deleted_at IS NULL AND status IN ('pending', 'confirmed');

-- Row Level Security
ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON appointments
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
