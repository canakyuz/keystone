-- 027: Row Level Security'yi tablo sahibine de zorunlu kıl.
--
-- SORUN
-- Önceki migration'ların hepsi "ALTER TABLE x ENABLE ROW LEVEL SECURITY" kullanıyor.
-- PostgreSQL'de ENABLE, tablonun SAHİBİNİ policy'lerden muaf tutar. Uygulama
-- neredeyse her kurulumda tabloları oluşturan rolle bağlandığı için (bu repodaki
-- docker-compose dahil) tenant izolasyon policy'leri sessizce devre dışı kalıyordu.
--
-- ÖLÇÜM (süper kullanıcı olmayan, tablo sahibi rolle)
--   ENABLE  -> iki farklı tenant'ın satırlarından 2'si de görünüyor
--   FORCE   -> yalnızca 1'i görünüyor, yani policy uygulanıyor
--
-- ÇÖZÜM
-- FORCE ROW LEVEL SECURITY, sahibi de policy'lere tabi kılar.
--
-- KALAN OPERASYONEL KOŞUL
-- Süper kullanıcılar RLS'i her koşulda atlar, FORCE bunu değiştirmez. Uygulama
-- bağlantısı ASLA süper kullanıcı olmamalıdır. Bkz. SECURITY.md.
--
-- NOT: tenants tablosu bilinçli olarak kapsam dışıdır. O tablo control-plane
-- verisidir ve tenant'a göre daraltılamaz; izolasyonu uygulama katmanında yapılır.

ALTER TABLE appointments     FORCE ROW LEVEL SECURITY;
ALTER TABLE assignments      FORCE ROW LEVEL SECURITY;
ALTER TABLE availabilities   FORCE ROW LEVEL SECURITY;
ALTER TABLE blog_categories  FORCE ROW LEVEL SECURITY;
ALTER TABLE blog_posts       FORCE ROW LEVEL SECURITY;
ALTER TABLE lessons          FORCE ROW LEVEL SECURITY;
ALTER TABLE payment_events   FORCE ROW LEVEL SECURITY;
ALTER TABLE payments         FORCE ROW LEVEL SECURITY;
ALTER TABLE projects         FORCE ROW LEVEL SECURITY;
ALTER TABLE refunds          FORCE ROW LEVEL SECURITY;
ALTER TABLE services         FORCE ROW LEVEL SECURITY;
ALTER TABLE sites            FORCE ROW LEVEL SECURITY;
ALTER TABLE students         FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules   FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_tools     FORCE ROW LEVEL SECURITY;
ALTER TABLE users            FORCE ROW LEVEL SECURITY;
ALTER TABLE websites         FORCE ROW LEVEL SECURITY;
