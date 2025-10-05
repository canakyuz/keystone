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

// setupRoutes fonksiyonu, uygulamanın tüm API rotalarını yapılandırır.
// Bu fonksiyon, sunucu başlatıldığında çağrılır ve hangi HTTP yolunun hangi işleyici (handler) fonksiyona karşılık geldiğini belirler.
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
	tenantScope fiber.Handler,
) {
	// /docs yolu, statik Swagger/OpenAPI dokümantasyon sayfasını sunar.
	app.Get("/docs", func(c *fiber.Ctx) error {
		return c.SendFile("web/static/docs/index.html")
	})
	app.Static("/docs/", "./web/static/docs")
	// /api/openapi.yaml yolu, API'nin tanım dosyasını sunar.
	app.Get("/api/openapi.yaml", func(c *fiber.Ctx) error {
		return c.SendFile("api/openapi.yaml")
	})

	// Health check (Sağlık Kontrolü) endpoint'i, servisin ayakta olup olmadığını kontrol etmek için kullanılır.
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":      "ok",
			"environment": cfg.Server.Environment,
			"message":     "NexSpaces API is running",
		})
	})

	// /api/v1 grubu, API'nin 1. versiyonu için bir ana grup oluşturur.
	v1 := app.Group("/api/v1")

	// Public (Herkese Açık) rotalar, kimlik doğrulaması gerektirmez.
	public := v1.Group("/public")
	public.Get("/websites/slug/:slug", websiteH.GetBySlug)

	// Auth (Kimlik Doğrulama) rotaları, kullanıcı giriş/kayıt işlemleri için kullanılır ve halka açıktır.
	auth := v1.Group("/auth")
	auth.Post("/register", authH.Register) // Kullanıcı kaydı
	auth.Post("/login", authH.Login)       // Kullanıcı girişi
	auth.Post("/logout", authH.Logout)     // Kullanıcı çıkışı

	// Korumalı kimlik doğrulama rotaları, JWT token ile kimlik doğrulaması gerektirir.
	authProtected := v1.Group("/auth", middleware.AuthMiddleware(cfg.Auth.JWTSecret))
	authProtected.Get("/me", authH.GetMe) // Mevcut giriş yapmış kullanıcı bilgilerini getirir.

	// Tenant (Kiracı) rotaları, kimlik doğrulaması gerektirir ve kiracıya özel işlemleri yönetir.
	tenants := v1.Group("/tenants", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	tenants.Post("/", tenantH.Create)                              // Yeni bir kiracı oluşturur.
	tenants.Get("/current", tenantH.GetCurrent)                    // Mevcut (aktif) kiracıyı getirir.
	tenants.Get("/stats", tenantH.GetStats)                        // Kiracı ile ilgili istatistikleri sunar.
	tenants.Get("/slug/:slug", tenantH.GetBySlug)                  // 'slug' ile bir kiracıyı bulur.
	tenants.Get("/:id", tenantH.GetByID)                           // ID ile bir kiracıyı bulur.
	tenants.Get("/", tenantH.List)                                 // Tüm kiracıları listeler.
	tenants.Patch("/:id", tenantH.Update)                          // Bir kiracının bilgilerini günceller.
	tenants.Post("/:id/suspend", tenantH.Suspend)                  // Bir kiracıyı askıya alır.
	tenants.Post("/:id/activate", tenantH.Activate)                // Askıya alınmış bir kiracıyı aktif eder.
	tenants.Post("/:id/upgrade", tenantH.UpgradePlan)              // Kiracının abonelik planını yükseltir.
	tenants.Post("/:id/domain", tenantH.SetCustomDomain)           // Kiracıya özel bir alan adı (domain) atar.
	tenants.Post("/:id/domain/verify", tenantH.VerifyCustomDomain) // Atanan özel alan adını doğrular.
	tenants.Patch("/:id/branding", tenantH.UpdateBranding)         // Kiracının marka kimliğini (logo, renkler vb.) günceller.
	tenants.Delete("/:id", tenantH.Delete)                         // Bir kiracıyı siler.

	// Upload (Dosya Yükleme) rotaları, kiracıya özeldir ve kimlik doğrulaması gerektirir.
	upload := v1.Group("/upload", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	upload.Post("/logo", uploadH.UploadLogo)       // Logo yükleme.
	upload.Post("/favicon", uploadH.UploadFavicon) // Favicon yükleme.
	upload.Post("/image", uploadH.UploadImage)     // Genel amaçlı resim yükleme.

	// Yüklenen dosyaların sunulması için statik bir yol. Herkese açıktır.
	app.Static("/uploads", "./uploads")

	// User (Kullanıcı) rotaları, kimlik doğrulaması gerektirir.
	users := v1.Group("/users", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	users.Post("/", userH.Create)                      // Yeni kullanıcı oluşturma (sadece admin).
	users.Get("/stats", userH.GetStats)                // Kullanıcı istatistikleri.
	users.Get("/:id", userH.GetByID)                   // ID ile kullanıcı getirme.
	users.Get("/", userH.List)                         // Tüm kullanıcıları listeleme.
	users.Patch("/:id", userH.Update)                  // Kullanıcı bilgilerini güncelleme.
	users.Post("/:id/password", userH.UpdatePassword)  // Kullanıcı şifresini güncelleme.
	users.Post("/:id/role", userH.UpdateRole)          // Kullanıcı rolünü güncelleme (sadece admin).
	users.Post("/:id/suspend", userH.Suspend)          // Kullanıcıyı askıya alma (sadece admin).
	users.Post("/:id/activate", userH.Activate)        // Kullanıcıyı aktif etme (sadece admin).
	users.Post("/:id/verify-email", userH.VerifyEmail) // Kullanıcı e-postasını doğrulama.
	users.Delete("/:id", userH.Delete)                 // Kullanıcıyı silme (sadece admin).

	// Website (Web Sitesi) rotaları, kiracıya özeldir.
	websites := v1.Group("/websites", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	websites.Get("/", websiteH.List)                // Web sitelerini listeler.
	websites.Post("/", websiteH.Create)             // Yeni web sitesi oluşturur.
	websites.Get("/:id", websiteH.GetByID)          // ID ile web sitesi getirir.
	websites.Patch("/:id", websiteH.Update)         // Web sitesini günceller.
	websites.Delete("/:id", websiteH.Delete)        // Web sitesini siler.
	websites.Post("/:id/publish", websiteH.Publish) // Web sitesini yayınlar.
	websites.Post("/:id/archive", websiteH.Archive) // Web sitesini arşivler.

	// Student (Öğrenci) rotaları, Dersler modülüne aittir ve kiracıya özeldir.
	students := v1.Group("/students", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	students.Post("/", studentH.Create)         // Yeni öğrenci oluşturur.
	students.Get("/stats", studentH.GetStats)   // Öğrenci istatistikleri.
	students.Get("/email", studentH.GetByEmail) // E-posta ile öğrenci bulur.
	students.Get("/:id", studentH.GetByID)      // ID ile öğrenci bulur.
	students.Get("/", studentH.List)            // Öğrencileri listeler.
	students.Put("/:id", studentH.Update)       // Öğrenci bilgilerini günceller.
	students.Delete("/:id", studentH.Delete)    // Öğrenciyi siler.

	// Lesson (Ders) rotaları, Dersler modülüne aittir ve kiracıya özeldir.
	lessons := v1.Group("/lessons", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	lessons.Post("/", lessonH.Create)             // Yeni ders oluşturur.
	lessons.Get("/stats", lessonH.GetStats)       // Ders istatistikleri.
	lessons.Get("/upcoming", lessonH.GetUpcoming) // Yaklaşan dersleri getirir.
	lessons.Get("/:id", lessonH.GetByID)          // ID ile ders bulur.
	lessons.Get("/", lessonH.List)                // Dersleri listeler.
	lessons.Put("/:id", lessonH.Update)           // Dersi günceller.
	lessons.Delete("/:id", lessonH.Delete)        // Dersi siler.

	// Öğrenciye özel ders rotaları.
	students.Get("/:student_id/lessons", lessonH.GetByStudent) // Bir öğrencinin tüm derslerini getirir.

	// Assignment (Ödev) rotaları, Dersler modülüne aittir ve kiracıya özeldir.
	assignments := v1.Group("/assignments", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	assignments.Post("/", assignmentH.Create)           // Yeni ödev oluşturur.
	assignments.Get("/stats", assignmentH.GetStats)     // Ödev istatistikleri.
	assignments.Get("/overdue", assignmentH.GetOverdue) // Gecikmiş ödevleri getirir.
	assignments.Get("/:id", assignmentH.GetByID)        // ID ile ödev bulur.
	assignments.Get("/", assignmentH.List)              // Ödevleri listeler.
	assignments.Put("/:id", assignmentH.Update)         // Ödevi günceller.
	assignments.Delete("/:id", assignmentH.Delete)      // Ödevi siler.

	// Öğrenciye özel ödev rotaları.
	students.Get("/:student_id/assignments", assignmentH.GetByStudent) // Bir öğrencinin tüm ödevlerini getirir.

	// Availability (Müsaitlik) rotaları, Rezervasyon modülüne aittir ve kiracıya özeldir.
	availabilities := v1.Group("/availabilities", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	availabilities.Post("/", availabilityH.Create)                  // Yeni müsaitlik durumu oluşturur.
	availabilities.Get("/stats", availabilityH.GetStats)            // Müsaitlik istatistikleri.
	availabilities.Get("/date-range", availabilityH.GetByDateRange) // Belirli bir tarih aralığındaki müsaitlikleri getirir.
	availabilities.Get("/:id", availabilityH.GetByID)               // ID ile müsaitlik durumu bulur.
	availabilities.Get("/", availabilityH.List)                     // Müsaitlik durumlarını listeler.
	availabilities.Put("/:id", availabilityH.Update)                // Müsaitlik durumunu günceller.
	availabilities.Delete("/:id", availabilityH.Delete)             // Müsaitlik durumunu siler.

	// Kullanıcıya özel müsaitlik rotaları.
	availabilities.Get("/user/:user_id", availabilityH.GetByUser) // Bir kullanıcının müsaitlik durumlarını getirir.

	// Appointment (Randevu) rotaları, Rezervasyon modülüne aittir ve kiracıya özeldir.
	appointments := v1.Group("/appointments", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	appointments.Post("/", appointmentH.Create)                  // Yeni randevu oluşturur.
	appointments.Get("/stats", appointmentH.GetStats)            // Randevu istatistikleri.
	appointments.Get("/upcoming", appointmentH.GetUpcoming)      // Yaklaşan randevuları getirir.
	appointments.Get("/date-range", appointmentH.GetByDateRange) // Belirli bir tarih aralığındaki randevuları getirir.
	appointments.Get("/client", appointmentH.GetByClient)        // Müşteri e-postasına göre randevuları getirir.
	appointments.Get("/:id", appointmentH.GetByID)               // ID ile randevu bulur.
	appointments.Get("/", appointmentH.List)                     // Randevuları listeler.
	appointments.Put("/:id", appointmentH.Update)                // Randevuyu günceller.
	appointments.Post("/:id/confirm", appointmentH.Confirm)      // Randevuyu onaylar.
	appointments.Post("/:id/cancel", appointmentH.Cancel)        // Randevuyu iptal eder.
	appointments.Post("/:id/complete", appointmentH.Complete)    // Randevuyu tamamlandı olarak işaretler.
	appointments.Delete("/:id", appointmentH.Delete)             // Randevuyu siler.

	// Kullanıcıya özel randevu rotaları.
	appointments.Get("/user/:user_id", appointmentH.GetByUser) // Bir kullanıcının randevularını getirir.

	// Service (Hizmet) rotaları, kiracıya özeldir.
	services := v1.Group("/services", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	services.Post("/", serviceH.Create)             // Yeni hizmet oluşturur.
	services.Get("/stats", serviceH.GetStats)       // Hizmet istatistikleri.
	services.Get("/featured", serviceH.GetFeatured) // Öne çıkan hizmetleri getirir.
	services.Get("/slug/:slug", serviceH.GetBySlug) // 'slug' ile hizmet bulur.
	services.Get("/:id", serviceH.GetByID)          // ID ile hizmet bulur.
	services.Get("/", serviceH.List)                // Hizmetleri listeler.
	services.Put("/:id", serviceH.Update)           // Hizmeti günceller.
	services.Delete("/:id", serviceH.Delete)        // Hizmeti siler.

	// Blog Category (Blog Kategori) rotaları, kiracıya özeldir.
	categories := v1.Group("/blog/categories", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	categories.Post("/", categoryH.Create)             // Yeni kategori oluşturur.
	categories.Get("/slug/:slug", categoryH.GetBySlug) // 'slug' ile kategori bulur.
	categories.Get("/:id", categoryH.GetByID)          // ID ile kategori bulur.
	categories.Get("/", categoryH.List)                // Kategorileri listeler.
	categories.Put("/:id", categoryH.Update)           // Kategoriyi günceller.
	categories.Delete("/:id", categoryH.Delete)        // Kategoriyi siler.

	// Blog Post (Blog Yazısı) rotaları, kiracıya özeldir.
	posts := v1.Group("/blog/posts", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	posts.Post("/", postH.Create)             // Yeni yazı oluşturur.
	posts.Get("/featured", postH.GetFeatured) // Öne çıkan yazıları getirir.
	posts.Get("/slug/:slug", postH.GetBySlug) // 'slug' ile yazı bulur.
	posts.Get("/tag/:tag", postH.GetByTag)    // Etikete göre yazıları getirir.
	posts.Get("/:id", postH.GetByID)          // ID ile yazı bulur.
	posts.Get("/", postH.List)                // Yazıları listeler.
	posts.Put("/:id", postH.Update)           // Yazıyı günceller.
	posts.Post("/:id/publish", postH.Publish) // Yazıyı yayınlar.
	posts.Post("/:id/archive", postH.Archive) // Yazıyı arşivler.
	posts.Delete("/:id", postH.Delete)        // Yazıyı siler.

	// Kategoriye özel yazı rotaları.
	categories.Get("/:category_id/posts", postH.GetByCategoryID) // Bir kategoriye ait yazıları getirir.

	// Payment (Ödeme) rotaları, kiracıya özeldir ve kimlik doğrulaması gerektirir.
	payments := v1.Group("/payments", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)
	payments.Post("/", paymentH.CreatePayment)                  // Yeni bir ödeme işlemi başlatır.
	payments.Post("/complete-3ds", paymentH.Complete3DSPayment) // 3D Secure doğrulamasını tamamlar.
	payments.Get("/:id", paymentH.GetPayment)                   // ID ile ödeme detayını getirir.
	payments.Get("/", paymentH.ListPayments)                    // Ödemeleri listeler.
	payments.Post("/:id/refund", paymentH.CreateRefund)         // Bir ödeme için iade talebi oluşturur.

	// Webhook rotaları, ödeme sağlayıcılardan gelen anlık bildirimleri işlemek için kullanılır. Halka açıktır.
	webhooks := v1.Group("/webhooks/payment")
	webhooks.Post("/:provider", webhookH.HandleWebhook)                   // Sağlayıcıya özel genel webhook işleyicisi.
	webhooks.Post("/iyzico/:tenant_id", webhookH.HandleIyzicoWebhook)     // Iyzico'dan gelen bildirimleri işler.
	webhooks.Post("/checkout/:tenant_id", webhookH.HandleCheckoutWebhook) // Checkout.com'dan gelen bildirimleri işler.

	// Registry (Kayıt Merkezi) rotaları, Modül ve Araç Pazaryeri'ni yönetir.
	registry := v1.Group("/registry")

	// Halka açık modül kataloğu rotaları, kimlik doğrulaması gerektirmez.
	registry.Get("/modules", moduleCatalogH.ListPublicModules)                       // Herkese açık modülleri listeler.
	registry.Get("/modules/search", moduleCatalogH.SearchModules)                    // Modüllerde arama yapar.
	registry.Get("/modules/popular", moduleCatalogH.GetPopularModules)               // Popüler modülleri getirir.
	registry.Get("/modules/top-rated", moduleCatalogH.GetTopRatedModules)            // En yüksek puanlı modülleri getirir.
	registry.Get("/modules/new", moduleCatalogH.GetNewModules)                       // Yeni eklenen modülleri getirir.
	registry.Get("/modules/free", moduleCatalogH.GetFreeModules)                     // Ücretsiz modülleri getirir.
	registry.Get("/modules/category/:category", moduleCatalogH.GetModulesByCategory) // Kategoriye göre modülleri getirir.
	registry.Get("/modules/slug/:slug", moduleCatalogH.GetModuleBySlug)              // 'slug' ile modül bulur.
	registry.Get("/modules/:id", moduleCatalogH.GetModuleByID)                       // ID ile modül bulur.

	// Halka açık araç kataloğu rotaları, kimlik doğrulaması gerektirmez.
	registry.Get("/tools", toolCatalogH.ListPublicTools)                       // Herkese açık araçları listeler.
	registry.Get("/tools/search", toolCatalogH.SearchTools)                    // Araçlarda arama yapar.
	registry.Get("/tools/popular", toolCatalogH.GetPopularTools)               // Popüler araçları getirir.
	registry.Get("/tools/top-rated", toolCatalogH.GetTopRatedTools)            // En yüksek puanlı araçları getirir.
	registry.Get("/tools/new", toolCatalogH.GetNewTools)                       // Yeni eklenen araçları getirir.
	registry.Get("/tools/free", toolCatalogH.GetFreeTools)                     // Ücretsiz araçları getirir.
	registry.Get("/tools/category/:category", toolCatalogH.GetToolsByCategory) // Kategoriye göre araçları getirir.
	registry.Get("/tools/slug/:slug", toolCatalogH.GetToolBySlug)              // 'slug' ile araç bulur.
	registry.Get("/tools/:id", toolCatalogH.GetToolByID)                       // ID ile araç bulur.

	// Kiracıya özel aktivasyon rotaları, kimlik doğrulaması gerektirir.
	tenantRegistry := v1.Group("/registry/tenant", middleware.AuthMiddleware(cfg.Auth.JWTSecret), tenantScope)

	// Mevcut kiracı için modül aktivasyon işlemleri.
	tenantRegistry.Get("/modules", activationH.GetActivatedModules)                             // Kiracının aktif modüllerini listeler.
	tenantRegistry.Post("/modules/install", activationH.InstallModule)                          // Bir modülü kurar (veritabanı kaydı).
	tenantRegistry.Post("/modules/:module_id/activate", activationH.ActivateModule)             // Bir modülü aktif hale getirir.
	tenantRegistry.Post("/modules/:module_id/deactivate", activationH.DeactivateModule)         // Bir modülü pasif hale getirir.
	tenantRegistry.Delete("/modules/:module_id", activationH.UninstallModule)                   // Bir modülü kaldırır.
	tenantRegistry.Post("/modules/:module_id/complete-setup", activationH.CompleteModuleSetup)  // Modül kurulumunun son adımlarını tamamlar.
	tenantRegistry.Get("/modules/:module_id/dependencies", activationH.CheckModuleDependencies) // Modülün bağımlılıklarını kontrol eder.

	// Mevcut kiracı için araç aktivasyon işlemleri.
	tenantRegistry.Get("/tools", activationH.GetActivatedTools)                           // Kiracının aktif araçlarını listeler.
	tenantRegistry.Post("/tools/install", activationH.InstallTool)                        // Bir aracı kurar.
	tenantRegistry.Post("/tools/:tool_id/activate", activationH.ActivateTool)             // Bir aracı aktif hale getirir.
	tenantRegistry.Post("/tools/:tool_id/deactivate", activationH.DeactivateTool)         // Bir aracı pasif hale getirir.
	tenantRegistry.Delete("/tools/:tool_id", activationH.UninstallTool)                   // Bir aracı kaldırır.
	tenantRegistry.Post("/tools/:tool_id/complete-setup", activationH.CompleteToolSetup)  // Araç kurulumunun son adımlarını tamamlar.
	tenantRegistry.Get("/tools/:tool_id/dependencies", activationH.CheckToolDependencies) // Aracın bağımlılıklarını kontrol eder.
}
