-- Rollback: Drop websites table

-- Drop RLS policies
DROP POLICY IF EXISTS tenant_isolation_policy ON websites;
DROP POLICY IF EXISTS public_websites_policy ON websites;

-- Disable RLS
ALTER TABLE websites DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_websites_tenant_id;
DROP INDEX IF EXISTS idx_websites_tenant_slug;
DROP INDEX IF EXISTS idx_websites_tenant_status;
DROP INDEX IF EXISTS idx_websites_tenant_type;
DROP INDEX IF EXISTS idx_websites_slug;
DROP INDEX IF EXISTS idx_websites_status;
DROP INDEX IF EXISTS idx_websites_custom_domain;
DROP INDEX IF EXISTS idx_websites_created_at;
DROP INDEX IF EXISTS idx_websites_published_at;

-- Drop trigger
DROP TRIGGER IF EXISTS websites_updated_at ON websites;

-- Drop table
DROP TABLE IF EXISTS websites;
