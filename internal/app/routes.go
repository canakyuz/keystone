package app

import (
	"github.com/canakyuz/keystone/internal/config"
	"github.com/canakyuz/keystone/internal/domain/user"
	authHandler "github.com/canakyuz/keystone/internal/handler/auth"
	registryHandler "github.com/canakyuz/keystone/internal/handler/registry"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	uploadHandler "github.com/canakyuz/keystone/internal/handler/upload"
	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	"github.com/canakyuz/keystone/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// setupRoutes wires every API route. It is called when the server starts and decides
// which HTTP path maps to which handler.
func setupRoutes(
	app *fiber.App,
	cfg *config.Config,
	authH *authHandler.Handler,
	tenantH *tenantHandler.Handler,
	userH *userHandler.Handler,
	uploadH *uploadHandler.Handler,
	moduleCatalogH *registryHandler.ModuleCatalogHandler,
	toolCatalogH *registryHandler.ToolCatalogHandler,
	activationH *registryHandler.ActivationHandler,
	tenantContextMiddleware fiber.Handler,
	tenantScope fiber.Handler,
	planRateLimit fiber.Handler,
	membership fiber.Handler,
) {
	// Authentication, then membership, then the plan-based limit, in that order.
	//
	// Each depends on the one before it. Membership reads the subject and tenant the
	// authenticator wrote, and replaces the role claim with the role the tenant's record
	// holds now. The limiter reads the tenant too; registering it globally, ahead of
	// authentication, is what once made the plan quotas dead code.
	//
	// Membership comes before the limiter so that a subject who is no longer a member
	// cannot spend its former tenant's quota. The cost is a primary-key lookup for a
	// request the limiter would have refused; the address limit in front of all of this
	// bounds how many of those one client can cause.
	authenticated := []fiber.Handler{
		middleware.AuthMiddleware(cfg.Auth.JWTSecret),
		membership,
		planRateLimit,
	}

	// Route guards. The role they check is the one Membership read from the database, not
	// the token's claim.
	var (
		owner        = string(user.RoleOwner)
		admin        = string(user.RoleAdmin)
		editor       = string(user.RoleEditor)
		admins       = middleware.RequireRole(owner, admin)
		editors      = middleware.RequireRole(owner, admin, editor)
		ownerOnly    = middleware.RequireRole(owner)
		ownTenant    = middleware.SameTenant("id")
		platformOnly = middleware.PlatformOnly()
	)

	// /docs serves the static Swagger/OpenAPI documentation page.
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("web/static/docs/index.html")
	})
	app.Static("/docs/", "./web/static/docs")
	// /api/openapi.yaml serves the API definition file.
	app.Get("/api/openapi.yaml", func(c *fiber.Ctx) error {
		return c.SendFile("api/openapi.yaml")
	})

	// The liveness and readiness probes are registered in internal/app/probes.go,
	// because they need database and Redis access.

	// /api/v1 is the root group for version 1 of the API.
	v1 := app.Group("/api/v1")

	// Authentication routes: login and registration. Public.
	auth := v1.Group("/auth")
	auth.Post("/register", authH.Register)
	auth.Post("/login", authH.Login)
	auth.Post("/logout", authH.Logout)

	// Protected auth routes; a JWT is required.
	authProtected := v1.Group("/auth", chain(authenticated)...)
	authProtected.Get("/me", authH.GetMe)

	// Tenant routes: authenticated, and scoped to one tenant.
	// Middleware order: Auth -> TenantContext -> TenantScope
	tenants := v1.Group("/tenants", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	// Tenant creation is NOT in this group, for two reasons.
	//
	// First: this group uses tenantContextMiddleware, which tries to resolve the
	// schema of the request's tenant. Since the new tenant does not exist yet, the
	// request would fail during schema resolution.
	//
	// Second: provisioning is no longer synchronous. The endpoint returns 202 Accepted
	// and an operation address the client polls. See
	// internal/app/operations.go and docs/INVARIANTS.md.
	tenants.Get("/current", tenantH.GetCurrent)

	// These act across tenants: counts over all of them, a lookup by any slug, a listing
	// of every tenant, and the lifecycle changes an operator makes. Every authenticated
	// caller could use them. See middleware.PlatformOnly.
	//
	// Registered ahead of "/:id" because Fiber matches in registration order, and "/stats"
	// would otherwise be taken for a tenant id.
	tenants.Get("/stats", platformOnly, tenantH.GetStats)
	tenants.Get("/slug/:slug", platformOnly, tenantH.GetBySlug)
	tenants.Get("/", platformOnly, tenantH.List)
	tenants.Post("/:id/suspend", platformOnly, tenantH.Suspend)
	tenants.Post("/:id/activate", platformOnly, tenantH.Activate)
	tenants.Post("/:id/upgrade", platformOnly, tenantH.UpgradePlan)

	// The id in the path must be the caller's own tenant. The handlers act on whatever id
	// they are given, and before SameTenant that was any tenant at all.
	tenants.Get("/:id", ownTenant, tenantH.GetByID)
	tenants.Patch("/:id", ownTenant, admins, tenantH.Update)
	tenants.Post("/:id/domain", ownTenant, admins, tenantH.SetCustomDomain)
	tenants.Post("/:id/domain/verify", ownTenant, admins, tenantH.VerifyCustomDomain)
	tenants.Patch("/:id/branding", ownTenant, admins, tenantH.UpdateBranding)
	tenants.Delete("/:id", ownTenant, ownerOnly, tenantH.Delete)

	// Upload routes: tenant-scoped and authenticated.
	upload := v1.Group("/upload", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	upload.Post("/logo", admins, uploadH.UploadLogo)
	upload.Post("/favicon", admins, uploadH.UploadFavicon)
	upload.Post("/image", editors, uploadH.UploadImage)

	// A static path serving uploaded files. Public.
	app.Static("/uploads", "./uploads")

	// User routes; authentication required.
	users := v1.Group("/users", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	users.Post("/", admins, userH.Create)
	users.Get("/stats", userH.GetStats)
	users.Get("/:id", userH.GetByID)
	users.Get("/", userH.List)
	users.Patch("/:id", middleware.SelfOrRole("id", owner, admin), userH.Update)
	// A password change belongs to the subject alone. It asks for the current password,
	// which no administrator has.
	users.Post("/:id/password", middleware.SelfOrRole("id"), userH.UpdatePassword)
	users.Post("/:id/role", admins, userH.UpdateRole)
	users.Post("/:id/suspend", admins, userH.Suspend)
	users.Post("/:id/activate", admins, userH.Activate)
	users.Post("/:id/verify-email", admins, userH.VerifyEmail)
	users.Delete("/:id", admins, userH.Delete)

	// The example business modules mount their own routes; see examples/verticals.
	// They are registered from cmd/server, after this function returns, because the
	// control plane does not import them.

	// Registry routes: the module and tool marketplace.
	registry := v1.Group("/registry")

	// Public module catalogue routes; no authentication required.
	registry.Get("/modules", moduleCatalogH.ListPublicModules)
	registry.Get("/modules/search", moduleCatalogH.SearchModules)
	registry.Get("/modules/popular", moduleCatalogH.GetPopularModules)
	registry.Get("/modules/top-rated", moduleCatalogH.GetTopRatedModules)
	registry.Get("/modules/new", moduleCatalogH.GetNewModules)
	registry.Get("/modules/free", moduleCatalogH.GetFreeModules)
	registry.Get("/modules/category/:category", moduleCatalogH.GetModulesByCategory)
	registry.Get("/modules/slug/:slug", moduleCatalogH.GetModuleBySlug)
	registry.Get("/modules/:id", moduleCatalogH.GetModuleByID)

	// Public tool catalogue routes; no authentication required.
	registry.Get("/tools", toolCatalogH.ListPublicTools)
	registry.Get("/tools/search", toolCatalogH.SearchTools)
	registry.Get("/tools/popular", toolCatalogH.GetPopularTools)
	registry.Get("/tools/top-rated", toolCatalogH.GetTopRatedTools)
	registry.Get("/tools/new", toolCatalogH.GetNewTools)
	registry.Get("/tools/free", toolCatalogH.GetFreeTools)
	registry.Get("/tools/category/:category", toolCatalogH.GetToolsByCategory)
	registry.Get("/tools/slug/:slug", toolCatalogH.GetToolBySlug)
	registry.Get("/tools/:id", toolCatalogH.GetToolByID)

	// Tenant-scoped activation routes; authentication required.
	tenantRegistry := v1.Group("/registry/tenant", chain(authenticated, tenantScope)...)

	// Module activation for the current tenant.
	tenantRegistry.Get("/modules", activationH.GetActivatedModules)
	tenantRegistry.Post("/modules/install", admins, activationH.InstallModule)
	tenantRegistry.Post("/modules/:module_id/activate", admins, activationH.ActivateModule)
	tenantRegistry.Post("/modules/:module_id/deactivate", admins, activationH.DeactivateModule)
	tenantRegistry.Delete("/modules/:module_id", admins, activationH.UninstallModule)
	tenantRegistry.Post("/modules/:module_id/complete-setup", admins, activationH.CompleteModuleSetup)
	tenantRegistry.Get("/modules/:module_id/dependencies", activationH.CheckModuleDependencies)

	// Tool activation for the current tenant.
	tenantRegistry.Get("/tools", activationH.GetActivatedTools)
	tenantRegistry.Post("/tools/install", admins, activationH.InstallTool)
	tenantRegistry.Post("/tools/:tool_id/activate", admins, activationH.ActivateTool)
	tenantRegistry.Post("/tools/:tool_id/deactivate", admins, activationH.DeactivateTool)
	tenantRegistry.Delete("/tools/:tool_id", admins, activationH.UninstallTool)
	tenantRegistry.Post("/tools/:tool_id/complete-setup", admins, activationH.CompleteToolSetup)
	tenantRegistry.Get("/tools/:tool_id/dependencies", activationH.CheckToolDependencies)
}

// chain appends group-specific middleware to a shared base without aliasing it.
//
// append() on a shared slice can write into the base's spare capacity, so two groups
// built from the same base would overwrite each other's middleware. Copying is cheap
// here and the alternative is a bug that only appears once a second group is added.
func chain(base []fiber.Handler, extra ...fiber.Handler) []fiber.Handler {
	out := make([]fiber.Handler, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)

	return out
}
