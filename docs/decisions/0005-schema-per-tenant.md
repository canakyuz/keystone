# 0005. Schema-per-tenant, RLS ile birlikte

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

Her müşterinin verisi diğerlerinden ayrı tutulmalı. Üç yaygın yaklaşım var ve
üçü de farklı şeyleri garanti ediyor.

## Korunması gereken kurallar

- Bir tenant başka bir tenant'ın verisini okuyamaz.
- Tenant bağlamı eksikse erişim reddedilir, açılmaz.
- Migration bütün tenant'lara uygulanabilir olmalıdır.

## Değerlendirilen yaklaşımlar

| Yaklaşım | Güçlü tarafı | Operasyon bedeli |
|---|---|---|
| Ortak tablo + `tenant_id` + RLS | Tek şemaya migration, ortak sorgulama kolay | Policy, rol ve sorgu doğruluğu kritik |
| Tenant başına şema | Mantıksal ayrım net | Şema sayısı ve migration yönetimi büyür |
| Tenant başına veritabanı | Ayrı yedekleme ve kaynak yönetimi | Bağlantı ve altyapı maliyeti artar |

## Karar

Tenant başına şema, artı paylaşılan tablolarda RLS.

Yönetim verisi (tenants, operations, provisioning_jobs) `public` şemasında
kalır ve RLS ile korunur. Tenant iş verisi kendi şemasına gider.

## Gerekçe

İkisi farklı problemleri çözüyor ve birbirinin yerine geçmiyor.

Şema ayrımı, tenant iş verisinin migration ve yedekleme birimini ayırıyor.
RLS ise paylaşılan yönetim tablolarında satır seviyesinde sınır çiziyor;
o tablolar şemaya bölünemez çünkü control plane hepsini birden sorgular.

## Bu kararın garanti ETMEDİĞİ şeyler

Bunlar önemli çünkü "izole" kelimesinin sınırını çiziyorlar.

**Kaynak izolasyonu yok.** Aynı PostgreSQL instance'ındaki ayrı şemalar CPU
ve I/O izolasyonu sağlamaz. Bir tenant'ın ağır sorgusu diğerlerini yavaşlatır.

**Şema ayrımı tek başına güvenlik sınırı değil.** Aynı uygulama rolü bütün
şemalara erişebiliyorsa, ayrım yalnızca bir isim alanı ayrımıdır. Güvenlik,
uygulamanın doğru şemaya yönelmesine bağlı kalır.

**RLS süper kullanıcıyı durdurmaz.** Uygulama süper kullanıcı olmayan bir
rolle bağlanmak zorunda. `FORCE ROW LEVEL SECURITY` tablo sahibini kapsar,
süper kullanıcıyı kapsamaz.

## Doğrulanması gereken noktalar

Bu yaklaşımın kanıtı, sırayla yapılan iki başarılı API çağrısı değil.

| Soru | Durum |
|---|---|
| İstemcinin verdiği tenant kimliği üyelik doğrulamasından geçiyor mu | Kısmi, membership tablosu yok |
| Tenant sorguları aynı bağlantı bağlamında mı çalışıyor | Evet, `ExecuteInTenantContext` bağlantıyı callback'e veriyor |
| `search_path` havuzda başka isteğe taşınabiliyor mu | Hayır, test ediliyor |
| Eksik tablo yüzünden `public` şemasına fallback oluyor mu | Test edilmedi |
| Şema adı güvenli üretiliyor mu | Evet, `pq.QuoteIdentifier` ve ad doğrulaması |
| Runtime rolü ile şema oluşturan rol ayrı mı | Hayır, henüz ayrılmadı |
| Migration bütün tenant'lara nasıl yayılıyor | Çözülmedi |

## Bu karar ne zaman yanlış hale gelir

- Tenant sayısı binleri geçerse. Şema başına migration maliyeti T×M adıma
  çıkar; eşzamanlılık artırmak toplam işi azaltmaz, yalnızca tamamlanma
  süresi ile veritabanı yükü arasında takas yapar.
- Bir müşteri kaynak izolasyonu talep ederse. O noktada tenant başına
  veritabanı veya ayrı instance gerekir.
- Tenant'lar arası raporlama gerekirse. Şemalar arası sorgu yazmak, ortak
  tablo yaklaşımına göre belirgin şekilde zahmetli.
