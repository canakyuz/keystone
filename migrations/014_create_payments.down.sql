-- Drop trigger
DROP TRIGGER IF EXISTS payments_updated_at_trigger ON payments;

-- Drop function
DROP FUNCTION IF EXISTS update_payments_updated_at();

-- Drop table (CASCADE to drop dependent objects)
DROP TABLE IF EXISTS payments CASCADE;
