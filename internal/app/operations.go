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
// Authorization on this endpoint therefore works differently: the caller is
// authenticated, but no tenant membership is checked. Creating a tenant should be a
// platform-level permission; right now it is protected by authentication alone, and
// that gap is recorded in INVARIANTS.md.
func registerOperationRoutes(app *fiber.App, jwtSecret string, h *operationHandler.Handler) {
	auth := middleware.AuthMiddleware(jwtSecret)

	v1 := app.Group("/api/v1")

	v1.Post("/tenants", auth, h.CreateTenant)
	v1.Get("/operations/:id", auth, h.GetOperation)
}
