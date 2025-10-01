package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"nexspaces-api/internal/config"
	authHandler "nexspaces-api/internal/handler/auth"
	lessonHandler "nexspaces-api/internal/handler/lesson"
	tenantHandler "nexspaces-api/internal/handler/tenant"
	userHandler "nexspaces-api/internal/handler/user"
	websiteHandler "nexspaces-api/internal/handler/website"
	lessonRepo "nexspaces-api/internal/repository/lesson"
	tenantRepo "nexspaces-api/internal/repository/tenant"
	userRepo "nexspaces-api/internal/repository/user"
	websiteRepo "nexspaces-api/internal/repository/website"
	lessonUsecase "nexspaces-api/internal/usecase/lesson"
	tenantUsecase "nexspaces-api/internal/usecase/tenant"
	userUsecase "nexspaces-api/internal/usecase/user"
	websiteUsecase "nexspaces-api/internal/usecase/website"
	"nexspaces-api/pkg/database"
	pkgLogger "nexspaces-api/pkg/logger"
	"nexspaces-api/pkg/validator"
)

// Application holds the application dependencies
type Application struct {
	config *config.Config
	db     *sql.DB
	app    *fiber.App
}

// NewApplication creates and initializes a new application
func NewApplication(cfg *config.Config) (*Application, error) {
	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Security.AllowedOrigins,
		AllowCredentials: cfg.Security.AllowCredentials,
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        cfg.Security.RateLimit.Requests,
		Expiration: cfg.Security.RateLimit.Duration,
	}))

	// Initialize shared dependencies
	appLogger := pkgLogger.New(pkgLogger.Config{
		Level:       cfg.Server.Environment,
		Environment: cfg.Server.Environment,
	})
	appValidator := validator.New()

	// Initialize repositories
	tenantRepository := tenantRepo.NewPostgresRepository(db)
	userRepository := userRepo.NewPostgresRepository(db)
	websiteRepository := websiteRepo.NewPostgresRepository(db)

	// Lesson module repositories
	studentRepository := lessonRepo.NewStudentPostgresRepository(db)
	lessonRepository := lessonRepo.NewLessonPostgresRepository(db)
	assignmentRepository := lessonRepo.NewAssignmentPostgresRepository(db)

	// Initialize services
	tenantService := tenantUsecase.NewService(tenantRepository, appValidator, appLogger)
	userService := userUsecase.NewService(userRepository, appValidator, appLogger, cfg.Auth.JWTSecret)
	websiteService := websiteUsecase.NewService(websiteRepository)

	// Lesson module services
	studentService := lessonUsecase.NewStudentService(studentRepository, *appLogger)
	lessonService := lessonUsecase.NewLessonService(lessonRepository, *appLogger)
	assignmentService := lessonUsecase.NewAssignmentService(assignmentRepository, *appLogger)

	// Initialize HTTP handlers
	authHTTPHandler := authHandler.NewHandler(userService)
	tenantHTTPHandler := tenantHandler.NewHandler(tenantService)
	userHTTPHandler := userHandler.NewHandler(userService)
	websiteHTTPHandler := websiteHandler.NewHandler(websiteService)

	// Lesson module handlers
	studentHTTPHandler := lessonHandler.NewStudentHandler(studentService)
	lessonHTTPHandler := lessonHandler.NewLessonHandler(lessonService)
	assignmentHTTPHandler := lessonHandler.NewAssignmentHandler(assignmentService)

	// Setup routes
	setupRoutes(app, cfg, authHTTPHandler, tenantHTTPHandler, userHTTPHandler, websiteHTTPHandler,
		studentHTTPHandler, lessonHTTPHandler, assignmentHTTPHandler)

	return &Application{
		config: cfg,
		db:     db,
		app:    app,
	}, nil
}

// Start starts the application server
func (a *Application) Start() error {
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port)
		log.Printf("🚀 Server starting on %s (environment: %s)", addr, a.config.Server.Environment)
		if err := a.app.Listen(addr); err != nil {
			log.Printf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	if err := a.app.Shutdown(); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// Close database connection
	if err := database.Close(a.db); err != nil {
		return fmt.Errorf("database close error: %w", err)
	}

	log.Println("✅ Server gracefully stopped")
	return nil
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   err.Error(),
		"code":    code,
		"path":    c.Path(),
		"method":  c.Method(),
	})
}
