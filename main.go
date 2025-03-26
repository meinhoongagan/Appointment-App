package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/meinhoongagan/appointment-app/db"

	"github.com/meinhoongagan/appointment-app/routes"
)

func main() {
	app := fiber.New()
	db.Init()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
	routes.SetupAuthRoutes(app)
	routes.SetupRBACRoutes(app)
	routes.SetupAppointmentRoutes(app)
	routes.SetupServiceRoutes(app)
	routes.SetupWorkingHoursRoutes(app)

	app.Listen(":3000")
	fmt.Println("Server started on port 3000")
}
