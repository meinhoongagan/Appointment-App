package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers/consumer"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupConsumerRoutes configures all consumer related routes
func SetupConsumerRoutes(app *fiber.App) {
	consumerGroup := app.Group("/consumer", middleware.Protected())
	consumerGroup.Get("/profile", consumer.GetUserProfile)
	consumerGroup.Post("/profile", consumer.CreateUserProfile)
	consumerGroup.Post("/profile/picture", consumer.UpdateUserProfilePicture)
	consumerGroup.Patch("/profile", consumer.UpdateUserProfile)
	consumerGroup.Delete("/profile", consumer.DeleteUserProfile)
}
