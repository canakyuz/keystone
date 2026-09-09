-- 028 geri alma.
-- WARNING: this removes read isolation from the users table entirely.

DROP POLICY IF EXISTS auth_lookup_policy ON users;

CREATE POLICY auth_policy ON users
    FOR SELECT
    USING (TRUE);
