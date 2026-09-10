// Package verticals wires the example business modules onto a running application.
//
// These modules — blog, booking, lessons, payments, services, websites — are a reference
// application, not part of the control plane. They exist to show what building on top of
// tenant provisioning and isolation looks like, and they are kept out of internal/ so
// that the interesting part of this repository is not buried under them.
//
// The dependency direction is the point. This package imports the control plane; the
// control plane does not import this package and does not know it exists. That is what
// makes "the core runs without the verticals" a fact the compiler enforces rather than a
// claim in a README: internal/app has no path to any of this code.
package verticals

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	blogHandler "github.com/canakyuz/keystone/examples/verticals/handler/blog"
	bookingHandler "github.com/canakyuz/keystone/examples/verticals/handler/booking"
	lessonHandler "github.com/canakyuz/keystone/examples/verticals/handler/lesson"
	paymentHandler "github.com/canakyuz/keystone/examples/verticals/handler/payment"
	serviceHandler "github.com/canakyuz/keystone/examples/verticals/handler/service"
	websiteHandler "github.com/canakyuz/keystone/examples/verticals/handler/website"
	"github.com/canakyuz/keystone/internal/app"
)

// Register mounts the example routes onto the application.
//
// It is called from cmd/server after the core application is built, which is the only
// place that knows about both halves. A deployment that wants the control plane alone
// drops this call and the modules go with it.
func Register(ext app.Extension, deps Dependencies) {
	v1 := ext.App.Group("/api/v1")

	authenticated := ext.Authenticated
	tenantContextMiddleware := ext.TenantContext
	tenantScope := ext.TenantScope

	websiteH := deps.Website
	studentH := deps.Student
	lessonH := deps.Lesson
	assignmentH := deps.Assignment
	availabilityH := deps.Availability
	appointmentH := deps.Appointment
	serviceH := deps.Service
	categoryH := deps.BlogCategory
	postH := deps.BlogPost
	paymentH := deps.Payment
	webhookH := deps.Webhook

	// The public lookup a published site is served through. No authentication: it is how
	// an anonymous visitor reaches published CMS content, which is what
	// websites.public_websites_policy in migration 030 deliberately allows across the
	// tenant boundary. See SECURITY.md.
	v1.Group("/public").Get("/websites/slug/:slug", websiteH.GetBySlug)

	// Website routes; tenant-scoped.
	websites := v1.Group("/websites", chain(authenticated, tenantContextMiddleware, tenantScope)...)
	websites.Get("/", websiteH.List)
	websites.Post("/", websiteH.Create)
	websites.Get("/:id", websiteH.GetByID)
	websites.Patch("/:id", websiteH.Update)
	websites.Delete("/:id", websiteH.Delete)
	websites.Post("/:id/publish", websiteH.Publish)
	websites.Post("/:id/archive", websiteH.Archive)

	// Student routes; part of the lessons module, tenant-scoped.
	students := v1.Group("/students", chain(authenticated, tenantScope)...)
	students.Post("/", studentH.Create)
	students.Get("/stats", studentH.GetStats)
	students.Get("/email", studentH.GetByEmail)
	students.Get("/:id", studentH.GetByID)
	students.Get("/", studentH.List)
	students.Put("/:id", studentH.Update)
	students.Delete("/:id", studentH.Delete)

	// Lesson routes; part of the lessons module, tenant-scoped.
	lessons := v1.Group("/lessons", chain(authenticated, tenantScope)...)
	lessons.Post("/", lessonH.Create)
	lessons.Get("/stats", lessonH.GetStats)
	lessons.Get("/upcoming", lessonH.GetUpcoming)
	lessons.Get("/:id", lessonH.GetByID)
	lessons.Get("/", lessonH.List)
	lessons.Put("/:id", lessonH.Update)
	lessons.Delete("/:id", lessonH.Delete)

	// Per-student lesson routes.
	students.Get("/:student_id/lessons", lessonH.GetByStudent)

	// Assignment routes; part of the lessons module, tenant-scoped.
	assignments := v1.Group("/assignments", chain(authenticated, tenantScope)...)
	assignments.Post("/", assignmentH.Create)
	assignments.Get("/stats", assignmentH.GetStats)
	assignments.Get("/overdue", assignmentH.GetOverdue)
	assignments.Get("/:id", assignmentH.GetByID)
	assignments.Get("/", assignmentH.List)
	assignments.Put("/:id", assignmentH.Update)
	assignments.Delete("/:id", assignmentH.Delete)

	// Per-student assignment routes.
	students.Get("/:student_id/assignments", assignmentH.GetByStudent)

	// Availability routes; part of the booking module, tenant-scoped.
	availabilities := v1.Group("/availabilities", chain(authenticated, tenantScope)...)
	availabilities.Post("/", availabilityH.Create)
	availabilities.Get("/stats", availabilityH.GetStats)
	availabilities.Get("/date-range", availabilityH.GetByDateRange)
	availabilities.Get("/:id", availabilityH.GetByID)
	availabilities.Get("/", availabilityH.List)
	availabilities.Put("/:id", availabilityH.Update)
	availabilities.Delete("/:id", availabilityH.Delete)

	// Per-user availability routes.
	availabilities.Get("/user/:user_id", availabilityH.GetByUser)

	// Appointment routes; part of the booking module, tenant-scoped.
	appointments := v1.Group("/appointments", chain(authenticated, tenantScope)...)
	appointments.Post("/", appointmentH.Create)
	appointments.Get("/stats", appointmentH.GetStats)
	appointments.Get("/upcoming", appointmentH.GetUpcoming)
	appointments.Get("/date-range", appointmentH.GetByDateRange)
	appointments.Get("/client", appointmentH.GetByClient)
	appointments.Get("/:id", appointmentH.GetByID)
	appointments.Get("/", appointmentH.List)
	appointments.Put("/:id", appointmentH.Update)
	appointments.Post("/:id/confirm", appointmentH.Confirm)
	appointments.Post("/:id/cancel", appointmentH.Cancel)
	appointments.Post("/:id/complete", appointmentH.Complete)
	appointments.Delete("/:id", appointmentH.Delete)

	// Per-user appointment routes.
	appointments.Get("/user/:user_id", appointmentH.GetByUser)

	// Service routes; tenant-scoped.
	services := v1.Group("/services", chain(authenticated, tenantScope)...)
	services.Post("/", serviceH.Create)
	services.Get("/stats", serviceH.GetStats)
	services.Get("/featured", serviceH.GetFeatured)
	services.Get("/slug/:slug", serviceH.GetBySlug)
	services.Get("/:id", serviceH.GetByID)
	services.Get("/", serviceH.List)
	services.Put("/:id", serviceH.Update)
	services.Delete("/:id", serviceH.Delete)

	// Blog category routes; tenant-scoped.
	categories := v1.Group("/blog/categories", chain(authenticated, tenantScope)...)
	categories.Post("/", categoryH.Create)
	categories.Get("/slug/:slug", categoryH.GetBySlug)
	categories.Get("/:id", categoryH.GetByID)
	categories.Get("/", categoryH.List)
	categories.Put("/:id", categoryH.Update)
	categories.Delete("/:id", categoryH.Delete)

	// Blog post routes; tenant-scoped.
	posts := v1.Group("/blog/posts", chain(authenticated, tenantScope)...)
	posts.Post("/", postH.Create)
	posts.Get("/featured", postH.GetFeatured)
	posts.Get("/slug/:slug", postH.GetBySlug)
	posts.Get("/tag/:tag", postH.GetByTag)
	posts.Get("/:id", postH.GetByID)
	posts.Get("/", postH.List)
	posts.Put("/:id", postH.Update)
	posts.Post("/:id/publish", postH.Publish)
	posts.Post("/:id/archive", postH.Archive)
	posts.Delete("/:id", postH.Delete)

	// Per-category post routes.
	categories.Get("/:category_id/posts", postH.GetByCategoryID)

	// Payment routes; tenant-scoped and authenticated.
	payments := v1.Group("/payments", chain(authenticated, tenantScope)...)
	payments.Post("/", paymentH.CreatePayment)
	payments.Post("/complete-3ds", paymentH.Complete3DSPayment)
	payments.Get("/:id", paymentH.GetPayment)
	payments.Get("/", paymentH.ListPayments)
	payments.Post("/:id/refund", paymentH.CreateRefund)

	// Webhook routes handle callbacks from the payment providers. Public.
	webhooks := v1.Group("/webhooks/payment")
	webhooks.Post("/:provider", webhookH.HandleWebhook)
	webhooks.Post("/iyzico/:tenant_id", webhookH.HandleIyzicoWebhook)
	webhooks.Post("/checkout/:tenant_id", webhookH.HandleCheckoutWebhook)
}

// chain appends group-specific middleware to a shared base without aliasing it.
//
// append() on a shared slice can write into the base's spare capacity, so two groups
// built from the same base would overwrite each other's middleware.
func chain(base []fiber.Handler, extra ...fiber.Handler) []fiber.Handler {
	out := make([]fiber.Handler, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)

	return out
}

// Dependencies are the handlers the example routes need.
type Dependencies struct {
	Website      *websiteHandler.Handler
	Student      *lessonHandler.StudentHandler
	Lesson       *lessonHandler.LessonHandler
	Assignment   *lessonHandler.AssignmentHandler
	Availability *bookingHandler.AvailabilityHandler
	Appointment  *bookingHandler.AppointmentHandler
	Service      *serviceHandler.ServiceHandler
	BlogCategory *blogHandler.CategoryHandler
	BlogPost     *blogHandler.PostHandler
	Payment      *paymentHandler.Handler
	Webhook      *paymentHandler.WebhookHandler
}

var _ = sql.ErrNoRows
