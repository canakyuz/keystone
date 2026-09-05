-- Migration: Create tools table (Registry System)
-- Description: Tool catalog - defines available tools in the platform
-- Author: Keystone Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tools table (catalog of all available tools)
CREATE TABLE IF NOT EXISTS tools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tool identification
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE, -- 'PAYMENT', 'CHAT', 'N8N', 'UPLOAD', etc.
    display_name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,

    -- Tool type and category
    category VARCHAR(50) NOT NULL, -- 'payment', 'communication', 'automation', 'storage', 'analytics'
    tool_type VARCHAR(50) NOT NULL DEFAULT 'global', -- 'global', 'module_specific'
    scope VARCHAR(50) NOT NULL DEFAULT 'cross_module', -- 'cross_module', 'module_bound'

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_public BOOLEAN NOT NULL DEFAULT TRUE, -- Visible in marketplace
    is_beta BOOLEAN NOT NULL DEFAULT FALSE,

    -- Versioning
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    min_platform_version VARCHAR(20), -- Minimum platform version required

    -- Pricing
    pricing_model VARCHAR(50) NOT NULL DEFAULT 'free', -- 'free', 'one_time', 'subscription', 'usage_based', 'transaction_fee'
    base_price DECIMAL(10,2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'USD',
    billing_cycle VARCHAR(20), -- 'monthly', 'yearly', 'per_transaction', null for free
    transaction_fee_percentage DECIMAL(5,2), -- For transaction-based pricing (e.g., 2.5%)
    transaction_fee_fixed DECIMAL(10,2), -- Fixed fee per transaction

    -- Features and capabilities
    features JSONB DEFAULT '[]'::JSONB, -- Array of feature descriptions
    capabilities JSONB DEFAULT '{}'::JSONB, -- Key capabilities and settings

    -- UI/UX
    icon VARCHAR(255), -- Icon URL or icon name
    cover_image VARCHAR(500), -- Cover image URL
    screenshots TEXT[], -- Array of screenshot URLs
    demo_url VARCHAR(500), -- Demo/preview URL
    documentation_url VARCHAR(500), -- Documentation URL

    -- Technical requirements
    requires_api_keys BOOLEAN DEFAULT FALSE, -- Requires external API keys
    requires_webhook BOOLEAN DEFAULT FALSE, -- Needs webhook configuration
    requires_storage BOOLEAN DEFAULT FALSE,
    requires_database BOOLEAN DEFAULT FALSE,
    database_tables TEXT[], -- Array of table names this tool creates

    -- External integrations
    integration_provider VARCHAR(100), -- e.g., 'Stripe', 'Iyzico', 'N8n', 'AWS S3'
    integration_type VARCHAR(50), -- 'native', 'third_party', 'custom'
    api_endpoints JSONB, -- API endpoints this tool exposes

    -- Limits and quotas (default values, can be overridden per tenant)
    default_limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"max_messages": 1000, "max_storage_gb": 10}
    rate_limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"requests_per_minute": 60}

    -- Configuration
    configuration_schema JSONB, -- JSON Schema for tool configuration
    default_configuration JSONB DEFAULT '{}'::JSONB, -- Default config values

    -- Metadata
    tags TEXT[], -- Searchable tags
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Statistics
    install_count INTEGER DEFAULT 0, -- Number of tenants using this tool
    rating DECIMAL(3,2) DEFAULT 0.00, -- Average rating (0.00 - 5.00)
    review_count INTEGER DEFAULT 0,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT tools_status_check CHECK (status IN ('active', 'inactive', 'deprecated', 'archived')),
    CONSTRAINT tools_type_check CHECK (tool_type IN ('global', 'module_specific')),
    CONSTRAINT tools_scope_check CHECK (scope IN ('cross_module', 'module_bound')),
    CONSTRAINT tools_pricing_check CHECK (pricing_model IN ('free', 'one_time', 'subscription', 'usage_based', 'transaction_fee')),
    CONSTRAINT tools_category_check CHECK (category IN ('payment', 'communication', 'automation', 'storage', 'analytics', 'notification', 'integration', 'security', 'other')),
    CONSTRAINT tools_integration_type_check CHECK (integration_type IN ('native', 'third_party', 'custom', 'none')),
    CONSTRAINT tools_name_length CHECK (LENGTH(name) >= 2 AND LENGTH(name) <= 100),
    CONSTRAINT tools_slug_length CHECK (LENGTH(slug) >= 2 AND LENGTH(slug) <= 50),
    CONSTRAINT tools_code_length CHECK (LENGTH(code) >= 2 AND LENGTH(code) <= 20),
    CONSTRAINT tools_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT tools_code_format CHECK (code ~ '^[A-Z0-9_]+$'),
    CONSTRAINT tools_rating_range CHECK (rating >= 0.00 AND rating <= 5.00),
    CONSTRAINT tools_price_positive CHECK (base_price >= 0),
    CONSTRAINT tools_transaction_fee_range CHECK (transaction_fee_percentage IS NULL OR (transaction_fee_percentage >= 0 AND transaction_fee_percentage <= 100))
);

-- Create indexes for better query performance
CREATE INDEX idx_tools_status ON tools(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_category ON tools(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_type ON tools(tool_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_scope ON tools(scope) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_pricing ON tools(pricing_model) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_public ON tools(is_public) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_slug ON tools(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_code ON tools(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_rating ON tools(rating DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_install_count ON tools(install_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_integration_provider ON tools(integration_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_tags ON tools USING GIN (tags);
CREATE INDEX idx_tools_features ON tools USING GIN (features);
CREATE INDEX idx_tools_metadata ON tools USING GIN (metadata);

-- Composite indexes for common queries
CREATE INDEX idx_tools_status_public ON tools(status, is_public) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_category_status ON tools(category, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tools_type_scope ON tools(tool_type, scope) WHERE deleted_at IS NULL;

-- Create trigger for updated_at
CREATE TRIGGER tools_updated_at
    BEFORE UPDATE ON tools
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE tools IS 'Tool catalog - defines all available tools (global and module-specific) in the platform';
COMMENT ON COLUMN tools.id IS 'Unique tool identifier (UUID)';
COMMENT ON COLUMN tools.code IS 'Tool code identifier (e.g., PAYMENT, CHAT, N8N)';
COMMENT ON COLUMN tools.slug IS 'URL-friendly tool identifier';
COMMENT ON COLUMN tools.category IS 'Tool category for marketplace organization';
COMMENT ON COLUMN tools.tool_type IS 'Tool type: global (cross-module) or module_specific';
COMMENT ON COLUMN tools.scope IS 'Tool scope: cross_module or module_bound';
COMMENT ON COLUMN tools.pricing_model IS 'How this tool is priced (includes transaction_fee for payment tools)';
COMMENT ON COLUMN tools.transaction_fee_percentage IS 'Transaction fee percentage (for payment tools)';
COMMENT ON COLUMN tools.integration_provider IS 'External provider (e.g., Stripe, Iyzico)';
COMMENT ON COLUMN tools.integration_type IS 'Type of integration: native, third_party, custom';
COMMENT ON COLUMN tools.requires_api_keys IS 'Whether tool needs external API keys';
COMMENT ON COLUMN tools.requires_webhook IS 'Whether tool needs webhook configuration';
COMMENT ON COLUMN tools.default_limits IS 'Default usage limits for tenants (JSONB)';
COMMENT ON COLUMN tools.rate_limits IS 'API rate limits (JSONB)';
COMMENT ON COLUMN tools.install_count IS 'Number of active tenant installations';
COMMENT ON COLUMN tools.rating IS 'Average user rating (0.00 - 5.00)';
