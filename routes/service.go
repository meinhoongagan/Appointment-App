package routes

import (
	"github.com/gofiber/fiber/v2"
	services "github.com/meinhoongagan/appointment-app/controllers/service"
	"github.com/meinhoongagan/appointment-app/middleware"
)

func SetupServiceRoutes(app *fiber.App) {
	service := app.Group("provider/services")
	service.Get("/", services.GetAllServices)
	service.Get("/:id", services.GetService)
	service.Get("/get-provider/service", middleware.Protected(), services.GetMyServices)
	service.Post("/", middleware.Protected(), middleware.RequirePermission("services", "create"), services.CreateService)
	service.Patch("/:id", middleware.Protected(), middleware.RequirePermission("services", "update"), services.UpdateService)
	service.Delete("/:id", middleware.Protected(), middleware.RequirePermission("services", "delete"), services.DeleteService)

	//_______________________________________________________________________________
	dashboard := app.Group("provider/dashboard", middleware.Protected())

	// Overview statistics
	dashboard.Get("/overview", services.GetDashboardOverview)

	// Recent appointments
	dashboard.Get("/recent-appointments", services.GetRecentAppointments)

	// Revenue summary
	dashboard.Get("/revenue", services.GetRevenueSummary)

	// Quick actions
	dashboard.Get("/quick-actions", services.GetQuickActions)

	//_______________________________________________________________________________
	providerAppointments := app.Group("/provider/appointments", middleware.Protected())

	// Upcoming appointments
	providerAppointments.Get("/upcoming", services.GetProviderUpcomingAppointments)

	// Appointment history
	providerAppointments.Get("/history", services.GetProviderAppointmentHistory)

	// Appointment management
	providerAppointments.Patch("/:id/status", middleware.RequirePermission("services", "update"), services.UpdateAppointmentStatus)
	providerAppointments.Patch("/:id/reschedule", middleware.RequirePermission("services", "update"), services.RescheduleAppointment)

	//_____________________________________________________________________
	profile := app.Group("/provider/profile", middleware.Protected())
	profile.Get("/", services.GetProviderProfile)
	profile.Patch("/", services.UpdateProviderProfile)

	// Business details
	profile.Get("/business", services.GetBusinessDetails)
	profile.Patch("/business", services.UpdateBusinessDetails)

	// Settings
	profile.Get("/settings", services.GetProviderSettings)
	profile.Patch("/settings", services.UpdateProviderSettings)

	// Working hours
	profile.Get("/working-hours", services.GetWorkingHours)
	profile.Patch("/working-hours", services.UpdateWorkingHours)

	// Time off
	profile.Get("/time-off", services.GetTimeOff)
	profile.Post("/time-off", services.AddTimeOff)
	profile.Delete("/time-off/:id", services.DeleteTimeOff)

}
