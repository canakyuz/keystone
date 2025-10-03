-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Payments table (tenant-scoped)
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Transaction details
    provider VARCHAR(50) NOT NULL, -- 'iyzico', 'checkout', 'stripe'
    provider_payment_id VARCHAR(255), -- PSP's transaction ID
    provider_reference VARCHAR(255), -- Additional PSP reference

    -- Amount & currency
    amount DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'TRY', -- ISO 4217

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending, processing, requires_action, succeeded, failed, canceled, refunded
    failure_code VARCHAR(100),
    failure_message TEXT,

    -- Payment method
    payment_method_type VARCHAR(50), -- card, wallet, bank_transfer
    card_brand VARCHAR(50), -- visa, mastercard, amex
    card_last4 VARCHAR(4),
    card_bin VARCHAR(6), -- First 6 digits
    card_exp_month INTEGER,
    card_exp_year INTEGER,

    -- 3DS (Secure Customer Authentication)
    requires_3ds BOOLEAN DEFAULT false,
    threeds_version VARCHAR(10), -- '1.0' or '2.0'
    threeds_html_content TEXT, -- iyzico 3DS redirect HTML
    threeds_callback_url VARCHAR(500),
    threeds_status VARCHAR(50), -- success, failed, abandoned

    -- Installment (Turkey specific)
    installment INTEGER DEFAULT 1,
    installment_rate DECIMAL(5,2) DEFAULT 0.00, -- Commission rate

    -- Customer information
    customer_email VARCHAR(255),
    customer_name VARCHAR(255),
    customer_ip_address INET,
    customer_user_agent TEXT,

    -- Billing address
    billing_address_line1 VARCHAR(255),
    billing_address_line2 VARCHAR(255),
    billing_city VARCHAR(100),
    billing_state VARCHAR(100),
    billing_postal_code VARCHAR(20),
    billing_country VARCHAR(2), -- ISO 3166-1 alpha-2

    -- Metadata & context
    description TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Related resources
    order_id UUID, -- Link to orders table (if exists)
    invoice_id UUID, -- Link to invoices table (if exists)

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    succeeded_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for performance
CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_provider ON payments(provider);
CREATE INDEX idx_payments_provider_payment_id ON payments(provider_payment_id);
CREATE INDEX idx_payments_customer_email ON payments(customer_email);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);
CREATE INDEX idx_payments_metadata ON payments USING GIN (metadata);

-- Composite indexes for common queries
CREATE INDEX idx_payments_tenant_status ON payments(tenant_id, status);
CREATE INDEX idx_payments_tenant_created ON payments(tenant_id, created_at DESC);

-- Row Level Security (RLS)
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

-- Policy: Tenants can only see their own payments
CREATE POLICY tenant_isolation_payments ON payments
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_payments_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER payments_updated_at_trigger
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_payments_updated_at();

-- Comments
COMMENT ON TABLE payments IS 'Multi-tenant payment transactions with PSP integration';
COMMENT ON COLUMN payments.provider IS 'Payment service provider: iyzico, checkout, stripe';
COMMENT ON COLUMN payments.requires_3ds IS '3D Secure authentication required (SCA compliance)';
COMMENT ON COLUMN payments.installment IS 'Number of installments (Turkey market)';
COMMENT ON COLUMN payments.metadata IS 'Additional structured data (JSONB)';
