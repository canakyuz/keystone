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
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"

	"github.com/canakyuz/keystone/internal/config"
	keystonegrpc "github.com/canakyuz/keystone/internal/grpc"
	authHandler "github.com/canakyuz/keystone/internal/handler/auth"
	blogHandler "github.com/canakyuz/keystone/internal/handler/blog"
	bookingHandler "github.com/canakyuz/keystone/internal/handler/booking"
	lessonHandler "github.com/canakyuz/keystone/internal/handler/lesson"
	operationHandler "github.com/canakyuz/keystone/internal/handler/operation"
	paymentHandler "github.com/canakyuz/keystone/internal/handler/payment"
	registryHandler "github.com/canakyuz/keystone/internal/handler/registry"
	serviceHandler "github.com/canakyuz/keystone/internal/handler/service"
	tenantHandler "github.com/canakyuz/keystone/internal/handler/tenant"
	uploadHandler "github.com/canakyuz/keystone/internal/handler/upload"
	userHandler "github.com/canakyuz/keystone/internal/handler/user"
	websiteHandler "github.com/canakyuz/keystone/internal/handler/website"
	"github.com/canakyuz/keystone/internal/middleware"
	providerPayment "github.com/canakyuz/keystone/internal/provider/payment"
	blogRepo "github.com/canakyuz/keystone/internal/repository/blog"
	bookingRepo "github.com/canakyuz/keystone/internal/repository/booking"
	lessonRepo "github.com/canakyuz/keystone/internal/repository/lesson"
	operationRepo "github.com/canakyuz/keystone/internal/repository/operation"
	paymentRepo "github.com/canakyuz/keystone/internal/repository/payment"
	registryRepo "github.com/canakyuz/keystone/internal/repository/registry"
	serviceRepo "github.com/canakyuz/keystone/internal/repository/service"
	templateRepo "github.com/canakyuz/keystone/internal/repository/template"
	tenantRepo "github.com/canakyuz/keystone/internal/repository/tenant"
	userRepo "github.com/canakyuz/keystone/internal/repository/user"
	websiteRepo "github.com/canakyuz/keystone/internal/repository/website"
	registryService "github.com/canakyuz/keystone/internal/service/registry"
	blogUsecase "github.com/canakyuz/keystone/internal/usecase/blog"
	bookingUsecase "github.com/canakyuz/keystone/internal/usecase/booking"
	lessonUsecase "github.com/canakyuz/keystone/internal/usecase/lesson"
	paymentUsecase "github.com/canakyuz/keystone/internal/usecase/payment"
	serviceUsecase "github.com/canakyuz/keystone/internal/usecase/service"
	tenantUsecase "github.com/canakyuz/keystone/internal/usecase/tenant"
	userUsecase "github.com/canakyuz/keystone/internal/usecase/user"
	websiteUsecase "github.com/canakyuz/keystone/internal/usecase/website"
	"github.com/canakyuz/keystone/pkg/database"
	pkgLogger "github.com/canakyuz/keystone/pkg/logger"
	"github.com/canakyuz/keystone/pkg/metrics"
	"github.com/canakyuz/keystone/pkg/ratelimit"
	"github.com/canakyuz/keystone/pkg/tracing"
	"github.com/canakyuz/keystone/pkg/validator"
)

// Application holds the core dependencies: configuration, the database connection
// and the web framework.
type Application struct {
	config *config.Config
	db     *sql.DB
	app    *fiber.App

	// grpcServer is nil when no address is configured.
	grpcServer *keystonegrpc.Server

	// shutdownTracing flushes buffered spans. Spans are exported in batches, so without
	// this the last few seconds of a run are lost — which is the window containing
	// whatever made someone restart the process.
	shutdownTracing func(context.Context) error
}

