-- 029: users.password -> users.password_hash
--
-- SORUN
-- 002_create_users.up.sql kolonu "password" olarak oluşturuyor, ancak
-- internal/repository/user/postgres.go altı ayrı sorguda "password_hash"
-- kolonunu okuyup yazıyor. Aradaki fark hiçbir migration'da giderilmemiş.
-- Sonuç: kullanıcı repository'sinin tamamı gerçek şemaya karşı
-- "column password_hash does not exist" ile başarısız oluyordu.
--
-- Kayma fark edilmedi çünkü testler migration dosyalarını değil, test
-- helper'ının içine elle kopyalanmış ayrı bir şemayı kullanıyordu.
--
-- ÇÖZÜM
-- Kolonu koda uydur. "password_hash" doğru isim: alan düz parola değil,
-- bcrypt özeti tutuyor ve bu ayrımın şemada görünür olması log/dump
-- incelemelerinde yanlış yorumu engeller.

ALTER TABLE users RENAME COLUMN password TO password_hash;

COMMENT ON COLUMN users.password_hash IS 'bcrypt ozeti. Duz parola asla saklanmaz.';
