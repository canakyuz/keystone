-- 037 rollback: plain http destinations become storable again.
ALTER TABLE tenant_webhook_endpoints DROP CONSTRAINT IF EXISTS tenant_webhook_endpoints_https_only;
ALTER TABLE tenant_webhook_endpoints DROP CONSTRAINT IF EXISTS tenant_webhook_endpoints_no_credentials;
