-- 028: narrow the unbounded read policy on the users table.
--
-- SORUN
-- 002_create_users.up.sql defined two permissive policies:
--
--   CREATE POLICY tenant_isolation_policy ON users FOR ALL
--     USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
--
--   CREATE POLICY auth_policy ON users FOR SELECT
--     USING (TRUE);
--
-- PostgreSQL combines permissive policies with OR. Because the second one was a
-- constant TRUE, the combined condition collapsed to TRUE on every SELECT. The
-- result: there was no tenant isolation at all on reads from the users table. Any
-- tenant context could read every tenant's user rows, password_hash included. FORCE
-- ROW LEVEL SECURITY does not fix this; the problem is in the policy logic, not in
-- the ownership exemption.
--
-- auth_policy's intent was legitimate: allow the login lookup (GetByEmailGlobal)
-- that happens before the tenant is known. But it implemented that as a permanent,
-- unconditional opening.
--
-- FIX
-- The widened visibility now requires an explicit session flag. The repository opens
-- that flag only for the global login lookup, with SET LOCAL, confined to a single
-- transaction. The privilege closes by itself when the transaction ends.

DROP POLICY IF EXISTS auth_policy ON users;

CREATE POLICY auth_lookup_policy ON users
    FOR SELECT
    USING (current_setting('app.auth_lookup', TRUE) = 'on');

COMMENT ON POLICY auth_lookup_policy ON users IS
    'Only for the login lookup performed before the tenant is known. The repository opens it per transaction with SET LOCAL app.auth_lookup.';
