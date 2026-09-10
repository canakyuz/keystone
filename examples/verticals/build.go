package verticals

import (
	"database/sql"

	blogHandler "github.com/canakyuz/keystone/examples/verticals/handler/blog"
	bookingHandler "github.com/canakyuz/keystone/examples/verticals/handler/booking"
	lessonHandler "github.com/canakyuz/keystone/examples/verticals/handler/lesson"
	paymentHandler "github.com/canakyuz/keystone/examples/verticals/handler/payment"
	serviceHandler "github.com/canakyuz/keystone/examples/verticals/handler/service"
	websiteHandler "github.com/canakyuz/keystone/examples/verticals/handler/website"
	providerPayment "github.com/canakyuz/keystone/examples/verticals/provider/payment"
	blogRepo "github.com/canakyuz/keystone/examples/verticals/repository/blog"
	bookingRepo "github.com/canakyuz/keystone/examples/verticals/repository/booking"
	lessonRepo "github.com/canakyuz/keystone/examples/verticals/repository/lesson"
	paymentRepo "github.com/canakyuz/keystone/examples/verticals/repository/payment"
	serviceRepo "github.com/canakyuz/keystone/examples/verticals/repository/service"
	websiteRepo "github.com/canakyuz/keystone/examples/verticals/repository/website"
	blogUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/blog"
	bookingUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/booking"
	lessonUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/lesson"
	paymentUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/payment"
	serviceUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/service"
	websiteUsecase "github.com/canakyuz/keystone/examples/verticals/usecase/website"
	"github.com/canakyuz/keystone/internal/config"
	"github.com/canakyuz/keystone/pkg/logger"
)

// Build constructs the example modules.
//
// This is the composition root for the reference application, and it used to live inside
// internal/app alongside the control plane's. Splitting it is what lets the control plane
// be built and shipped without any of this: the core composition root no longer names a
// single one of these packages, so the compiler will not let the dependency creep back.
func Build(db *sql.DB, cfg *config.Config, log *logger.Logger) Dependencies {
	// Repositories.
	websiteRepository := websiteRepo.NewPostgresRepository(db)
	studentRepository := lessonRepo.NewStudentPostgresRepository(db)
	lessonRepository := lessonRepo.NewLessonPostgresRepository(db)
	assignmentRepository := lessonRepo.NewAssignmentPostgresRepository(db)
	availabilityRepository := bookingRepo.NewAvailabilityPostgresRepository(db)
	appointmentRepository := bookingRepo.NewAppointmentPostgresRepository(db)
	serviceRepository := serviceRepo.NewServicePostgresRepository(db)
	postRepository := blogRepo.NewPostRepository(db)
	categoryRepository := blogRepo.NewCategoryRepository(db)
	paymentRepository := paymentRepo.NewPostgresRepository(db)

	// The payment orchestrator, which fronts several providers (Iyzico, Checkout.com).
	paymentOrchestrator := providerPayment.NewOrchestrator(&cfg.Payment)

	if cfg.Payment.Iyzico.Enabled {
		paymentOrchestrator.RegisterProvider("iyzico", providerPayment.NewIyzicoProvider(&cfg.Payment.Iyzico))
	}
	if cfg.Payment.Checkout.Enabled {
		paymentOrchestrator.RegisterProvider("checkout", providerPayment.NewCheckoutProvider(&cfg.Payment.Checkout))
	}

	// Usecases.
	websiteService := websiteUsecase.NewService(websiteRepository)
	studentService := lessonUsecase.NewStudentService(studentRepository, *log)
	lessonService := lessonUsecase.NewLessonService(lessonRepository, *log)
	assignmentService := lessonUsecase.NewAssignmentService(assignmentRepository, *log)
	availabilityService := bookingUsecase.NewAvailabilityService(availabilityRepository, *log)
	appointmentService := bookingUsecase.NewAppointmentService(appointmentRepository, *log)
	serviceService := serviceUsecase.NewServiceService(serviceRepository, *log)
	postService := blogUsecase.NewPostService(postRepository, *log)
	categoryService := blogUsecase.NewCategoryService(categoryRepository, *log)
	paymentService := paymentUsecase.NewService(paymentRepository, paymentOrchestrator)

	// Handlers.
	return Dependencies{
		Website:      websiteHandler.NewHandler(websiteService),
		Student:      lessonHandler.NewStudentHandler(studentService),
		Lesson:       lessonHandler.NewLessonHandler(lessonService),
		Assignment:   lessonHandler.NewAssignmentHandler(assignmentService),
		Availability: bookingHandler.NewAvailabilityHandler(availabilityService),
		Appointment:  bookingHandler.NewAppointmentHandler(appointmentService),
		Service:      serviceHandler.NewServiceHandler(serviceService),
		BlogCategory: blogHandler.NewCategoryHandler(categoryService, *log),
		BlogPost:     blogHandler.NewPostHandler(postService, *log),
		Payment:      paymentHandler.NewHandler(paymentService),
		Webhook:      paymentHandler.NewWebhookHandler(paymentService),
	}
}
