# Değişmez Kurallar

Bu belge sistemin verdiği sözleri listeler. Her kuralın karşısında onu
uygulayan kod ve doğrulayan test bulunur. Karşılığı olmayan kural bu listede
slogan olarak durmaz, "henüz yok" olarak işaretlenir.

Durum sütunu üç değer alır. **Uygulanıyor**: kod ve doğrulama var.
**Kısmi**: bir bölümü var, eksiği açıkça yazılı. **Yok**: henüz uygulanmadı.

Kararların gerekçeleri ve alternatifleri için
[decisions/](decisions/) altındaki karar kayıtlarına bakın.

---

## 1. Yetkilendirme doğrulanmış subject ve tenant üyeliğine dayanır

**Durum:** Kısmi

İstek hangi tenant adına yapılıyorsa, o bilgi yalnızca doğrulanmış JWT
claim'inden okunur. `X-Tenant-ID` başlığı ve `tenant_id` query parametresi
yalnızca `ENVIRONMENT=development` iken ve açık bir anahtarla kabul edilir.

- Kod: `internal/middleware/tenant_context.go`, `extractTenantID`
- Test: `internal/middleware/tenant_source_test.go`

Bu kural daha önce ihlal ediliyordu. `extractTenantID`, `c.Locals("user")`
anahtarını okuyordu; `AuthMiddleware` böyle bir anahtar hiç yazmıyor. JWT yolu
hiçbir zaman çalışmadı ve her istek sessizce başlığa düştü. Geçerli token
taşıyan herhangi bir kullanıcı başka tenant'ın verisini okuyabiliyordu.

**Eksik:** Üyelik (membership) tablosu henüz yok. Şu an tenant kimliği JWT'den
geliyor ama "bu subject bu tenant'ın üyesi mi" sorusu ayrı bir kayıttan
doğrulanmıyor. Aynı insanın bir tenant'ta yönetici, başkasında görüntüleyici
olması bu tablo eklenmeden temsil edilemez.

**Eksik:** `POST /api/v1/tenants` yalnızca kimlik doğrulamasıyla korunuyor.
Yeni tenant oluşturma platform seviyesinde bir yetki olmalı, ama böyle bir
yetki modeli henüz yok. Bu uç tenant üyeliği aramaz, arayamaz da: tenant
henüz mevcut değil.

---

## 2. Aynı idempotency anahtarı farklı içerikle yeniden kullanılamaz

**Durum:** Uygulanıyor

Aynı kapsam ve anahtarla gelen istek, gövdesi aynıysa mevcut operasyonu
döndürür. Gövdesi farklıysa çakışma hatası döner. Sessizce eski sonucu
döndürmek, istemciye göndermediği isteğin işlendiğini düşündürürdü.

Benzersizliği veritabanı zorlar. Önce okuyup sonra yazmak tek başına yarış
koşulunu engellemez; aynı anda gelen iki istek kontrolü birlikte geçebilir.

- Kod: `internal/repository/operation/postgres.go`, `Create`
- Şema: `migrations/031_create_operation_model.up.sql`, `idx_idempotency_scope_key`
- Testler: `TestCreate_SameKeySameBodyReplays`,
  `TestCreate_SameKeyDifferentBodyConflicts`,
  `TestCreate_ConcurrentSameKeyProducesOneOperation`

**Sınır:** Garanti süresizdir değildir. Anahtarın saklama süresi dolduktan
sonra aynı anahtar yeni bir işlem yaratır. Varsayılan 24 saat.

---

## 3. Bir iş için yalnızca güncel lease sahibi sonuç kaydedebilir

**Durum:** Uygulanıyor

Her sahiplenmede `fence` bir artar. Sonuç bildirimi güncel fence değeriyle
yapılmak zorundadır. Lease süresi dolup iş devredildikten sonra geri dönen
eski worker'ın bildirimi reddedilir.

Yalnızca `lease_owner` kontrolü yetmez: eski worker kendi adını bilir ve o
kontrolü geçerdi.

- Kod: `internal/repository/operation/claim.go`, `completeInTx`
- Testler: `TestComplete_StaleWorkerIsRejected`, `TestRenewLease_RejectsStaleFence`

**Sınır:** Fencing, metadata güncellemesini korur. Eski worker'ın harici bir
sistemde yan etki üretmesini kendiliğinden engellemez. Kurulum adımlarının
güvenle tekrarlanabilir olması ayrıca gerekir.

---

## 4. Kurulum tamamlanmadan tenant `active` olamaz

**Durum:** Uygulanıyor

Tenant kaydı `pending` durumunda yazılır. `active` olması worker'ın kurulum
adımlarını tamamlamasına bağlıdır. Şeması hazır olmayan bir tenant istek
kabul edemez.

Tenant aktifleştirme ile operasyon kapatma aynı transaction içindedir. Ayrı
yazılsaydı, aralarında süreç kapandığında kullanıcıya "tamamlandı" görünen
ama tenant'ı kullanılamaz bir kayıt kalabilirdi.

