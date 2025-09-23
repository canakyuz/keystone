package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/redis/v3"
	_ "github.com/lib/pq"

	"nexspaces-api/internal/adapters/http/handlers"
	"nexspaces-api/internal/adapters/http/middleware"
	"nexspaces-api/internal/adapters/persistence/postgres"
	"nexspaces-api/internal/config"
	"nexspaces-api/internal/core/usecases/tenant"
	"nexspaces-api/internal/core/usecases/user"
	templateUC "nexspaces-api/internal/core/usecases/template"
	subscriptionUC "nexspaces-api/internal/core/usecases/subscription"
	"nexspaces-api/internal/shared/validation"
)

// Application represents the main application
type Application struct {
	config     *config.Config
	fiber      *fiber.App
	db         *sql.DB
	redis      *redis.Storage
	validator  *validation.Validator
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config) (*Application, error) {
	app := &Application{
		config:    cfg,
		validator: validation.NewValidator(),
	}

	if err := app.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := app.initRedis(); err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	if err := app.initFiber(); err != nil {
		return nil, fmt.Errorf("failed to initialize Fiber: %w", err)
	}

	if err := app.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	return app, nil
}

// initDatabase initializes PostgreSQL database connection
func (a *Application) initDatabase() error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		a.config.Database.Host,
		a.config.Database.Port,
		a.config.Database.User,
		a.config.Database.Password,
		a.config.Database.Name,
		a.config.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(a.config.Database.MaxOpenConns)
	db.SetMaxIdleConns(a.config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(a.config.Database.ConnMaxLifetime) * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	log.Println("✅ Database connection established")
	return nil
}

// initRedis initializes Redis connection
func (a *Application) initRedis() error {
	if a.config.Redis.URL == "" {
		log.Println("⚠️ Redis URL not configured, using in-memory storage")
		return nil
	}

	redisStore := redis.New(redis.Config{
		URL:      a.config.Redis.URL,
		Database: a.config.Redis.Database,
		Username: a.config.Redis.Username,
		Password: a.config.Redis.Password,
	})

	a.redis = redisStore
	log.Println("✅ Redis connection established")
	return nil
}

// initFiber initializes the Fiber app with middleware
func (a *Application) initFiber() error {
	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		AppName:                 "NexSpaces API",
		ErrorHandler:            middleware.ErrorHandlerMiddleware(),
		DisableStartupMessage:   a.config.Server.Environment == "production",
		BodyLimit:               10 * 1024 * 1024, // 10MB
		ReadTimeout:             30 * time.Second,
		WriteTimeout:            30 * time.Second,
		IdleTimeout:             120 * time.Second,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          a.config.Security.TrustedProxies,
	})

	// Apply security middleware
	securityConfig := middleware.SecurityConfig{
		AllowedOrigins:     a.config.Security.AllowedOrigins,
		AllowCredentials:   a.config.Security.AllowCredentials,
		MaxRequestSize:     a.config.Security.MaxRequestSize,
		RateLimitRequests:  a.config.Security.RateLimit.Requests,
		RateLimitDuration:  time.Duration(a.config.Security.RateLimit.Duration) * time.Minute,
		RedisURL:           a.config.Redis.URL,
		JWTSecret:          a.config.Auth.JWTSecret,
		EnableCSP:          a.config.Security.EnableCSP,
		EnableHSTS:         a.config.Security.EnableHSTS,
		TrustedProxies:     a.config.Security.TrustedProxies,
	}

	securityMiddlewares := middleware.NewSecurityMiddleware(securityConfig)
	for _, mw := range securityMiddlewares {
		app.Use(mw)
	}

	// Add custom security middleware
	app.Use(middleware.RequestValidationMiddleware())
	app.Use(middleware.XSSProtectionMiddleware())
	app.Use(middleware.SQLInjectionProtectionMiddleware())

	a.fiber = app
	log.Println("✅ Fiber app initialized with security middleware")
	return nil
}

