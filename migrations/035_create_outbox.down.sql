-- 035 rollback: undelivered events are lost with the table.
DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS tenant_webhook_endpoints CASCADE;
