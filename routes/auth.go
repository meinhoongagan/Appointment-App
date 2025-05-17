package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupAuthRoutes configures all authentication related routes
func SetupAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")

	// Public routes
	auth.Post("/register", controllers.Register)
	auth.Post("/login", controllers.Login)

	// Protected routes
	auth.Get("/me", middleware.Protected(), controllers.GetUserProfile)
	auth.Post("/logout", middleware.Protected(), controllers.Logout)
	auth.Post("/refresh", controllers.RefreshToken)

	//Get user by ID
	auth.Get("/user/:id", middleware.Protected(), controllers.GetUserByID)
}
