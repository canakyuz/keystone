package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/pkg/authn"
)

// JWTClaims is the verified content of a token.
//
// It is an alias rather than a copy: the gRPC interceptor and this middleware must agree
// on what a token says, and two structs that drift apart is how a transport ends up
// trusting a field the other one validates. See pkg/authn.
type JWTClaims = authn.Claims

// AuthMiddleware validates JWT tokens.
//
// The verification itself lives in pkg/authn, shared with the gRPC interceptor. This
// function is the HTTP half: pull the header out, put the claims into the Fiber context,
// turn a failure into a status code.
//
// The response says only that authentication failed. Distinguishing "no header" from
// "expired" from "bad signature" tells an unauthenticated caller which of the four they
// hit, and none of them can act on the difference.
func AuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := authn.Verify(c.Get("Authorization"), jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthenticated",
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// OptionalAuth records the caller's identity when they present one, and lets the request
// through when they do not.
//
// It differs from AuthMiddleware only in what it does with a failure. It must not differ
// in what counts as a failure, which is why both go through pkg/authn: an endpoint that
// accepted a token the strict path rejects would be a way in.
func OptionalAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := authn.Verify(c.Get("Authorization"), jwtSecret)
		if err != nil {
			return c.Next()
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("tenant_id", claims.TenantID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role")
		if userRole == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access forbidden",
			})
		}

		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// GetUserID retrieves user ID from context
func GetUserID(c *fiber.Ctx) string {
	if userID := c.Locals("user_id"); userID != nil {
		return userID.(string)
	}
	return ""
}

// GetTenantID retrieves tenant ID from context
func GetTenantID(c *fiber.Ctx) string {
	if tenantID := c.Locals("tenant_id"); tenantID != nil {
		return tenantID.(string)
	}
	return ""
}

// GetUserRole retrieves user role from context
func GetUserRole(c *fiber.Ctx) string {
	if role := c.Locals("role"); role != nil {
		return role.(string)
	}
	return ""
}
