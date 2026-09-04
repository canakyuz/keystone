-- 028 geri alma.
-- DİKKAT: bu, users tablosundaki okuma izolasyonunu tamamen kaldırır.

DROP POLICY IF EXISTS auth_lookup_policy ON users;

CREATE POLICY auth_policy ON users
    FOR SELECT
    USING (TRUE);
