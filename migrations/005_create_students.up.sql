-- Create students table
CREATE TABLE IF NOT EXISTS students (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    date_of_birth DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    level VARCHAR(20) NOT NULL,
    grade VARCHAR(50),
    school VARCHAR(200),
    parent_name VARCHAR(200),
    parent_email VARCHAR(255),
    parent_phone VARCHAR(50),
    current_gpa DECIMAL(3,2),
    target_gpa DECIMAL(3,2),
    subjects TEXT[],
    goals TEXT,
    notes TEXT,
    enrollment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    last_lesson_date TIMESTAMP,
    total_lessons INTEGER DEFAULT 0,
    completed_lessons INTEGER DEFAULT 0,
    cancelled_lessons INTEGER DEFAULT 0,
    attendance_rate DECIMAL(5,2) DEFAULT 0,
    average_score DECIMAL(5,2) DEFAULT 0,
    avatar VARCHAR(500),
    metadata JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMP,
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_students_tenant_id ON students(tenant_id);
CREATE INDEX idx_students_email ON students(email);
CREATE INDEX idx_students_status ON students(status);
CREATE INDEX idx_students_level ON students(level);

ALTER TABLE students ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON students
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
