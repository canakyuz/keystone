package middleware

import (
	"context"
	"errors"
	"fmt"

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

		// Only not-found is the caller's problem. Anything else is returned as a plain error,
		// which the application's error handler logs and answers with a generic 500. It used
		// to be wrapped in a 403 carrying err.Error(), so a database failure reached the
		// client as "failed to get tenant: pq: ..." and read as a permission refusal.
		t, err := repo.GetByID(c.Context(), tenantID)
		if errors.Is(err, tenant.ErrTenantNotFound) {
			return fiber.NewError(fiber.StatusForbidden, "tenant not found")
		}
		if err != nil {
			return fmt.Errorf("tenant scope: could not load tenant: %w", err)
		}

		if t.SchemaName == "" {
			return fiber.NewError(fiber.StatusPreconditionFailed, "tenant schema not provisioned")
		}

		if err := manager.SetTenantScope(c.Context(), t.SchemaName); err != nil {
			return fmt.Errorf("tenant scope: %w", err)
		}

		defer func() {
			_ = manager.ResetTenantScope(c.Context())
		}()

		return c.Next()
	}
}
