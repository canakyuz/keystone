-- Migration: Create modules table (Registry System)
-- Description: Module catalog - defines available modules in the platform
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create modules table (catalog of all available modules)
CREATE TABLE IF NOT EXISTS modules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Module identification
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE, -- 'LMS', 'CMS', 'BOOKING', etc.
    display_name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,

    -- Module type and category
    category VARCHAR(50) NOT NULL, -- 'education', 'content', 'commerce', 'hospitality', 'management'
    module_type VARCHAR(50) NOT NULL DEFAULT 'standard', -- 'standard', 'premium', 'enterprise'

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_public BOOLEAN NOT NULL DEFAULT TRUE, -- Visible in marketplace
    is_beta BOOLEAN NOT NULL DEFAULT FALSE,

    -- Versioning
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    min_platform_version VARCHAR(20), -- Minimum platform version required

    -- Pricing
    pricing_model VARCHAR(50) NOT NULL DEFAULT 'free', -- 'free', 'one_time', 'subscription', 'usage_based'
    base_price DECIMAL(10,2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'USD',
    billing_cycle VARCHAR(20), -- 'monthly', 'yearly', 'one_time', null for free

    -- Features and capabilities
    features JSONB DEFAULT '[]'::JSONB, -- Array of feature descriptions
    capabilities JSONB DEFAULT '{}'::JSONB, -- Key capabilities and limits

    -- UI/UX
    icon VARCHAR(255), -- Icon URL or icon name
    cover_image VARCHAR(500), -- Cover image URL
    screenshots TEXT[], -- Array of screenshot URLs
    demo_url VARCHAR(500), -- Demo/preview URL
    documentation_url VARCHAR(500), -- Documentation URL

    -- Requirements and dependencies
    requires_database BOOLEAN DEFAULT TRUE,
    requires_storage BOOLEAN DEFAULT FALSE,
    requires_email BOOLEAN DEFAULT FALSE,
    database_tables TEXT[], -- Array of table names this module creates

    -- Limits and quotas (default values, can be overridden per tenant)
    default_limits JSONB DEFAULT '{}'::JSONB, -- e.g., {"max_lessons": 100, "max_students": 50}

    -- Installation
    installation_notes TEXT,
    configuration_schema JSONB, -- JSON Schema for module configuration

    -- Metadata
    tags TEXT[], -- Searchable tags
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Statistics
    install_count INTEGER DEFAULT 0, -- Number of tenants using this module
    rating DECIMAL(3,2) DEFAULT 0.00, -- Average rating (0.00 - 5.00)
    review_count INTEGER DEFAULT 0,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT modules_status_check CHECK (status IN ('active', 'inactive', 'deprecated', 'archived')),
    CONSTRAINT modules_type_check CHECK (module_type IN ('standard', 'premium', 'enterprise')),
    CONSTRAINT modules_pricing_check CHECK (pricing_model IN ('free', 'one_time', 'subscription', 'usage_based')),
    CONSTRAINT modules_category_check CHECK (category IN ('education', 'content', 'commerce', 'hospitality', 'management', 'communication', 'analytics', 'other')),
    CONSTRAINT modules_name_length CHECK (LENGTH(name) >= 2 AND LENGTH(name) <= 100),
    CONSTRAINT modules_slug_length CHECK (LENGTH(slug) >= 2 AND LENGTH(slug) <= 50),
    CONSTRAINT modules_code_length CHECK (LENGTH(code) >= 2 AND LENGTH(code) <= 20),
    CONSTRAINT modules_slug_format CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT modules_code_format CHECK (code ~ '^[A-Z0-9_]+$'),
    CONSTRAINT modules_rating_range CHECK (rating >= 0.00 AND rating <= 5.00),
    CONSTRAINT modules_price_positive CHECK (base_price >= 0)
);

-- Create indexes for better query performance
CREATE INDEX idx_modules_status ON modules(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_category ON modules(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_type ON modules(module_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_pricing ON modules(pricing_model) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_public ON modules(is_public) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_slug ON modules(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_code ON modules(code) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_rating ON modules(rating DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_install_count ON modules(install_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_tags ON modules USING GIN (tags);
CREATE INDEX idx_modules_features ON modules USING GIN (features);
CREATE INDEX idx_modules_metadata ON modules USING GIN (metadata);

-- Composite indexes for common queries
CREATE INDEX idx_modules_status_public ON modules(status, is_public) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_category_status ON modules(category, status) WHERE deleted_at IS NULL;

-- Create trigger for updated_at
CREATE TRIGGER modules_updated_at
    BEFORE UPDATE ON modules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE modules IS 'Module catalog - defines all available modules in the platform marketplace';
COMMENT ON COLUMN modules.id IS 'Unique module identifier (UUID)';
COMMENT ON COLUMN modules.code IS 'Module code identifier (e.g., LMS, CMS, BOOKING)';
COMMENT ON COLUMN modules.slug IS 'URL-friendly module identifier';
COMMENT ON COLUMN modules.category IS 'Module category for marketplace organization';
COMMENT ON COLUMN modules.module_type IS 'Module tier: standard, premium, enterprise';
COMMENT ON COLUMN modules.pricing_model IS 'How this module is priced';
COMMENT ON COLUMN modules.features IS 'Array of feature descriptions (JSONB)';
COMMENT ON COLUMN modules.capabilities IS 'Module capabilities and limits (JSONB)';
COMMENT ON COLUMN modules.default_limits IS 'Default usage limits for tenants (JSONB)';
COMMENT ON COLUMN modules.database_tables IS 'List of database tables this module creates';
COMMENT ON COLUMN modules.install_count IS 'Number of active tenant installations';
COMMENT ON COLUMN modules.rating IS 'Average user rating (0.00 - 5.00)';
