package consumer

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
)

type UserProfile struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Email       string             `json:"email"`
	UserDetials models.UserDetails `json:"user_details"`
}

// GetUserProfile returns the profile of the logged-in user
func GetUserProfile(c *fiber.Ctx) error {
	// Get user from context (set by middleware)
	userID := c.Locals("userID").(uint)
	var userProfile UserProfile
	var user models.User
	var userDetails models.UserDetails
	// Use Preload to load the associated Role and UserDetails
	// load user details with favorite services
	if err := db.DB.Preload("FavoriteServices").Where("user_id = ?", userID).First(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User details not found",
		})
	}

	// load user details
	if err := db.DB.Where("user_id = ?", userID).First(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User details not found",
		})
	}

	// load user profile
	userProfile.ID = user.ID
	userProfile.Name = user.Name
	userProfile.Email = user.Email
	userProfile.UserDetials = userDetails

	return c.JSON(userProfile)
}

type UserDetailsInput struct {
	FavoriteServiceIDs []uint `json:"favorite_service_ids"`
}

func CreateUserProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var input UserDetailsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var services []models.Service
	if len(input.FavoriteServiceIDs) > 0 {
		if err := db.DB.Where("id IN ?", input.FavoriteServiceIDs).Find(&services).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch favorite services",
			})
		}
	}

	userDetails := models.UserDetails{
		UserID:           userID,
		FavoriteServices: services,
	}

	if err := db.DB.Create(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user details",
		})
	}

	// Preload favorite services for full response
	var createdDetails models.UserDetails
	if err := db.DB.Preload("FavoriteServices").First(&createdDetails, userDetails.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch created user details",
		})
	}

	return c.JSON(createdDetails)
}

func UpdateUserProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	var input UserDetailsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var services []models.Service
	if len(input.FavoriteServiceIDs) > 0 {
		if err := db.DB.Where("id IN ?", input.FavoriteServiceIDs).Find(&services).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch favorite services",
			})
		}
	}

	userDetails := models.UserDetails{
		UserID:           userID,
		FavoriteServices: services,
	}

	if err := db.DB.Save(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user details",
		})
	}

	return c.JSON(userDetails)
}

func UpdateUserProfilePicture(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Get the file from the request
	file, err := c.FormFile("profile_picture")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get profile picture",
		})
	}

	// Open the file for Cloudinary upload
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open profile picture",
		})
	}
	defer f.Close()

	// Generate a unique public ID for Cloudinary (e.g., userID_timestamp)
	publicID := fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())

	// Upload to Cloudinary
	secureURL, err := utils.UploadToCloudinary(f, publicID, "profile_pictures")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload profile picture to Cloudinary",
		})
	}

	userDetails := models.UserDetails{
		UserID:         userID,
		ProfilePicture: secureURL,
	}

	if err := db.DB.Save(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile picture",
		})
	}

	return c.JSON(userDetails)
}

func DeleteUserProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Delete user details
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.UserDetails{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user details",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User details deleted successfully",
	})
}
