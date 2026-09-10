-- 036: the audit trail.
--
-- PROBLEM
-- Rule 7 in docs/INVARIANTS.md was marked "not yet" because there was no audit trail at
-- all. Who changed what was recorded only in operations.created_by, which says who asked
-- for a provisioning and nothing about what came of it.
--
-- WRITTEN IN THE SAME TRANSACTION AS THE CHANGE
-- The entry goes in beside the state change it describes, for the same reason the outbox
-- event does: an audit trail that can disagree with the data is worse than none, because
-- somebody will trust it. Written afterwards, a crash in between leaves a change nobody
-- can account for; written before, a rollback leaves a record of something that never
-- happened.
--
-- APPEND ONLY
-- There is no UPDATE or DELETE path in the application, and the revoke below removes the
-- privilege rather than relying on nobody writing that code. A trail that can be edited
-- by the thing it is auditing answers no question worth asking.
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Who. NULL means the system acted on its own, a worker completing a job for example,
    -- which is different from an unknown actor and is worth being able to tell apart.
    actor_id UUID,
    actor_type VARCHAR(20) NOT NULL DEFAULT 'system'
        CHECK (actor_type IN ('user', 'system', 'api')),

    -- What, and to what.
    action VARCHAR(100) NOT NULL,
    subject_type VARCHAR(50) NOT NULL,
    subject_id UUID NOT NULL,

    -- Enough context to explain the entry without joining to rows that may since have
    -- changed. An audit record that depends on current data to be readable is not a
    -- record of the past.
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- The diagnostic chain, so an entry can be tied to the request and the trace that
    -- produced it. See migrations 033 and 034.
    request_id VARCHAR(64),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT audit_log_action_not_blank CHECK (length(trim(action)) > 0)
);

-- The query an auditor actually runs: what happened to this tenant, most recent first.
CREATE INDEX idx_audit_log_tenant_time ON audit_log(tenant_id, created_at DESC);

-- And the other one: everything that happened to this object.
CREATE INDEX idx_audit_log_subject ON audit_log(subject_type, subject_id, created_at DESC);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON audit_log
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- Append only, enforced by the database rather than by convention.
REVOKE UPDATE, DELETE ON audit_log FROM PUBLIC;
REVOKE UPDATE, DELETE ON audit_log FROM application_user;

COMMENT ON TABLE audit_log IS
    'Append-only trail, written in the same transaction as the change it describes.';
