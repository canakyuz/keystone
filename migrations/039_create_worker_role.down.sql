DROP POLICY IF EXISTS worker_policy ON audit_log;
DROP POLICY IF EXISTS worker_policy ON tenant_webhook_endpoints;
DROP POLICY IF EXISTS worker_policy ON outbox_events;
DROP POLICY IF EXISTS worker_policy ON provisioning_jobs;
DROP POLICY IF EXISTS worker_policy ON operations;

REVOKE ALL ON tenants, operations, provisioning_jobs, outbox_events, tenant_webhook_endpoints, audit_log
    FROM keystone_worker;

DO $$
BEGIN
    EXECUTE format('REVOKE CREATE ON DATABASE %I FROM keystone_worker', current_database());
END
$$;

REVOKE USAGE ON SCHEMA public FROM keystone_worker;

-- The role itself stays. Roles belong to the cluster, and another database in it may
-- still hold grants to this one.
