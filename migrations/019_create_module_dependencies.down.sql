-- Migration: Rollback module_dependencies table
-- Description: Drop module_dependencies table and related objects
-- Author: Keystone Team
-- Date: 2025-10-05

-- Drop trigger
DROP TRIGGER IF EXISTS module_dependencies_updated_at ON module_dependencies;

-- Drop indexes
DROP INDEX IF EXISTS idx_module_deps_module_id;
DROP INDEX IF EXISTS idx_module_deps_depends_module;
DROP INDEX IF EXISTS idx_module_deps_depends_tool;
DROP INDEX IF EXISTS idx_module_deps_type;
DROP INDEX IF EXISTS idx_module_deps_scope;
DROP INDEX IF EXISTS idx_module_deps_priority;
DROP INDEX IF EXISTS idx_module_deps_install_order;
DROP INDEX IF EXISTS idx_module_deps_auto_install;
DROP INDEX IF EXISTS idx_module_deps_module_type;
DROP INDEX IF EXISTS idx_module_deps_module_order;
DROP INDEX IF EXISTS idx_module_deps_unique_module_dep;

-- Drop table
DROP TABLE IF EXISTS module_dependencies CASCADE;
