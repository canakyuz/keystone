-- Migration: 002_add_plan_and_tool_schema
-- Description: Adds tables for plans, tools (modules), and globalization to support the dynamic architecture.

BEGIN;

-- ===============================
-- PLANS TABLE
-- Defines the subscription plans available.
-- ===============================
CREATE TABLE plans (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    monthly_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    yearly_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    features TEXT[] DEFAULT '{}',
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_plans_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===============================
-- TOOLS TABLE
-- Master list of all available tools/modules in the system.
-- ===============================
CREATE TABLE tools (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ===============================
-- PLAN_TOOLS TABLE
-- Maps which tools are included in which plan (Many-to-Many).
-- ===============================
CREATE TABLE plan_tools (
    plan_id INTEGER NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    tool_id INTEGER NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, tool_id)
);

-- ===============================
-- TENANT_TOOL_SETTINGS TABLE
-- Stores tenant-specific configurations for each enabled tool. This is the core of the dynamic module system.
-- ===============================
CREATE TABLE tenant_tool_settings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tool_id INTEGER NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, tool_id)
);

-- Enable RLS
ALTER TABLE tenant_tool_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_tenant_tool_settings ON tenant_tool_settings
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE TRIGGER update_tenant_tool_settings_updated_at
    BEFORE UPDATE ON tenant_tool_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===============================
-- SITES TABLE
-- Manages sites created by tenants from templates.
-- ===============================
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE RESTRICT,
    site_config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Enable RLS
ALTER TABLE sites ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_sites ON sites
    FOR ALL TO application_user
    USING (tenant_id = current_setting('app.current_tenant')::UUID);

CREATE TRIGGER update_sites_updated_at
    BEFORE UPDATE ON sites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===============================
-- GLOBALIZATION TABLES
-- ===============================
CREATE TABLE languages (
    code VARCHAR(5) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE translations (
    id BIGSERIAL PRIMARY KEY,
    lang_code VARCHAR(5) NOT NULL REFERENCES languages(code) ON DELETE CASCADE,
    translation_key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    UNIQUE (lang_code, translation_key)
);

-- Insert some default languages
INSERT INTO languages (code, name) VALUES ('en-US', 'English'), ('tr-TR', 'Türkçe');

COMMIT;
