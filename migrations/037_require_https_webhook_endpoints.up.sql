-- 037: webhook destinations must be https.
--
-- A cheap constraint at the last place a bad value can be stopped. It is deliberately not
-- the defence: the real one is in pkg/outbound, which checks the address the connection
-- actually resolves to at the moment it dials.
--
-- The reason the constraint cannot be the defence is that a URL is not an address. A
-- hostname that resolves to a public address when it is registered can resolve to
-- 169.254.169.254 by the time a delivery goes out, and no constraint over the text can
-- see that. What this catches is the careless case — plain http, a literal loopback
-- address — before it ever reaches the queue, and it does so even for a row inserted by
-- something that forgot to call the validator.
--
-- Existing rows are left alone rather than rewritten. There are none in any deployment
-- yet, and a migration that silently edits a customer's configuration is worse than one
-- that refuses to run.
ALTER TABLE tenant_webhook_endpoints
    ADD CONSTRAINT tenant_webhook_endpoints_https_only
    CHECK (url LIKE 'https://%');

ALTER TABLE tenant_webhook_endpoints
    ADD CONSTRAINT tenant_webhook_endpoints_no_credentials
    CHECK (position('@' in split_part(url, '/', 3)) = 0);

COMMENT ON CONSTRAINT tenant_webhook_endpoints_https_only ON tenant_webhook_endpoints IS
    'Deliveries describe a tenant; plain http would send that in clear.';
