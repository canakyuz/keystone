package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/logger"

	"errors"
	"github.com/canakyuz/keystone/pkg/tenantctx"
	"sync/atomic"
)

// The context keys moved to pkg/tenantctx so that the repository layer does not
// have to import the HTTP middleware package; see pkg/tenantctx. The names here
// are kept for backward compatibility.
type TenantContextKey = tenantctx.Key

const (
	// TenantSchemaKey is the context key holding schema_name.
	TenantSchemaKey = tenantctx.SchemaKey
	// TenantIDKey is the context key holding tenant_id.
	TenantIDKey = tenantctx.IDKey
)

// TenantContextMiddleware puts the tenant information into the context on every
// request.
//
// Flow:
//  1. Extract tenant_id from the request (JWT claims, or the X-Tenant-ID header).
//  2. Resolve schema_name from tenant_id through the cache (Redis, DB fallback).
//  3. Store the schema in both the Fiber context and the Go context.
//  4. Let the handlers use it.
//
// WARNING: this middleware must run AFTER the authentication middleware.
func TenantContextMiddleware(schemaCache *TenantSchemaCache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Step 1: take the tenant id from the request.
		tenantID := extractTenantID(c)
		if tenantID == "" {
			if schemaCache.logger != nil {
				schemaCache.logger.Warn("no tenant_id on request")
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "tenant_id is required",
			})
		}

		// Step 2: resolve schema_name from the tenant id, through the cache.
		schemaName, err := schemaCache.GetTenantSchema(c.Context(), tenantID)
		if err != nil {
			if schemaCache.logger != nil {
				schemaCache.logger.WithFields(logger.Fields{
					"tenant_id": tenantID,
					"error":     err.Error(),
				}).Error("tenant schema not found")
			}

			// Return 404 when the tenant does not exist.
			// TenantSchemaCache converts sql.ErrNoRows into ErrTenantNotFound: a missing
			// tenant is negatively cached, so a second request never reaches the database
			// and the error no longer comes from the sql layer.
			if errors.Is(err, ErrTenantNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "tenant not found",
				})
			}

			// Anything else is a 500.
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "could not resolve tenant",
			})
		}

		// Step 3: store the tenant information in the context.
		//
		// c.Locals() is Fiber's own per-request context.
		c.Locals("tenant_id", tenantID)
		c.Locals("tenant_schema", schemaName)

		// Also put it on the Go context, for the repository layer.
		ctx := context.WithValue(c.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, TenantSchemaKey, schemaName)
		c.SetUserContext(ctx)

		if schemaCache.logger != nil {
			schemaCache.logger.WithFields(logger.Fields{
				"tenant_id":   tenantID,
				"schema_name": schemaName,
			}).Debug("tenant context set")
		}

		// Hand off to the next middleware or handler.
		return c.Next()
	}
}

// extractTenantID pulls the tenant id out of the request.
//
// Order of precedence:
//  1. The JWT claim (the only trusted source).
//  2. The X-Tenant-ID header (development and test only).
//  3. The query parameter (development and test only).
// allowUntrustedTenantHeader records whether selecting the tenant through the
// X-Tenant-ID header or the tenant_id query parameter is permitted.
//
// It defaults to false and is only opened by AllowUntrustedTenantSource.
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

// GetTenantSchemaFromContext reads the tenant schema from the Go context.
//
// Usage, from the repository layer:
//
//	schema := middleware.GetTenantSchemaFromContext(ctx)
func GetTenantSchemaFromContext(ctx context.Context) string {
	return tenantctx.Schema(ctx)
}

// GetTenantIDFromContext reads the tenant id from the Go context.
func GetTenantIDFromContext(ctx context.Context) string {
	return tenantctx.ID(ctx)
}
