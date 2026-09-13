package app

import (
	"github.com/gofiber/fiber/v2"

	operationHandler "github.com/canakyuz/keystone/internal/handler/operation"
	"github.com/canakyuz/keystone/internal/middleware"
)

// registerOperationRoutes registers the tenant provisioning and operation lookup
// endpoints.
//
// WHY not inside setupRoutes: that function's parameter list is already past twenty.
// Adding another dependency would make it longer still. These endpoints live in a
// separate function carrying their own dependencies.
//
// NO TENANT CONTEXT
// POST /tenants does not use tenantContextMiddleware, because the tenant does not
// exist yet. The middleware would try to resolve the schema of a tenant that is not
// there and the request would fail with a 404.
//
// Authorization on this endpoint therefore stops short of the new tenant: the caller must
// be an active member of the tenant its token names, and must hold the platform permission
// that migration 040 records. Creating a tenant for somebody else is an operator action,
// and no tenant role grants it.
func registerOperationRoutes(
	app *fiber.App, jwtSecret string, membership, platformOnly fiber.Handler, h *operationHandler.Handler,
) {
	authenticated := []fiber.Handler{middleware.AuthMiddleware(jwtSecret), membership}

	v1 := app.Group("/api/v1")

	v1.Post("/tenants", chain(authenticated, platformOnly, h.CreateTenant)...)
	v1.Get("/operations/:id", chain(authenticated, h.GetOperation)...)
}
