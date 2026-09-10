-- 034: carry the originating trace context on the operation.
--
-- Migration 033 added request_id, which is enough to join log lines. This is the same
-- idea for traces: the W3C traceparent of the request that created the operation, so the
-- worker's span can point back at it.
--
-- WHY a column rather than a header: the HTTP hop has headers, this hop does not. The
-- producer writes a row and the consumer reads it back, in another process, minutes or
-- hours later and possibly several times. The only channel between them is the row.
--
-- The worker turns this into a span LINK rather than a parent. A parent would keep the
-- request's trace open until the job finally succeeds, and a trace only completes when
-- all its spans do, so a job that keeps failing would leave a trace that never closes.
-- See pkg/tracing.LinkFrom.
--
-- Nullable: operations created before this migration, and any created while tracing is
-- switched off, have nothing to carry.
ALTER TABLE operations ADD COLUMN IF NOT EXISTS trace_context VARCHAR(128);

COMMENT ON COLUMN operations.trace_context IS
    'W3C traceparent of the request that created this operation. Linked, not parented.';
