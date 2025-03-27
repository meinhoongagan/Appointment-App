package routes 

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

func SetupServiceRoutes(app *fiber.App) {
	service := app.Group("/services")
	service.Get("/",controllers.GetAllServices)
	service.Get("/:id",controllers.GetService)
	service.Post("/", middleware.Protected(), middleware.RequirePermission("services", "create") ,controllers.CreateService)
	service.Put("/:id", middleware.Protected(), middleware.RequirePermission("services", "update") ,controllers.UpdateService)	
	service.Delete("/:id", middleware.Protected(), middleware.RequirePermission("services", "delete") ,controllers.DeleteService)
}