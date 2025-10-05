-- Revert Migration 025: Remove schema-per-tenant metadata

DROP FUNCTION IF EXISTS reset_tenant_search_path();

ALTER TABLE tenants
    ALTER COLUMN schema_name DROP NOT NULL;

UPDATE tenants SET schema_name = NULL;

DROP FUNCTION IF EXISTS generate_schema_name(TEXT);

DROP INDEX IF EXISTS idx_tenants_schema_name;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS schema_name;