// setupRoutes sets up all application routes
func (a *Application) setupRoutes() error {
	// Initialize repositories
	tenantRepo := postgres.NewTenantRepository(a.db)
	userRepo := postgres.NewUserRepository(a.db)
	templateRepo := postgres.NewTemplateRepository(a.db)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(userRepo, a.config.Auth.JWTSecret)
	tenantMiddleware := middleware.NewTenantMiddleware(tenantRepo)

	// Initialize use cases
	createTenantUC := tenant.NewCreateTenantUseCase(
		tenantRepo,
		userRepo,
		nil, // subscription repo
		nil, // billing service
		nil, // notification service
		nil, // event bus
	)

	createUserUC := user.NewCreateUserUseCase(
		userRepo,
		tenantRepo,
		nil, // auth service
		nil, // notification service
		nil, // event bus
	)

	updateUserUC := user.NewUpdateUserUseCase(
		userRepo,
		tenantRepo,
		nil, // event bus
	)

	deactivateUserUC := user.NewDeactivateUserUseCase(
		userRepo,
		nil, // event bus
	)

	createTemplateUC := templateUC.NewCreateTemplateUseCase(
		templateRepo,
		userRepo,
		tenantRepo,
		nil, // event bus
	)

	publishTemplateUC := templateUC.NewPublishTemplateUseCase(
		templateRepo,
		userRepo,
		nil, // event bus
	)

	installTemplateUC := templateUC.NewInstallTemplateUseCase(
		templateRepo,
		userRepo,
		tenantRepo,
		nil, // event bus
	)

	searchTemplatesUC := templateUC.NewSearchTemplatesUseCase(
		templateRepo,
		userRepo,
	)

	// Initialize handlers
	tenantHandler := handlers.NewTenantHandler(createTenantUC, a.validator)
	userHandler := handlers.NewUserHandler(createUserUC, updateUserUC, deactivateUserUC, a.validator)
	templateHandler := handlers.NewTemplateHandler(
		createTemplateUC,
		publishTemplateUC,
		installTemplateUC,
		searchTemplatesUC,
		a.validator,
	)

	// API routes
	api := a.fiber.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// Public tenant creation (no auth required)
	api.Post("/tenants", tenantHandler.CreateTenant)

	// Public template search (no auth required, but optional auth)
	api.Get("/templates/search", authMiddleware.OptionalAuth, templateHandler.SearchTemplates)

	// Tenant-specific routes (require tenant resolution from subdomain/domain)
	tenantRoutes := api.Group("/tenants/:tenantId")
	tenantRoutes.Use(tenantMiddleware.ResolveTenantFromParam("tenantId"))
	tenantRoutes.Use(authMiddleware.Authenticate)
	tenantRoutes.Use(tenantMiddleware.ValidateTenantAccess)

	// User management
	userRoutes := tenantRoutes.Group("/users")
	userRoutes.Post("/", authMiddleware.RequirePermission("manage_users"), userHandler.CreateUser)
	userRoutes.Get("/", userHandler.GetUsers)
	userRoutes.Get("/:userId", userHandler.GetUser)
	userRoutes.Put("/:userId", authMiddleware.RequirePermission("manage_users"), userHandler.UpdateUser)
	userRoutes.Delete("/:userId", authMiddleware.RequirePermission("manage_users"), userHandler.DeactivateUser)

	// Template management
	templateRoutes := tenantRoutes.Group("/templates")
	templateRoutes.Post("/", authMiddleware.RequirePermission("manage_templates"), templateHandler.CreateTemplate)
	templateRoutes.Post("/:templateId/publish", authMiddleware.RequirePermission("publish_templates"), templateHandler.PublishTemplate)
	templateRoutes.Post("/:templateId/install", authMiddleware.RequirePermission("install_templates"), templateHandler.InstallTemplate)

	log.Println("✅ Routes configured")
	return nil
}

// Start starts the application server
func (a *Application) Start() error {
	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%d", a.config.Server.Port)
		log.Printf("🚀 Server starting on port %d", a.config.Server.Port)
		log.Printf("🔗 Environment: %s", a.config.Server.Environment)

		if err := a.fiber.Listen(addr); err != nil {
			log.Printf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	return a.Shutdown()
}

// Shutdown gracefully shuts down the application
func (a *Application) Shutdown() error {
	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown Fiber server
	if err := a.fiber.ShutdownWithContext(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
		return err
	}

	// Close database connection
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("❌ Error closing database: %v", err)
			return err
		}
		log.Println("✅ Database connection closed")
	}

	// Close Redis connection
	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			log.Printf("❌ Error closing Redis: %v", err)
			return err
		}
		log.Println("✅ Redis connection closed")
	}

	log.Println("✅ Server shutdown complete")
	return nil
}

// GetDB returns the database connection
func (a *Application) GetDB() *sql.DB {
	return a.db
}

// GetFiber returns the Fiber app instance
func (a *Application) GetFiber() *fiber.App {
	return a.fiber
}