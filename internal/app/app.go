package app

import (
	"context"
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
	"github.com/redis/go-redis/v9"

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
	providerPayment "github.com/canakyuz/keystone/internal/provider/payment"
	blogRepo "github.com/canakyuz/keystone/internal/repository/blog"
	bookingRepo "github.com/canakyuz/keystone/internal/repository/booking"
	lessonRepo "github.com/canakyuz/keystone/internal/repository/lesson"
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
	"github.com/canakyuz/keystone/pkg/validator"
)

// Application struct'ı, uygulamanın temel bağımlılıklarını (konfigürasyon, veritabanı bağlantısı, web framework) bir arada tutar.
// Bu, bağımlılıkların uygulama genelinde düzenli bir şekilde yönetilmesini sağlar.
type Application struct {
	config *config.Config
	db     *sql.DB
	app    *fiber.App
}

// NewApplication, yeni bir uygulama örneği oluşturur ve başlatır.
// Bu "yapıcı" (constructor) fonksiyon, uygulamanın çalışması için gereken tüm bileşenleri (veritabanı, loglama, rotalar vb.) birbirine bağlar.
func NewApplication(cfg *config.Config) (*Application, error) {
	// Veritabanını başlat.
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("veritabanına bağlanırken hata oluştu: %w", err)
	}

	// 🎓 REDIS CLIENT: In-memory cache için
	// Connection pooling: Default 10 connections
	// Health check: Ping komutu ile bağlantı kontrolü
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Redis bağlantısını test et
	// 🎓 GO KONSEPT: context.Background() - root context, no timeout
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis bağlantısı başarısız: %v (Cache devre dışı, DB fallback aktif)", err)
		// Redis hatası fatal değil, DB fallback var
	} else {
		log.Println("✅ Redis bağlantısı başarılı")
	}

	// Fiber (web framework) uygulamasını oluştur.
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorHandler: customErrorHandler, // Hata yönetimi için özel bir fonksiyon belirle.
	})

	// Global Middleware (Ara Katman) tanımlamaları.
	// Bu middleware'ler gelen her istek için çalıştırılır.
	app.Use(recover.New())            // Panik durumlarında sunucunun çökmesini engeller ve 500 hatası döner.
	app.Use(helmet.New())             // Güvenlikle ilgili temel HTTP başlıklarını (header) ekler.
	app.Use(logger.New(logger.Config{ // Gelen istekleri konsola loglar.
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{ // Cross-Origin Resource Sharing ayarları. Farklı domain'lerden gelen isteklere izin verir.
		AllowOrigins:     cfg.Security.AllowedOrigins,
		AllowCredentials: cfg.Security.AllowCredentials,
	}))
	app.Use(limiter.New(limiter.Config{ // İstek limiti (rate limiting) uygular, brute-force saldırılarını önler.
		Max:        cfg.Security.RateLimit.Requests,
		Expiration: cfg.Security.RateLimit.Duration,
	}))

	// OpenAPI (Swagger) tanımına göre istekleri doğrulayan middleware'i ayarla.
	openAPIMiddleware, err := newOpenAPIMiddleware("api/openapi.yaml")
	if err != nil {
		return nil, fmt.Errorf("OpenAPI middleware oluşturulamadı: %w", err)
	}
	app.Use(openAPIMiddleware)

	// NOT: tenantContextMiddleware'i authentication sonrasında ekleyeceğiz
	// çünkü JWT'den tenant_id çıkarmak için önce auth middleware çalışmalı

	// Paylaşılan bağımlılıkları başlat.
	appLogger := pkgLogger.New(pkgLogger.Config{ // Uygulama genelinde kullanılacak loglama servisi.
		Level:       cfg.Server.Environment,
		Environment: cfg.Server.Environment,
	})
	appValidator := validator.New() // Veri doğrulama (validation) servisi.

	// Repository (Veri Erişim Katmanı) katmanını başlat.
	// Repository'ler veritabanı ile doğrudan iletişim kuran yapılardır.
	tenantRepository := tenantRepo.NewPostgresRepository(db)
	tenantConnectionManager := database.NewTenantConnectionManager(db, appLogger)

	// 🎓 TENANT SCHEMA CACHE: Redis + DB fallback cache layer
	// Performance: ~10-20ms latency kazancı (cache hit)
	// TTL: 10 dakika (tenant schema nadiren değişir)
	tenantSchemaCache := middleware.NewTenantSchemaCache(redisClient, db, appLogger)

	userRepository := userRepo.NewPostgresRepository(db, tenantConnectionManager)
	websiteRepository := websiteRepo.NewPostgresRepository(db)

	// Dersler modülü için repository'ler.
	studentRepository := lessonRepo.NewStudentPostgresRepository(db)
	lessonRepository := lessonRepo.NewLessonPostgresRepository(db)
	assignmentRepository := lessonRepo.NewAssignmentPostgresRepository(db)

	// Rezervasyon modülü için repository'ler.
	availabilityRepository := bookingRepo.NewAvailabilityPostgresRepository(db)
	appointmentRepository := bookingRepo.NewAppointmentPostgresRepository(db)

	// Hizmet modülü için repository'ler.
	serviceRepository := serviceRepo.NewServicePostgresRepository(db)

	// Blog modülü için repository'ler.
	postRepository := blogRepo.NewPostRepository(db)
	categoryRepository := blogRepo.NewCategoryRepository(db)

	// Ödeme modülü için repository.
	paymentRepository := paymentRepo.NewPostgresRepository(db)

	// Kayıt Merkezi (Registry) modülü için repository'ler.
	moduleRepository := registryRepo.NewModuleRepository(db)
	toolRepository := registryRepo.NewToolRepository(db)
	tenantModuleRepository := registryRepo.NewTenantModuleRepository(db)
	tenantToolRepository := registryRepo.NewTenantToolRepository(db)

	// Ödeme Orkestratörünü (Payment Orchestrator) başlat.
	// Bu yapı, birden fazla ödeme sağlayıcısını (Iyzico, Checkout.com vb.) yönetir.
	paymentOrchestrator := providerPayment.NewOrchestrator(&cfg.Payment)

	// Yapılandırmada aktif olan ödeme sağlayıcılarını kaydet.
	if cfg.Payment.Iyzico.Enabled {
		iyzicoProvider := providerPayment.NewIyzicoProvider(&cfg.Payment.Iyzico)
		paymentOrchestrator.RegisterProvider("iyzico", iyzicoProvider)
	}
	if cfg.Payment.Checkout.Enabled {
		checkoutProvider := providerPayment.NewCheckoutProvider(&cfg.Payment.Checkout)
		paymentOrchestrator.RegisterProvider("checkout", checkoutProvider)
	}

	// Service/Usecase (İş Mantığı Katmanı) katmanını başlat.
	// Usecase'ler, uygulamanın iş kurallarını ve mantığını içerir.
	schemaTemplateRepository := templateRepo.NewFileSystemRepository("templates/tenants")
	tenantProvisioningService := tenantUsecase.NewProvisioningService(db, schemaTemplateRepository, appLogger)
	tenantService := tenantUsecase.NewService(tenantRepository, appValidator, appLogger, tenantProvisioningService)
	userService := userUsecase.NewService(userRepository, appValidator, appLogger, cfg.Auth.JWTSecret)
	websiteService := websiteUsecase.NewService(websiteRepository)

	// Dersler modülü için servisler.
	studentService := lessonUsecase.NewStudentService(studentRepository, *appLogger)
	lessonService := lessonUsecase.NewLessonService(lessonRepository, *appLogger)
	assignmentService := lessonUsecase.NewAssignmentService(assignmentRepository, *appLogger)

	// Rezervasyon modülü için servisler.
	availabilityService := bookingUsecase.NewAvailabilityService(availabilityRepository, *appLogger)
	appointmentService := bookingUsecase.NewAppointmentService(appointmentRepository, *appLogger)

	// Hizmet modülü için servisler.
	serviceService := serviceUsecase.NewServiceService(serviceRepository, *appLogger)

	// Blog modülü için servisler.
	postService := blogUsecase.NewPostService(postRepository, *appLogger)
	categoryService := blogUsecase.NewCategoryService(categoryRepository, *appLogger)

	// Ödeme servisi.
	paymentService := paymentUsecase.NewService(paymentRepository, paymentOrchestrator)

	// Kayıt Merkezi (Registry) servisleri.
	moduleCatalogService := registryService.NewModuleCatalogService(moduleRepository)
	toolCatalogService := registryService.NewToolCatalogService(toolRepository)
	dependencyCheckerService := registryService.NewDependencyCheckerService(db, moduleRepository, toolRepository, tenantModuleRepository, tenantToolRepository)
	tenantActivationService := registryService.NewTenantActivationService(moduleRepository, toolRepository, tenantModuleRepository, tenantToolRepository, dependencyCheckerService)

	// Tenant context middleware (her request için tenant isolation)
	// 🎓 CACHE-AWARE: Redis cache kullanarak schema lookup performance optimize edildi
	tenantContextMiddleware := middleware.TenantContextMiddleware(tenantSchemaCache)

	// Eski tenant manager (backward compatibility)
	tenantManager := database.NewTenantManager(db)
	tenantScopeMiddleware := middleware.TenantScope(tenantRepository, tenantManager)

	// Handler (Sunum Katmanı) katmanını başlat.
	// Handler'lar, HTTP isteklerini alır, ilgili servisleri çağırır ve HTTP cevapları döner.
	authHTTPHandler := authHandler.NewHandler(userService)
	tenantHTTPHandler := tenantHandler.NewHandler(tenantService)
	userHTTPHandler := userHandler.NewHandler(userService)
	websiteHTTPHandler := websiteHandler.NewHandler(websiteService)

	// Dersler modülü için handler'lar.
	studentHTTPHandler := lessonHandler.NewStudentHandler(studentService)
	lessonHTTPHandler := lessonHandler.NewLessonHandler(lessonService)
	assignmentHTTPHandler := lessonHandler.NewAssignmentHandler(assignmentService)

	// Rezervasyon modülü için handler'lar.
	availabilityHTTPHandler := bookingHandler.NewAvailabilityHandler(availabilityService)
	appointmentHTTPHandler := bookingHandler.NewAppointmentHandler(appointmentService)

	// Hizmet modülü için handler'lar.
	serviceHTTPHandler := serviceHandler.NewServiceHandler(serviceService)

	// Blog modülü için handler'lar.
	postHTTPHandler := blogHandler.NewPostHandler(postService, *appLogger)
	categoryHTTPHandler := blogHandler.NewCategoryHandler(categoryService, *appLogger)

	// Dosya yükleme handler'ı.
	uploadHTTPHandler := uploadHandler.NewHandler(appLogger)

	// Ödeme handler'ları.
	paymentHTTPHandler := paymentHandler.NewHandler(paymentService)
	webhookHTTPHandler := paymentHandler.NewWebhookHandler(paymentService)

	// Kayıt Merkezi (Registry) handler'ları.
	moduleCatalogHTTPHandler := registryHandler.NewModuleCatalogHandler(moduleCatalogService)
	toolCatalogHTTPHandler := registryHandler.NewToolCatalogHandler(toolCatalogService)
	activationHTTPHandler := registryHandler.NewActivationHandler(tenantActivationService, dependencyCheckerService)

	// Rotaları ayarla. Bu fonksiyon, hangi endpoint'in hangi handler'a gideceğini belirler.
	setupRoutes(app, cfg, authHTTPHandler, tenantHTTPHandler, userHTTPHandler, uploadHTTPHandler, websiteHTTPHandler,
		studentHTTPHandler, lessonHTTPHandler, assignmentHTTPHandler,
		availabilityHTTPHandler, appointmentHTTPHandler,
		serviceHTTPHandler,
		postHTTPHandler, categoryHTTPHandler,
		paymentHTTPHandler, webhookHTTPHandler,
		moduleCatalogHTTPHandler, toolCatalogHTTPHandler, activationHTTPHandler,
		tenantContextMiddleware, tenantScopeMiddleware)

	// Hazırlanan uygulama örneğini geri döndür.
	return &Application{
		config: cfg,
		db:     db,
		app:    app,
	}, nil
}

