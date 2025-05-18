package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
	"golang.org/x/crypto/bcrypt"
)

// GetProviderProfile retrieves the provider's profile information
func GetProviderProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var provider models.User
	if err := db.DB.Preload("Role").First(&provider, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider profile not found",
		})
	}

	return c.JSON(fiber.Map{
		"profile": provider,
	})
}

// UpdateProviderProfile updates the provider's personal information
func UpdateProviderProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Find existing provider
	var provider models.User
	if err := db.DB.First(&provider, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Parse update data
	updateData := make(map[string]interface{})
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Remove fields that shouldn't be updated directly
	fieldsToIgnore := []string{"id", "ID", "role", "Role", "RoleID", "role_id", "password"}
	for _, field := range fieldsToIgnore {
		delete(updateData, field)
	}

	// Update provider profile
	if err := db.DB.Model(&provider).Updates(updateData).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	// Refresh provider data
	if err := db.DB.Preload("Role").First(&provider, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated profile",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"profile": provider,
	})
}

// GetBusinessDetails retrieves the provider's business details
func GetBusinessDetails(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Assuming there's a BusinessDetails model linked to the provider
	var businessDetails models.BusinessDetails
	if err := db.DB.Where("provider_id = ?", userID).First(&businessDetails).Error; err != nil {
		// If not found, return empty details rather than error
		return c.JSON(fiber.Map{
			"business_details": models.BusinessDetails{
				ProviderID: userID,
			},
		})
	}

	return c.JSON(fiber.Map{
		"business_details": businessDetails,
	})
}

// UpdateBusinessDetails updates the provider's business details
func UpdateBusinessDetails(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var businessDetails models.BusinessDetails
	// Try to find existing business details
	result := db.DB.Where("provider_id = ?", userID).First(&businessDetails)

	// Parse update data
	updatedDetails := new(models.BusinessDetails)
	if err := c.BodyParser(updatedDetails); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Ensure provider ID is set correctly
	updatedDetails.ProviderID = userID

	// If business details exist, update them
	if result.RowsAffected > 0 {
		if err := db.DB.Model(&businessDetails).Updates(updatedDetails).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update business details",
			})
		}
	} else {
		// If not exists, create new business details
		if err := db.DB.Create(updatedDetails).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create business details",
			})
		}
	}

	// Get the updated/created business details
	db.DB.Where("provider_id = ?", userID).First(&businessDetails)

	return c.JSON(fiber.Map{
		"message":          "Business details updated successfully",
		"business_details": businessDetails,
	})
}

