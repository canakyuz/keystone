package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TenantContext middleware extracts tenant ID and sets it in context
// For development: uses X-Tenant-ID header or default tenant
// For production: should extract from JWT token
func TenantContext(defaultTenantID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get tenant ID from header (development mode)
		tenantIDStr := c.Get("X-Tenant-ID", defaultTenantID)

		// TODO: In production, extract from JWT claims instead:
		// claims := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)
		// tenantIDStr = claims["tenant_id"].(string)

		// Validate UUID
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid tenant ID",
			})
		}

		// Set tenant ID in context
		c.Locals("tenant_id", tenantID)

		return c.Next()
	}
}
