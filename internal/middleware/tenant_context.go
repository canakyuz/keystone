package middleware

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/logger"

	"github.com/canakyuz/keystone/pkg/tenantctx"
	"sync/atomic"
)

// Context anahtarları pkg/tenantctx'e taşındı. Repository katmanının HTTP
// middleware paketini import etmesi gerekmesin diye; bkz. pkg/tenantctx.
// Buradaki adlar geriye dönük uyumluluk için korunuyor.
type TenantContextKey = tenantctx.Key

const (
	// TenantSchemaKey, context'te schema_name'i sakladığımız key
	TenantSchemaKey = tenantctx.SchemaKey
	// TenantIDKey, context'te tenant_id'yi sakladığımız key
	TenantIDKey = tenantctx.IDKey
)

// TenantContextMiddleware, her request için tenant bilgilerini context'e ekler.
//
// Çalışma Akışı:
// 1. Request'ten tenant_id'yi çıkar (JWT claims veya X-Tenant-ID header'ından)
// 2. tenant_id ile cache-aware schema_name lookup yap (Redis + DB fallback)
// 3. Schema bilgisini hem Fiber context'ine hem de Go context'ine kaydet
// 4. Handler'ların bu bilgiyi kullanmasına izin ver
//
// UYARI: Bu middleware, authentication middleware'inden SONRA çalışmalıdır.
//
// 🎓 PERFORMANS: Cache kullanımıyla ~10-20ms latency kazancı (DB query bypass)
func TenantContextMiddleware(schemaCache *TenantSchemaCache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Adım 1: Tenant ID'yi request'ten al
		tenantID := extractTenantID(c)
		if tenantID == "" {
			if schemaCache.logger != nil {
				schemaCache.logger.Warn("Request'te tenant_id bulunamadı")
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "tenant_id gerekli",
			})
		}

		// Adım 2: Tenant ID ile schema_name lookup yap (Cache-aware)
		// 🎓 PERFORMANS: Cache hit -> ~1ms, Cache miss -> ~6-21ms
		schemaName, err := schemaCache.GetTenantSchema(c.Context(), tenantID)
		if err != nil {
			if schemaCache.logger != nil {
				schemaCache.logger.WithFields(logger.Fields{
					"tenant_id": tenantID,
					"error":     err.Error(),
				}).Error("Tenant schema bulunamadı")
			}

			// Tenant bulunamadıysa 404 dön
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "tenant bulunamadı",
				})
			}

			// Diğer hatalar 500
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "tenant bilgisi alınamadı",
			})
		}

		// Adım 3: Tenant bilgilerini context'e kaydet
		//
		// c.Locals(): Fiber'in kendi context sistemi (request lifecycle boyunca)
		c.Locals("tenant_id", tenantID)
		c.Locals("tenant_schema", schemaName)

		// Go context'ine de ekle (repository layer'da kullanmak için)
		ctx := context.WithValue(c.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, TenantSchemaKey, schemaName)
		c.SetUserContext(ctx)

		if schemaCache.logger != nil {
			schemaCache.logger.WithFields(logger.Fields{
				"tenant_id":   tenantID,
				"schema_name": schemaName,
			}).Debug("Tenant context ayarlandı")
		}

		// Sonraki middleware/handler'a geç
		return c.Next()
	}
}

// extractTenantID, request'ten tenant_id bilgisini çıkarır.
//
// Öncelik sırası:
// 1. JWT claims'den (önerilen - güvenli)
// 2. X-Tenant-ID header'ından (development/test)
// 3. Query parameter'dan (ÖNERILMEZ)
// allowUntrustedTenantHeader, X-Tenant-ID header'i ve tenant_id query
// parametresiyle tenant secilmesine izin verilip verilmedigini tutar.
//
// Varsayilan false'tur ve yalnizca AllowUntrustedTenantSource ile acilir.
// Uretimde ASLA acilmamalidir.
var allowUntrustedTenantHeader atomic.Bool

// AllowUntrustedTenantSource, yalnizca yerel gelistirme ve testler icin
// header/query uzerinden tenant secmeyi acar.
//
// Bu bir kolaylik degil, bilincli bir guvenlik anahtaridir: acikken kimlik
// dogrulamasindan gecmis herhangi bir kullanici, baska bir tenant'in kimligini
// header'a yazarak o tenant'in verisine erisebilir.
func AllowUntrustedTenantSource(allow bool) {
	allowUntrustedTenantHeader.Store(allow)
}

// extractTenantID, request'in hangi tenant adina yapildigini belirler.
//
// GUVENLIK: Tek guvenilir kaynak, AuthMiddleware'in dogrulanmis JWT'den
// yazdigi c.Locals("tenant_id") degeridir.
//
// Onceki hali c.Locals("user") anahtarini okuyup map[string]interface{}'e
// cevirmeye calisiyordu. AuthMiddleware boyle bir anahtar hic yazmiyor; claim'i
// dogrudan c.Locals("tenant_id") olarak koyuyor. Dolayisiyla JWT yolu hicbir
// zaman calismiyor ve her istek sessizce X-Tenant-ID header'ina dusuyordu.
// Sonuc: gecerli bir token tasiyan herhangi bir kullanici, header'i degistirerek
// istedigi tenant'in verisini okuyabiliyordu.
func extractTenantID(c *fiber.Ctx) string {
	// 1) Dogrulanmis JWT claim'i. Tek guvenilir kaynak.
	if tenantID, ok := c.Locals("tenant_id").(string); ok && tenantID != "" {
		return tenantID
	}

	// 2) Header ve query yalnizca acikca izin verildiginde okunur.
	if !allowUntrustedTenantHeader.Load() {
		return ""
	}

	if headerTenantID := c.Get("X-Tenant-ID"); headerTenantID != "" {
		return headerTenantID
	}

	return c.Query("tenant_id")
}

// GetTenantSchemaFromContext, Go context'inden tenant schema'sını alır.
//
// Kullanım (Repository layer):
//
//	schema := middleware.GetTenantSchemaFromContext(ctx)
func GetTenantSchemaFromContext(ctx context.Context) string {
	return tenantctx.Schema(ctx)
}

// GetTenantIDFromContext, Go context'inden tenant ID'yi alır.
func GetTenantIDFromContext(ctx context.Context) string {
	return tenantctx.ID(ctx)
}
