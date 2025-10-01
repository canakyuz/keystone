-- Create lessons table
CREATE TABLE IF NOT EXISTS lessons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    subject VARCHAR(100) NOT NULL,
    topic VARCHAR(200),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    type VARCHAR(20) NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration INTEGER NOT NULL,
    location VARCHAR(255),
    meeting_url VARCHAR(500),
    materials TEXT[],
    homework TEXT,
    notes TEXT,
    student_notes TEXT,
    performance_score INTEGER,
    attendance_status VARCHAR(20),
    homework_completed BOOLEAN DEFAULT FALSE,
    next_lesson_topic VARCHAR(200),
    next_lesson_date TIMESTAMP,
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_lessons_tenant_id ON lessons(tenant_id);
CREATE INDEX idx_lessons_student_id ON lessons(student_id);
CREATE INDEX idx_lessons_status ON lessons(status);
CREATE INDEX idx_lessons_scheduled_at ON lessons(scheduled_at);

ALTER TABLE lessons ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON lessons
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
