package app

import (
	"github.com/gofiber/fiber/v2"
	"nexpaces-api/internal/config"
	authHandler "nexpaces-api/internal/handler/auth"
	blogHandler "nexpaces-api/internal/handler/blog"
	bookingHandler "nexpaces-api/internal/handler/booking"
	lessonHandler "nexpaces-api/internal/handler/lesson"
	paymentHandler "nexpaces-api/internal/handler/payment"
	registryHandler "nexpaces-api/internal/handler/registry"
	serviceHandler "nexpaces-api/internal/handler/service"
	tenantHandler "nexpaces-api/internal/handler/tenant"
	uploadHandler "nexpaces-api/internal/handler/upload"
	userHandler "nexpaces-api/internal/handler/user"
	websiteHandler "nexpaces-api/internal/handler/website"
	"nexpaces-api/internal/middleware"
)

// setupRoutes configures all application routes
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
) {
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("web/static/docs/index.html")
	})
	app.Static("/docs/", "./web/static/docs")
	app.Get("/api/openapi.yaml", func(c *fiber.Ctx) error {
		return c.SendFile("api/openapi.yaml")
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"environment": cfg.Server.Environment,
			"message":     "NexSpaces API is running",
		})
	})

	// API v1 group
	v1 := app.Group("/api/v1")

	// Public routes (no authentication required)
	public := v1.Group("/public")
	public.Get("/websites/slug/:slug", websiteH.GetBySlug)

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register", authH.Register)
	auth.Post("/login", authH.Login)
	auth.Post("/logout", authH.Logout)

	// Protected auth routes
	authProtected := v1.Group("/auth", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	authProtected.Get("/me", authH.GetMe)

	// Tenant routes (authentication required)
	tenants := v1.Group("/tenants", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	tenants.Post("/", tenantH.Create)                              // Create tenant
	tenants.Get("/current", tenantH.GetCurrent)                    // Get current tenant
	tenants.Get("/stats", tenantH.GetStats)                        // Tenant statistics
	tenants.Get("/slug/:slug", tenantH.GetBySlug)                  // Get by slug
	tenants.Get("/:id", tenantH.GetByID)                           // Get by ID
	tenants.Get("/", tenantH.List)                                 // List tenants
	tenants.Patch("/:id", tenantH.Update)                          // Update tenant
	tenants.Post("/:id/suspend", tenantH.Suspend)                  // Suspend tenant
	tenants.Post("/:id/activate", tenantH.Activate)                // Activate tenant
	tenants.Post("/:id/upgrade", tenantH.UpgradePlan)              // Upgrade plan
	tenants.Post("/:id/domain", tenantH.SetCustomDomain)           // Set custom domain
	tenants.Post("/:id/domain/verify", tenantH.VerifyCustomDomain) // Verify domain
	tenants.Patch("/:id/branding", tenantH.UpdateBranding)         // Update branding
	tenants.Delete("/:id", tenantH.Delete)                         // Delete tenant

	// Upload routes (tenant-scoped, authentication required)
	upload := v1.Group("/upload", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	upload.Post("/logo", uploadH.UploadLogo)       // Upload logo
	upload.Post("/favicon", uploadH.UploadFavicon) // Upload favicon
	upload.Post("/image", uploadH.UploadImage)     // Upload general image

	// Serve uploaded files (public access)
	app.Static("/uploads", "./uploads")

	// User routes (authentication required)
	users := v1.Group("/users", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	users.Post("/", userH.Create)                      // Create user (admin only)
	users.Get("/stats", userH.GetStats)                // User statistics
	users.Get("/:id", userH.GetByID)                   // Get user by ID
	users.Get("/", userH.List)                         // List users
	users.Patch("/:id", userH.Update)                  // Update user
	users.Post("/:id/password", userH.UpdatePassword)  // Update password
	users.Post("/:id/role", userH.UpdateRole)          // Update role (admin only)
	users.Post("/:id/suspend", userH.Suspend)          // Suspend user (admin only)
	users.Post("/:id/activate", userH.Activate)        // Activate user (admin only)
	users.Post("/:id/verify-email", userH.VerifyEmail) // Verify email
	users.Delete("/:id", userH.Delete)                 // Delete user (admin only)

	// Website routes (tenant-scoped)
	websites := v1.Group("/websites", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	websites.Get("/", websiteH.List)
	websites.Post("/", websiteH.Create)
	websites.Get("/:id", websiteH.GetByID)
	websites.Patch("/:id", websiteH.Update)
	websites.Delete("/:id", websiteH.Delete)
	websites.Post("/:id/publish", websiteH.Publish)
	websites.Post("/:id/archive", websiteH.Archive)

	// Student routes (Lessons module - tenant-scoped)
	students := v1.Group("/students", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	students.Post("/", studentH.Create)         // Create student
	students.Get("/stats", studentH.GetStats)   // Student statistics
	students.Get("/email", studentH.GetByEmail) // Get by email
	students.Get("/:id", studentH.GetByID)      // Get student by ID
	students.Get("/", studentH.List)            // List students
	students.Put("/:id", studentH.Update)       // Update student
	students.Delete("/:id", studentH.Delete)    // Delete student

	// Lesson routes (Lessons module - tenant-scoped)
	lessons := v1.Group("/lessons", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	lessons.Post("/", lessonH.Create)             // Create lesson
	lessons.Get("/stats", lessonH.GetStats)       // Lesson statistics
	lessons.Get("/upcoming", lessonH.GetUpcoming) // Get upcoming lessons
	lessons.Get("/:id", lessonH.GetByID)          // Get lesson by ID
	lessons.Get("/", lessonH.List)                // List lessons
	lessons.Put("/:id", lessonH.Update)           // Update lesson
	lessons.Delete("/:id", lessonH.Delete)        // Delete lesson

	// Student-specific lesson routes
	students.Get("/:student_id/lessons", lessonH.GetByStudent) // Get lessons by student

	// Assignment routes (Lessons module - tenant-scoped)
	assignments := v1.Group("/assignments", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	assignments.Post("/", assignmentH.Create)           // Create assignment
	assignments.Get("/stats", assignmentH.GetStats)     // Assignment statistics
	assignments.Get("/overdue", assignmentH.GetOverdue) // Get overdue assignments
	assignments.Get("/:id", assignmentH.GetByID)        // Get assignment by ID
	assignments.Get("/", assignmentH.List)              // List assignments
	assignments.Put("/:id", assignmentH.Update)         // Update assignment
	assignments.Delete("/:id", assignmentH.Delete)      // Delete assignment

	// Student-specific assignment routes
	students.Get("/:student_id/assignments", assignmentH.GetByStudent) // Get assignments by student

	// Availability routes (Booking module - tenant-scoped)
	availabilities := v1.Group("/availabilities", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	availabilities.Post("/", availabilityH.Create)                  // Create availability
	availabilities.Get("/stats", availabilityH.GetStats)            // Availability statistics
	availabilities.Get("/date-range", availabilityH.GetByDateRange) // Get by date range
	availabilities.Get("/:id", availabilityH.GetByID)               // Get availability by ID
	availabilities.Get("/", availabilityH.List)                     // List availabilities
	availabilities.Put("/:id", availabilityH.Update)                // Update availability
	availabilities.Delete("/:id", availabilityH.Delete)             // Delete availability

	// User-specific availability routes
	availabilities.Get("/user/:user_id", availabilityH.GetByUser) // Get availabilities by user

	// Appointment routes (Booking module - tenant-scoped)
	appointments := v1.Group("/appointments", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	appointments.Post("/", appointmentH.Create)                  // Create appointment
	appointments.Get("/stats", appointmentH.GetStats)            // Appointment statistics
	appointments.Get("/upcoming", appointmentH.GetUpcoming)      // Get upcoming appointments
	appointments.Get("/date-range", appointmentH.GetByDateRange) // Get by date range
	appointments.Get("/client", appointmentH.GetByClient)        // Get by client email
	appointments.Get("/:id", appointmentH.GetByID)               // Get appointment by ID
	appointments.Get("/", appointmentH.List)                     // List appointments
	appointments.Put("/:id", appointmentH.Update)                // Update appointment
	appointments.Post("/:id/confirm", appointmentH.Confirm)      // Confirm appointment
	appointments.Post("/:id/cancel", appointmentH.Cancel)        // Cancel appointment
	appointments.Post("/:id/complete", appointmentH.Complete)    // Complete appointment
	appointments.Delete("/:id", appointmentH.Delete)             // Delete appointment

	// User-specific appointment routes

	// Service routes (tenant-scoped)
	services := v1.Group("/services", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	services.Post("/", serviceH.Create)                        // Create service
	services.Get("/stats", serviceH.GetStats)                  // Service statistics
	services.Get("/featured", serviceH.GetFeatured)            // Get featured services
	services.Get("/slug/:slug", serviceH.GetBySlug)            // Get by slug
	services.Get("/:id", serviceH.GetByID)                     // Get service by ID
	services.Get("/", serviceH.List)                           // List services
	services.Put("/:id", serviceH.Update)                      // Update service
	services.Delete("/:id", serviceH.Delete)                   // Delete service
	appointments.Get("/user/:user_id", appointmentH.GetByUser) // Get appointments by user

	// Blog Category routes (tenant-scoped)
	categories := v1.Group("/blog/categories", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	categories.Post("/", categoryH.Create)             // Create category
	categories.Get("/slug/:slug", categoryH.GetBySlug) // Get by slug
	categories.Get("/:id", categoryH.GetByID)          // Get category by ID
	categories.Get("/", categoryH.List)                // List categories
	categories.Put("/:id", categoryH.Update)           // Update category
	categories.Delete("/:id", categoryH.Delete)        // Delete category

	// Blog Post routes (tenant-scoped)
	posts := v1.Group("/blog/posts", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	posts.Post("/", postH.Create)             // Create post
	posts.Get("/featured", postH.GetFeatured) // Get featured posts
	posts.Get("/slug/:slug", postH.GetBySlug) // Get by slug
	posts.Get("/tag/:tag", postH.GetByTag)    // Get by tag
	posts.Get("/:id", postH.GetByID)          // Get post by ID
	posts.Get("/", postH.List)                // List posts
	posts.Put("/:id", postH.Update)           // Update post
	posts.Post("/:id/publish", postH.Publish) // Publish post
	posts.Post("/:id/archive", postH.Archive) // Archive post
	posts.Delete("/:id", postH.Delete)        // Delete post

	// Category-specific post routes
	categories.Get("/:category_id/posts", postH.GetByCategoryID) // Get posts by category

	// Payment routes (tenant-scoped, authentication required)
	payments := v1.Group("/payments", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	payments.Post("/", paymentH.CreatePayment)                // Create payment
	payments.Post("/complete-3ds", paymentH.Complete3DSPayment) // Complete 3DS authentication
	payments.Get("/:id", paymentH.GetPayment)                  // Get payment by ID
	payments.Get("/", paymentH.ListPayments)                   // List payments
	payments.Post("/:id/refund", paymentH.CreateRefund)        // Create refund

	// Webhook routes (public, no authentication)
	webhooks := v1.Group("/webhooks/payment")
	webhooks.Post("/:provider", webhookH.HandleWebhook)                 // Generic webhook (provider-specific)
	webhooks.Post("/iyzico/:tenant_id", webhookH.HandleIyzicoWebhook)   // iyzico webhook (tenant-specific URL)
	webhooks.Post("/checkout/:tenant_id", webhookH.HandleCheckoutWebhook) // Checkout.com webhook (tenant-specific URL)

	// Registry routes - Module & Tool Marketplace
	registry := v1.Group("/registry")

	// Public module catalog (no authentication)
	registry.Get("/modules", moduleCatalogH.ListPublicModules)           // List public modules
	registry.Get("/modules/search", moduleCatalogH.SearchModules)        // Search modules
	registry.Get("/modules/popular", moduleCatalogH.GetPopularModules)   // Popular modules
	registry.Get("/modules/top-rated", moduleCatalogH.GetTopRatedModules) // Top rated modules
	registry.Get("/modules/new", moduleCatalogH.GetNewModules)           // New modules
	registry.Get("/modules/free", moduleCatalogH.GetFreeModules)         // Free modules
	registry.Get("/modules/category/:category", moduleCatalogH.GetModulesByCategory) // Modules by category
	registry.Get("/modules/slug/:slug", moduleCatalogH.GetModuleBySlug) // Get module by slug
	registry.Get("/modules/:id", moduleCatalogH.GetModuleByID)           // Get module by ID

	// Public tool catalog (no authentication)
	registry.Get("/tools", toolCatalogH.ListPublicTools)           // List public tools
	registry.Get("/tools/search", toolCatalogH.SearchTools)        // Search tools
	registry.Get("/tools/popular", toolCatalogH.GetPopularTools)   // Popular tools
	registry.Get("/tools/top-rated", toolCatalogH.GetTopRatedTools) // Top rated tools
	registry.Get("/tools/new", toolCatalogH.GetNewTools)           // New tools
	registry.Get("/tools/free", toolCatalogH.GetFreeTools)         // Free tools
	registry.Get("/tools/category/:category", toolCatalogH.GetToolsByCategory) // Tools by category
	registry.Get("/tools/slug/:slug", toolCatalogH.GetToolBySlug) // Get tool by slug
	registry.Get("/tools/:id", toolCatalogH.GetToolByID)           // Get tool by ID

	// Tenant-specific activation routes (authentication required)
	tenantRegistry := v1.Group("/registry/tenant", middleware.AuthMiddleware(cfg.Auth.JWTSecret))

	// Module activation for current tenant
	tenantRegistry.Get("/modules", activationH.GetActivatedModules)                        // List activated modules
	tenantRegistry.Post("/modules/install", activationH.InstallModule)                     // Install module
	tenantRegistry.Post("/modules/:module_id/activate", activationH.ActivateModule)        // Activate module
	tenantRegistry.Post("/modules/:module_id/deactivate", activationH.DeactivateModule)    // Deactivate module
	tenantRegistry.Delete("/modules/:module_id", activationH.UninstallModule)              // Uninstall module
	tenantRegistry.Post("/modules/:module_id/complete-setup", activationH.CompleteModuleSetup) // Complete module setup
	tenantRegistry.Get("/modules/:module_id/dependencies", activationH.CheckModuleDependencies) // Check module dependencies

	// Tool activation for current tenant
	tenantRegistry.Get("/tools", activationH.GetActivatedTools)                          // List activated tools
	tenantRegistry.Post("/tools/install", activationH.InstallTool)                       // Install tool
	tenantRegistry.Post("/tools/:tool_id/activate", activationH.ActivateTool)            // Activate tool
	tenantRegistry.Post("/tools/:tool_id/deactivate", activationH.DeactivateTool)        // Deactivate tool
	tenantRegistry.Delete("/tools/:tool_id", activationH.UninstallTool)                  // Uninstall tool
	tenantRegistry.Post("/tools/:tool_id/complete-setup", activationH.CompleteToolSetup) // Complete tool setup
	tenantRegistry.Get("/tools/:tool_id/dependencies", activationH.CheckToolDependencies) // Check tool dependencies
}
