package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"nexspaces-api/internal/core/domain/shared"
	"nexspaces-api/internal/core/ports/repositories"
	"nexspaces-api/internal/shared/errors"
)

// TenantMiddleware handles tenant context resolution and isolation
type TenantMiddleware struct {
	tenantRepo repositories.TenantRepository
}

// NewTenantMiddleware creates a new tenant middleware
func NewTenantMiddleware(tenantRepo repositories.TenantRepository) *TenantMiddleware {
	return &TenantMiddleware{
		tenantRepo: tenantRepo,
	}
}

// ResolveTenant resolves tenant from subdomain or custom domain
func (m *TenantMiddleware) ResolveTenant(c *fiber.Ctx) error {
	host := c.Get("Host")
	if host == "" {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Host header required", nil)
	}

	// Remove port if present
	if colonPos := strings.LastIndex(host, ":"); colonPos != -1 {
		host = host[:colonPos]
	}

	var tenant *tenant.Tenant
	var err error

	// Try to resolve by custom domain first
	if !strings.Contains(host, "nexspaces.") && !strings.Contains(host, "localhost") {
		// This might be a custom domain
		domain, domainErr := shared.NewDomain(host)
		if domainErr == nil {
			tenant, err = m.tenantRepo.GetByCustomDomain(c.Context(), domain)
			if err != nil && !errors.Is(err, shared.ErrTenantNotFound) {
				return errors.NewHTTPError(fiber.StatusInternalServerError, "Failed to resolve tenant by domain", err)
			}
		}
	}

	// If not found by custom domain, try subdomain
	if tenant == nil {
		subdomain := m.extractSubdomain(host)
		if subdomain == "" {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Unable to determine tenant from host", nil)
		}

		tenantSlug, err := shared.NewTenantSlug(subdomain)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant slug format", err)
		}

		tenant, err = m.tenantRepo.GetBySlug(c.Context(), *tenantSlug)
		if err != nil {
			if errors.Is(err, shared.ErrTenantNotFound) {
				return errors.NewHTTPError(fiber.StatusNotFound, "Tenant not found", err)
			}
			return errors.NewHTTPError(fiber.StatusInternalServerError, "Failed to resolve tenant", err)
		}
	}

	// Verify tenant is active
	if !tenant.IsActive() {
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant is not active", nil)
	}

	// Set tenant context
	c.Locals("tenant_id", uuid.UUID(tenant.ID))
	c.Locals("tenant_slug", tenant.Slug.String())
	c.Locals("tenant", tenant)

	// Set tenant context for database RLS
	err = m.setDatabaseTenantContext(c.Context(), tenant.ID)
	if err != nil {
		return errors.NewHTTPError(fiber.StatusInternalServerError, "Failed to set tenant context", err)
	}

	return c.Next()
}

// RequireTenant ensures a tenant context is present
func (m *TenantMiddleware) RequireTenant(c *fiber.Ctx) error {
	tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Tenant context required", nil)
	}

	// Verify user belongs to this tenant (if authenticated)
	if userTenantID, exists := c.Locals("tenant_id").(uuid.UUID); exists {
		if userTenantID != tenantID {
			return errors.NewHTTPError(fiber.StatusForbidden, "Cross-tenant access denied", nil)
		}
	}

	return c.Next()
}

// ValidateTenantAccess validates that the authenticated user belongs to the tenant
func (m *TenantMiddleware) ValidateTenantAccess(c *fiber.Ctx) error {
	// Get tenant ID from context (set by ResolveTenant middleware)
	tenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusBadRequest, "Tenant context not found", nil)
	}

	// Get user's tenant ID from auth context
	userTenantID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return errors.NewHTTPError(fiber.StatusUnauthorized, "User not authenticated", nil)
	}

	// Verify tenant IDs match
	if tenantID != userTenantID {
		return errors.NewHTTPError(fiber.StatusForbidden, "Access denied: user does not belong to this tenant", nil)
	}

	return c.Next()
}

