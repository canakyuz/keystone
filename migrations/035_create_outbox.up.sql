-- 035: transactional outbox and the endpoints it delivers to.
--
-- PROBLEM
-- Rule 8 in docs/INVARIANTS.md says an external webhook failure must not roll back a
-- completed provisioning. Until now that rule was marked "not yet" because there was no
-- delivery at all, and the naive way to add it would break the rule on the first try:
-- calling an HTTP endpoint inside the transaction that completes the job means a slow or
-- dead receiver holds a database transaction open, and a failed call either rolls back
-- work that actually succeeded or is swallowed and lost.
--
-- THE OUTBOX
-- The event is written in the same transaction as the business data. That single fact is
-- the whole pattern: either both the provisioning and the intent to notify commit, or
-- neither does. There is no window where the tenant is active and nobody will ever be
-- told, and none where a notification goes out for work that was rolled back.
--
-- Delivery is then a separate, retryable step that reads committed rows. It can be slow,
-- it can fail, it can run on another machine, and none of that touches the transaction
-- that produced the event.
--
-- WHAT THIS DOES NOT GIVE
-- Exactly-once delivery, which is not available over HTTP. A receiver that accepts a
-- request and then fails before acknowledging will be sent it again. Delivery carries an
-- Idempotency-Key so the receiver can recognise the repeat; making use of it is the
-- receiver's half of the contract.

-- 1) TENANT_WEBHOOK_ENDPOINTS ---------------------------------------------
--
-- Where a tenant wants to be told. Resolved when the event is written rather than when it
-- is delivered, so each row has exactly one destination and its own retry state: an
-- endpoint that is down cannot delay delivery to one that is up.
CREATE TABLE IF NOT EXISTS tenant_webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    url TEXT NOT NULL,

    -- Used to sign the delivery so the receiver can tell a real notification from anyone
    -- who guessed the URL.
    secret TEXT NOT NULL,

    -- A disabled endpoint stops receiving without losing its history, which is what an
    -- operator wants while a receiver is being repaired.
    active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_webhook_endpoints_url_not_blank CHECK (length(trim(url)) > 0)
);

CREATE INDEX idx_webhook_endpoints_tenant ON tenant_webhook_endpoints(tenant_id) WHERE active;

ALTER TABLE tenant_webhook_endpoints ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_webhook_endpoints FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON tenant_webhook_endpoints
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- 2) OUTBOX_EVENTS ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- What the event is about, and what happened to it.
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,

    payload JSONB NOT NULL DEFAULT '{}'::jsonb,

    endpoint_id UUID NOT NULL REFERENCES tenant_webhook_endpoints(id) ON DELETE CASCADE,

    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'delivering', 'delivered', 'dead')),

    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 8,
    next_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- The same lease and fence the provisioning queue uses, and for the same reasons: a
    -- delivery process can die without saying so, and a worker that lost its lease must
    -- not be able to mark an event delivered after another worker has taken it over.
    -- See docs/decisions/0003-lease-and-fencing.md.
    lease_owner VARCHAR(255),
    lease_expires_at TIMESTAMP WITH TIME ZONE,
    fence BIGINT NOT NULL DEFAULT 0,

    last_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,

    -- A delivered event has a delivery time. Without this a partial update leaves a row
    -- that claims success and cannot say when.
    CONSTRAINT outbox_delivered_has_timestamp
        CHECK (status <> 'delivered' OR delivered_at IS NOT NULL),

    CONSTRAINT outbox_delivering_has_lease
        CHECK (status <> 'delivering' OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL))
);

-- The set the claim query scans: undelivered events only.
--
-- Delivered and dead rows accumulate forever and are never claimed again, so leaving them
-- in the index would make the claim cost grow with the total number of events ever
-- emitted rather than with the number waiting.
CREATE INDEX idx_outbox_claimable
    ON outbox_events(next_attempt_at)
    WHERE status IN ('pending', 'delivering');

CREATE INDEX idx_outbox_tenant ON outbox_events(tenant_id, created_at DESC);

ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_policy ON outbox_events
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

COMMENT ON TABLE outbox_events IS
    'Events written in the same transaction as the work they describe, delivered separately.';
COMMENT ON COLUMN outbox_events.fence IS
    'Incremented on every claim. Used to reject a late report from a worker.';
