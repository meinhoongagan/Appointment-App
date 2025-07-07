package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupAuthRoutes configures all authentication related routes
func SetupAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")

	auth.Post("/register", controllers.Register)

	auth.Post("/login", controllers.Login)

	auth.Get("/me", middleware.Protected(), controllers.GetUserProfile)

	auth.Post("/logout", middleware.Protected(), controllers.Logout)

	auth.Post("/refresh", controllers.RefreshToken)

	auth.Get("/user/:id", middleware.Protected(), controllers.GetUserByID)

	auth.Post("/send-otp", controllers.SendOTP)

	auth.Post("/otp/verify/", controllers.VerifyOTP)

	auth.Post("/reset-password/:token", controllers.ResetPassword)
}
