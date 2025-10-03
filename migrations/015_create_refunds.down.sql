-- Drop trigger
DROP TRIGGER IF EXISTS refunds_updated_at_trigger ON refunds;

-- Drop function
DROP FUNCTION IF EXISTS update_refunds_updated_at();

-- Drop table
DROP TABLE IF EXISTS refunds CASCADE;