Tenant'ı `active` yapmak worker handler'ının işi değildir; aktifleştirme işin
tamamlandı işaretlenmesiyle aynı transaction içindedir.

Worker'ın yaptığı ara durum yazımları (`provisioning`, `failed`) fencing
korumasının dışındadır. Bu yüzden geçişler kaynak duruma koşulludur: aktif bir
tenant geri `provisioning` durumuna çekilemez. Lease'ini kaybetmiş eski bir
worker'ın gecikmiş çağrısı sessizce etkisiz kalır.

- Kod: `internal/repository/operation/claim.go`, `CompleteSuccess`
- Kod: `internal/repository/operation/provision.go`, `insertPendingTenant`
- Kod: `internal/repository/tenant/lifecycle.go`, korumalı geçişler
- Kod: `internal/worker/provision_handler.go`, `Handle`
- Şema: `migrations/032_extend_tenant_lifecycle_states.up.sql`
- Test: `TestCompleteSuccess_ActivatesTenantInSameTransaction`

---

## 5. Tenant bağlamı eksikse tenant verisine erişim reddedilir

**Durum:** Uygulanıyor

Row Level Security policy'leri fail-closed'dır. Tenant bağlamı ayarlanmamışsa
sonuç boş kümedir: ne hata ne de tüm satırlar.

- Şema: `migrations/030_harden_tenant_isolation_policies.up.sql`
- Şema: `migrations/027_force_row_level_security.up.sql`
- Kod: `internal/repository/user/postgres.go`, context'ten şema okuma
- Testler: `test/security/rls_test.go`

Bu kural daha önce üç ayrı şekilde ihlal ediliyordu. `users` tablosunda
`USING (TRUE)` policy'si okuma izolasyonunu tamamen kaldırıyordu. On yedi
tabloda `FORCE` eksikti, dolayısıyla tablo sahibi rolü policy'lerden muaftı.
`sites` policy'si fail-open'dı: bağlam yokken tüm satırlar görünüyordu.

**İşletim koşulu:** Uygulama süper kullanıcı olmayan bir rolle bağlanmalıdır.
Süper kullanıcılar RLS'i her koşulda atlar.

---

## 6. Bir tenant'ın yükü bütün worker kapasitesini tüketemez

**Durum:** Uygulanıyor

Worker hem toplamda hem tenant başına eşzamanlılık sınırı uygular. Yalnızca
toplam sınır olsaydı, çok işi olan bir tenant bütün yuvaları doldurup
diğerlerini bekletebilirdi.

- Kod: `internal/worker/provisioner.go`, `reserveTenant`
- Testler: `TestRun_RespectsMaxConcurrent`, `TestRun_RespectsPerTenantLimit`

**Sınır:** Şu an basit bir tenant başına sayaç var. Tek bir tenant'ın kuyruğu
doldurması küçük tenant'ları geciktiriyorsa adil zamanlama gerekir; henüz yok.

---

## 7. Başarılı iş durumu ile audit kaydı aynı transaction içinde tutulur

**Durum:** Yok

Audit tablosu henüz oluşturulmadı. Hangi değişikliği kimin yaptığı şu an
yalnızca `operations.created_by` alanında tutuluyor, ayrı bir denetim izi yok.

---

## 8. Harici webhook başarısızlığı tamamlanmış kurulumu geri almaz

**Durum:** Yok

Webhook teslimatı henüz uygulanmadı. Bu kural, teslimat eklendiğinde
transactional outbox ile karşılanacak: olay iş verisiyle aynı transaction'da
yazılır, teslimat ayrı ve yeniden denenebilir bir adım olur.

---

## Kapanma davranışı

Bir kural değil ama aynı sınıfta bir söz: worker kapanırken önce yeni iş
almayı bırakır, çalışan işlere sınırlı süre verir, süre dolunca iptal eder.
Tamamlanamayan işler lease süresi dolunca başka bir worker tarafından
devralınır.

Context iptali, daha önce commit edilmiş yan etkileri geri almaz. İptal
"durmayı dene" demektir, "yapılanı sil" demez.

- Kod: `internal/worker/provisioner.go`, `drain`
- Testler: `TestRun_GracefulShutdownWaitsForRunningJobs`,
  `TestRun_ShutdownGraceExpiryCancelsJobs`

---

## Verilmeyen sözler

Bunlar bilinçli olarak garanti edilmez.

**"En fazla bir kez çalışır" garantisi yoktur.** İş yeniden çalışabilir. Lease
süresi dolan bir iş başka bir worker'a geçer ve adımlar baştan çalışır.
Handler'ların güvenle tekrarlanabilir olması gerekir.

**Süreçler arası önbellek tutarlılığı anlıktır değildir.** L1 katmanı süreç
içidir. Bir kaydı geçersiz kılmak yalnızca o süreçte anlık etki eder; diğer
replikalar kendi L1 TTL'leri dolana kadar eski değeri görebilir.

**Rate limit Redis düştüğünde uygulanmaz.** Varsayılan davranış isteği
geçirmektir. Limitleyici bir kullanılabilirlik aracı değil, kötüye kullanım
frenidir.
