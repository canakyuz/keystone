-- 032: add the provisioning states to the tenant lifecycle.
--
-- PROBLEM
-- The schema allowed only four states: active, suspended, inactive, trial. That was
-- enough while tenant creation was synchronous; a tenant either existed or it did
-- not.
--
-- Once provisioning became asynchronous, the interval in between became visible. If a
-- tenant record exists but its schema is not ready, its status cannot be 'active': if
-- its requests were accepted they would be routed at a schema that does not exist.
--
-- STATE MACHINE
--
--   pending ---> provisioning ---> active <---> suspended
--                     |
--                     +---------> failed
--
-- 'failed' and 'inactive' are kept apart. The first says provisioning failed, the
-- second says a working tenant was shut down. Merging them into one value would leave
-- the question "can provisioning be retried?" unanswerable.
--
-- A new operation can be opened for a failed tenant; the tenant stays 'failed' while
-- the new operation starts 'pending'. That is the reason operation status is kept
-- separate from tenant status (see migration 031).

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;

ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN (
        'pending',       -- record created, provisioning not started
        'provisioning',  -- the worker is preparing the schema
        'active',        -- may accept requests
        'suspended',     -- temporarily stopped
        'inactive',      -- shut down
        'failed',        -- provisioning failed
        'trial'
    ));

COMMENT ON COLUMN tenants.status IS
    'Lifecycle state. Only a tenant in the active state accepts requests.';
