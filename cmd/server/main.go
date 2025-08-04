package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"nexspaces-api/internal/auth"
	"nexspaces-api/internal/config"
	"nexspaces-api/internal/handlers"
	"nexspaces-api/internal/middleware"
	"nexspaces-api/pkg/database"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database connections
	db, err := database.NewPostgreSQL(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	defer db.Close()

	redis, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redis.Close()

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWT.Secret)
	authMiddleware := middleware.NewAuthMiddleware(jwtService, redis)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, redis, jwtService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Global middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Tenant-ID, X-Tenant-Slug",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	app.Use(authMiddleware.TenantResolver)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		// Check database health
		if err := db.Health(); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "error",
				"service": "nexspaces-api",
				"database": "unhealthy",
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "nexspaces-api",
			"database": "healthy",
		})
	})

	// API routes
	api := app.Group("/api/v1")
	
	// Public auth routes (no auth required)
	auth := api.Group("/auth")
	auth.Post("/validate", authHandler.ValidateToken)     // Validate Better Auth token
	auth.Post("/refresh", authHandler.RefreshToken)       // Refresh token
	
	// Protected auth routes (auth required)
	authProtected := api.Group("/auth", authMiddleware.RequireAuth)
	authProtected.Get("/me", authHandler.Me)              // Get current user
	authProtected.Post("/logout", authHandler.Logout)     // Logout
	authProtected.Post("/switch-tenant", authHandler.SwitchTenant) // Switch tenant
	
	// Tenant routes (public)
	tenants := api.Group("/tenants")
	tenants.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Get tenants - public"})
	})
	
	// Protected tenant routes
	tenantsProtected := api.Group("/tenants", authMiddleware.RequireAuth)
	tenantsProtected.Post("/", authMiddleware.RequireRole("admin", "owner"), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Create tenant - admin only"})
	})
	tenantsProtected.Get("/:tenantId", authMiddleware.RequireTenantAccess, func(c *fiber.Ctx) error {
		tenantSlug := c.Locals("tenant_slug")
		return c.JSON(fiber.Map{
			"message": "Get tenant details",
			"tenant_id": c.Params("tenantId"),
			"tenant_slug": tenantSlug,
		})
	})

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("🛑 Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	// Start server
	log.Printf("🚀 Server starting on port %s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Println("Server stopped")
	}
}