func UploadBusinessMedia(c *fiber.Ctx) error {
	// Assume provider_id is stored in Locals from JWT middleware
	providerID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Message: "Provider ID not found",
			Error:   "Authentication required or provider_id missing",
		})
	}

	// Check if BusinessDetails exists
	var businessDetails models.BusinessDetails
	if err := db.DB.Where("provider_id = ?", providerID).First(&businessDetails).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Business details not found",
			Error:   err.Error(),
		})
	}

	// Parse multipart form (max 10 MB)
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Failed to parse multipart form",
			Error:   err.Error(),
		})
	}

	// Create temporary directory for uploads
	tempDir := "uploads"
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to create temp directory",
			Error:   err.Error(),
		})
	}

	// Handle profile picture
	profileFiles := form.File["profile_picture"]
	if len(profileFiles) > 0 {
		profileFile := profileFiles[0]
		// Validate file type
		allowedTypes := map[string]bool{"image/jpeg": true, "image/png": true}
		if !allowedTypes[profileFile.Header.Get("Content-Type")] {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Message: "Invalid profile picture type. Only JPEG/PNG allowed",
			})
		}

		tempPath := filepath.Join(tempDir, profileFile.Filename)
		if err := c.SaveFile(profileFile, tempPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Message: "Failed to save profile picture",
				Error:   err.Error(),
			})
		}
		defer os.Remove(tempPath) // Clean up

		publicID := fmt.Sprintf("provider_%d_profile", providerID)
		url, err := utils.UploadToCloudinary(tempPath, publicID, "provider_profiles")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Message: "Failed to upload profile picture to Cloudinary",
				Error:   err.Error(),
			})
		}
		businessDetails.ProfilePictureURL = url
	}

	// Handle certificates
	certificateFiles := form.File["certificates"]
	var certificateURLs []string

	// Safely unmarshal existing certificate URLs if not empty
	if businessDetails.CertificateURLs != "" {
		if err := json.Unmarshal([]byte(businessDetails.CertificateURLs), &certificateURLs); err != nil {
			// If there's an error, initialize as empty array rather than failing
			certificateURLs = []string{}
			log.Printf("Error parsing existing certificate URLs: %v. Initializing empty array.", err)
		}
	}

	for i, certFile := range certificateFiles {
		// Validate file type
		if certFile.Header.Get("Content-Type") != "application/pdf" {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Message: "Invalid certificate type. Only PDF allowed",
			})
		}

		tempPath := filepath.Join(tempDir, certFile.Filename)
		if err := c.SaveFile(certFile, tempPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Message: "Failed to save certificate",
				Error:   err.Error(),
			})
		}
		defer os.Remove(tempPath) // Clean up

		publicID := fmt.Sprintf("provider_%d_cert_%d", providerID, i+1)
		// Upload certificate without resizing by passing nil for the transformation
		// Modify utils.UploadToCloudinary to accept nil transformation for PDFs
		url, err := utils.UploadToCloudinary(tempPath, publicID, "certificates")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Message: "Failed to upload certificate to Cloudinary",
				Error:   err.Error(),
			})
		}
		certificateURLs = append(certificateURLs, url)
	}

	// Update CertificateURLs - ensure it's always a valid JSON string array
	if len(certificateURLs) > 0 {
		urlsJSON, err := json.Marshal(certificateURLs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
				Message: "Failed to serialize certificate URLs",
				Error:   err.Error(),
			})
		}
		businessDetails.CertificateURLs = string(urlsJSON)
	} else if businessDetails.CertificateURLs == "" {
		// Ensure we have a valid empty JSON array if there are no certificates
		businessDetails.CertificateURLs = "[]"
	}

	// Save updates to database
	if err := db.DB.Save(&businessDetails).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to update business details",
			Error:   err.Error(),
		})
	}

	// Send confirmation email
	var provider models.User
	if err := db.DB.First(&provider, providerID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Provider not found",
			Error:   err.Error(),
		})
	}

	emailBody := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your profile media has been updated successfully.</p>
		<p><strong>Details:</strong></p>
		<ul>
			<li><strong>Profile Picture:</strong> %s</li>
			<li><strong>Certificates:</strong> %d uploaded</li>
		</ul>
		<p>Best regards,</p>
		<p>Your Appointment Team</p>
	`, provider.Name, businessDetails.ProfilePictureURL, len(certificateURLs))
	if err := utils.SendEmail(provider.Email, "Profile Media Updated", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to send confirmation email",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Media uploaded successfully",
		"profile_picture":  businessDetails.ProfilePictureURL,
		"certificate_urls": certificateURLs,
	})
}

// GetProviderSettings retrieves the provider's settings
func GetProviderSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var settings models.ProviderSettings
	if err := db.DB.Where("provider_id = ?", userID).First(&settings).Error; err != nil {
		// If settings not found, return default settings
		return c.JSON(fiber.Map{
			"settings": models.ProviderSettings{
				ProviderID:           userID,
				NotificationsEnabled: true,
				AutoConfirmBookings:  false,
				AdvanceBookingDays:   30,
			},
		})
	}

	return c.JSON(fiber.Map{
		"settings": settings,
	})
}

// UpdateProviderSettings updates the provider's settings
func UpdateProviderSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var settings models.ProviderSettings
	// Try to find existing settings
	result := db.DB.Where("provider_id = ?", userID).First(&settings)

	// Parse update data
	updatedSettings := new(models.ProviderSettings)
	if err := c.BodyParser(updatedSettings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Ensure provider ID is set correctly
	updatedSettings.ProviderID = userID

	// If settings exist, update them
	if result.RowsAffected > 0 {
		if err := db.DB.Model(&settings).Updates(updatedSettings).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update settings",
			})
		}
	} else {
		// If not exists, create new settings
		if err := db.DB.Create(updatedSettings).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create settings",
			})
		}
	}

	// Get the updated/created settings
	db.DB.Where("provider_id = ?", userID).First(&settings)

	return c.JSON(fiber.Map{
		"message":  "Settings updated successfully",
		"settings": settings,
	})
}

// GetWorkingHours retrieves the provider's working hours
func GetWorkingHours(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var workingHours []models.WorkingHours
	if err := db.DB.Where("provider_id = ?", userID).Find(&workingHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve working hours",
		})
	}

	// If no working hours are found, return default template
	if len(workingHours) == 0 {
		defaultHours := []models.WorkingHours{
			{ProviderID: userID, DayOfWeek: 1, StartTime: "09:00", EndTime: "17:00"},
			{ProviderID: userID, DayOfWeek: 2, StartTime: "09:00", EndTime: "17:00"},
			{ProviderID: userID, DayOfWeek: 3, StartTime: "09:00", EndTime: "17:00"},
			{ProviderID: userID, DayOfWeek: 4, StartTime: "09:00", EndTime: "17:00"},
			{ProviderID: userID, DayOfWeek: 5, StartTime: "09:00", EndTime: "17:00"},
			{ProviderID: userID, DayOfWeek: 6, StartTime: "00:00", EndTime: "00:00"},
			{ProviderID: userID, DayOfWeek: 0, StartTime: "00:00", EndTime: "00:00"},
		}
		return c.JSON(fiber.Map{
			"working_hours": defaultHours,
		})
	}

	return c.JSON(fiber.Map{
		"working_hours": workingHours,
	})
}

// UpdateWorkingHours updates the provider's working hours
func UpdateWorkingHours(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse update data
	var updatedHours []models.WorkingHours
	if err := c.BodyParser(&updatedHours); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Delete existing working hours for the provider
	if err := db.DB.Where("provider_id = ?", userID).Delete(&models.WorkingHours{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update working hours",
		})
	}

	// Set provider ID for all entries
	for i := range updatedHours {
		updatedHours[i].ProviderID = userID
	}

	// Create new working hours
	if err := db.DB.Create(&updatedHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update working hours",
		})
	}

	// Get the updated working hours
	var workingHours []models.WorkingHours
	db.DB.Where("provider_id = ?", userID).Find(&workingHours)

	return c.JSON(fiber.Map{
		"message":       "Working hours updated successfully",
		"working_hours": workingHours,
	})
}

func CreateReceptionist(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse request body into User struct
	receptionist := new(models.User)
	if err := c.BodyParser(receptionist); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(receptionist.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}
	receptionist.Password = string(hashedPassword)

	// Assign role ID for receptionist
	receptionist.RoleID = 4

	// Create the receptionist user
	if err := db.DB.Create(receptionist).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create receptionist: " + err.Error(),
		})
	}

	// Find the provider (assumed to be the authenticated user)
	var provider models.User
	if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Create receptionist settings entry
	receptionistSettings := models.ReceptionistSettings{
		ReceptionistID: receptionist.ID,
		ProviderID:     provider.ID,
	}
	if err := db.DB.Create(&receptionistSettings).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create receptionist settings: " + err.Error(),
		})
	}

	return c.JSON(receptionist)
}

func GetReceptionistList(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Find the provider
	var provider models.User
	if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	var receptionistSettings []models.ReceptionistSettings
	if err := db.DB.Preload("Receptionist").Preload("Provider").Find(&receptionistSettings, "provider_id = ?", provider.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch receptionists: " + err.Error(),
		})
	}
	// If no receptionists are found, return an empty list
	if len(receptionistSettings) == 0 {
		return c.JSON(fiber.Map{
			"receptionists": []models.ReceptionistSettings{},
		})
	}
	//find receptionist by receptionist id
	var receptionistIDs []uint
	for _, setting := range receptionistSettings {
		receptionistIDs = append(receptionistIDs, setting.ReceptionistID)
	}

	var receptionists []models.User
	if err := db.DB.Where("id IN ?", receptionistIDs).Find(&receptionists).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch receptionists: " + err.Error(),
		})
	}

	return c.JSON(receptionists)
}
func GetReceptionistByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse receptionist ID from URL parameter
	receptionistID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid receptionist ID",
		})
	}

	// Find the provider
	var provider models.User
	if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	var receptionist models.User
	if err := db.DB.Where("id = ? AND role_id = 4", receptionistID).First(&receptionist).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Receptionist not found",
		})
	}

	return c.JSON(receptionist)
}

func DeleteReceptionist(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse receptionist ID from URL parameter
	receptionistID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid receptionist ID",
		})
	}

	// Find the provider
	var provider models.User
	if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Delete the receptionist settings
	if err := db.DB.Where("receptionist_id = ? AND provider_id = ?", receptionistID, provider.ID).Delete(&models.ReceptionistSettings{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete receptionist settings: " + err.Error(),
		})
	}

	// Delete the receptionist user
	if err := db.DB.Delete(&models.User{}, receptionistID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete receptionist: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Receptionist deleted successfully",
	})
}
