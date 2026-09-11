-- 039: a database role for the worker, holding only what the worker does.
--
-- PROBLEM
-- The worker claims jobs from provisioning_jobs and events from outbox_events. Both carry
-- the fail-closed tenant policy, and the worker has no tenant: it serves all of them.
-- Under the non-superuser role SECURITY.md requires, its claim matched no row, so it
-- found no work while jobs waited. It ran only as a superuser, which put a credential
-- that ignores every policy into the process that also delivers webhooks to addresses
-- tenants supply. The API and the worker also shared one role, so the privilege boundary
-- ADR-0001 relied on did not exist.
--
-- FIX
-- keystone_worker is a group role. A deployment creates a login role and grants this one
-- to it. It holds:
--
--   tenants                   SELECT, UPDATE           status and schema name
--   operations                SELECT, UPDATE           closing an operation
--   provisioning_jobs         SELECT, UPDATE           claim, lease, result
--   outbox_events             SELECT, INSERT, UPDATE   append, claim, result
--   tenant_webhook_endpoints  SELECT                   where to deliver
--   audit_log                 INSERT                   the trail, and nothing else
--   the database              CREATE                   CREATE SCHEMA for a new tenant
--
-- and nothing on users, payments or any other table.
--
-- RLS is crossed with policies naming this role on exactly the tables above, not with
-- BYPASSRLS. BYPASSRLS applies to every table the role is ever granted; a policy applies
-- to one. If somebody later grants this role SELECT on users, the fail-closed tenant
-- policy still answers with nothing.

DO $$
BEGIN
    CREATE ROLE keystone_worker NOLOGIN;
EXCEPTION
    -- Several databases in one cluster can run this at the same moment; see migration 001.
    WHEN duplicate_object OR unique_violation THEN NULL;
END
$$;

GRANT USAGE ON SCHEMA public TO keystone_worker;

GRANT SELECT, UPDATE         ON tenants                  TO keystone_worker;
GRANT SELECT, UPDATE         ON operations               TO keystone_worker;
GRANT SELECT, UPDATE         ON provisioning_jobs        TO keystone_worker;
GRANT SELECT, INSERT, UPDATE ON outbox_events            TO keystone_worker;
GRANT SELECT                 ON tenant_webhook_endpoints TO keystone_worker;
GRANT INSERT                 ON audit_log                TO keystone_worker;

DO $$
BEGIN
    EXECUTE format('GRANT CREATE ON DATABASE %I TO keystone_worker', current_database());
END
$$;

-- These admit every row, for this role only. The grants above decide which commands the
-- worker may run; the policies decide that the tenant policy does not hide the rows from
-- a process that serves every tenant.
--
-- They look like the USING (TRUE) policy migration 028 removed from users, and the
-- difference is the whole point: that one named no role, so it applied to everyone.
CREATE POLICY worker_policy ON operations
    FOR ALL TO keystone_worker USING (TRUE) WITH CHECK (TRUE);
CREATE POLICY worker_policy ON provisioning_jobs
    FOR ALL TO keystone_worker USING (TRUE) WITH CHECK (TRUE);
CREATE POLICY worker_policy ON outbox_events
    FOR ALL TO keystone_worker USING (TRUE) WITH CHECK (TRUE);
CREATE POLICY worker_policy ON tenant_webhook_endpoints
    FOR SELECT TO keystone_worker USING (TRUE);
CREATE POLICY worker_policy ON audit_log
    FOR INSERT TO keystone_worker WITH CHECK (TRUE);
