-- 040: who may act across tenants.
--
-- PROBLEM
-- Listing every tenant, counting them all, looking one up by slug, and suspending,
-- reactivating or repricing any of them are operator actions, not something a tenant's own
-- administrator does. They were open to every authenticated caller, and closing them was
-- the only honest answer available at the time: there was nobody to grant them to.
-- middleware.PlatformOnly refused everyone.
--
-- FIX
-- This table is the grant. A row says one subject may act across tenants. The API reads it
-- on those routes and nowhere else, so the permission cannot be inferred from a tenant role:
-- an owner of one tenant is still nobody at the platform level until a row exists here.
--
-- Deliberately not tenant data, so it carries no RLS. Tenant policies exist to keep one
-- tenant out of another's rows; this table is the record of the few subjects that are
-- allowed to cross that line, and hiding it from the API that has to read it would only
-- mean checking the permission somewhere weaker. The worker has no grant on it (see 039)
-- because the worker never answers a request.
CREATE TABLE IF NOT EXISTS platform_operators (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,

    -- Why this subject has it, and who said so. An unexplained standing permission is the
    -- one nobody dares revoke.
    note TEXT,
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE platform_operators IS
    'Subjects allowed to act across tenants. Checked by the API on the platform routes only.';
