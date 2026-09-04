# Keystone

Go ile yazılmış multi-tenant SaaS control plane. Tenant sağlama,
schema-per-tenant izolasyonu ve Row Level Security'yi tek bir yerde toplar.

[![CI](https://github.com/canakyuz/keystone/actions/workflows/ci.yml/badge.svg)](https://github.com/canakyuz/keystone/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Ne yapar

Bir SaaS ürününde her müşteri için ayrı veri alanı açmak, o alanı yalıtmak ve
yalıtımın gerçekten çalıştığını kanıtlamak gerekir. Keystone bu üç işi yapar.

- **Tenant sağlama.** Yeni tenant için PostgreSQL şeması oluşturur, migration'ları
  uygular, kayıt defterine yazar.
- **İki katmanlı izolasyon.** Tenant'a özel şema artı paylaşılan tablolarda RLS.
- **Doğrulanmış izolasyon.** İddialar `test/security/` altında, süper kullanıcı
  olmayan bir rolle ve gerçek PostgreSQL üzerinde sınanır.

## Neden ilginç

Bu repo tenant izolasyonunu yalnızca iddia etmiyor, sınıyor. İzolasyonu gerçekten
ölçen testler yazıldığında beş ayrı hata ortaya çıktı ve hepsi kayıt altında.

En çarpıcısı şuydu: `users` tablosunda iki permissive policy vardı.

```sql
CREATE POLICY tenant_isolation_policy ON users FOR ALL
  USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

CREATE POLICY auth_policy ON users FOR SELECT
  USING (TRUE);
```

PostgreSQL permissive policy'leri OR ile birleştirir. İkincisi sabit `TRUE`
olduğu için birleşik koşul her `SELECT`'te doğruya düşüyordu. `users` tablosunda
okuma yönünde hiçbir tenant izolasyonu yoktu.

Tam liste ve düzeltmeler: [SECURITY.md](SECURITY.md).

## Mimari

```
HTTP (Fiber)
    |
    v
Handler  --->  Usecase  --->  Repository  --->  PostgreSQL
                                  |
                                  +-- TenantConnectionManager
                                      havuzdan bir baglanti ayirir,
                                      search_path'i tenant semasina alir,
                                      AYNI baglantiyi callback'e verir
```

Bağımlılık yönü daima içeri doğrudur. `internal/domain` hiçbir dış katmanı
bilmez. Tenant context anahtarları `pkg/tenantctx` içinde, yaprak bir pakette
durur; böylece repository katmanı HTTP middleware'ini import etmek zorunda kalmaz.

## Hızlı başlangıç

```bash
git clone https://github.com/canakyuz/keystone.git
cd keystone
cp .env.example .env

docker compose up -d postgres
go run ./cmd/server
```

Sağlık kontrolü:

```bash
curl localhost:8080/health
```

## Test

```bash
go test ./...              # tamami
go test ./test/security/   # yalnizca izolasyon iddialari
```

Testler gerçek PostgreSQL kullanır. Her test kendi izole veritabanını açar ve
sonunda düşürür, dolayısıyla paralel koşu güvenlidir. Postgres erişilemiyorsa
ilgili testler atlanır.

Şemanın tek kaynağı `migrations/` dizinidir. Test helper'ı bu dosyaları
doğrudan çalıştırır, kopya tutmaz.

## İşletim koşulu

Uygulama veritabanına **süper kullanıcı olmayan** bir rolle bağlanmalıdır.
PostgreSQL'de süper kullanıcılar RLS'i her koşulda atlar. Ayrıntı:
[SECURITY.md](SECURITY.md).

## Durum

Çekirdek çalışır durumda: tenant sağlama, kullanıcı yönetimi, RLS zorlaması,
migration runner. Dikey modüller (blog, rezervasyon, ders, ödeme) referans
uygulama olarak repoda durur ve control plane'in üzerine nasıl özellik
inşa edildiğini gösterir.

Yol haritası: gRPC sözleşmeleri, transactional outbox, webhook teslimatı,
ölçülmüş yük testi sonuçları.

## Lisans

MIT. Bkz. [LICENSE](LICENSE).