// NewApplication builds and wires an application instance: database, logging,
// middleware, repositories, services, handlers and routes. This is the composition
// root.
func NewApplication(cfg *config.Config) (*Application, error) {
	// Start the database.
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("could not connect to the database: %w", err)
	}

	// Redis client, backing the L2 cache tier.
	// Connection pooling: Default 10 connections
	// Check the connection with a ping.
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test the Redis connection.
	// Root context, no timeout.
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("redis connection failed: %v (cache disabled, falling back to the database)", err)
		// A Redis failure is not fatal; the database fallback covers it.
	} else {
		log.Println("redis connected")
	}

	// Create the Fiber application.
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorHandler: customErrorHandler,
	})

	appLogger := pkgLogger.New(pkgLogger.Config{
		Level:       cfg.Server.Environment,
		Environment: cfg.Server.Environment,
	})

	// The limiter and the plan cache are built before the rate limit middleware.
	tenantPlanCache := middleware.NewTenantPlanCache(redisClient, db)

	// If Redis is unreachable we fall back to the in-process limiter. That is correct
	// on a single replica; across several replicas the enforced limit is multiplied by
	// the replica count. It is a fallback, not the intended configuration.
	var rateLimiter ratelimit.Limiter = ratelimit.NewMemory()
	if redisClient != nil {
		rateLimiter = ratelimit.NewRedis(redisClient, "ratelimit:")
	}

	// The metrics registry is built before anything that records into it.
	metricsRegistry := metrics.New()

	// Tracing is installed before the middleware that starts spans. With no endpoint
	// configured this installs the propagator and a no-op tracer, so an untraced
	// deployment still forwards a caller's trace context rather than breaking it.
	shutdownTracing, err := tracing.Init(context.Background(), tracing.Config{
		Endpoint:    cfg.Tracing.Endpoint,
		ServiceName: "keystone-api",
		Environment: cfg.Server.Environment,
		SampleRatio: cfg.Tracing.SampleRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialise tracing: %w", err)
	}

	registerProbes(app, cfg, db, redisClient)
	registerMetrics(app, metricsRegistry)

	// Global middleware, run for every incoming request.
	//
	// The order matters. Metrics come first so that a request rejected by the rate
	// limiter is still counted: during an incident, when most requests are being
	// rejected, is exactly when the dashboards must not disagree with reality.
	app.Use(recover.New()) // Turns a panic into a 500 instead of a crash.
	app.Use(middleware.Metrics(metricsRegistry))
	app.Use(middleware.Tracing("keystone-api"))
	app.Use(helmet.New()) // Baseline security headers.

	// Structured request logging with a correlation id. The id is taken from the
	// incoming X-Request-ID when the caller supplies one, so a trace started upstream is
	// not broken here, and generated otherwise.
	app.Use(middleware.Logger(appLogger))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Security.AllowedOrigins,
		AllowCredentials: cfg.Security.AllowCredentials,
	}))
	// Rate limiting. A shared token bucket is used instead of Fiber's built-in limiter:
	// the built-in one counts in-process by default, so across three replicas it
	// enforces three times the configured limit. Here the limit also follows the
	// tenant's plan, and the key is the tenant rather than the IP.
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Scope:   middleware.ScopeIP,
		Limiter: rateLimiter,
		Plans:   tenantPlanCache,
		Metrics: metricsRegistry,
		SkipPaths: map[string]bool{
			"/health": true,
			"/ready":  true,
		},
		Logger: appLogger,
	}))

	// Middleware validating requests against the OpenAPI definition.
	openAPIMiddleware, err := newOpenAPIMiddleware("api/openapi.yaml")
	if err != nil {
		return nil, fmt.Errorf("could not create the OpenAPI middleware: %w", err)
	}
	app.Use(openAPIMiddleware)

	// The plan-based limit runs inside the authenticated groups, not here. It needs the
	// tenant, and the tenant is not known until the authenticator has run.
	planRateLimit := middleware.RateLimit(middleware.RateLimitConfig{
		Scope:   middleware.ScopeTenant,
		Limiter: rateLimiter,
		Plans:   tenantPlanCache,
		Metrics: metricsRegistry,
		Logger:  appLogger,
	})

	// Tenant provisioning and operation lookup endpoints.
	//
	// Registered here, after the global middleware, so they are traced, counted, logged
	// and rate limited like everything else. They used to be registered before it, which
	// in Fiber means the middleware never runs for them: the single most important
	// endpoint in a control plane was the one with no observability and no rate limit,
	// and the operation rows it created carried neither a request id nor a trace context.
	//
	// What they still do not use is tenantContextMiddleware, which is applied per group
	// rather than globally. That exemption is the real constraint — the tenant does not
	// exist yet, so resolving its schema would fail — and it survives this move.
	operationRepository := operationRepo.New(db)
	registerOperationRoutes(app, cfg.Auth.JWTSecret,
		operationHandler.New(operationRepository, appLogger))

	// Note: tenantContextMiddleware is added after authentication, because extracting
	// tenant_id from the JWT requires the auth middleware to have run first.

	// Build the shared dependencies.
	appValidator := validator.New()

	// The repository layer, which talks to the database directly.
	tenantRepository := tenantRepo.NewPostgresRepository(db)
	tenantConnectionManager := database.NewTenantConnectionManager(db, appLogger)

	// Tenant schema cache: Redis with a database fallback.
	// TTL is 10 minutes: a tenant's schema rarely changes.
	tenantSchemaCache := middleware.NewTenantSchemaCache(redisClient, db, appLogger, metricsRegistry)

	userRepository := userRepo.NewPostgresRepository(db, tenantConnectionManager)
	websiteRepository := websiteRepo.NewPostgresRepository(db)

	// Repositories for the lessons module.
	studentRepository := lessonRepo.NewStudentPostgresRepository(db)
	lessonRepository := lessonRepo.NewLessonPostgresRepository(db)
	assignmentRepository := lessonRepo.NewAssignmentPostgresRepository(db)

	// Repositories for the booking module.
	availabilityRepository := bookingRepo.NewAvailabilityPostgresRepository(db)
	appointmentRepository := bookingRepo.NewAppointmentPostgresRepository(db)

	// Repositories for the services module.
	serviceRepository := serviceRepo.NewServicePostgresRepository(db)

	// Repositories for the blog module.
	postRepository := blogRepo.NewPostRepository(db)
	categoryRepository := blogRepo.NewCategoryRepository(db)

	// Repository for the payment module.
	paymentRepository := paymentRepo.NewPostgresRepository(db)

	// Repositories for the registry module.
	moduleRepository := registryRepo.NewModuleRepository(db)
	toolRepository := registryRepo.NewToolRepository(db)
	tenantModuleRepository := registryRepo.NewTenantModuleRepository(db)
	tenantToolRepository := registryRepo.NewTenantToolRepository(db)

	// The payment orchestrator, which fronts several providers (Iyzico, Checkout.com).
	paymentOrchestrator := providerPayment.NewOrchestrator(&cfg.Payment)

	// Register the payment providers enabled in the configuration.
	if cfg.Payment.Iyzico.Enabled {
		iyzicoProvider := providerPayment.NewIyzicoProvider(&cfg.Payment.Iyzico)
		paymentOrchestrator.RegisterProvider("iyzico", iyzicoProvider)
	}
	if cfg.Payment.Checkout.Enabled {
		checkoutProvider := providerPayment.NewCheckoutProvider(&cfg.Payment.Checkout)
		paymentOrchestrator.RegisterProvider("checkout", checkoutProvider)
	}

	// The usecase layer, holding the business rules.
	schemaTemplateRepository := templateRepo.NewFileSystemRepository("templates/tenants")
	tenantProvisioningService := tenantUsecase.NewProvisioningService(db, schemaTemplateRepository, appLogger)
	tenantService := tenantUsecase.NewService(tenantRepository, appValidator, appLogger, tenantProvisioningService)
	userService := userUsecase.NewService(userRepository, appValidator, appLogger, cfg.Auth.JWTSecret)
	websiteService := websiteUsecase.NewService(websiteRepository)

	// Services for the lessons module.
	studentService := lessonUsecase.NewStudentService(studentRepository, *appLogger)
	lessonService := lessonUsecase.NewLessonService(lessonRepository, *appLogger)
	assignmentService := lessonUsecase.NewAssignmentService(assignmentRepository, *appLogger)

	// Services for the booking module.
	availabilityService := bookingUsecase.NewAvailabilityService(availabilityRepository, *appLogger)
	appointmentService := bookingUsecase.NewAppointmentService(appointmentRepository, *appLogger)

	// Services for the services module.
	serviceService := serviceUsecase.NewServiceService(serviceRepository, *appLogger)

	// Services for the blog module.
	postService := blogUsecase.NewPostService(postRepository, *appLogger)
	categoryService := blogUsecase.NewCategoryService(categoryRepository, *appLogger)

	// Payment service.
	paymentService := paymentUsecase.NewService(paymentRepository, paymentOrchestrator)

	// Registry services.
	moduleCatalogService := registryService.NewModuleCatalogService(moduleRepository)
	toolCatalogService := registryService.NewToolCatalogService(toolRepository)
	dependencyCheckerService := registryService.NewDependencyCheckerService(db, moduleRepository, toolRepository, tenantModuleRepository, tenantToolRepository)
	tenantActivationService := registryService.NewTenantActivationService(moduleRepository, toolRepository, tenantModuleRepository, tenantToolRepository, dependencyCheckerService)

	// Tenant context middleware: tenant isolation on every request.
	// Cache-aware: schema lookups go through the cache rather than the database.
	//
	// SECURITY: selecting the tenant through the X-Tenant-ID header is only enabled in
	// development. In production the verified JWT claim is the only valid source;
	// otherwise any user holding a valid token could switch to another tenant.
	isDevelopment := cfg.Server.Environment == "development"
	middleware.AllowUntrustedTenantSource(isDevelopment)
	if !isDevelopment {
		appLogger.Info("tenant source restricted to the JWT claim")
	}

	tenantContextMiddleware := middleware.TenantContextMiddleware(tenantSchemaCache)

	// The legacy tenant manager, kept for backward compatibility.
	tenantManager := database.NewTenantManager(db)
	tenantScopeMiddleware := middleware.TenantScope(tenantRepository, tenantManager)

	// The handler layer: it takes HTTP requests, calls the services and writes
	// responses.
	authHTTPHandler := authHandler.NewHandler(userService)
	tenantHTTPHandler := tenantHandler.NewHandler(tenantService)
	userHTTPHandler := userHandler.NewHandler(userService)
	websiteHTTPHandler := websiteHandler.NewHandler(websiteService)

	// Handlers for the lessons module.
	studentHTTPHandler := lessonHandler.NewStudentHandler(studentService)
	lessonHTTPHandler := lessonHandler.NewLessonHandler(lessonService)
	assignmentHTTPHandler := lessonHandler.NewAssignmentHandler(assignmentService)

	// Handlers for the booking module.
	availabilityHTTPHandler := bookingHandler.NewAvailabilityHandler(availabilityService)
	appointmentHTTPHandler := bookingHandler.NewAppointmentHandler(appointmentService)

	// Handlers for the services module.
	serviceHTTPHandler := serviceHandler.NewServiceHandler(serviceService)

	// Handlers for the blog module.
	postHTTPHandler := blogHandler.NewPostHandler(postService, *appLogger)
	categoryHTTPHandler := blogHandler.NewCategoryHandler(categoryService, *appLogger)

	// Upload handler.
	uploadHTTPHandler := uploadHandler.NewHandler(appLogger)

	// Payment handlers.
	paymentHTTPHandler := paymentHandler.NewHandler(paymentService)
	webhookHTTPHandler := paymentHandler.NewWebhookHandler(paymentService)

	// Registry handlers.
	moduleCatalogHTTPHandler := registryHandler.NewModuleCatalogHandler(moduleCatalogService)
	toolCatalogHTTPHandler := registryHandler.NewToolCatalogHandler(toolCatalogService)
	activationHTTPHandler := registryHandler.NewActivationHandler(tenantActivationService, dependencyCheckerService)

	// Wire the routes.
	setupRoutes(app, cfg, authHTTPHandler, tenantHTTPHandler, userHTTPHandler, uploadHTTPHandler, websiteHTTPHandler,
		studentHTTPHandler, lessonHTTPHandler, assignmentHTTPHandler,
		availabilityHTTPHandler, appointmentHTTPHandler,
		serviceHTTPHandler,
		postHTTPHandler, categoryHTTPHandler,
		paymentHTTPHandler, webhookHTTPHandler,
		moduleCatalogHTTPHandler, toolCatalogHTTPHandler, activationHTTPHandler,
		tenantContextMiddleware, tenantScopeMiddleware, planRateLimit)

	// The typed surface runs in this process, on its own port. It calls the same
	// repositories as the REST handlers, so the guarantees have one implementation and two
	// ways in; see internal/grpc.
	var grpcServer *keystonegrpc.Server
	if cfg.GRPC.Addr != "" {
		grpcServer = keystonegrpc.New(
			keystonegrpc.Config{
				Addr:       cfg.GRPC.Addr,
				JWTSecret:  cfg.Auth.JWTSecret,
				Reflection: cfg.GRPC.Reflection,
			},
			tenantSchemaCache,
			keystonegrpc.NewOperationService(operationRepository),
			keystonegrpc.NewUserService(userRepository),
			metricsRegistry,
			appLogger,
		)
	}

	// Return the assembled application.
	return &Application{
		grpcServer:      grpcServer,
		shutdownTracing: shutdownTracing,
		config:          cfg,
		db:              db,
		app:             app,
	}, nil
}

