-- Payment webhook events table (audit trail + idempotency)
CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Provider details
    provider VARCHAR(50) NOT NULL, -- 'iyzico', 'checkout', 'stripe'
    provider_event_id VARCHAR(255) NOT NULL, -- PSP's event ID (for idempotency)

    -- Event data
    event_type VARCHAR(100) NOT NULL, -- payment.succeeded, payment.failed, refund.succeeded
    event_version VARCHAR(20), -- API version
    payload JSONB NOT NULL, -- Full webhook payload

    -- Related resources
    payment_id UUID REFERENCES payments(id) ON DELETE SET NULL,
    refund_id UUID REFERENCES refunds(id) ON DELETE SET NULL,

    -- Processing status
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_error TEXT,
    retry_count INTEGER DEFAULT 0,

    -- Request metadata
    request_ip_address INET,
    request_headers JSONB,
    signature_valid BOOLEAN DEFAULT false,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_payment_events_tenant_id ON payment_events(tenant_id);
CREATE INDEX idx_payment_events_provider ON payment_events(provider);
CREATE INDEX idx_payment_events_provider_event_id ON payment_events(provider_event_id);
CREATE INDEX idx_payment_events_event_type ON payment_events(event_type);
CREATE INDEX idx_payment_events_payment_id ON payment_events(payment_id);
CREATE INDEX idx_payment_events_processed ON payment_events(processed);
CREATE INDEX idx_payment_events_created_at ON payment_events(created_at DESC);
CREATE INDEX idx_payment_events_payload ON payment_events USING GIN (payload);

-- Unique constraint for idempotency (prevent duplicate event processing)
CREATE UNIQUE INDEX idx_payment_events_idempotency
    ON payment_events(provider, provider_event_id);

-- Composite indexes
CREATE INDEX idx_payment_events_tenant_type ON payment_events(tenant_id, event_type);

-- Row Level Security (RLS)
ALTER TABLE payment_events ENABLE ROW LEVEL SECURITY;

-- Policy: Tenants can only see their own events
CREATE POLICY tenant_isolation_payment_events ON payment_events
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_payment_events_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER payment_events_updated_at_trigger
    BEFORE UPDATE ON payment_events
    FOR EACH ROW
    EXECUTE FUNCTION update_payment_events_updated_at();

-- Comments
COMMENT ON TABLE payment_events IS 'Webhook events from payment providers (audit + idempotency)';
COMMENT ON COLUMN payment_events.provider_event_id IS 'PSP event ID for idempotency check';
COMMENT ON COLUMN payment_events.processed IS 'Event processing status';
COMMENT ON COLUMN payment_events.signature_valid IS 'Webhook signature verification result';
