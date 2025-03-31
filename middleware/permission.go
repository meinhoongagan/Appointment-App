package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// RequirePermission checks if the user has the required permission
func RequirePermission(resource string, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user from context (set by JWT middleware)
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		userID := uint(claims["id"].(float64))
		fmt.Println("User ID from JWT:", userID)
		// Get user with role and permissions from database
		var dbUser models.User
		if err := db.DB.Preload("Role.Permissions").First(&dbUser, userID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		fmt.Println("User from DB:", dbUser)
		// Check if user has required permission
		hasPermission := false
		for _, permission := range dbUser.Role.Permissions {
			if permission.Resource == resource && permission.Action == action {
				hasPermission = true
				break
			}
		}
		fmt.Println("Required permission:", resource, action)
		fmt.Println("Has permission:", hasPermission)
		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to perform this action",
			})
		}

		return c.Next()
	}
}

// RequireRole checks if the user has the required role
func RequireRole(roleName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user from context (set by JWT middleware)
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		userID := uint(claims["id"].(float64))

		// Get user with role from database
		var dbUser models.User
		if err := db.DB.Preload("Role").First(&dbUser, userID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Check if user has required role
		if dbUser.Role.Name != roleName {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have the required role to perform this action",
			})
		}

		return c.Next()
	}
}
