-- Drop trigger
DROP TRIGGER IF EXISTS payment_events_updated_at_trigger ON payment_events;

-- Drop function
DROP FUNCTION IF EXISTS update_payment_events_updated_at();

-- Drop table
DROP TABLE IF EXISTS payment_events CASCADE;
