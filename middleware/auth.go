package middleware

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
)

func Protected() fiber.Handler {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "solid_secret_key" // Replace with secure key in production
	}

	return jwtware.New(jwtware.Config{
		SigningKey:   []byte(secret),
		ErrorHandler: jwtError,
		SuccessHandler: func(c *fiber.Ctx) error {
			// Extensive debugging
			fmt.Println("JWT Middleware Triggered")

			// Safely extract user token
			userToken := c.Locals("user")
			if userToken == nil {
				fmt.Println("No user token found in locals")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "No authentication token",
				})
			}

			// Convert to JWT token
			token, ok := userToken.(*jwt.Token)
			if !ok {
				fmt.Println("Failed to convert user to JWT token")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token",
				})
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				fmt.Println("Failed to extract token claims")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token claims",
				})
			}

			// Debug print full claims
			fmt.Printf("Full Token Claims: %+v\n", claims)

			// Extract user ID
			userID, err := extractUserID(claims)
			if err != nil {
				fmt.Println("User ID extraction error:", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid user ID in token",
				})
			}

			// Extract role
			role, err := extractRole(claims)
			if err != nil {
				fmt.Println("Role extraction error:", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid role in token",
				})
			}

			// Set locals
			c.Locals("userID", userID)
			c.Locals("role", role)

			fmt.Printf("Set userID: %d, role: %s\n", userID, role)

			return c.Next()
		},
	})
}

// extractUserID handles multiple potential formats of user ID in token
func extractUserID(claims jwt.MapClaims) (uint, error) {
	idVal := claims["id"]
	if idVal == nil {
		return 0, fmt.Errorf("no ID found in claims")
	}

	switch v := idVal.(type) {
	case float64:
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("could not parse ID string: %v", err)
		}
		return uint(parsed), nil
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	default:
		return 0, fmt.Errorf("unsupported ID type: %T", v)
	}
}

// extractRole handles multiple potential formats of role in token
func extractRole(claims jwt.MapClaims) (string, error) {
	roleVal := claims["role"]
	if roleVal == nil {
		return "", fmt.Errorf("no role found in claims")
	}

	switch v := roleVal.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		if roleName, ok := v["name"].(string); ok {
			return roleName, nil
		}
		return "", fmt.Errorf("could not extract role name")
	default:
		return "", fmt.Errorf("unsupported role type: %T", v)
	}
}

// jwtError handles JWT errors
func jwtError(c *fiber.Ctx, err error) error {
	fmt.Println("JWT Error:", err)
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error":   "Unauthorized",
		"message": "Invalid or expired token",
	})
}