// Start runs the application server.
func (a *Application) Start() error {
	// Graceful shutdown: give in-flight requests time to finish when the server stops.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // Listen for interrupt (Ctrl+C) and terminate signals.

	// The gRPC listener binds synchronously, before the HTTP one starts, so a port
	// already in use is reported at startup rather than swallowed by a goroutine, where it
	// would leave a process that looks healthy and answers nothing on one of its ports.
	if a.grpcServer != nil {
		if err := a.grpcServer.Start(); err != nil {
			return fmt.Errorf("could not start the grpc server: %w", err)
		}
	}

	// Start the server in its own goroutine so the main one is not blocked.
	go func() {
		addr := fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port)
		log.Printf("starting server on %s (environment: %s)", addr, a.config.Server.Environment)
		if err := a.app.Listen(addr); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// Wait until the shutdown signal arrives.
	<-quit
	log.Println("shutting down server")

	// Shut the Fiber server down gracefully.
	if err := a.app.Shutdown(); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// Drain the gRPC server alongside the HTTP one.
	if a.grpcServer != nil {
		a.grpcServer.Stop(10 * time.Second)
	}

	// Flush the buffered spans before the process exits.
	if a.shutdownTracing != nil {
		flushCtx, cancelFlush := context.WithTimeout(context.Background(), 5*time.Second)
		if err := a.shutdownTracing(flushCtx); err != nil {
			log.Printf("could not flush traces: %v", err)
		}
		cancelFlush()
	}

	// Close the database connection.
	if err := database.Close(a.db); err != nil {
		return fmt.Errorf("database shutdown failed: %w", err)
	}

	log.Println("server stopped gracefully")
	return nil
}

// customErrorHandler catches errors from anywhere in the application and writes a
// JSON response in a consistent shape.
func customErrorHandler(c *fiber.Ctx, err error) error {
	// The default status is 500.
	code := fiber.StatusInternalServerError

	// If this is a Fiber error, use the status it carries (404, for instance).
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Write the JSON error response.
	return c.Status(code).JSON(fiber.Map{
		"error":  err.Error(),
		"code":   code,
		"path":   c.Path(),
		"method": c.Method(),
	})
}
