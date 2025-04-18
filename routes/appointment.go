package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupAppointmentRoutes configures all appointment related routes
func SetupAppointmentRoutes(app *fiber.App) {
	appointment := app.Group("/appointments")
	appointment.Get("/", controllers.GetAllAppointments)
	appointment.Get("/:id", controllers.GetAppointment)
	appointment.Post("/", middleware.Protected(), middleware.RequirePermission("appointments", "create"), controllers.CreateAppointment)
	appointment.Patch("/:id", middleware.Protected(), middleware.RequirePermission("appointments", "update"), controllers.UpdateAppointment)
	appointment.Delete("/:id", middleware.Protected(), middleware.RequirePermission("appointments", "delete"), controllers.DeleteAppointment)
}
