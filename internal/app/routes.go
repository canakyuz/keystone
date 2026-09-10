package app

import (
	"github.com/canakyuz/keystone/internal/config"
	authHandler "github.com/canakyuz/keystone/internal/handler/auth"
	blogHandler "github.com/canakyuz/keystone/internal/handler/blog"
	bookingHandler "github.com/canakyuz/keystone/internal/handler/booking"
	lessonHandler "github.com/canakyuz/keystone/internal/handler/lesson"
	paymentHandler "github.com/canakyuz/keystone/internal/handler/payment"
	registryHandler "github.com/canakyuz/keystone/internal/handler/registry"
	serviceHandler "github.com/canakyuz/keystone/internal/handler/service"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	uploadHandler "github.com/canakyuz/keystone/internal/handler/upload"
	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	websiteHandler "github.com/canakyuz/keystone/internal/handler/website"
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
	websiteH *websiteHandler.Handler,
	studentH *lessonHandler.StudentHandler,
	lessonH *lessonHandler.LessonHandler,
	assignmentH *lessonHandler.AssignmentHandler,
	availabilityH *bookingHandler.AvailabilityHandler,
	appointmentH *bookingHandler.AppointmentHandler,
	serviceH *serviceHandler.ServiceHandler,
	postH *blogHandler.PostHandler,
	categoryH *blogHandler.CategoryHandler,
	paymentH *paymentHandler.Handler,
	webhookH *paymentHandler.WebhookHandler,
	moduleCatalogH *registryHandler.ModuleCatalogHandler,
	toolCatalogH *registryHandler.ToolCatalogHandler,
	activationH *registryHandler.ActivationHandler,
	tenantContextMiddleware fiber.Handler,
	tenantScope fiber.Handler,
	planRateLimit fiber.Handler,
) {
	// Authentication, then the plan-based limit, in that order.
	//
	// They are bundled because the second depends on the first: the limiter reads the
	// tenant the authenticator wrote. Registering the limiter globally, ahead of
	// authentication, is what made the plan quotas dead code — every authenticated
	// tenant fell back to the anonymous limit.
	authenticated := []fiber.Handler{
		middleware.AuthMiddleware(cfg.Auth.JWTSecret),
		planRateLimit,
	}

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

	// Public routes; no authentication required.
	public := v1.Group("/public")
	public.Get("/websites/slug/:slug", websiteH.GetBySlug)

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
	tenants.Get("/stats", tenantH.GetStats)
	tenants.Get("/slug/:slug", tenantH.GetBySlug)
	tenants.Get("/:id", tenantH.GetByID)
	tenants.Get("/", tenantH.List)
	tenants.Patch("/:id", tenantH.Update)
	tenants.Post("/:id/suspend", tenantH.Suspend)
	tenants.Post("/:id/activate", tenantH.Activate)
	tenants.Post("/:id/upgrade", tenantH.UpgradePlan)
	tenants.Post("/:id/domain", tenantH.SetCustomDomain)
	tenants.Post("/:id/domain/verify", tenantH.VerifyCustomDomain)
	tenants.Patch("/:id/branding", tenantH.UpdateBranding)
	tenants.Delete("/:id", tenantH.Delete)

	// Upload routes: tenant-scoped and authenticated.
	upload := v1.Group("/upload", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	upload.Post("/logo", uploadH.UploadLogo)
	upload.Post("/favicon", uploadH.UploadFavicon)
	upload.Post("/image", uploadH.UploadImage)

	// A static path serving uploaded files. Public.
	app.Static("/uploads", "./uploads")

	// User routes; authentication required.
	users := v1.Group("/users", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	users.Post("/", userH.Create)
	users.Get("/stats", userH.GetStats)
	users.Get("/:id", userH.GetByID)
	users.Get("/", userH.List)
	users.Patch("/:id", userH.Update)
	users.Post("/:id/password", userH.UpdatePassword)
	users.Post("/:id/role", userH.UpdateRole)
	users.Post("/:id/suspend", userH.Suspend)
	users.Post("/:id/activate", userH.Activate)
	users.Post("/:id/verify-email", userH.VerifyEmail)
	users.Delete("/:id", userH.Delete)

	// Website routes; tenant-scoped.
	websites := v1.Group("/websites", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	websites.Get("/", websiteH.List)
	websites.Post("/", websiteH.Create)
	websites.Get("/:id", websiteH.GetByID)
	websites.Patch("/:id", websiteH.Update)
	websites.Delete("/:id", websiteH.Delete)
	websites.Post("/:id/publish", websiteH.Publish)
	websites.Post("/:id/archive", websiteH.Archive)

	// Student routes; part of the lessons module, tenant-scoped.
	students := v1.Group("/students", chain(authenticated, tenantScope)...)
	students.Post("/", studentH.Create)
	students.Get("/stats", studentH.GetStats)
	students.Get("/email", studentH.GetByEmail)
	students.Get("/:id", studentH.GetByID)
	students.Get("/", studentH.List)
	students.Put("/:id", studentH.Update)
	students.Delete("/:id", studentH.Delete)

	// Lesson routes; part of the lessons module, tenant-scoped.
	lessons := v1.Group("/lessons", chain(authenticated, tenantScope)...)
	lessons.Post("/", lessonH.Create)
	lessons.Get("/stats", lessonH.GetStats)
	lessons.Get("/upcoming", lessonH.GetUpcoming)
	lessons.Get("/:id", lessonH.GetByID)
	lessons.Get("/", lessonH.List)
	lessons.Put("/:id", lessonH.Update)
	lessons.Delete("/:id", lessonH.Delete)

	// Per-student lesson routes.
	students.Get("/:student_id/lessons", lessonH.GetByStudent)

	// Assignment routes; part of the lessons module, tenant-scoped.
	assignments := v1.Group("/assignments", chain(authenticated, tenantScope)...)
	assignments.Post("/", assignmentH.Create)
	assignments.Get("/stats", assignmentH.GetStats)
	assignments.Get("/overdue", assignmentH.GetOverdue)
	assignments.Get("/:id", assignmentH.GetByID)
	assignments.Get("/", assignmentH.List)
	assignments.Put("/:id", assignmentH.Update)
	assignments.Delete("/:id", assignmentH.Delete)

	// Per-student assignment routes.
	students.Get("/:student_id/assignments", assignmentH.GetByStudent)

	// Availability routes; part of the booking module, tenant-scoped.
	availabilities := v1.Group("/availabilities", chain(authenticated, tenantScope)...)
	availabilities.Post("/", availabilityH.Create)
	availabilities.Get("/stats", availabilityH.GetStats)
	availabilities.Get("/date-range", availabilityH.GetByDateRange)
	availabilities.Get("/:id", availabilityH.GetByID)
	availabilities.Get("/", availabilityH.List)
	availabilities.Put("/:id", availabilityH.Update)
	availabilities.Delete("/:id", availabilityH.Delete)

	// Per-user availability routes.
	availabilities.Get("/user/:user_id", availabilityH.GetByUser)

	// Appointment routes; part of the booking module, tenant-scoped.
	appointments := v1.Group("/appointments", chain(authenticated, tenantScope)...)
	appointments.Post("/", appointmentH.Create)
	appointments.Get("/stats", appointmentH.GetStats)
	appointments.Get("/upcoming", appointmentH.GetUpcoming)
	appointments.Get("/date-range", appointmentH.GetByDateRange)
	appointments.Get("/client", appointmentH.GetByClient)
	appointments.Get("/:id", appointmentH.GetByID)
	appointments.Get("/", appointmentH.List)
	appointments.Put("/:id", appointmentH.Update)
	appointments.Post("/:id/confirm", appointmentH.Confirm)
	appointments.Post("/:id/cancel", appointmentH.Cancel)
	appointments.Post("/:id/complete", appointmentH.Complete)
	appointments.Delete("/:id", appointmentH.Delete)

	// Per-user appointment routes.
	appointments.Get("/user/:user_id", appointmentH.GetByUser)

	// Service routes; tenant-scoped.
	services := v1.Group("/services", chain(authenticated, tenantScope)...)
	services.Post("/", serviceH.Create)
	services.Get("/stats", serviceH.GetStats)
	services.Get("/featured", serviceH.GetFeatured)
	services.Get("/slug/:slug", serviceH.GetBySlug)
	services.Get("/:id", serviceH.GetByID)
	services.Get("/", serviceH.List)
	services.Put("/:id", serviceH.Update)
	services.Delete("/:id", serviceH.Delete)

	// Blog category routes; tenant-scoped.
	categories := v1.Group("/blog/categories", chain(authenticated, tenantScope)...)
	categories.Post("/", categoryH.Create)
	categories.Get("/slug/:slug", categoryH.GetBySlug)
	categories.Get("/:id", categoryH.GetByID)
	categories.Get("/", categoryH.List)
	categories.Put("/:id", categoryH.Update)
	categories.Delete("/:id", categoryH.Delete)

	// Blog post routes; tenant-scoped.
	posts := v1.Group("/blog/posts", chain(authenticated, tenantScope)...)
	posts.Post("/", postH.Create)
	posts.Get("/featured", postH.GetFeatured)
	posts.Get("/slug/:slug", postH.GetBySlug)
	posts.Get("/tag/:tag", postH.GetByTag)
	posts.Get("/:id", postH.GetByID)
	posts.Get("/", postH.List)
	posts.Put("/:id", postH.Update)
	posts.Post("/:id/publish", postH.Publish)
	posts.Post("/:id/archive", postH.Archive)
	posts.Delete("/:id", postH.Delete)

	// Per-category post routes.
	categories.Get("/:category_id/posts", postH.GetByCategoryID)

	// Payment routes; tenant-scoped and authenticated.
	payments := v1.Group("/payments", chain(authenticated, tenantScope)...)
	payments.Post("/", paymentH.CreatePayment)
	payments.Post("/complete-3ds", paymentH.Complete3DSPayment)
	payments.Get("/:id", paymentH.GetPayment)
	payments.Get("/", paymentH.ListPayments)
	payments.Post("/:id/refund", paymentH.CreateRefund)

	// Webhook routes handle callbacks from the payment providers. Public.
	webhooks := v1.Group("/webhooks/payment")
	webhooks.Post("/:provider", webhookH.HandleWebhook)
	webhooks.Post("/iyzico/:tenant_id", webhookH.HandleIyzicoWebhook)
	webhooks.Post("/checkout/:tenant_id", webhookH.HandleCheckoutWebhook)

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
	tenantRegistry.Post("/modules/install", activationH.InstallModule)
	tenantRegistry.Post("/modules/:module_id/activate", activationH.ActivateModule)
	tenantRegistry.Post("/modules/:module_id/deactivate", activationH.DeactivateModule)
	tenantRegistry.Delete("/modules/:module_id", activationH.UninstallModule)
	tenantRegistry.Post("/modules/:module_id/complete-setup", activationH.CompleteModuleSetup)
	tenantRegistry.Get("/modules/:module_id/dependencies", activationH.CheckModuleDependencies)

	// Tool activation for the current tenant.
	tenantRegistry.Get("/tools", activationH.GetActivatedTools)
	tenantRegistry.Post("/tools/install", activationH.InstallTool)
	tenantRegistry.Post("/tools/:tool_id/activate", activationH.ActivateTool)
	tenantRegistry.Post("/tools/:tool_id/deactivate", activationH.DeactivateTool)
	tenantRegistry.Delete("/tools/:tool_id", activationH.UninstallTool)
	tenantRegistry.Post("/tools/:tool_id/complete-setup", activationH.CompleteToolSetup)
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
