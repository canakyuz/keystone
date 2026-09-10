-- ============================================================================
-- Migration Rollback: 026 - CMS Page Builder Tables
-- Description: drop the CMS tables and their related functions
-- ============================================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_pages_updated_at ON pages;
DROP TRIGGER IF EXISTS trigger_sections_updated_at ON sections;
DROP TRIGGER IF EXISTS trigger_components_updated_at ON components;
DROP TRIGGER IF EXISTS trigger_reorder_sections ON sections;
DROP TRIGGER IF EXISTS trigger_reorder_components ON components;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at();
DROP FUNCTION IF EXISTS reorder_sections();
DROP FUNCTION IF EXISTS reorder_components();
DROP FUNCTION IF EXISTS get_translation(JSONB, VARCHAR);

-- Drop tables (CASCADE will remove dependent objects)
DROP TABLE IF EXISTS components CASCADE;
DROP TABLE IF EXISTS sections CASCADE;
DROP TABLE IF EXISTS pages CASCADE;
DROP TABLE IF EXISTS languages CASCADE;

-- ============================================================================
-- Rollback complete.
-- ============================================================================
