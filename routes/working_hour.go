package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupWorkingHourRoutes configures all working hour related routes
func SetupWorkingHourRoutes(app *fiber.App) {
	workingHour := app.Group("/working-hours")
	workingHour.Get("/", controllers.GetAllWorkingHours)
	workingHour.Get("/:id", controllers.GetWorkingHour)
	workingHour.Post("/", middleware.Protected(), middleware.RequirePermission("working-hours", "create"), controllers.CreateWorkingHour)
	workingHour.Patch("/:id", middleware.Protected(), middleware.RequirePermission("working-hours", "update"), controllers.UpdateWorkingHour)
	workingHour.Delete("/:id", middleware.Protected(), middleware.RequirePermission("working-hours", "delete"), controllers.DeleteWorkingHour)
	// workingHour.Post("/upload", middleware.Protected(), middleware.RequirePermission("working-hours", "create"), utils.UploadWorkingHours)
}
