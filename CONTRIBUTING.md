# Katkı rehberi

## Geliştirme ortamı

Gerekenler: Go 1.24+, PostgreSQL 16+, Docker (opsiyonel).

```
git clone https://github.com/canakyuz/keystone.git
cd keystone
cp .env.example .env
go build ./...
```

## Test

Testler gerçek bir PostgreSQL'e karşı koşar. Mock veritabanı kullanılmaz;
bu repodaki hataların çoğu yalnızca gerçek bir veritabanında görünür.

```
go test ./...
```

Postgres erişilemiyorsa veritabanı gerektiren testler atlanır, hata vermez.
Bağlantı ayarları ortam değişkenleriyle ezilebilir: `TEST_DB_HOST`,
`TEST_DB_PORT`, `TEST_DB_USER`, `TEST_DB_PASSWORD`, `TEST_DB_NAME`.

Her test kendi izole veritabanını oluşturur ve sonunda düşürür, dolayısıyla
paralel koşu güvenlidir.

## Şema değişiklikleri

Şemanın tek doğruluk kaynağı `migrations/` dizinidir. Test helper'ı bu
dosyaları doğrudan çalıştırır. Şemayı başka bir yere kopyalamayın; bu repo
daha önce tam olarak bu yüzden fark edilmeyen bir kaymaya düştü.

Migration'lar geriye dönük uyumlu olmalıdır. Bir kolonu kaldırmadan önce
yazımını durdurun, sonra ayrı bir sürümde düşürün.

## RLS policy yazarken

Policy'ler fail-closed olmalıdır. Tenant context ayarlanmamışsa sonuç boş küme
olmalı, ne hata ne de tüm satırlar.

```sql
-- dogru
USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)

-- yanlis: context yoksa TRUE'ya duser
USING (tenant_id = COALESCE(NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID, tenant_id))
```

Permissive policy'ler OR ile birleşir. Bir tabloya `USING (TRUE)` eklemek, o
tablodaki diğer tüm izolasyon policy'lerini etkisiz kılar.

## Commit formatı

Conventional commits, tek satır.

```
feat(tenant): add schema provisioning
fix(rls): force row level security on tenant tables
```

## Gönderim öncesi

```
gofmt -l .
go vet ./...
go test ./...
```
