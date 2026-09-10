-- 034 rollback: the worker loses the link back to the originating request.
ALTER TABLE operations DROP COLUMN IF EXISTS trace_context;
