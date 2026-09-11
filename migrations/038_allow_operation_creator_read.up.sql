-- 038: let the subject who asked for an operation read it back.
--
-- PROBLEM
-- operations carries the fail-closed tenant policy from migration 031. The status
-- lookup, GET /api/v1/operations/:id, runs without tenant context: the caller is polling
-- a tenant that is still being provisioned and that it is not yet a member of. Under the
-- non-superuser role the lookup matched no row, so every poll answered 404, and so did
-- every idempotent replay, which reads the operation the same way. Nobody saw it because
-- the repository tests connect as a superuser, and RLS never applies to a superuser.
--
-- Dropping the policy for this query would have turned the lookup into a read of any
-- tenant's operation by id. Visibility is tied to the subject who created the operation
-- instead.
--
-- FIX
-- A second policy, SELECT only. The repository sets app.current_subject from the verified
-- token subject with set_config(..., true), which lasts until the end of the transaction.
-- Permissive policies combine with OR, so reads that carry a tenant context are unchanged,
-- and writes are still governed by the tenant policy alone.
CREATE POLICY operation_creator_read_policy ON operations
    FOR SELECT
    USING (created_by = NULLIF(current_setting('app.current_subject', TRUE), '')::UUID);

COMMENT ON POLICY operation_creator_read_policy ON operations IS
    'The creator of an operation can read it without tenant context. Set per transaction by the repository from the verified subject.';
