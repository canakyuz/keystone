-- 041: a record of every file the API stores.
--
-- PROBLEM
-- Uploads were written to disk and nowhere else. Nobody could say who stored a file or when,
-- a tenant's files could not be listed or counted, and uploads were the one change made
-- through the API that the audit trail did not see (the limit under rule 7 in
-- docs/INVARIANTS.md).
--
-- ORDER OF WRITES
-- A file system and a database cannot share a transaction, so one of them goes first. The
-- file does: then the row and its audit entry commit together, and when that transaction
-- fails the handler removes the file. A crash between the two leaves a file with no row,
-- which nothing points to and which a sweep can find. The other order would leave rows
-- pointing at files that do not exist.
CREATE TABLE IF NOT EXISTS uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    category VARCHAR(20) NOT NULL CHECK (category IN ('logo', 'favicon', 'image')),

    -- Where the file is served from. The server chose every part of it, so it holds nothing
    -- the uploader wrote.
    path TEXT NOT NULL UNIQUE,

    -- Read from the file's own bytes, not from what the client declared.
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),

    -- The client's file name is not kept. It is text a person chose and can hold a name or
    -- anything else, and the audit entry beside this row is append-only.
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_uploads_tenant_time ON uploads(tenant_id, created_at DESC);

ALTER TABLE uploads ENABLE ROW LEVEL SECURITY;
ALTER TABLE uploads FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON uploads
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

COMMENT ON TABLE uploads IS
    'One row per stored file, written with its audit entry after the file reaches disk.';
