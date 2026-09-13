package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/canakyuz/keystone/internal/authz"
)

// Membership admits an authenticated subject only while the tenant's own record says it
// is an active member, and replaces the role claim with the role that record holds.
//
// It belongs directly after AuthMiddleware and ahead of anything that reads the role. The
// token says who the subject was when it logged in; this says who it is now. Before it
// existed, a suspended user, a deleted one, and a real user of another tenant holding a
// token that names this one were all let in.
func Membership(members authz.MemberLookup) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, err := authz.VerifyMember(c.UserContext(), members, GetTenantID(c), GetUserID(c))

		switch {
		case errors.Is(err, authz.ErrNotMember):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case err != nil:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not verify membership"})
		}

		c.Locals("role", string(role))

		return c.Next()
	}
}

// SameTenant refuses a request whose path names a tenant other than the caller's.
//
// The tenant handlers take the id from the path. Before this, the owner of one tenant
// could read, change, suspend or delete any other by putting its id there. The answer is
// 404 rather than 403, so the response does not confirm that the other tenant exists.
func SameTenant(param string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Params(param) != GetTenantID(c) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
		}

		return c.Next()
	}
}

// PlatformOnly admits only a subject recorded in platform_operators.
//
// Listing every tenant, looking one up by slug, suspending, reactivating and changing a
// plan are operator actions. None of them is something a tenant's own administrator should
// do, to its own tenant or to anyone else's, so no tenant role opens them: an owner is
// nobody at this level until a row grants it. See migration 040.
func PlatformOnly(operators authz.PlatformLookup) fiber.Handler {
	return func(c *fiber.Ctx) error {
		allowed, err := authz.IsPlatformOperator(c.UserContext(), operators, GetUserID(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "could not verify the platform permission",
			})
		}

		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "requires a platform permission"})
		}

		return c.Next()
	}
}

// SelfOrRole lets a subject act on its own user record, and on anyone else's only while it
// holds one of the given roles. With no roles, only the subject itself is let through.
func SelfOrRole(param string, roles ...string) fiber.Handler {
	requireRole := RequireRole(roles...)

	return func(c *fiber.Ctx) error {
		if id := c.Params(param); id != "" && id == GetUserID(c) {
			return c.Next()
		}

		return requireRole(c)
	}
}
