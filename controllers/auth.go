package controllers

import (
	"os"
	"time"

	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"golang.org/x/crypto/bcrypt"
)

// Register handles user registration
func Register(c *fiber.Ctx) error {
	user := new(models.User)

	if err := c.BodyParser(user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Validate input
	if user.Email == "" || user.Password == "" || user.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields",
		})
	}

	// Check if user already exists
	var existingUser models.User
	if db.DB.Where("email = ?", user.Email).First(&existingUser).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User with this email already exists",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}
	user.Password = string(hashedPassword)

	// Set default role if not provided
	if user.RoleID == 0 {
		// Find the client role
		var clientRole models.Role
		if err := db.DB.Where("name = ?", "client").First(&clientRole).Error; err != nil {
			// Log the error for debugging purposes
			log.Printf("Error finding client role: %v", err)

			// Return a more informative error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to assign default role. Role 'client' not found.",
			})
		}

		user.RoleID = clientRole.ID
		user.Role = clientRole
		log.Printf("Assigned default role with ID: %d", clientRole.ID)
	}

	if user.RoleID != 0 {
		// Find the role
		var role models.Role
		if err := db.DB.First(&role, user.RoleID).Error; err != nil {
			// Log the error for debugging purposes
			log.Printf("Error finding role: %v", err)

			// Return a more informative error
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to assign role. Role not found.",
			})
		}

		user.Role = role
		log.Printf("Assigned role with ID: %d", role.ID)
	}

	// Create user
	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user: " + err.Error(),
		})
	}

	// Remove password from response
	user.Password = ""

	return c.Status(fiber.StatusCreated).JSON(user)
}

// Login handles user authentication
func Login(c *fiber.Ctx) error {
	type LoginInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	input := new(LoginInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Find user
	var user models.User
	if db.DB.Where("email = ?", input.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	role := models.Role{}
	if err := db.DB.First(&role, user.RoleID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch role",
		})
	}

	// Create access token
	claims := jwt.MapClaims{
		"id":      user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24 hour expiration
		"role":    role,
		"role_id": user.RoleID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Get secret from environment or use a default (in production, always use environment variable)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your_secret_key" // Replace with secure key in production
	}

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Create refresh token with longer expiration
	refreshClaims := jwt.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 day expiration
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(secret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"token":        tokenString,
		"refreshToken": refreshTokenString,
		"user": fiber.Map{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.Role.Name,
			"role_id": user.Role.ID,
		},
	})
}

// GetUserProfile returns the current user's profile
func GetUserProfile(c *fiber.Ctx) error {
	// Get user from context (set by middleware)
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["id"].(float64)

	var userProfile models.User
	db.DB.Where("id = ?", uint(userID)).First(&userProfile)

	// Don't send password
	userProfile.Password = ""

	return c.JSON(userProfile)
}

// Logout doesn't actually invalidate the token as JWTs are stateless
// For a more secure implementation, you'd need to use a token blacklist
func Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Successfully logged out",
	})
}

// RefreshToken generates a new access token using a refresh token
func RefreshToken(c *fiber.Ctx) error {
	type RefreshRequest struct {
		RefreshToken string `json:"refreshToken"`
	}

	refreshRequest := new(RefreshRequest)
	if err := c.BodyParser(refreshRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Parse and validate the refresh token
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your_secret_key"
	}

	token, err := jwt.Parse(refreshRequest.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	// Create new access token
	claims := token.Claims.(jwt.MapClaims)
	newClaims := jwt.MapClaims{
		"id":    claims["id"],
		"email": claims["email"],
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := newToken.SignedString([]byte(secret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token": tokenString,
	})
}
