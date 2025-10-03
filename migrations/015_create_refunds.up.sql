-- Refunds table (tenant-scoped)
CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,

    -- Provider details
    provider VARCHAR(50) NOT NULL, -- 'iyzico', 'checkout', 'stripe'
    provider_refund_id VARCHAR(255), -- PSP's refund ID

    -- Amount
    amount DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'TRY',

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending, processing, succeeded, failed, canceled
    failure_code VARCHAR(100),
    failure_message TEXT,

    -- Reason
    reason VARCHAR(255), -- customer_request, duplicate, fraudulent, etc.
    description TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    succeeded_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX idx_refunds_tenant_id ON refunds(tenant_id);
CREATE INDEX idx_refunds_payment_id ON refunds(payment_id);
CREATE INDEX idx_refunds_status ON refunds(status);
CREATE INDEX idx_refunds_provider_refund_id ON refunds(provider_refund_id);
CREATE INDEX idx_refunds_created_at ON refunds(created_at DESC);

-- Composite indexes
CREATE INDEX idx_refunds_tenant_status ON refunds(tenant_id, status);

-- Row Level Security (RLS)
ALTER TABLE refunds ENABLE ROW LEVEL SECURITY;

-- Policy: Tenants can only see their own refunds
CREATE POLICY tenant_isolation_refunds ON refunds
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_refunds_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER refunds_updated_at_trigger
    BEFORE UPDATE ON refunds
    FOR EACH ROW
    EXECUTE FUNCTION update_refunds_updated_at();

-- Comments
COMMENT ON TABLE refunds IS 'Payment refunds (partial or full)';
COMMENT ON COLUMN refunds.reason IS 'Refund reason code';
