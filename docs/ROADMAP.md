# Keystone Yol Haritası

Amaç: repoya on dakika ayıran bir Go mühendisinin "bu kişi kıdemli" demesini
sağlamak. Bu belge o hedefi ölçülebilir fazlara böler.

---

## Başarı ölçütü

Bir işe alım mühendisi repoya girdiğinde şu sırayı izler: README, sonra bir test
çalıştırma denemesi, sonra en ilginç görünen pakete bakma. Her fazın çıktısı bu
üç adımdan birini iyileştirmeli.

Bitiş hedefi, aşağıdakilerin hepsinin tek komutla doğrulanabilir olması.

| İddia | Kanıt |
|---|---|
| Tenant izolasyonu çalışıyor | `go test ./test/security/` |
| İzolasyon HTTP'den veritabanına kadar bütün | `go test ./test/e2e/` |
| Eşzamanlılık doğru yazılmış | `go test -race ./internal/worker/` |
| Performans iddiası ölçülmüş | README'de P50/P95/P99 tablosu |
| Servis sözleşmesi tanımlı | `buf lint` + `grpcurl` örneği |

---

## Mevcut durum

2026-09-04 ölçümü.

| Ölçüm | Değer | Yorum |
|---|---|---|
| Kaynak dosya | 137 | Hacim yeterli |
| Test dosyası | 9 | Kapsam dar |
| HTTP katmanı testi | 0 | En büyük boşluklardan biri |
| `go func` | 3 | Eşzamanlılık sinyali yok denecek kadar az |
| Kanal | 1 | Aynı |
| JSONB dışı `interface{}` | 71 | Çoğu DTO'da, somut tip olmalı |
| Türkçe yorum satırı | 268 / 3077 | Karışık dil |
| Dikey modül | 12 | Çekirdek 3 tanesi, gerisi dikkat dağıtıyor |

Güçlü yan: bugün kapatılan altı izolasyon hatası. Bu iş kıdemli seviyede ama
şu an SECURITY.md'nin içinde gömülü duruyor.

---

## Faz 1: Uçtan uca izolasyon kanıtı

**Süre:** 2-3 gün
**Neden önce bu:** En ucuz faz ve zaten yapılmış işi tamamlıyor. Şu an
izolasyon veritabanı katmanında kanıtlı, ama HTTP'den veritabanına kadar olan
zincir sınanmamış. Tenant middleware'i request'ten şemayı çözüyor ve bütün
izolasyon buna dayanıyor, tek bir testi yok.

**Yapılacaklar**

1. `test/e2e/` paketi aç. Fiber'ın `app.Test()` metodu ile gerçek uygulamayı
   ayağa kaldır, gerçek PostgreSQL'e bağla.
2. Tenant middleware testleri:
   - tenant header'ı yoksa 401
   - bilinmeyen tenant için 404
   - geçersiz schema adı denemesinde 400, SQL injection denemesi reddedilmeli
3. Çapraz tenant testi: tenant A'nın token'ı ile tenant B'nin kaynağını isteyen
   HTTP çağrısı veri döndürmemeli.
4. Middleware'in `search_path`'i istek sonunda sıfırladığını doğrula. Havuza
   kirli bağlantı dönmemeli.

**Dokunulacak yerler:** `internal/middleware/tenant_context.go`,
`internal/app/routes.go`, yeni `test/e2e/`

**Bitti sayılır:** `go test ./test/e2e/` yeşil ve README'de tek komutla
izolasyonu kanıtlayan bir bölüm var.

---

## Faz 2: Transactional outbox ve teslimat worker'ı

**Süre:** 1-2 hafta
**Neden:** Repodaki en büyük boşluk eşzamanlılık. 137 dosyada üç `go func` var.
Go ilanlarının hepsi distributed systems ve concurrency ölçüyor. Bu faz yapay
bir demo değil, ürünün gerçekten ihtiyacı olan bir parça üzerinden worker pool,
context iptali, graceful shutdown ve backoff'u tek seferde gösteriyor.

