package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/meinhoongagan/appointment-app/db"

	"github.com/meinhoongagan/appointment-app/routes"

	"github.com/meinhoongagan/appointment-app/cron"

	"github.com/meinhoongagan/appointment-app/redis"
)

func main() {
	app := fiber.New()
	db.Init()
	redis.InitRedis()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
	routes.SetupAuthRoutes(app)
	routes.SetupRBACRoutes(app)
	routes.SetupServiceRoutes(app)
	routes.SetupAppointmentRoutes(app)
	routes.SetupConsumerRoutes(app)

	// Initialize cron jobs
	cron.StartCronJobs()

	app.Listen(":8000")
	fmt.Println("Server started on port 8000")
}
