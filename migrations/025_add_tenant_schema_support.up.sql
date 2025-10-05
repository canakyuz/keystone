-- Migration 025: Add schema-per-tenant support metadata
-- Adds schema_name column and helper function to generate unique schema names

ALTER TABLE tenants
    ADD COLUMN schema_name VARCHAR(63);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_schema_name
    ON tenants(schema_name)
    WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION generate_schema_name(base TEXT)
RETURNS TEXT AS $$
DECLARE
    slug TEXT;
    candidate TEXT;
    counter INT := 0;
BEGIN
    IF base IS NULL OR LENGTH(TRIM(base)) = 0 THEN
        RAISE EXCEPTION 'base value cannot be empty';
    END IF;

    slug := lower(regexp_replace(base, '[^a-zA-Z0-9]+', '_', 'g'));
    slug := regexp_replace(slug, '_+', '_', 'g');
    slug := trim(both '_' FROM slug);
    IF slug = '' THEN
        slug := 'tenant';
    END IF;

    candidate := 'tenant_' || slug;

    WHILE EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = candidate)
       OR EXISTS (SELECT 1 FROM tenants WHERE schema_name = candidate) LOOP
        counter := counter + 1;
        candidate := 'tenant_' || slug || '_' || counter;
    END LOOP;

    RETURN candidate;
END;
$$ LANGUAGE plpgsql;

-- Backfill existing tenants
UPDATE tenants
SET schema_name = generate_schema_name(COALESCE(slug, id::TEXT))
WHERE schema_name IS NULL;

ALTER TABLE tenants
    ALTER COLUMN schema_name SET NOT NULL;

-- Ensure search_path resets cleanly between requests
CREATE OR REPLACE FUNCTION reset_tenant_search_path()
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('search_path', 'public', true);
END;
$$ LANGUAGE plpgsql;
