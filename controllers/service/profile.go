package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

// Get Buisness Details and Provider Details By ID
func GetProviderDetailsByID(c *fiber.Ctx) error {
	type Profile struct {
		ProviderID uint
		Name       string
		Email      string
	}
	type details struct {
		BusinessDetails models.BusinessDetails
		Provider        Profile
	}

	id := c.Params("id")
	//search profile by id and get business details
	var provider models.User
	if err := db.DB.Preload("Role").First(&provider, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}
	var profile Profile
	profile.ProviderID = provider.ID
	profile.Name = provider.Name
	profile.Email = provider.Email
	var businessDetails models.BusinessDetails
	if err := db.DB.Where("provider_id = ?", id).First(&businessDetails).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Business details not found",
		})
	}

	return c.JSON(fiber.Map{
		"profile": details{
			BusinessDetails: businessDetails,
			Provider:        profile,
		},
	})
}

// GetAllServicesByProviderID retrieves all services by provider ID
func GetAllServicesByProviderID(c *fiber.Ctx) error {
	providerID := c.Params("id")
	var services []models.Service
	if err := db.DB.Where("provider_id = ?", providerID).Find(&services).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No services found for this provider",
		})
	}
	return c.JSON(fiber.Map{
		"services": services,
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
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No working hours found",
		})
	}

	return c.JSON(fiber.Map{
		"working_hours": workingHours,
	})
}

func CreateWorkingHours(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse input
	var inputHours []models.WorkingHours
	if err := c.BodyParser(&inputHours); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input: " + err.Error(),
		})
	}

	// Validate input
	if len(inputHours) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one working hours entry is required",
		})
	}

	// Check for duplicate days and validate times
	daySet := make(map[models.DayOfWeek]bool)
	for i, wh := range inputHours {
		// Validate day_of_week
		if wh.DayOfWeek < models.Sunday || wh.DayOfWeek > models.Saturday {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid day_of_week at index %d: must be 0-6", i),
			})
		}
		if daySet[wh.DayOfWeek] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Duplicate day_of_week %d at index %d", wh.DayOfWeek, i),
			})
		}
		daySet[wh.DayOfWeek] = true

		// Validate start_time and end_time
		startTime, err := time.Parse("15:04", wh.StartTime)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid start_time at index %d: must be HH:MM", i),
			})
		}
		endTime, err := time.Parse("15:04", wh.EndTime)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid end_time at index %d: must be HH:MM", i),
			})
		}
		if !endTime.After(startTime) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("end_time must be after start_time at index %d", i),
			})
		}

		// Validate break times if provided
		if wh.BreakStart != nil && wh.BreakEnd != nil {
			breakStart, err := time.Parse("15:04", *wh.BreakStart)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break_start at index %d: must be HH:MM", i),
				})
			}
			breakEnd, err := time.Parse("15:04", *wh.BreakEnd)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break_end at index %d: must be HH:MM", i),
				})
			}
			if !breakStart.After(startTime) || !breakEnd.After(breakStart) || !endTime.After(breakEnd) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break times at index %d: must be within working hours", i),
				})
			}
		} else if (wh.BreakStart != nil) != (wh.BreakEnd != nil) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Both break_start and break_end must be provided or omitted at index %d", i),
			})
		}
	}

	// Set provider ID
	for i := range inputHours {
		inputHours[i].ProviderID = userID
	}

	// Check if working hours already exist
	var existingHours []models.WorkingHours
	if err := db.DB.Where("provider_id = ?", userID).Find(&existingHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing working hours",
		})
	}
	if len(existingHours) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Working hours already exist; use update endpoint to modify",
		})
	}

	// Create working hours
	if err := db.DB.Create(&inputHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create working hours: " + err.Error(),
		})
	}

	// Retrieve created working hours
	var createdHours []models.WorkingHours
	if err := db.DB.Where("provider_id = ?", userID).Find(&createdHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve created working hours",
		})
	}

	return c.JSON(fiber.Map{
		"message":       "Working hours created successfully",
		"working_hours": createdHours,
	})
}