**Başlangıç noktası hazır:** `internal/domain/webhook/entity.go` 278 satır ve
`RetryCount`, `NeedsRetry`, `Processed` alanlarıyla modellenmiş. Ama
repository'si ve usecase'i yok, yani şu an ölü kod. Önce bunu ödeme özelinden
genel bir outbox modeline çıkar.

**Yapılacaklar**

1. Migration: `outbox_events` tablosu.
   Kolonlar: `id`, `tenant_id`, `aggregate_type`, `aggregate_id`, `event_type`,
   `payload JSONB`, `status`, `attempts`, `next_attempt_at`, `locked_by`,
   `locked_at`, `created_at`, `delivered_at`.
   Index: `(status, next_attempt_at)` partial, `WHERE status = 'pending'`.
   RLS: `tenant_id` üzerinden, fail-closed, migration 030'daki forma uygun.

2. Yazma tarafı: iş verisi ile olay aynı transaction'da yazılır. Bu outbox
   deseninin bütün noktası. Olay kaybolmaz, iş verisi de yarım kalmaz.

3. Okuma tarafı: `SELECT ... FOR UPDATE SKIP LOCKED LIMIT n`.
   Bu tek satır, birden fazla worker'ın aynı olayı almasını veritabanı
   seviyesinde engeller. Uygulama tarafında kilit yönetmeye gerek kalmaz.

4. Worker pool: `internal/worker/outbox.go`
   - N goroutine, yapılandırılabilir
   - `context.Context` ile iptal, `errgroup` ile hata toplama
   - SIGTERM'de graceful shutdown, uçuştaki teslimatlar tamamlanır
   - full jitter'lı exponential backoff, thundering herd'ü engellemek için
   - max attempt sonrası dead letter durumu

5. Teslimat: HTTP webhook, `Idempotency-Key` header'ı ile at-least-once.
   Alıcı tarafta tekrar teslimatı ayırt edebilsin.

6. Testler (`-race` zorunlu):
   - 8 worker, 1000 olay, hiçbir olay iki kez teslim edilmemeli
   - backoff aralıkları beklenen eğriyi izlemeli
   - shutdown sırasında uçuştaki teslimat yarıda kesilmemeli
   - context iptali worker'ları sızdırmadan durdurmalı (goroutine sayımı)

**Bitti sayılır:** `go test -race ./internal/worker/` yeşil ve README'de
`SKIP LOCKED` seçiminin neden yapıldığını anlatan kısa bir bölüm var.

---

## Faz 3: gRPC ve protobuf sözleşmeleri

**Süre:** 1 hafta
**Neden:** Paylaştığın ilanların hepsi REST ve gRPC'yi birlikte istiyor. Ayrıca
tip güvenli sözleşme, mevcut 71 `interface{}` kullanımının bir kısmını doğal
olarak ortadan kaldırır.

**Yapılacaklar**

1. `proto/keystone/v1/` altında tenant, identity ve entitlement servisleri.
2. `buf` ile codegen ve lint. `buf.yaml`, `buf.gen.yaml`, CI'ya `buf lint` adımı.
3. gRPC sunucusu Fiber ile yan yana, ayrı portta.
4. Interceptor zinciri: tenant context, auth, logging, panic recovery.
   Bunlar HTTP middleware ile aynı mantığı paylaşmalı, kopyalanmamalı.
5. Server reflection aç, README'ye `grpcurl` örneği koy.

**Dokunulacak yerler:** yeni `proto/`, yeni `internal/grpc/`,
`internal/app/app.go`

**Bitti sayılır:** `buf lint` temiz, `grpcurl -plaintext localhost:9090 list`
servisleri döküyor, aynı izolasyon testleri gRPC üzerinden de geçiyor.

---

## Faz 4: Gözlemlenebilirlik ve ölçüm

**Süre:** 3-5 gün
**Neden:** CLAUDE.md "P95 <200ms" diyor ama bu şu an ölçülmemiş bir iddia.
Somut sayı iddiadan güçlüdür. Ayrıca tracing olmadan outbox worker'ının
davranışını göstermek zor.

