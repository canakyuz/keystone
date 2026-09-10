-- 029 rollback.
ALTER TABLE users RENAME COLUMN password_hash TO password;
