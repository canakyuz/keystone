-- Migration: Create module_dependencies table
-- Description: Defines dependencies between modules (e.g., E-commerce depends on Payment)
-- Author: Keystone Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create module_dependencies table
CREATE TABLE IF NOT EXISTS module_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Module relationship
    module_id UUID NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    depends_on_module_id UUID REFERENCES modules(id) ON DELETE CASCADE,
    depends_on_tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,

    -- Dependency type
    dependency_type VARCHAR(50) NOT NULL DEFAULT 'required', -- 'required', 'optional', 'recommended'
    dependency_scope VARCHAR(50) NOT NULL DEFAULT 'runtime', -- 'runtime', 'installation', 'feature'

    -- Versioning constraints
    min_version VARCHAR(20), -- Minimum required version
    max_version VARCHAR(20), -- Maximum compatible version

    -- Priority and ordering
    priority INTEGER DEFAULT 0, -- Higher priority dependencies installed first
    install_order INTEGER DEFAULT 0, -- Installation sequence

    -- Configuration
    auto_install BOOLEAN DEFAULT FALSE, -- Auto-install dependency when parent is installed
    allow_disable BOOLEAN DEFAULT TRUE, -- Can this dependency be disabled independently

    -- Metadata
    description TEXT, -- Why this dependency exists
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT module_deps_type_check CHECK (dependency_type IN ('required', 'optional', 'recommended')),
    CONSTRAINT module_deps_scope_check CHECK (dependency_scope IN ('runtime', 'installation', 'feature')),
    CONSTRAINT module_deps_one_target CHECK (
        (depends_on_module_id IS NOT NULL AND depends_on_tool_id IS NULL) OR
        (depends_on_module_id IS NULL AND depends_on_tool_id IS NOT NULL)
    ), -- Must depend on either a module OR a tool, not both
    CONSTRAINT module_deps_no_self_reference CHECK (module_id != depends_on_module_id),
    CONSTRAINT module_deps_priority_positive CHECK (priority >= 0),
    CONSTRAINT module_deps_order_positive CHECK (install_order >= 0)
);

-- Create indexes for better query performance
CREATE INDEX idx_module_deps_module_id ON module_dependencies(module_id);
CREATE INDEX idx_module_deps_depends_module ON module_dependencies(depends_on_module_id);
CREATE INDEX idx_module_deps_depends_tool ON module_dependencies(depends_on_tool_id);
CREATE INDEX idx_module_deps_type ON module_dependencies(dependency_type);
CREATE INDEX idx_module_deps_scope ON module_dependencies(dependency_scope);
CREATE INDEX idx_module_deps_priority ON module_dependencies(priority DESC);
CREATE INDEX idx_module_deps_install_order ON module_dependencies(install_order);
CREATE INDEX idx_module_deps_auto_install ON module_dependencies(auto_install) WHERE auto_install = TRUE;

-- Composite indexes for common queries
CREATE INDEX idx_module_deps_module_type ON module_dependencies(module_id, dependency_type);
CREATE INDEX idx_module_deps_module_order ON module_dependencies(module_id, install_order);

-- Unique constraint: A module can't have duplicate dependencies
CREATE UNIQUE INDEX idx_module_deps_unique_module_dep ON module_dependencies(
    module_id,
    COALESCE(depends_on_module_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(depends_on_tool_id, '00000000-0000-0000-0000-000000000000'::UUID)
) WHERE depends_on_module_id IS NOT NULL OR depends_on_tool_id IS NOT NULL;

-- Create trigger for updated_at
CREATE TRIGGER module_dependencies_updated_at
    BEFORE UPDATE ON module_dependencies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE module_dependencies IS 'Defines dependencies between modules and tools (e.g., E-commerce module requires Payment tool)';
COMMENT ON COLUMN module_dependencies.module_id IS 'The module that has the dependency';
COMMENT ON COLUMN module_dependencies.depends_on_module_id IS 'The module this depends on (mutually exclusive with depends_on_tool_id)';
COMMENT ON COLUMN module_dependencies.depends_on_tool_id IS 'The tool this depends on (mutually exclusive with depends_on_module_id)';
COMMENT ON COLUMN module_dependencies.dependency_type IS 'Dependency type: required (must have), optional, recommended';
COMMENT ON COLUMN module_dependencies.dependency_scope IS 'When dependency is needed: runtime, installation, feature';
COMMENT ON COLUMN module_dependencies.auto_install IS 'Whether to automatically install dependency with parent module';
COMMENT ON COLUMN module_dependencies.install_order IS 'Installation sequence (lower numbers installed first)';
COMMENT ON COLUMN module_dependencies.priority IS 'Dependency priority (higher = more important)';
