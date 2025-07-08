package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/meinhoongagan/appointment-app/cron"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/docs"

	// _ "github.com/meinhoongagan/appointment-app/docs"
	"github.com/meinhoongagan/appointment-app/redis"
	"github.com/meinhoongagan/appointment-app/routes"
)

func main() {
	app := fiber.New()

	// Configure Swagger based on environment
	configureSwagger()
	db.Init()
	redis.InitRedis()

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	// @title Appointment App API
	// @version 1.0
	// @description This is the API documentation for the Appointment App.
	// @BasePath /
	// @schemes http https
	// @securityDefinitions.apikey BearerAuth
	// @in header
	// @name Authorization

	// @host appointment-app-a395.onrender.com
	// @x-servers [{"url": "https://appointment-app-a395.onrender.com", "description": "Production"}, {"url": "http://localhost:8000", "description": "Development"}]

	app.Get("/swagger/*", swagger.HandlerDefault)

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

func configureSwagger() {
	if os.Getenv("ENV") == "production" || os.Getenv("RENDER") != "" {
		docs.SwaggerInfo.Host = "appointment-app-a395.onrender.com"
		docs.SwaggerInfo.Schemes = []string{"https"}
	} else {
		docs.SwaggerInfo.Host = "localhost:8000"
		docs.SwaggerInfo.Schemes = []string{"http"}
	}
}
