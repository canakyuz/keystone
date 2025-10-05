-- Migration: Rollback modules table
-- Description: Drop modules table and related objects
-- Author: NexSpaces Team
-- Date: 2025-10-05

-- Drop trigger
DROP TRIGGER IF EXISTS modules_updated_at ON modules;

-- Drop indexes
DROP INDEX IF EXISTS idx_modules_status;
DROP INDEX IF EXISTS idx_modules_category;
DROP INDEX IF EXISTS idx_modules_type;
DROP INDEX IF EXISTS idx_modules_pricing;
DROP INDEX IF EXISTS idx_modules_public;
DROP INDEX IF EXISTS idx_modules_slug;
DROP INDEX IF EXISTS idx_modules_code;
DROP INDEX IF EXISTS idx_modules_rating;
DROP INDEX IF EXISTS idx_modules_install_count;
DROP INDEX IF EXISTS idx_modules_tags;
DROP INDEX IF EXISTS idx_modules_features;
DROP INDEX IF EXISTS idx_modules_metadata;
DROP INDEX IF EXISTS idx_modules_status_public;
DROP INDEX IF EXISTS idx_modules_category_status;

-- Drop table
DROP TABLE IF EXISTS modules CASCADE;
