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

	"nexpaces-api/internal/config"
	authHandler "nexpaces-api/internal/handler/auth"
	blogHandler "nexpaces-api/internal/handler/blog"
	bookingHandler "nexpaces-api/internal/handler/booking"
	lessonHandler "nexpaces-api/internal/handler/lesson"
	paymentHandler "nexpaces-api/internal/handler/payment"
	serviceHandler "nexpaces-api/internal/handler/service"
	tenantHandler "nexpaces-api/internal/handler/tenant"
	uploadHandler "nexpaces-api/internal/handler/upload"
	userHandler "nexpaces-api/internal/handler/user"
	websiteHandler "nexpaces-api/internal/handler/website"
	blogRepo "nexpaces-api/internal/repository/blog"
	bookingRepo "nexpaces-api/internal/repository/booking"
	lessonRepo "nexpaces-api/internal/repository/lesson"
	paymentRepo "nexpaces-api/internal/repository/payment"
	serviceRepo "nexpaces-api/internal/repository/service"
	tenantRepo "nexpaces-api/internal/repository/tenant"
	userRepo "nexpaces-api/internal/repository/user"
	websiteRepo "nexpaces-api/internal/repository/website"
	blogUsecase "nexpaces-api/internal/usecase/blog"
	bookingUsecase "nexpaces-api/internal/usecase/booking"
	lessonUsecase "nexpaces-api/internal/usecase/lesson"
	paymentUsecase "nexpaces-api/internal/usecase/payment"
	serviceUsecase "nexpaces-api/internal/usecase/service"
	tenantUsecase "nexpaces-api/internal/usecase/tenant"
	userUsecase "nexpaces-api/internal/usecase/user"
	websiteUsecase "nexpaces-api/internal/usecase/website"
	providerPayment "nexpaces-api/internal/provider/payment"
	"nexpaces-api/pkg/database"
	pkgLogger "nexpaces-api/pkg/logger"
	"nexpaces-api/pkg/validator"
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

	openAPIMiddleware, err := newOpenAPIMiddleware("api/openapi.yaml")
	if err != nil {
		return nil, fmt.Errorf("setup OpenAPI middleware: %w", err)
	}
	app.Use(openAPIMiddleware)

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

	// Booking module repositories
	availabilityRepository := bookingRepo.NewAvailabilityPostgresRepository(db)
	appointmentRepository := bookingRepo.NewAppointmentPostgresRepository(db)

	// Service module repositories
	serviceRepository := serviceRepo.NewServicePostgresRepository(db)

	// Blog module repositories
	postRepository := blogRepo.NewPostRepository(db)
	categoryRepository := blogRepo.NewCategoryRepository(db)

	// Payment module repository
	paymentRepository := paymentRepo.NewPostgresRepository(db)

	// Initialize payment orchestrator
	paymentOrchestrator := providerPayment.NewOrchestrator(&cfg.Payment)

	// Register payment providers
	if cfg.Payment.Iyzico.Enabled {
		iyzicoProvider := providerPayment.NewIyzicoProvider(&cfg.Payment.Iyzico)
		paymentOrchestrator.RegisterProvider("iyzico", iyzicoProvider)
	}
	if cfg.Payment.Checkout.Enabled {
		checkoutProvider := providerPayment.NewCheckoutProvider(&cfg.Payment.Checkout)
		paymentOrchestrator.RegisterProvider("checkout", checkoutProvider)
	}

	// Initialize services
	tenantService := tenantUsecase.NewService(tenantRepository, appValidator, appLogger)
	userService := userUsecase.NewService(userRepository, appValidator, appLogger, cfg.Auth.JWTSecret)
	websiteService := websiteUsecase.NewService(websiteRepository)

	// Lesson module services
	studentService := lessonUsecase.NewStudentService(studentRepository, *appLogger)
	lessonService := lessonUsecase.NewLessonService(lessonRepository, *appLogger)
	assignmentService := lessonUsecase.NewAssignmentService(assignmentRepository, *appLogger)

	// Booking module services
	availabilityService := bookingUsecase.NewAvailabilityService(availabilityRepository, *appLogger)
	appointmentService := bookingUsecase.NewAppointmentService(appointmentRepository, *appLogger)

	// Service module services
	serviceService := serviceUsecase.NewServiceService(serviceRepository, *appLogger)

	// Blog module services
	postService := blogUsecase.NewPostService(postRepository, *appLogger)
	categoryService := blogUsecase.NewCategoryService(categoryRepository, *appLogger)

	// Payment service
	paymentService := paymentUsecase.NewService(paymentRepository, paymentOrchestrator)

	// Initialize HTTP handlers
	authHTTPHandler := authHandler.NewHandler(userService)
	tenantHTTPHandler := tenantHandler.NewHandler(tenantService)
	userHTTPHandler := userHandler.NewHandler(userService)
	websiteHTTPHandler := websiteHandler.NewHandler(websiteService)

	// Lesson module handlers
	studentHTTPHandler := lessonHandler.NewStudentHandler(studentService)
	lessonHTTPHandler := lessonHandler.NewLessonHandler(lessonService)
	assignmentHTTPHandler := lessonHandler.NewAssignmentHandler(assignmentService)

	// Booking module handlers
	availabilityHTTPHandler := bookingHandler.NewAvailabilityHandler(availabilityService)
	appointmentHTTPHandler := bookingHandler.NewAppointmentHandler(appointmentService)

	// Service module handlers
	serviceHTTPHandler := serviceHandler.NewServiceHandler(serviceService)

	// Blog module handlers
	postHTTPHandler := blogHandler.NewPostHandler(postService, *appLogger)
	categoryHTTPHandler := blogHandler.NewCategoryHandler(categoryService, *appLogger)

	// Upload handler
	uploadHTTPHandler := uploadHandler.NewHandler(appLogger)

	// Payment handlers
	paymentHTTPHandler := paymentHandler.NewHandler(paymentService)
	webhookHTTPHandler := paymentHandler.NewWebhookHandler(paymentService)

	// Setup routes
	setupRoutes(app, cfg, authHTTPHandler, tenantHTTPHandler, userHTTPHandler, uploadHTTPHandler, websiteHTTPHandler,
		studentHTTPHandler, lessonHTTPHandler, assignmentHTTPHandler,
		availabilityHTTPHandler, appointmentHTTPHandler,
		serviceHTTPHandler,
		postHTTPHandler, categoryHTTPHandler,
		paymentHTTPHandler, webhookHTTPHandler)

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
		"error":  err.Error(),
		"code":   code,
		"path":   c.Path(),
		"method": c.Method(),
	})
}
