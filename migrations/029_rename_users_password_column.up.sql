-- 029: users.password -> users.password_hash
--
-- SORUN
-- 002_create_users.up.sql creates the column as "password", but
-- internal/repository/user/postgres.go reads and writes "password_hash" across six
-- separate queries. No migration ever reconciled the difference. The result: the
-- entire user repository failed against the real schema with
-- "column password_hash does not exist".
--
-- The drift went unnoticed because the tests did not use the migration files; they
-- used a separate schema copied by hand into the test helper.
--
-- FIX
-- Make the column match the code. "password_hash" is the right name: the field holds
-- a bcrypt digest, not a plaintext password, and making that distinction visible in
-- the schema prevents misreading it during a log or dump review.

ALTER TABLE users RENAME COLUMN password TO password_hash;

COMMENT ON COLUMN users.password_hash IS 'bcrypt ozeti. Duz parola asla saklanmaz.';
