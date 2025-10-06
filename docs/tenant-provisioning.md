# Tenant Provisioning & Isolation Architecture

Bu rehber, NexSpaces ekosisteminde tenant verisinin nasıl oluşturulduğunu, hangi panellerin hangi modülleri kullandığını ve izolasyon stratejisinin modül temelli olarak nasıl uygulanması gerektiğini açıklar. Doküman, hem `nexpaces-api` (backend) hem `nexpaces-web` (frontend) kod tabanındaki mevcut durumu referans alır ve geliştirme planı önerir.

---

## 1. Platform Panelleri ve Veri Akışı

`nexpaces-web/README.md` üç ana paneli tanımlar:

| Panel | Dosya yolu | Açıklama | Veri Kaynağı |
|-------|------------|---------|--------------|
| **Marketing** | `nexpaces-web/src/app/(client)` | Kayıt öncesi satış, landing, fiyatlandırma sayfaları. Tenant gerektirmez. | Statik içerik, registry kataloğu (`modules`, `tools`). |
| **Platform Admin** | `nexpaces-web/src/app/(platform)` | Global yönetim paneli (tenant provisioning, modül yönetimi, sistem sağlığı). | `tenants`, `tenant_modules`, `tenant_tools`, log/metrics tabloları. |
| **Tenant Workspace** | `nexpaces-web/src/app/[tenant]` | Müşteri panelleri (`*.nexpaces.com`). Aktif modüllere göre CRM, LMS, ERP vb. ekranlar açılır. | Tenant izole verileri (schema veya ayrı DB). |

Bu yapı, backend tarafında üç veri katmanı gerektirir:
1. **Global katalog** – Modül ve tool tanımları (`modules`, `tools`, `module_dependencies`, `tool_dependencies`).
2. **Tenant meta verisi** – Tenant kaydı, plan, şema bilgisi (`tenants`, `tenant_modules`, `tenant_tools`).
3. **Tenant domain verisi** – LMS, CRM, ERP vb. domain tabloları (tenant izolasyonu gerekir).

---

## 2. Mevcut Backend Bileşenleri

| Bileşen | Dosya | Açıklama |
|---------|-------|----------|
| Tenant domain modeli | `internal/domain/tenant/entity.go` | `Tenant` yapısı, plan enumları, `SetSchemaName`, validasyon. |
| Tenant hizmeti | `internal/usecase/tenant/service.go:32-97` | `Create` metodu slug/e-posta benzersizliği, schema adı üretimi, kayıt ve provisioning çağrısı. |
| Provisioning servisi | `internal/usecase/tenant/provisioning.go` | `GenerateSchemaName`, `ProvisionTenantSchema` (şema oluştur, template uygula, `search_path` ayarla). |
| Tenant API uç noktası | `internal/handler/tenant/handler.go:19-50` | `POST /api/v1/tenants` endpoint’i tenant yaratır. |
| Registry şeması | `migrations/017-022_*` + `docs/REGISTRY_SCHEMA_SUMMARY.md` | Modül & tool katalogları. |
| Tenant -> schema mapping | `migrations/025_add_tenant_schema_support.up.sql` | `schema_name` kolonu ve `generate_schema_name(base text)` fonksiyonu. |
| Development seed | `scripts/seed/dev_seed.sql` | Local kurucu tenant ve panel test kullanıcıları. `Makefile`’de `seed-dev` hedefi ile çalışır. |

> Not: Migration 024 (test tenant seed) kaldırıldı; production migration zinciri artık veri eklemiyor.

---

## 3. Panel/Modül Bazlı İzolasyon Haritası

`docs/MODULE_INVENTORY.md` ve `docs/REGISTRY_SCHEMA_SUMMARY.md` modülleri listeler. Bu modüllerin panel kullanımına göre izolasyon seviyesi şöyle belirlenebilir:

| Panel / Modül Grubu | Temel Tablolar | Önerilen İzolasyon |
|---------------------|----------------|--------------------|
| **CMS / Landing / Blog** (Marketing site içerikleri) | `blog_posts`, `blog_categories`, `websites` (yalnızca public içerik) | Shared schema (`public`). Tenant verisi `tenant_id` ile ayrılır. |
| **CRM / Projects / LMS / Booking / Service Catalog / OMS** (Tenant workspace çekirdeği) | `lessons`, `students`, `assignments`, `projects`, `appointments`, `services`… | Schema-per-tenant (`CREATE SCHEMA tenant_<slug>`). Bu tablolar RLS + `search_path` ile izole edilir. |
| **ERP (Finans, HR), Payments, Analytics** (yüksek regülasyon) | `payments`, `refunds`, `payment_events`, gelecekte ERP tabloları | Database-per-tenant. Ayrı PostgreSQL veritabanı açıp migration çalıştırmak gerekir. |
| **Platform Admin Meta** | `tenants`, `tenant_modules`, `tenant_tools`, registry tabloları | Global veritabanı; shared schema. |

Bu eşleştirme kod seviyesinde konfigüre edilmelidir. Örneğin `configs/tenant_isolation.yaml` gibi bir dosyada modül kodu → izolasyon seviyesi haritalanabilir.

---

## 4. Tenant Provisioning Akışı (Detay)