// Start, uygulama sunucusunu başlatır.
func (a *Application) Start() error {
	// Graceful Shutdown (Zarif Kapatma) mekanizmasını ayarla.
	// Bu, sunucu kapanırken mevcut işlemleri bitirmesi için zaman tanır.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // Kesme (Ctrl+C) veya Terminate sinyallerini dinle.

	// Sunucuyu ayrı bir goroutine içinde başlat. Bu, ana thread'i bloklamaz.
	go func() {
		addr := fmt.Sprintf("%s:%s", a.config.Server.Host, a.config.Server.Port)
		log.Printf("🚀 Sunucu %s üzerinde başlatılıyor (ortam: %s)", addr, a.config.Server.Environment)
		if err := a.app.Listen(addr); err != nil {
			log.Printf("❌ Sunucu hatası: %v", err)
		}
	}()

	// Kapatma sinyali gelene kadar bekle.
	<-quit
	log.Println("🛑 Sunucu kapatılıyor...")

	// Fiber sunucusunu zarif bir şekilde kapat.
	if err := a.app.Shutdown(); err != nil {
		return fmt.Errorf("sunucu kapatma hatası: %w", err)
	}

	// Veritabanı bağlantısını kapat.
	if err := database.Close(a.db); err != nil {
		return fmt.Errorf("veritabanı kapatma hatası: %w", err)
	}

	log.Println("✅ Sunucu zarif bir şekilde durduruldu")
	return nil
}

// customErrorHandler, uygulama genelinde oluşan hataları yakalayan ve standart bir formatta JSON cevabı dönen fonksiyondur.
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Varsayılan hata kodu 500 (Internal Server Error).
	code := fiber.StatusInternalServerError

	// Gelen hatanın bir Fiber hatası olup olmadığını kontrol et.
	// Eğer öyleyse, o hatanın kendi kodunu kullan (örn: 404 Not Found).
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	// Hata detaylarını içeren JSON cevabını oluştur ve gönder.
	return c.Status(code).JSON(fiber.Map{
		"error":  err.Error(),
		"code":   code,
		"path":   c.Path(),
		"method": c.Method(),
	})
}
