# Güvenlik

## Açık bildirimi

Bir güvenlik açığı bulursanız lütfen public issue açmayın.
İletişim: canakyuz@wesan.co

## Tenant izolasyonu nasıl uygulanıyor

Keystone iki katmanlı izolasyon kullanır.

1. **Schema-per-tenant.** Her tenant'ın kendi PostgreSQL şeması vardır.
   `TenantConnectionManager` havuzdan tek bir bağlantı ayırır, o bağlantıda
   `search_path`'i tenant şemasına alır ve callback'e **aynı bağlantıyı** verir.
2. **Row Level Security.** Paylaşılan `public` şemasındaki tenant'a ait tablolar
   RLS policy'leri ile korunur. Policy'ler `app.current_tenant` oturum
   değişkenini okur.

## İşletim koşulları

Bu koşullar sağlanmazsa izolasyon garanti edilmez.

**Uygulama bağlantısı süper kullanıcı olmamalıdır.** PostgreSQL'de süper
kullanıcılar RLS policy'lerini her koşulda atlar. `FORCE ROW LEVEL SECURITY`
bunu değiştirmez. Uygulama için ayrı, süper kullanıcı olmayan bir rol açın.

**`ENVIRONMENT` production olmalıdır.** `development` iken `X-Tenant-ID`
header'ı ve `tenant_id` query parametresi ile tenant seçilebilir. Bu yalnızca
yerel geliştirme kolaylığıdır. Üretimde açık kalırsa kimlik doğrulamasından
geçmiş herhangi bir kullanıcı başka tenant'ın verisine erişir.

**Tenant context'e giren her sorgu callback'in verdiği bağlantıyı kullanmalıdır.**
`ExecuteInTenantContext` içinde havuz (`*sql.DB`) üzerinden sorgu çalıştırmak
sessizce başka bir bağlantıya düşer ve o bağlantının `search_path`'i bu tenant'a
ayarlı değildir.

## Bilinen ve bilinçli tasarım kararları

Aşağıdakiler açık değil, kasıtlı davranıştır.

- `websites.public_websites_policy` yayınlanmış siteleri tenant sınırından
  bağımsız okunabilir kılar. Public CMS içeriği için amaçlanan davranıştır.
- `projects.public_projects_policy` aynı şekilde tamamlanmış ve öne çıkarılmış
  projeleri açar.
- `users.auth_lookup_policy`, tenant bilinmeden yapılan login aramasına izin
  verir. Yalnızca `app.auth_lookup` oturum bayrağı açıkken uygulanır ve
  repository bu bayrağı `SET LOCAL` ile tek bir transaction'a hapseder.

## Geçmişte kapatılan açıklar

Bu repo, izolasyon iddiasını doğrulayan testler yazıldığında ortaya çıkan
hataların kaydını tutar. Ayrıntılar ilgili migration dosyalarının başındaki
yorumlardadır.

| Sorun | Etki | Düzeltme |
|---|---|---|
| `users` üzerinde `USING (TRUE)` policy'si | Okuma izolasyonu tamamen yoktu; her tenant tüm kullanıcıları, parola özetleri dahil okuyabiliyordu | `028` |
| 17 tabloda `ENABLE`, `FORCE` yok | Tablo sahibi rolüyle bağlanan uygulama policy'lerden muaftı | `027` |
| `sites` policy'si `COALESCE` ile fail-open | Context ayarlanmadığında tüm satırlar görünüyordu | `030` |
| `payments`, `refunds`, `payment_events` policy'lerinde argümansız `current_setting` | Ayarsız oturumda sert hata | `030` |
| `ExecuteInTenantContext` callback'e bağlantıyı vermiyordu | `search_path` sorguların koştuğu bağlantıya uygulanmıyordu | `pkg/database` |
| `extractTenantID` yanlış context anahtarını okuyordu | JWT tenant claim'i hiç kullanılmıyordu; geçerli token taşıyan herkes `X-Tenant-ID` header'ı ile başka tenant'a geçebiliyordu | `internal/middleware` |

## Test etme

İzolasyon iddiaları `test/security/` altında, süper kullanıcı olmayan bir rolle
ve gerçek PostgreSQL üzerinde doğrulanır.

```
go test ./test/security/ -v
```