**Yapılacaklar**

1. OpenTelemetry: HTTP isteği, usecase, repository ve SQL için span'lar.
   Outbox teslimatı, olayı üreten isteğin trace'ine bağlanmalı.
2. Prometheus metrikleri: endpoint başına RED (rate, errors, duration),
   `tenant_id` etiketli. Outbox için kuyruk derinliği ve teslimat gecikmesi.
3. zerolog zaten var, trace id korelasyonu ekle.
4. k6 senaryosu: tenant oluşturma, login, listeleme karışımı.
5. Sonuçları README'ye tablo olarak koy. Donanım ve senaryo koşulları yazılsın,
   yoksa sayı anlamsız olur.

**Bitti sayılır:** README'de P50/P95/P99 tablosu ve onu üreten k6 dosyası
repoda.

---

## Faz 5: Odaklama, dikey modülleri ayır

**Süre:** 3-4 gün
**Neden:** 137 kaynak dosyanın ilginç olanı yaklaşık 15 tanesi. Blog,
rezervasyon, ders, ödeme ve site modülleri hacim katıyor ama derinlik katmıyor.
Bir inceleyicinin dikkatini çekirdekten uzaklaştırıyor.

**Neden sonda:** Bu bir refactor ve `internal/app/app.go` ile
`internal/app/routes.go` içinde 111 referansa dokunuyor. Çekirdek güçlenmeden
yapılırsa geriye az şey kalır.

**Yapılacaklar**

1. Dikey modülleri `examples/verticals/` altına taşı.
2. Composition root'u ikiye ayır: çekirdek uygulama ve örnek uygulama.
   Bugün tek bir `app.go` her şeyi kuruyor.
3. README'de çekirdek ile örnek arasındaki sınırı net anlat.

**Bitti sayılır:** `go build ./...` yeşil, çekirdek uygulama dikey modüller
olmadan ayağa kalkıyor.

---

## Faz 6: Dil ve cila

**Süre:** 2-3 gün

**Sürekli kural:** Faz 1'den itibaren yeni yazılan her kod yorumu, commit
mesajı ve dokümantasyon İngilizce. Bu faz yalnızca mevcut 268 Türkçe yorum
satırını ve dokümanları çevirir, birikmiş borcu kapatır.

**Yapılacaklar**

1. README, SECURITY.md, CONTRIBUTING.md İngilizce.
2. Dışa açık her sembol için godoc yorumu.
3. Mimari diyagramı.
4. CHANGELOG ve SemVer etiketleme.
5. `interface{}` temizliği: DTO'lardaki 71 kullanımı somut tiplere çevir.

---

## Kapsam dışı

Bunlar bilinçli olarak yapılmayacak.

- **Kubernetes ve Helm.** İlanlarda geçiyor ama bir repoya Helm chart koymak
  kolay ve kopyalanabilir. Doğru yazılmış bir `SKIP LOCKED` worker'ı değil.
  Sinyal değeri düşük, önce Faz 2 bitsin.
- **Çok bölgeli dağıtım.** Ölçek problemi yokken çözmek YAGNI.
- **Admin arayüzü.** Bu bir backend portföyü. Frontend eklemek odağı böler.
- **Yeni dikey modül.** Zaten 12 tane var ve bunlar sorunun kendisi.

---

## Sıralama gerekçesi

Faz 1 önce çünkü en ucuz ve zaten yapılmış işi tamamlıyor. Bugün kapatılan altı
izolasyon hatasının hikayesi HTTP testleri olmadan yarım kalıyor.

Faz 2 ikinci çünkü en büyük boşluk orada ve en çok zaman alan iş o. Erken
başlamak gerekiyor.

Faz 5 sonlarda çünkü geri dönüşü zor bir refactor. Çekirdek güçlendikten sonra
yapılmalı.

Faz 6 en sonda çünkü çeviri her an yapılabilir ve teknik derinliği geciktirmesi
anlamsız. Ama yeni kodun İngilizce yazılması Faz 1'de başlar, yoksa borç büyür.
