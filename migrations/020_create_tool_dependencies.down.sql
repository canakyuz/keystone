-- Migration: Rollback tool_dependencies table
-- Description: Drop tool_dependencies table and related objects
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Drop trigger
DROP TRIGGER IF EXISTS tool_dependencies_updated_at ON tool_dependencies;

-- Drop indexes
DROP INDEX IF EXISTS idx_tool_deps_tool_id;
DROP INDEX IF EXISTS idx_tool_deps_depends_on;
DROP INDEX IF EXISTS idx_tool_deps_type;
DROP INDEX IF EXISTS idx_tool_deps_scope;
DROP INDEX IF EXISTS idx_tool_deps_priority;
DROP INDEX IF EXISTS idx_tool_deps_install_order;
DROP INDEX IF EXISTS idx_tool_deps_auto_install;
DROP INDEX IF EXISTS idx_tool_deps_tool_type;
DROP INDEX IF EXISTS idx_tool_deps_tool_order;

-- Drop table
DROP TABLE IF EXISTS tool_dependencies CASCADE;
