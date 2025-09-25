package app

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	swagger "github.com/gofiber/swagger"

	"nexspaces-api/internal/config"
)

type Application struct {
	config *config.Config
	app    *fiber.App
}

func NewApplication(cfg *config.Config) (*Application, error) {
	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
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

	// Add global middleware
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Security.AllowedOrigins,
		AllowCredentials: cfg.Security.AllowCredentials,
	}))

	// Rate limiting
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.Security.RateLimit.Requests,
		Expiration: cfg.Security.RateLimit.Duration,
	}))

	// Setup routes
	setupRoutes(app)

	return &Application{
		config: cfg,
		app:    app,
	}, nil
}

func (a *Application) Start() error {
	// Create a channel to receive OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port)
		log.Printf("Server starting on %s", addr)
		if err := a.app.Listen(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	return a.app.Shutdown()
}

func setupRoutes(app *fiber.App) {
	// Swagger docs
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check
	// @Summary Show the status of server.
	// @Description get the status of server.
	// @Tags root
	// @Accept */*
	// @Produce json
	// @Success 200 {object} map[string]interface{}
	// @Router /health [get]
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "NexSpaces API is running",
		})
	})

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Authentication routes
	auth := v1.Group("/auth")
	auth.Post("/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Login endpoint - TODO"})
	})
	auth.Post("/register", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Register endpoint - TODO"})
	})
	auth.Post("/logout", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Logout endpoint - TODO"})
	})

	// Protected routes (require authentication)
	protected := v1.Group("/")

	// Tenant routes
	tenants := protected.Group("/tenants")
	tenants.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Get tenants - TODO"})
	})
	tenants.Post("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Create tenant - TODO"})
	})

	// User routes
	users := protected.Group("/users")
	users.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Get users - TODO"})
	})
	users.Post("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Create user - TODO"})
	})

	// Template routes
	templates := protected.Group("/templates")
	templates.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Get templates - TODO"})
	})
	templates.Post("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Create template - TODO"})
	})
	templates.Get("/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Get template by ID - TODO"})
	})
}