// UpdateWorkingHours updates the provider's working hours
func UpdateWorkingHours(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse input
	var inputHours []models.WorkingHours
	if err := c.BodyParser(&inputHours); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input: expected an array of working hours, " + err.Error(),
		})
	}

	// Validate input
	if len(inputHours) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one working hours entry is required",
		})
	}

	// Check for duplicate days and validate times
	daySet := make(map[models.DayOfWeek]bool)
	for i, wh := range inputHours {
		// Validate day_of_week
		if wh.DayOfWeek < models.Sunday || wh.DayOfWeek > models.Saturday {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid day_of_week at index %d: must be 0-6", i),
			})
		}
		if daySet[wh.DayOfWeek] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Duplicate day_of_week %d at index %d", wh.DayOfWeek, i),
			})
		}
		daySet[wh.DayOfWeek] = true

		// Validate start_time and end_time
		startTime, err := time.Parse("15:04", wh.StartTime)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid start_time at index %d: must be HH:MM", i),
			})
		}
		endTime, err := time.Parse("15:04", wh.EndTime)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Invalid end_time at index %d: must be HH:MM", i),
			})
		}
		if !endTime.After(startTime) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("end_time must be after start_time at index %d", i),
			})
		}

		// Validate break times if provided
		if wh.BreakStart != nil && wh.BreakEnd != nil {
			breakStart, err := time.Parse("15:04", *wh.BreakStart)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break_start at index %d: must be HH:MM", i),
				})
			}
			breakEnd, err := time.Parse("15:04", *wh.BreakEnd)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break_end at index %d: must be HH:MM", i),
				})
			}
			if !breakStart.After(startTime) || !breakEnd.After(breakStart) || !endTime.After(breakEnd) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid break times at index %d: must be within working hours", i),
				})
			}
		} else if (wh.BreakStart != nil) != (wh.BreakEnd != nil) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("Both break_start and break_end must be provided or omitted at index %d", i),
			})
		}
	}

	// Perform updates, creates, and deletes in a transaction
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Fetch existing working hours
		var existingHours []models.WorkingHours
		if err := tx.Where("provider_id = ?", userID).Find(&existingHours).Error; err != nil {
			return fmt.Errorf("failed to fetch existing working hours: %v", err)
		}

		// Create a map of existing hours by day_of_week for quick lookup
		existingMap := make(map[models.DayOfWeek]models.WorkingHours)
		for _, h := range existingHours {
			existingMap[h.DayOfWeek] = h
		}

		// Process input hours
		for _, input := range inputHours {
			input.ProviderID = userID
			if existing, exists := existingMap[input.DayOfWeek]; exists {
				// Update existing record
				if err := tx.Model(&existing).Updates(models.WorkingHours{
					StartTime:  input.StartTime,
					EndTime:    input.EndTime,
					BreakStart: input.BreakStart,
					BreakEnd:   input.BreakEnd,
				}).Error; err != nil {
					return fmt.Errorf("failed to update working hours for day %d: %v", input.DayOfWeek, err)
				}
			} else {
				// Create new record
				if err := tx.Create(&input).Error; err != nil {
					return fmt.Errorf("failed to create working hours for day %d: %v", input.DayOfWeek, err)
				}
			}
			// Remove from existingMap to track which days remain
			delete(existingMap, input.DayOfWeek)
		}

		// Delete any remaining existing hours not in the input
		for day := range existingMap {
			if err := tx.Where("provider_id = ? AND day_of_week = ?", userID, day).Delete(&models.WorkingHours{}).Error; err != nil {
				return fmt.Errorf("failed to delete working hours for day %d: %v", day, err)
			}
		}

		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update working hours: " + err.Error(),
		})
	}

	// Retrieve updated working hours
	var workingHours []models.WorkingHours
	if err := db.DB.Where("provider_id = ?", userID).Find(&workingHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated working hours: " + err.Error(),
		})
	}

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
