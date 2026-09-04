// Package tenantctx, tenant bilgisinin Go context'i üzerinden taşınmasını tanımlar.
//
// NEDEN ayrı bir paket: bu anahtarlar daha önce internal/middleware içinde
// duruyordu. Repository katmanı onları okumak için HTTP middleware paketini
// import etmek zorunda kalıyordu; yani veri erişim katmanı taşıma katmanına
// bağımlıydı. Bu, clean architecture'ın bağımlılık yönünü tersine çevirir ve
// pratikte test helper'ları devreye girdiğinde import döngüsüne yol açtı.
//
// tenantctx bir yaprak pakettir: hiçbir proje paketini import etmez, dolayısıyla
// her katmandan güvenle kullanılabilir.
package tenantctx

import "context"

// Key, context anahtarları için kullanılan tiptir.
// Düz string yerine ayrı bir tip kullanılır ki başka paketlerin yazdığı
// anahtarlarla çakışma olmasın.
type Key string

const (
	// SchemaKey, tenant'ın PostgreSQL schema adını taşır.
	SchemaKey Key = "tenant_schema"
	// IDKey, tenant'ın kimliğini taşır.
	IDKey Key = "tenant_id"
)

// WithSchema, schema adını context'e yerleştirir.
func WithSchema(ctx context.Context, schemaName string) context.Context {
	return context.WithValue(ctx, SchemaKey, schemaName)
}

// WithID, tenant kimliğini context'e yerleştirir.
func WithID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, IDKey, tenantID)
}

// Schema, context'teki schema adını döner. Yoksa boş string.
func Schema(ctx context.Context) string {
	schema, _ := ctx.Value(SchemaKey).(string)
	return schema
}

// ID, context'teki tenant kimliğini döner. Yoksa boş string.
func ID(ctx context.Context) string {
	id, _ := ctx.Value(IDKey).(string)
	return id
}
