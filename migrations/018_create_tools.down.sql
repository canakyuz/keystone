-- Migration: Rollback tools table
-- Description: Drop tools table and related objects
-- Author: Keystone Team
-- Date: 2025-10-05

-- Drop trigger
DROP TRIGGER IF EXISTS tools_updated_at ON tools;

-- Drop indexes
DROP INDEX IF EXISTS idx_tools_status;
DROP INDEX IF EXISTS idx_tools_category;
DROP INDEX IF EXISTS idx_tools_type;
DROP INDEX IF EXISTS idx_tools_scope;
DROP INDEX IF EXISTS idx_tools_pricing;
DROP INDEX IF EXISTS idx_tools_public;
DROP INDEX IF EXISTS idx_tools_slug;
DROP INDEX IF EXISTS idx_tools_code;
DROP INDEX IF EXISTS idx_tools_rating;
DROP INDEX IF EXISTS idx_tools_install_count;
DROP INDEX IF EXISTS idx_tools_integration_provider;
DROP INDEX IF EXISTS idx_tools_tags;
DROP INDEX IF EXISTS idx_tools_features;
DROP INDEX IF EXISTS idx_tools_metadata;
DROP INDEX IF EXISTS idx_tools_status_public;
DROP INDEX IF EXISTS idx_tools_category_status;
DROP INDEX IF EXISTS idx_tools_type_scope;

-- Drop table
DROP TABLE IF EXISTS tools CASCADE;
