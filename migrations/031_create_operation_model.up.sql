-- 031: a durable operation model for long-running work.
--
-- PROBLEM
-- Tenant provisioning ran synchronously: the HTTP request waited until the schema was
-- created. That had two consequences. When the request timed out, the client had no
-- way to learn the state of the work. And when the process died mid-provisioning,
-- nobody could take the half-finished job over.
--
-- FIX
-- Three tables: what is being done (operations), who will do it
-- (provisioning_jobs), and what happens when the same request arrives again
-- (idempotency_keys).

-- 1) OPERATIONS -----------------------------------------------------------
--
-- The user-facing record. The client receives this record's address and polls it for
-- the status.
--
-- WHY separate from the tenant status: a tenant's state and a single operation's
-- state are not the same thing. After a failed provisioning a new operation can be
-- opened for the same tenant; the tenant stays 'failed' while the new operation
-- starts 'pending'. Holding both in one column would make that impossible.
CREATE TABLE IF NOT EXISTS operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- The kind of work, e.g. 'tenant.provision'.
    kind VARCHAR(64) NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    -- The error is split into a machine-readable code and a human-facing message. The
    -- client branches on the code and shows the message to the user.
    error_code VARCHAR(64),
    error_message TEXT,

    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT operations_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),

    -- A completed operation MUST have a completion time.
    -- Without this constraint, a partial update silently leaves an inconsistent row.
    CONSTRAINT operations_completed_at_present
        CHECK (
            (status IN ('succeeded', 'failed') AND completed_at IS NOT NULL)
            OR (status IN ('pending', 'running') AND completed_at IS NULL)
        )
);

CREATE INDEX idx_operations_tenant ON operations(tenant_id, created_at DESC);
CREATE INDEX idx_operations_pending ON operations(status) WHERE status IN ('pending', 'running');

-- 2) PROVISIONING_JOBS ----------------------------------------------------
--
-- The unit of work a worker claims. Operations faces the user; this table faces the
-- executor.
CREATE TABLE IF NOT EXISTS provisioning_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation_id UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,

    -- When the job may run. Pushed back on a retry.
    next_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- LEASE
    -- A worker claims the job for a bounded time, not forever.
    -- With a persistent 'processing' flag, a process dying mid-provisioning would leave
    -- the job stuck behind that flag forever and nobody could take it over.
    -- A job whose lease expires can be claimed again.
    lease_owner VARCHAR(128),
    lease_expires_at TIMESTAMP WITH TIME ZONE,

    -- FENCING TOKEN
    -- A counter incremented on every claim.
    --
    -- WHY it is needed: once the lease has expired and the job has moved to another
    -- worker, the old worker can come back and report "completed". Checking lease_owner
    -- alone is not enough; the old worker knows its own name. A result must be reported
    -- with the current fence value, so a late report is rejected.
    fence BIGINT NOT NULL DEFAULT 0,

    last_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT provisioning_jobs_status_check
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'dead')),

    CONSTRAINT provisioning_jobs_attempts_bounded
        CHECK (attempts >= 0 AND attempts <= max_attempts),

    -- A running job must have an owner and a lease expiry.
    CONSTRAINT provisioning_jobs_running_has_lease
        CHECK (
            status <> 'running'
            OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
        )
);

-- The set the claim query scans: runnable jobs only.
-- The partial index leaves completed jobs out of the scan, so claim cost stays flat as
-- the table grows.
CREATE INDEX idx_jobs_claimable
    ON provisioning_jobs(next_attempt_at)
    WHERE status IN ('pending', 'running');

CREATE INDEX idx_jobs_operation ON provisioning_jobs(operation_id);

-- An operation cannot have more than one active job at a time.
CREATE UNIQUE INDEX idx_jobs_one_active_per_operation
    ON provisioning_jobs(operation_id)
    WHERE status IN ('pending', 'running');

-- 3) IDEMPOTENCY_KEYS -----------------------------------------------------
--
-- Returns the same logical operation when the same request arrives again.
--
-- WHY a unique constraint: the "check whether it exists, insert if not" sequence does
-- not prevent the race on its own. Two requests can pass the check at the same moment
-- and create two separate operations. The database enforces uniqueness; the application
-- catches the violation and returns the existing record.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- The scope the key is valid in. One customer's key must not match another
    -- customer's request.
    scope VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,

    -- A normalised digest of the request.
    -- The same key arriving with a different body is reported as a conflict; silently
    -- returning the old result would mislead the client.
    request_fingerprint CHAR(64) NOT NULL,

    operation_id UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- The retention window. Once it expires the same key creates a new operation; the
    -- idempotency guarantee is therefore not indefinite, and this window must be stated
    -- in the API documentation.
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX idx_idempotency_scope_key
    ON idempotency_keys(scope, idempotency_key);

CREATE INDEX idx_idempotency_expiry ON idempotency_keys(expires_at);

-- RLS: operation and job records are tenant data.
-- They follow the fail-closed form from migration 030.
ALTER TABLE operations ENABLE ROW LEVEL SECURITY;
ALTER TABLE operations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON operations
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE provisioning_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE provisioning_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON provisioning_jobs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

COMMENT ON COLUMN provisioning_jobs.fence IS
    'Incremented on every claim. Used to reject a late report from a worker.';
COMMENT ON COLUMN provisioning_jobs.lease_expires_at IS
    'Lease bitisi. Suresi dolan is baska bir worker tarafindan devralinabilir.';
