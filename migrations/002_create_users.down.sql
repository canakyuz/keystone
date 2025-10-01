-- Rollback: Drop users table

-- Drop RLS policies
DROP POLICY IF EXISTS tenant_isolation_policy ON users;
DROP POLICY IF EXISTS auth_policy ON users;

-- Disable RLS
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_tenant_id;
DROP INDEX IF EXISTS idx_users_tenant_email;
DROP INDEX IF EXISTS idx_users_tenant_role;
DROP INDEX IF EXISTS idx_users_tenant_status;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_created_at;

-- Drop trigger
DROP TRIGGER IF EXISTS users_updated_at ON users;

-- Drop table
DROP TABLE IF EXISTS users;
