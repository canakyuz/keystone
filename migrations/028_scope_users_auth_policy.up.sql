-- 028: users tablosundaki sınırsız okuma policy'sini daralt.
--
-- SORUN
-- 002_create_users.up.sql iki permissive policy tanımlıyordu:
--
--   CREATE POLICY tenant_isolation_policy ON users FOR ALL
--     USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
--
--   CREATE POLICY auth_policy ON users FOR SELECT
--     USING (TRUE);
--
-- PostgreSQL permissive policy'leri OR ile birleştirir. İkincisi sabit TRUE
-- olduğu için birleşik koşul her SELECT'te TRUE'ya düşüyordu. Sonuç: users
-- tablosunda okuma yönünde hiçbir tenant izolasyonu yoktu. Herhangi bir tenant
-- bağlamı, tüm tenant'ların kullanıcı satırlarını (password_hash dahil)
-- okuyabiliyordu. FORCE ROW LEVEL SECURITY bunu düzeltmez; sorun policy
-- mantığındadır, sahiplik muafiyetinde değil.
--
-- auth_policy'nin amacı meşruydu: tenant henüz bilinmeden yapılan login
-- aramasına (GetByEmailGlobal) izin vermek. Ama bunu kalıcı ve koşulsuz bir
-- açıklık olarak uyguluyordu.
--
-- ÇÖZÜM
-- Genişletilmiş görünürlük artık açık bir oturum bayrağı ister. Repository
-- bu bayrağı yalnızca global login araması için, SET LOCAL ile ve tek bir
-- transaction sınırında açar. Transaction bitince yetki kendiliğinden kapanır.

DROP POLICY IF EXISTS auth_policy ON users;

CREATE POLICY auth_lookup_policy ON users
    FOR SELECT
    USING (current_setting('app.auth_lookup', TRUE) = 'on');

COMMENT ON POLICY auth_lookup_policy ON users IS
    'Yalnızca tenant bilinmeden yapılan login aramasi icin. Repository SET LOCAL app.auth_lookup ile transaction basina acar.';
