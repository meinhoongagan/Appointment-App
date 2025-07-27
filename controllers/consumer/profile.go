package consumer

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
)

// UserProfile represents the user profile response
type UserProfile struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Email       string             `json:"email"`
	UserDetails models.UserDetails `json:"user_details" swag:"type:object"`
}

// GetUserProfile returns the profile of the logged-in user
// @Summary Get user profile
// @Description Retrieve the profile details of the authenticated user, including user details and favorite services
// @Tags consumer
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserProfile "User profile details"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid user token"
// @Failure 404 {object} fiber.Map{error=string} "User details not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /consumer/profile [get]
func GetUserProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID in token",
		})
	}
	var userProfile UserProfile
	var user models.User
	var userDetails models.UserDetails
	if err := db.DB.Preload("FavoriteServices").Where("user_id = ?", userID).First(&userDetails).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User details not found",
		})
	}
	if err := db.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	userProfile.ID = user.ID
	userProfile.Name = user.Name
	userProfile.Email = user.Email
	userProfile.UserDetails = userDetails
	return c.JSON(userProfile)
}

// UserDetailsInput represents the input for creating/updating user profile
type UserDetailsInput struct {
	FavoriteServiceIDs []uint `json:"favorite_service_ids" swag:"type:array,items.type:integer"`
}

// CreateUserProfile creates a new user profile
// @Summary Create user profile
// @Description Create user details for the authenticated user, including favorite services
// @Tags consumer
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body UserDetailsInput true "User details input"
// @Success 201 {object} models.UserDetails "Created user details"
// @Failure 400 {object} fiber.Map{error=string} "Invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid user token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /consumer/profile [post]
func CreateUserProfile(c *fiber.Ctx) error {
	userProfileChan := make(chan models.UserDetails)

	go func() {
		userID, ok := c.Locals("userID").(uint)
		if !ok {
			userProfileChan <- models.UserDetails{}
			return
		}
		var input UserDetailsInput
		if err := c.BodyParser(&input); err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		var services []models.Service
		if len(input.FavoriteServiceIDs) > 0 {
			if err := db.DB.Where("id IN ?", input.FavoriteServiceIDs).Find(&services).Error; err != nil {
				userProfileChan <- models.UserDetails{}
				return
			}
		}
		userDetails := models.UserDetails{
			UserID:           userID,
			FavoriteServices: services,
		}
		if err := db.DB.Create(&userDetails).Error; err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		var createdDetails models.UserDetails
		if err := db.DB.Preload("FavoriteServices").First(&createdDetails, userDetails.ID).Error; err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		userProfileChan <- createdDetails
	}()

	createdDetails := <-userProfileChan
	if createdDetails.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user details",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(createdDetails)
}

// UpdateUserProfile updates the user profile
// @Summary Update user profile
// @Description Update the user details for the authenticated user, including favorite services
// @Tags consumer
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body UserDetailsInput true "Updated user details input"
// @Success 200 {object} models.UserDetails "Updated user details"
// @Failure 400 {object} fiber.Map{error=string} "Invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid user token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /consumer/profile [patch]
func UpdateUserProfile(c *fiber.Ctx) error {
	userProfileChan := make(chan models.UserDetails)

	go func() {
		userID, ok := c.Locals("userID").(uint)
		if !ok {
			userProfileChan <- models.UserDetails{}
			return
		}
		var input UserDetailsInput
		if err := c.BodyParser(&input); err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		var services []models.Service
		if len(input.FavoriteServiceIDs) > 0 {
			if err := db.DB.Where("id IN ?", input.FavoriteServiceIDs).Find(&services).Error; err != nil {
				userProfileChan <- models.UserDetails{}
				return
			}
		}
		var userDetails models.UserDetails
		if err := db.DB.Preload("FavoriteServices").First(&userDetails, userID).Error; err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		userDetails.FavoriteServices = services
		if err := db.DB.Save(&userDetails).Error; err != nil {
			userProfileChan <- models.UserDetails{}
			return
		}
		userProfileChan <- userDetails
	}()

	updatedDetails := <-userProfileChan
	if updatedDetails.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user details",
		})
	}
	return c.JSON(updatedDetails)
}

// UpdateUserProfilePicture updates the user's profile picture
// @Summary Update user profile picture
// @Description Upload and update the profile picture for the authenticated user using Cloudinary
// @Tags consumer
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param profile_picture formData file true "Profile picture file"
// @Success 200 {object} models.UserDetails "Updated user details with profile picture URL"
// @Failure 400 {object} fiber.Map{error=string} "Failed to get profile picture"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid user token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /consumer/profile/picture [post]
func UpdateUserProfilePicture(c *fiber.Ctx) error {
	userProfilePictureChan := make(chan models.UserDetails)

	go func() {
		userID, ok := c.Locals("userID").(uint)
		if !ok {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		file, err := c.FormFile("profile_picture")
		if err != nil {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		f, err := file.Open()
		if err != nil {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		defer f.Close()
		publicID := fmt.Sprintf("user_%d_%d", userID, time.Now().Unix())
		secureURL, err := utils.UploadToCloudinary(f, publicID, "profile_pictures")
		if err != nil {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		var userDetails models.UserDetails
		if err := db.DB.Preload("FavoriteServices").First(&userDetails, userID).Error; err != nil {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		userDetails.ProfilePicture = secureURL
		if err := db.DB.Save(&userDetails).Error; err != nil {
			userProfilePictureChan <- models.UserDetails{}
			return
		}
		userProfilePictureChan <- userDetails
	}()

	updatedDetails := <-userProfilePictureChan
	if updatedDetails.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile picture",
		})
	}
	return c.JSON(updatedDetails)
}

// DeleteUserProfile deletes the user profile
// @Summary Delete user profile
// @Description Delete the user details for the authenticated user
// @Tags consumer
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} fiber.Map{message=string} "User details deleted successfully"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid user token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /consumer/profile [delete]
func DeleteUserProfile(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID in token",
		})
	}
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.UserDetails{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user details",
		})
	}
	return c.JSON(fiber.Map{
		"message": "User details deleted successfully",
	})
}