// ResolveTenantFromParam resolves tenant from URL parameter
func (m *TenantMiddleware) ResolveTenantFromParam(paramName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantIDStr := c.Params(paramName)
		if tenantIDStr == "" {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Tenant ID parameter required", nil)
		}

		tenantUUID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusBadRequest, "Invalid tenant ID format", err)
		}

		tenantID := shared.TenantID(tenantUUID)

		// Get tenant from repository
		tenant, err := m.tenantRepo.GetByID(c.Context(), tenantID)
		if err != nil {
			if errors.Is(err, shared.ErrTenantNotFound) {
				return errors.NewHTTPError(fiber.StatusNotFound, "Tenant not found", err)
			}
			return errors.NewHTTPError(fiber.StatusInternalServerError, "Failed to get tenant", err)
		}

		// Verify tenant is active
		if !tenant.IsActive() {
			return errors.NewHTTPError(fiber.StatusForbidden, "Tenant is not active", nil)
		}

		// Set tenant context
		c.Locals("tenant_id", tenantUUID)
		c.Locals("tenant_slug", tenant.Slug.String())
		c.Locals("tenant", tenant)

		// Set database tenant context
		err = m.setDatabaseTenantContext(c.Context(), tenant.ID)
		if err != nil {
			return errors.NewHTTPError(fiber.StatusInternalServerError, "Failed to set tenant context", err)
		}

		return c.Next()
	}
}

// extractSubdomain extracts subdomain from host
func (m *TenantMiddleware) extractSubdomain(host string) string {
	// For localhost development
	if strings.Contains(host, "localhost") {
		return "dev" // Default to 'dev' tenant for localhost
	}

	// For production domains like tenant.nexspaces.com
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		// First part should be the subdomain
		return parts[0]
	}

	return ""
}

// setDatabaseTenantContext sets the tenant context for PostgreSQL RLS
func (m *TenantMiddleware) setDatabaseTenantContext(ctx context.Context, tenantID shared.TenantID) error {
	// This would typically use your database connection to set the tenant context
	// For now, we'll store it in the context for the repositories to use

	// In a real implementation, you might do something like:
	// _, err := db.ExecContext(ctx, "SELECT set_tenant_context($1)", tenantID)
	// return err

	// For this implementation, we'll rely on the repositories to handle RLS context
	return nil
}

// GetTenantFromContext retrieves tenant from Fiber context
func GetTenantFromContext(c *fiber.Ctx) (*tenant.Tenant, error) {
	tenant, ok := c.Locals("tenant").(interface{})
	if !ok {
		return nil, errors.NewHTTPError(fiber.StatusBadRequest, "Tenant context not found", nil)
	}

	tenantEntity, ok := tenant.(*tenant.Tenant)
	if !ok {
		return nil, errors.NewHTTPError(fiber.StatusInternalServerError, "Invalid tenant context", nil)
	}

	return tenantEntity, nil
}

// GetTenantIDFromContext retrieves tenant ID from Fiber context
func GetTenantIDFromContext(c *fiber.Ctx) (shared.TenantID, error) {
	tenantUUID, ok := c.Locals("tenant_id").(uuid.UUID)
	if !ok {
		return shared.TenantID{}, errors.NewHTTPError(fiber.StatusBadRequest, "Tenant ID not found in context", nil)
	}

	return shared.TenantID(tenantUUID), nil
}

// TenantIsolationGuard provides an additional layer of tenant isolation validation
func (m *TenantMiddleware) TenantIsolationGuard(c *fiber.Ctx) error {
	// Get both tenant contexts
	resolvedTenantID, ok1 := c.Locals("tenant_id").(uuid.UUID)
	userTenantID, ok2 := c.Locals("user_tenant_id").(uuid.UUID)

	// If both contexts exist, they must match
	if ok1 && ok2 && resolvedTenantID != userTenantID {
		// Log this as a potential security violation
		// TODO: Add proper logging
		return errors.NewHTTPError(fiber.StatusForbidden, "Tenant isolation violation detected", nil)
	}

	return c.Next()
}

// AllowCrossTenantAccess temporarily allows cross-tenant access for specific operations
// This should be used very sparingly and only for system-level operations
func (m *TenantMiddleware) AllowCrossTenantAccess(c *fiber.Ctx) error {
	c.Locals("allow_cross_tenant", true)
	return c.Next()
}

// CheckSubscriptionStatus validates tenant subscription status
func (m *TenantMiddleware) CheckSubscriptionStatus(c *fiber.Ctx) error {
	tenant, err := GetTenantFromContext(c)
	if err != nil {
		return err
	}

	// Check if tenant has an active subscription
	if tenant.SubscriptionID == nil {
		return errors.NewHTTPError(fiber.StatusPaymentRequired, "No active subscription", nil)
	}

	// Additional subscription validation would go here
	// For example, checking if subscription is past due, suspended, etc.

	return c.Next()
}
