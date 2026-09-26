package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/internal/database"
	"github.com/canakyuz/keystone/internal/domain/tenant"
)

// TenantLookup reads the tenant whose schema a request is scoped to.
type TenantLookup interface {
	GetByID(ctx context.Context, id string) (*tenant.Tenant, error)
}

// TenantScope ensures the database session is scoped to the tenant schema before handling the request.
func TenantScope(repo TenantLookup, manager *database.TenantManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := GetTenantID(c)
		if tenantID == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "tenant context missing")
		}

		tenant, err := repo.GetByID(c.Context(), tenantID)
		if err != nil {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}

		if tenant.SchemaName == "" {
			return fiber.NewError(fiber.StatusPreconditionFailed, "tenant schema not provisioned")
		}

		if err := manager.SetTenantScope(c.Context(), tenant.SchemaName); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		defer func() {
			_ = manager.ResetTenantScope(c.Context())
		}()

		return c.Next()
	}
}