1. **Request** – Platform admin panelinden `POST /api/v1/tenants` çalışır (`internal/handler/tenant/handler.go:19-50`). Payload’a modül listesi eklenmesi planlanıyor.
2. **Validasyon** – `tenant.Service.Create` slug ve e-posta benzersizliğini kontrol eder (`service.go:38-56`).
3. **Schema Adı** – `ProvisioningService.GenerateSchemaName` `SELECT generate_schema_name($1)` çalıştırır (`provisioning.go:21-37`).
4. **Tenant Kaydı** – Repository `INSERT INTO tenants ... schema_name` yazar (`internal/repository/tenant/postgres.go:24-52`).
5. **İzolasyon Seçimi** – `ProvisionTenantSchema` şemayı oluşturur (`provisioning.go:49-109`). İzolasyon haritasına göre:
   - Shared → `schema_name` olarak `public` tutulur, `CREATE SCHEMA` atlanır.
   - Schema → `CREATE SCHEMA IF NOT EXISTS tenant_<slug>`; template SQL uygulanır (`templates.GetTemplateByPlan`).
   - Database → (Gelecek geliştirme) `CREATE DATABASE tenant_<slug>` ve migration tetikleme.
6. **Modül Aktivasyonu** – Tenant’ın seçtiği modüller `tenant_modules` tablosuna yazılır. Dev seed scriptindeki örnek LMS aktivasyonu referans alınabilir (`scripts/seed/dev_seed.sql` bölümü).
7. **Loglama** – Başarılı provisioning `tenant_id`, `schema_name`, `plan` bilgileri ile loglanır (`provisioning.go:97-108`).
8. **Owner Kullanıcı** – Şu an API owner kayıtlarını `POST /api/v1/auth/register` ile beklentiyor; request’te `tenant_id` zorunlu (`internal/usecase/user/dto.go:12-18`). Gelecek adımda provisioning yanıtı onboarding token döndürerek owner kaydıyla entegre edilecek.

---

## 5. Development Seed ve Test

- `make db-reset` → tüm migration’lar (`scripts/run_migrations.sh`).
- `make seed-dev` → `scripts/seed/dev_seed.sql` dosyasını çalıştırır ve aşağıdakileri oluşturur:
  - Kurucu tenant (`canakyuz`) + owner (`DevPass123!`).
  - `canakyuz-dev` tenantı + sektör bazlı kullanıcılar (`DevPass123!`).
  - LMS modülü aktivasyonu ve örnek öğrenci (`students` tablosu).
  - Her tenant için `schema_name` alınır ve `CREATE SCHEMA IF NOT EXISTS <schema_name>` çalıştırılır.

> Production veya staging pipeline’ında `seed-dev` **çalıştırılmamalıdır**; yalnızca local/dev ortamı içindir.

Test önerileri:
- `make seed-dev` sonrasında `make db-shell` ile `SELECT email, role, tenant_id FROM users ORDER BY tenant_id, email;` çalıştırarak kullanıcı setini doğrulayın.
- `scripts/seed/dev_seed.sql` tarafından açılan hesaplarla `/auth/login` endpointini smoke test edin.

---

## 6. Enterprise (Database-per-Tenant) Planı

1. **Provision Tenant Database**
   - Yönetici bağlantısından `CREATE DATABASE tenant_<slug>` komutu çalıştırın.
   - Uygulama rolüne `GRANT ALL ON DATABASE` verin.
   - `scripts/run_migrations.sh` benzeri bir komutla yeni DB’de migration’ları uygulayın (parametrik hale getirmek gerekiyor).
2. **TenantConnectionManager**
   - README’de referans verilen bileşen; gelen request’te `tenant.Plan` ve `tenant.SchemaName` bilgisine bakarak uygun DSN ve `search_path` ayarlamalı.
   - Schema-per-tenant’ta `SET LOCAL search_path TO tenant_<slug>, public`; database-per-tenant’ta connection pool başka DSN’e yönlenir.
3. **Plan Yükseltme**
   - `tenant.Service.UpgradePlan` ( `service.go:204-220` ) schema → database geçişinde veri taşıma işlemlerini tetikleyecek şekilde genişletilmeli.
   - Taşımada `pg_dump`/`pg_restore` veya ETL kullanın; downtime ve rollback planı yapın.
4. **Observability & Backup**
   - Health endpointleri her tenant için plan, schema/db adı, son migration zamanı gibi bilgileri döndürmeli.
   - Yedekleme stratejisi: schema-per-tenant için `--schema=tenant_<slug>`, database-per-tenant için `pg_dump tenant_<slug>`.

---

## 7. Yapılacaklar & Önerilen Adımlar

1. `CreateTenantRequest` payload’una modül listesi ekleyin & izolasyon haritasını konfigurasyona taşıyın.
2. `ProvisioningService` içine database-per-tenant senaryosu için `ProvisionTenantDatabase` ve `MigrateSchemaToDatabase` fonksiyonlarını ekleyin.
3. `TenantConnectionManager`’ı implement edip middleware katmanında her request’te doğru `search_path` / bağlantı seçimini yapın.
4. Platform admin panelinde ( `nexpaces-web/src/app/(platform)` ) tenant detayı ekranına plan + izolasyon bilgilerini ekleyin.
5. Seed scriptini enterprise denemeleri için genişletin (mock ayrı DB oluşturma).
6. Dokümantasyon (README, API docs) ve OpenAPI şemalarını izolasyon ve onboarding süreçlerini yansıtacak şekilde güncelleyin.

Bu döküman, NexSpaces kapsamındaki panel tipleri ve modül envanteri ile uyumlu olacak şekilde tenant provisioning sürecinin nasıl yönetileceğini ortaya koyar. Kod değişiklikleri yapıldıkça `docs/tenant-provisioning.md` dosyası da güncellenmelidir.
