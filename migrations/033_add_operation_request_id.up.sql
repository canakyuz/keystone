-- 033: carry the originating request id on the operation.
--
-- PROBLEM
-- Provisioning is asynchronous, so the request that asked for a tenant and the worker
-- that builds it are in different processes, minutes apart, writing to different logs.
-- Diagnosing a failed provisioning meant matching timestamps by eye.
--
-- The diagnostic chain the design calls for is
--   request_id -> operation_id -> job_id -> attempt -> worker -> migration step
-- and every link existed except the first. INVARIANTS.md and a comment in
-- internal/worker/provision_handler.go both recorded it as missing.
--
-- FIX
-- The API records the correlation id it already generates for every request, and the
-- worker reads it back when it claims the job. One column closes the chain.
--
-- It is nullable on purpose: operations created before this migration have no request
-- id, and inventing one would be worse than an honest NULL. It is also not a foreign
-- key to anything; the value is a log correlation token, not a reference.
ALTER TABLE operations ADD COLUMN IF NOT EXISTS request_id VARCHAR(64);

COMMENT ON COLUMN operations.request_id IS
    'Correlation id of the HTTP request that created this operation. Log correlation only.';
