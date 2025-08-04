package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"nexspaces-api/internal/auth"
	"nexspaces-api/pkg/database"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
	redis      *database.RedisClient
}

func NewAuthMiddleware(jwtService *auth.JWTService, redis *database.RedisClient) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		redis:      redis,
	}
}

// RequireAuth middleware validates JWT token and sets user context
func (am *AuthMiddleware) RequireAuth(c *fiber.Ctx) error {
	// Get token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	// Validate token
	claims, err := am.jwtService.ValidateBetterAuthToken(authHeader)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid token",
			"details": err.Error(),
		})
	}

	// Check if session is still valid in Redis (optional session store check)
	if claims.SessionID != "" {
		sessionKey := "session:" + claims.SessionID
		exists, err := am.redis.Exists(context.Background(), sessionKey).Result()
		if err != nil || exists == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session expired or invalid",
			})
		}
	}

	// Set user context
	c.Locals("user", claims)
	c.Locals("user_id", claims.UserID)
	c.Locals("tenant_id", claims.TenantID)
	c.Locals("user_email", claims.Email)
	c.Locals("user_role", claims.Role)
	c.Locals("tenant_role", claims.TenantRole)

	return c.Next()
}

// RequireTenantAccess middleware ensures user has access to specific tenant
func (am *AuthMiddleware) RequireTenantAccess(c *fiber.Ctx) error {
	// Get user from context (must be authenticated first)
	user, ok := c.Locals("user").(*auth.JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Get tenant ID from URL params or headers
	requestedTenant := c.Params("tenantId")
	if requestedTenant == "" {
		requestedTenant = c.Get("X-Tenant-ID")
	}

	// Check if user has access to this tenant
	if user.TenantID != requestedTenant {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied to this tenant",
		})
	}

	return c.Next()
}

// RequireRole middleware checks if user has specific role
func (am *AuthMiddleware) RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*auth.JWTClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role || user.TenantRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":          "Insufficient permissions",
				"required_roles": roles,
				"user_role":      user.Role,
				"tenant_role":    user.TenantRole,
			})
		}

		return c.Next()
	}
}

// OptionalAuth middleware validates token if present but doesn't require it
func (am *AuthMiddleware) OptionalAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Next()
	}

	// Try to validate token
	claims, err := am.jwtService.ValidateBetterAuthToken(authHeader)
	if err != nil {
		// Invalid token, but we continue without authentication
		return c.Next()
	}

	// Set user context if token is valid
	c.Locals("user", claims)
	c.Locals("user_id", claims.UserID)
	c.Locals("tenant_id", claims.TenantID)
	c.Locals("authenticated", true)

	return c.Next()
}

// TenantResolver middleware resolves tenant from subdomain/domain
func (am *AuthMiddleware) TenantResolver(c *fiber.Ctx) error {
	// Get host header
	host := c.Get("Host")

	// Extract tenant slug from subdomain
	var tenantSlug string

	// Handle subdomain: abc.nexpaces.com -> abc
	if strings.Contains(host, ".") {
		parts := strings.Split(host, ".")
		if len(parts) >= 2 && parts[0] != "www" {
			tenantSlug = parts[0]
		}
	}

	// Handle custom domain (requires database lookup)
	if tenantSlug == "" {
		// TODO: Lookup tenant by custom domain
		// tenantSlug = lookupTenantByDomain(host)
	}

	// Set tenant context
	if tenantSlug != "" {
		c.Locals("tenant_slug", tenantSlug)
		c.Set("X-Tenant-Slug", tenantSlug)
	}

	return c.Next()
}
