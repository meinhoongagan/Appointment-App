package controllers

import (
	"fmt"
	"os"
	"time"

	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/redis"
	"github.com/meinhoongagan/appointment-app/utils"
	"golang.org/x/crypto/bcrypt"
)

// Register handles user registration
// Public routes
// Register handles user registration
// @Summary Register a new user
// @Description Registers a user with name, email, and password. Assigns a default role if not provided.
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body object{name=string,email=string,password=string,role=string} true "User info"
// @Success 201 {object} models.User
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/register [post]
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

	// Generate OTP and send email (same as SendOTP)
	otp := utils.GenerateOTP()
	user.OTP = otp
	user.OTPExpiresAt = time.Now().Add(10 * time.Minute)
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save OTP",
		})
	}
	if err := utils.SendEmail(user.Email, "Your OTP Code", fmt.Sprintf("Your OTP code is: %s ,Valid for 10 minutes", otp)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP email",
		})
	}

	// Remove password from response
	user.Password = ""

	return c.Status(fiber.StatusCreated).JSON(user)
}

// Login handles user authentication
// Login handles user authentication
// @Summary Login a user
// @Description Authenticates a user and returns access + refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body object{email=string,password=string} true "Login credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
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

	// check if verified
	if !user.IsVerified {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Please verify your account",
		})
	}

	//Find Role_id From User Table
	var role models.Role
	if err := db.DB.Where("id = ?", user.RoleID).First(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to find user role",
		})
	}

	fmt.Println("User Role ID:", role.ID)

	// Create access token
	claims := jwt.MapClaims{
		"id":      user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24 hour expiration
		"role":    role,
		"role_id": role.ID,
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
			"role":    role,
			"role_id": role.ID,
		},
	})
}

// GetUserProfile returns the current user's profile

// Protected routes
// GetUserProfile returns the current user's profile
// @Summary Get current user profile
// @Description Get the profile of the authenticated user
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Router /auth/me [get]
func GetUserProfile(c *fiber.Ctx) error {
	// Get user from context (set by middleware)
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["id"].(float64)

	var userProfile models.User

	// Use Preload to load the associated Role
	if err := db.DB.Preload("Role").Where("id = ?", uint(userID)).First(&userProfile).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Don't send password in response
	userProfile.Password = ""

	return c.JSON(userProfile)
}

// Logout doesn't actually invalidate the token as JWTs are stateless
// Logout a user
// @Summary Logout user
// @Description Logs out user (stateless in JWT; no actual invalidation)
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Successfully logged out",
	})
}

// GetUserByID returns a user by ID
// Get user by ID
// GetUserByID gets a user by ID
// @Summary Get user by ID
// @Description Returns user information for given ID
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 404 {object} map[string]string
// @Router /auth/user/{id} [get]
func GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User

	if err := db.DB.Preload("Role").Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Don't send password in response
	user.Password = ""

	return c.JSON(user)
}

// Send OTP
// SendOTP sends an OTP to user's email
// @Summary Send OTP
// @Description Sends OTP to the provided email for verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body object{email=string} true "Email for OTP"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /auth/send-otp [post]
func SendOTP(c *fiber.Ctx) error {
	type OTPRequest struct {
		Email string `json:"email"`
	}

	otpRequest := new(OTPRequest)
	if err := c.BodyParser(otpRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	var user models.User
	if db.DB.Where("email = ?", otpRequest.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Generate OTP
	otp := utils.GenerateOTP()
	user.OTP = otp
	user.OTPExpiresAt = time.Now().Add(10 * time.Minute) // Set OTP expiration time
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save OTP",
		})
	}
	// Send OTP via email
	if err := utils.SendEmail(otpRequest.Email, "Your OTP Code", fmt.Sprintf("Your OTP code is: %s ,Valid for 10 minutes", otp)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send OTP email",
		})
	}

	return c.JSON(fiber.Map{
		"message": "OTP sent successfully",
	})
}

// VerifyOTP verifies the OTP for a user

// Verify OTP
// VerifyOTP verifies OTP for a user
// @Summary Verify OTP
// @Description Verifies OTP sent to the email
// @Tags Auth
// @Accept json
// @Produce json
// @Param action query string false "Optional action e.g. reset"
// @Param data body object{email=string,otp=string} true "OTP verification data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /auth/otp/verify/ [post]
func VerifyOTP(c *fiber.Ctx) error {
	type OTPRequest struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	action := c.Query("action")

	otpRequest := new(OTPRequest)
	if err := c.BodyParser(otpRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}
	var token string
	if action == "reset" {
		token = utils.GenerateUUID()
		redis.Client.Set(redis.Ctx, otpRequest.Email, token, time.Minute*10)
	}

	var user models.User
	if db.DB.Where("email = ?", otpRequest.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if OTP is expired
	if user.OTPExpiresAt.Before(time.Now()) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "OTP expired",
		})
	}

	if user.OTP != otpRequest.OTP {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid OTP",
		})
	}

	// Update user to verified
	user.IsVerified = true
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save user",
		})
	}
	// Remove OTP from user
	user.OTP = ""
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save user",
		})
	}
	//Send Token only if not empty
	if token != "" {
		return c.JSON(fiber.Map{
			"message": "OTP verified successfully",
			"token":   token,
		})
	} else {
		return c.JSON(fiber.Map{
			"message": "OTP verified successfully",
		})
	}
}

// ResetPassword handles password reset

// Reset Password
// ResetPassword resets a user's password
// @Summary Reset password
// @Description Resets the password using a token from OTP verification
// @Tags Auth
// @Accept json
// @Produce json
// @Param token path string true "Reset token"
// @Param data body object{email=string,new_password=string} true "Password reset data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /auth/reset-password/{token} [post]
func ResetPassword(c *fiber.Ctx) error {
	var requestBody struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&requestBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	token := c.Params("token")
	// Verify Token if valid
	if token != redis.Client.Get(redis.Ctx, requestBody.Email).Val() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Find user
	var user models.User
	if db.DB.Where("email = ?", requestBody.Email).First(&user).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestBody.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}
	user.Password = string(hashedPassword)
	// Save new password
	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save user",
		})
	}
	// Send confirmation email
	if err := utils.SendEmail(requestBody.Email, "Password Reset Confirmation", "Your password has been reset successfully."); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send confirmation email",
		})
	}
	if !user.IsVerified {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "OTP verification required",
		})
	}
	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// RefreshToken generates a new access token using a refresh token
// RefreshToken generates new access token
// @Summary Refresh access token
// @Description Generates a new access token using a refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param data body object{refreshToken=string} true "Refresh token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
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
