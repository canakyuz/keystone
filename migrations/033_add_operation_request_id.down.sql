-- 033 rollback: the diagnostic chain loses its first link.
ALTER TABLE operations DROP COLUMN IF EXISTS request_id;
