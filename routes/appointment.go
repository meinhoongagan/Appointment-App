package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers/consumer"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupAppointmentRoutes configures all appointment related routes
func SetupAppointmentRoutes(app *fiber.App) {
	appointment := app.Group("/appointments", middleware.Protected())
	appointment.Get("/", consumer.GetAllAppointments)
	appointment.Get("/:id", consumer.GetAppointment)
	appointment.Get("/service/:id", consumer.GetServiceDetails)
	appointment.Post("/", middleware.Protected(), middleware.RequirePermission("appointments", "create"), consumer.CreateAppointment)
	appointment.Patch("/:id", middleware.Protected(), middleware.RequirePermission("appointments", "update"), consumer.UpdateAppointment)
	appointment.Delete("/:id", middleware.Protected(), middleware.RequirePermission("appointments", "delete"), consumer.DeleteAppointment)

	//_______________________________________________________________________________
	//Provider appointments
	providers := app.Group("/providers", middleware.Protected())

	providers.Get("/", consumer.GetAllProviders)
	providers.Get("/:id", consumer.GetProviderDetails)
	providers.Get("/:id/services", consumer.GetProviderServices)
	providers.Get("/search/service", consumer.SearchProviders)
	providers.Get("/category/:categoryId", consumer.GetProvidersByCategory)
	providers.Get("/featured", consumer.GetFeaturedProviders)
	providers.Get("/nearby", consumer.GetNearbyProviders)
	providers.Get("/available-time-slots/:provider_id", consumer.GetAvailableSlots)

	//Reviews________________________________________________________________
	reviewRoutes := app.Group("/reviews", middleware.Protected())

	reviewRoutes.Post("/", consumer.CreateReview)

	app.Get("/providers/:id/reviews", consumer.GetProviderReviews)

	reviewRoutes.Put("/:id", consumer.UpdateReview)

	reviewRoutes.Delete("/:id", consumer.DeleteReview)

	app.Get("/providers/:id/review-stats", consumer.GetProviderReviewStats)
}
