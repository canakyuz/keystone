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

## Önbellek

Tenant çözümlemesi her istekte çalışır, dolayısıyla en sıcak yoldur.
`pkg/cache` iki katmanlı bir önbellek sunar: L1 süreç içi, L2 Redis, en altta
veritabanı.

Kapatılan sorunlar:

- **Önbellek yığılması.** Soğuk bir anahtara aynı anda gelen N istek N adet
  veritabanı sorgusuna dönüşüyordu. `singleflight` ile tek sorguya iniyor.
  Test yüz eşzamanlı isteğin tek yükleme yaptığını doğruluyor.
- **Negatif önbellekleme yoktu.** Var olmayan rastgele tenant kimlikleriyle
  yapılan istek seli her seferinde veritabanına iniyordu. Ucuz bir yük
  yükseltme vektörü.
- **Sınırsız goroutine.** Önbellek doldurma her istekte yeni bir goroutine
  açıyordu, panic recovery de yoktu.
- **Redis tek hata noktasıydı.** L1 katmanı sayesinde Redis düştüğünde servis
  çalışmaya devam ediyor.
- **İsabet oranı ölçülmüyordu.** `Stats()` ile ölçülebiliyor.

TTL'e jitter eklenir. Aynı anda oluşturulan anahtarlar aynı anda düşerse
sona erme anında toplu bir ıska dalgası oluşur.

## İstek limiti

`pkg/ratelimit`, Redis üzerinde paylaşılan bir token bucket uygular. Oku,
hesapla, yaz dizisi tek bir Lua betiğinde çalışır, dolayısıyla iki replika
aynı tokeni harcayamaz.

Fiber'ın yerleşik limiter'ı kullanılmaz. O, varsayılan olarak süreç içi
sayar: üç replikada, replika başına yüz istek ayarı gerçekte üç yüz istek
demektir. Ayarlanan değer ile uygulanan değer arasında replika sayısı kadar
fark oluşur.

Limit kiracı planına göre belirlenir ve anahtar IP yerine tenant'tır.

| Plan | Dakikalık istek |
|---|---|
| free | 60 |
| starter | 300 |
| pro | 1.200 |
| enterprise | 6.000 |
| kimlik doğrulanmamış | 30 |

Redis erişilemediğinde varsayılan davranış isteği geçirmektir. Limitleyici bir
kullanılabilirlik aracı değil, kötüye kullanım frenidir. Redis düştüğünde tüm
trafiği reddetmek, önlemeye çalıştığı kesintiyi kendi eliyle yaratır.

Eşzamanlılık testi iki yüz eşzamanlı istekten tam olarak kapasite kadarının
geçtiğini doğrular.

## Sağlık uçları

İki ayrı uç, iki ayrı soru.

| Uç | Soru | Başarısız olursa | Bağımlılıklara bakar |
|---|---|---|---|
| `/health` | Süreç ayakta mı | Container yeniden başlar | Hayır |
| `/ready` | İstek karşılayabilir mi | Load balancer trafiği keser | Evet |

`/health` bilinçli olarak veritabanına bakmaz. Veritabanı geçici olarak
düştüğünde sağlıklı süreçleri yeniden başlatmak, kurtarma sırasında bağlantı
fırtınası yaratır.

`/ready` için Redis zorunlu değildir. Önbellek ve limitleyici Redis olmadan
süreç içi yollarına düşerek çalışır.

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
