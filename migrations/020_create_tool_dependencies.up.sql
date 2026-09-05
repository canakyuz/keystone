-- Migration: Create tool_dependencies table
-- Description: Defines dependencies between tools (e.g., Refund tool requires Payment tool)
-- Author: Keystone Team
-- Date: 2025-10-05

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tool_dependencies table
CREATE TABLE IF NOT EXISTS tool_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Tool relationship
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    depends_on_tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,

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
    auto_install BOOLEAN DEFAULT TRUE, -- Auto-install dependency when parent is installed
    allow_disable BOOLEAN DEFAULT FALSE, -- Can this dependency be disabled independently (usually false for tools)

    -- Metadata
    description TEXT, -- Why this dependency exists
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,

    -- Constraints
    CONSTRAINT tool_deps_type_check CHECK (dependency_type IN ('required', 'optional', 'recommended')),
    CONSTRAINT tool_deps_scope_check CHECK (dependency_scope IN ('runtime', 'installation', 'feature')),
    CONSTRAINT tool_deps_no_self_reference CHECK (tool_id != depends_on_tool_id),
    CONSTRAINT tool_deps_priority_positive CHECK (priority >= 0),
    CONSTRAINT tool_deps_order_positive CHECK (install_order >= 0),
    CONSTRAINT tool_deps_unique_dependency UNIQUE (tool_id, depends_on_tool_id)
);

-- Create indexes for better query performance
CREATE INDEX idx_tool_deps_tool_id ON tool_dependencies(tool_id);
CREATE INDEX idx_tool_deps_depends_on ON tool_dependencies(depends_on_tool_id);
CREATE INDEX idx_tool_deps_type ON tool_dependencies(dependency_type);
CREATE INDEX idx_tool_deps_scope ON tool_dependencies(dependency_scope);
CREATE INDEX idx_tool_deps_priority ON tool_dependencies(priority DESC);
CREATE INDEX idx_tool_deps_install_order ON tool_dependencies(install_order);
CREATE INDEX idx_tool_deps_auto_install ON tool_dependencies(auto_install) WHERE auto_install = TRUE;

-- Composite indexes for common queries
CREATE INDEX idx_tool_deps_tool_type ON tool_dependencies(tool_id, dependency_type);
CREATE INDEX idx_tool_deps_tool_order ON tool_dependencies(tool_id, install_order);

-- Create trigger for updated_at
CREATE TRIGGER tool_dependencies_updated_at
    BEFORE UPDATE ON tool_dependencies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE tool_dependencies IS 'Defines dependencies between tools (e.g., Refund tool requires Payment tool)';
COMMENT ON COLUMN tool_dependencies.tool_id IS 'The tool that has the dependency';
COMMENT ON COLUMN tool_dependencies.depends_on_tool_id IS 'The tool this depends on';
COMMENT ON COLUMN tool_dependencies.dependency_type IS 'Dependency type: required (must have), optional, recommended';
COMMENT ON COLUMN tool_dependencies.dependency_scope IS 'When dependency is needed: runtime, installation, feature';
COMMENT ON COLUMN tool_dependencies.auto_install IS 'Whether to automatically install dependency with parent tool';
COMMENT ON COLUMN tool_dependencies.allow_disable IS 'Can dependency be disabled independently (usually false for tools)';
COMMENT ON COLUMN tool_dependencies.install_order IS 'Installation sequence (lower numbers installed first)';
COMMENT ON COLUMN tool_dependencies.priority IS 'Dependency priority (higher = more important)';
