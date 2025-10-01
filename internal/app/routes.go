package app

import (
	"github.com/gofiber/fiber/v2"
	"nexspaces-api/internal/config"
	authHandler "nexspaces-api/internal/handler/auth"
	tenantHandler "nexspaces-api/internal/handler/tenant"
	userHandler "nexspaces-api/internal/handler/user"
	websiteHandler "nexspaces-api/internal/handler/website"
	"nexspaces-api/internal/middleware"
)

// setupRoutes configures all application routes
func setupRoutes(
	app *fiber.App,
	cfg *config.Config,
	authH *authHandler.Handler,
	tenantH *tenantHandler.Handler,
	userH *userHandler.Handler,
	websiteH *websiteHandler.Handler,
) {
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
	tenants.Post("/", tenantH.Create)                                    // Create tenant
	tenants.Get("/current", tenantH.GetCurrent)                          // Get current tenant
	tenants.Get("/stats", tenantH.GetStats)                              // Tenant statistics
	tenants.Get("/slug/:slug", tenantH.GetBySlug)                        // Get by slug
	tenants.Get("/:id", tenantH.GetByID)                                 // Get by ID
	tenants.Get("/", tenantH.List)                                       // List tenants
	tenants.Patch("/:id", tenantH.Update)                                // Update tenant
	tenants.Post("/:id/suspend", tenantH.Suspend)                        // Suspend tenant
	tenants.Post("/:id/activate", tenantH.Activate)                      // Activate tenant
	tenants.Post("/:id/upgrade", tenantH.UpgradePlan)                    // Upgrade plan
	tenants.Post("/:id/domain", tenantH.SetCustomDomain)                 // Set custom domain
	tenants.Post("/:id/domain/verify", tenantH.VerifyCustomDomain)       // Verify domain
	tenants.Delete("/:id", tenantH.Delete)                               // Delete tenant

	// User routes (authentication required)
	users := v1.Group("/users", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	users.Post("/", userH.Create)                            // Create user (admin only)
	users.Get("/stats", userH.GetStats)                      // User statistics
	users.Get("/:id", userH.GetByID)                         // Get user by ID
	users.Get("/", userH.List)                               // List users
	users.Patch("/:id", userH.Update)                        // Update user
	users.Post("/:id/password", userH.UpdatePassword)        // Update password
	users.Post("/:id/role", userH.UpdateRole)                // Update role (admin only)
	users.Post("/:id/suspend", userH.Suspend)                // Suspend user (admin only)
	users.Post("/:id/activate", userH.Activate)              // Activate user (admin only)
	users.Post("/:id/verify-email", userH.VerifyEmail)       // Verify email
	users.Delete("/:id", userH.Delete)                       // Delete user (admin only)

	// Website routes (tenant-scoped)
	websites := v1.Group("/websites", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	websites.Get("/", websiteH.List)
	websites.Post("/", websiteH.Create)
	websites.Get("/:id", websiteH.GetByID)
	websites.Patch("/:id", websiteH.Update)
	websites.Delete("/:id", websiteH.Delete)
	websites.Post("/:id/publish", websiteH.Publish)
	websites.Post("/:id/archive", websiteH.Archive)
}
