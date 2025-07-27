package service

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
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
// @Summary Get provider profile
// @Description Retrieve the authenticated provider's personal profile information, including role details
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{profile=models.User} "Provider profile"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "Provider profile not found"
// @Router /provider/profile [get]
func GetProviderProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type profileResult struct {
		Provider models.User
		Err      error
	}

	resultChan := make(chan profileResult)

	go func() {
		var result profileResult

		if err := db.DB.Preload("Role").First(&result.Provider, userID).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider profile not found",
		})
	}

	return c.JSON(fiber.Map{
		"profile": result.Provider,
	})
}

// GetProviderDetailsByID retrieves provider details and business details by provider ID
// @Summary Get provider details by ID
// @Description Retrieve provider profile and business details for a specific provider by ID
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Provider ID"
// @Success 200 {object} object{profile=object{ProviderID=uint,Name=string,Email=string,BusinessDetails=models.BusinessDetails}} "Provider and business details"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "Provider or business details not found"
// @Router /provider/profile/{id} [get]
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

	type detailsResult struct {
		Provider        models.User
		BusinessDetails models.BusinessDetails
		Err             error
		ErrType         string
	}

	resultChan := make(chan detailsResult)

	go func() {
		var result detailsResult

		// Get provider details
		if err := db.DB.Preload("Role").First(&result.Provider, id).Error; err != nil {
			result.Err = err
			result.ErrType = "provider"
			resultChan <- result
			return
		}

		// Get business details
		if err := db.DB.Where("provider_id = ?", id).First(&result.BusinessDetails).Error; err != nil {
			result.Err = err
			result.ErrType = "business"
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		if result.ErrType == "provider" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		} else {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Business details not found",
			})
		}
	}

	var profile Profile
	profile.ProviderID = result.Provider.ID
	profile.Name = result.Provider.Name
	profile.Email = result.Provider.Email

	return c.JSON(fiber.Map{
		"profile": details{
			BusinessDetails: result.BusinessDetails,
			Provider:        profile,
		},
	})
}

// GetAllServicesByProviderID retrieves all services by provider ID
// @Summary Get all services by provider ID
// @Description Retrieve a list of all services offered by a specific provider
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Provider ID"
// @Success 200 {object} object{services=[]models.Service} "List of services"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "No services found for this provider"
// @Router /provider/profile/services/{id} [get]
func GetAllServicesByProviderID(c *fiber.Ctx) error {
	providerID := c.Params("id")

	type servicesResult struct {
		Services []models.Service
		Err      error
	}

	resultChan := make(chan servicesResult)

	go func() {
		var result servicesResult

		if err := db.DB.Where("provider_id = ?", providerID).Find(&result.Services).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No services found for this provider",
		})
	}

	return c.JSON(fiber.Map{
		"services": result.Services,
	})
}

// UpdateProviderProfile updates the provider's personal information
// @Summary Update provider profile
// @Description Update the authenticated provider's personal profile information (e.g., name, email)
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "Updated profile information (fields like name, email; id, role, password ignored)"
// @Success 200 {object} object{message=string,profile=models.User} "Profile updated successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile [patch]
func UpdateProviderProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

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

	type updateResult struct {
		Provider models.User
		Err      error
		ErrType  string
	}

	resultChan := make(chan updateResult)

	go func() {
		var result updateResult

		// Find existing provider
		var provider models.User
		if err := db.DB.First(&provider, userID).Error; err != nil {
			result.Err = err
			result.ErrType = "not_found"
			resultChan <- result
			return
		}

		// Update provider profile
		if err := db.DB.Model(&provider).Updates(updateData).Error; err != nil {
			result.Err = err
			result.ErrType = "update_failed"
			resultChan <- result
			return
		}

		// Refresh provider data
		if err := db.DB.Preload("Role").First(&result.Provider, userID).Error; err != nil {
			result.Err = err
			result.ErrType = "refresh_failed"
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		case "update_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update profile",
			})
		case "refresh_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve updated profile",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"profile": result.Provider,
	})
}

// GetBusinessDetails retrieves the provider's business details
// @Summary Get business details
// @Description Retrieve the authenticated provider's business details; returns default empty details if none exist
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{business_details=models.BusinessDetails} "Business details"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Router /provider/profile/business [get]
func GetBusinessDetails(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type businessResult struct {
		Details models.BusinessDetails
		Found   bool
		Err     error
	}

	resultChan := make(chan businessResult)

	go func() {
		var result businessResult

		// Assuming there's a BusinessDetails model linked to the provider
		var businessDetails models.BusinessDetails
		if err := db.DB.Where("provider_id = ?", userID).First(&businessDetails).Error; err != nil {
			// Not found is not considered an error in this case
			result.Details = models.BusinessDetails{
				ProviderID: userID,
			}
			result.Found = false
		} else {
			result.Details = businessDetails
			result.Found = true
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve business details",
		})
	}

	return c.JSON(fiber.Map{
		"business_details": result.Details,
	})
}

// UpdateBusinessDetails updates the provider's business details
// @Summary Update business details
// @Description Update or create the authenticated provider's business details
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.BusinessDetails true "Updated business details"
// @Success 200 {object} object{message=string,business_details=models.BusinessDetails} "Business details updated"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile/business [patch]
func UpdateBusinessDetails(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse update data
	updatedDetails := new(models.BusinessDetails)
	if err := c.BodyParser(updatedDetails); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Ensure provider ID is set correctly
	updatedDetails.ProviderID = userID

	type updateBusinessResult struct {
		Details models.BusinessDetails
		Err     error
		ErrType string
	}

	resultChan := make(chan updateBusinessResult)

	go func() {
		var result updateBusinessResult

		var businessDetails models.BusinessDetails
		// Try to find existing business details
		dbResult := db.DB.Where("provider_id = ?", userID).First(&businessDetails)

		// If business details exist, update them
		if dbResult.RowsAffected > 0 {
			if err := db.DB.Model(&businessDetails).Updates(updatedDetails).Error; err != nil {
				result.Err = err
				result.ErrType = "update_failed"
				resultChan <- result
				return
			}
		} else {
			// If not exists, create new business details
			if err := db.DB.Create(updatedDetails).Error; err != nil {
				result.Err = err
				result.ErrType = "create_failed"
				resultChan <- result
				return
			}
		}

		// Get the updated/created business details
		if err := db.DB.Where("provider_id = ?", userID).First(&result.Details).Error; err != nil {
			result.Err = err
			result.ErrType = "retrieve_failed"
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "update_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update business details",
			})
		case "create_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create business details",
			})
		case "retrieve_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve updated business details",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":          "Business details updated successfully",
		"business_details": result.Details,
	})
}

// UploadBusinessMedia uploads profile picture and certificates for the provider
// @Summary Upload business media
// @Description Upload a profile picture (JPEG/PNG) and/or certificates (PDF) for the authenticated provider's business details
// @Tags provider-profile
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param profile_picture formData file false "Profile picture (JPEG or PNG)"
// @Param certificates formData file false "Certificates (PDF)" collectionFormat=multi
// @Success 200 {object} object{message=string,profile_picture=string,certificate_urls=[]string} "Media uploaded successfully"
// @Failure 400 {object} utils.ErrorResponse{Message=string,Error=string} "Bad request - invalid file type or form data"
// @Failure 401 {object} utils.ErrorResponse{Message=string,Error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} utils.ErrorResponse{Message=string,Error=string} "Business details or provider not found"
// @Failure 500 {object} utils.ErrorResponse{Message=string,Error=string} "Internal server error"
// @Router /provider/profile/business/upload-media [post]
func UploadBusinessMedia(c *fiber.Ctx) error {
	// Assume provider_id is stored in Locals from JWT middleware
	providerID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ErrorResponse{
			Message: "Provider ID not found",
			Error:   "Authentication required or provider_id missing",
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

	// Validate profile picture file type
	profileFiles := form.File["profile_picture"]
	var profileFile *multipart.FileHeader
	if len(profileFiles) > 0 {
		profileFile = profileFiles[0]
		allowedTypes := map[string]bool{"image/jpeg": true, "image/png": true}
		if !allowedTypes[profileFile.Header.Get("Content-Type")] {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Message: "Invalid profile picture type. Only JPEG/PNG allowed",
			})
		}
	}

	// Validate certificate file types
	certificateFiles := form.File["certificates"]
	for _, certFile := range certificateFiles {
		if certFile.Header.Get("Content-Type") != "application/pdf" {
			return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
				Message: "Invalid certificate type. Only PDF allowed",
			})
		}
	}

	type uploadResult struct {
		BusinessDetails   models.BusinessDetails
		Provider          models.User
		ProfilePictureURL string
		CertificateURLs   []string
		Err               error
		ErrType           string
		ErrMessage        string
	}

	resultChan := make(chan uploadResult)

	go func() {
		var result uploadResult

		// Check if BusinessDetails exists
		var businessDetails models.BusinessDetails
		if err := db.DB.Where("provider_id = ?", providerID).First(&businessDetails).Error; err != nil {
			result.Err = err
			result.ErrType = "business_details_not_found"
			result.ErrMessage = "Business details not found"
			resultChan <- result
			return
		}

		// Handle profile picture
		if profileFile != nil {
			tempPath := filepath.Join(tempDir, profileFile.Filename)
			if err := c.SaveFile(profileFile, tempPath); err != nil {
				result.Err = err
				result.ErrType = "save_profile_picture_failed"
				result.ErrMessage = "Failed to save profile picture"
				resultChan <- result
				return
			}
			defer os.Remove(tempPath) // Clean up

			publicID := fmt.Sprintf("provider_%d_profile", providerID)
			url, err := utils.UploadToCloudinary(tempPath, publicID, "provider_profiles")
			if err != nil {
				result.Err = err
				result.ErrType = "upload_profile_picture_failed"
				result.ErrMessage = "Failed to upload profile picture to Cloudinary"
				resultChan <- result
				return
			}
			businessDetails.ProfilePictureURL = url
			result.ProfilePictureURL = url
		}

		// Handle certificates
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
			tempPath := filepath.Join(tempDir, certFile.Filename)
			if err := c.SaveFile(certFile, tempPath); err != nil {
				result.Err = err
				result.ErrType = "save_certificate_failed"
				result.ErrMessage = "Failed to save certificate"
				resultChan <- result
				return
			}
			defer os.Remove(tempPath) // Clean up

			publicID := fmt.Sprintf("provider_%d_cert_%d", providerID, i+1)
			// Upload certificate without resizing by passing nil for the transformation
			url, err := utils.UploadToCloudinary(tempPath, publicID, "certificates")
			if err != nil {
				result.Err = err
				result.ErrType = "upload_certificate_failed"
				result.ErrMessage = "Failed to upload certificate to Cloudinary"
				resultChan <- result
				return
			}
			certificateURLs = append(certificateURLs, url)
		}

		// Update CertificateURLs - ensure it's always a valid JSON string array
		if len(certificateURLs) > 0 {
			urlsJSON, err := json.Marshal(certificateURLs)
			if err != nil {
				result.Err = err
				result.ErrType = "serialize_urls_failed"
				result.ErrMessage = "Failed to serialize certificate URLs"
				resultChan <- result
				return
			}
			businessDetails.CertificateURLs = string(urlsJSON)
		} else if businessDetails.CertificateURLs == "" {
			// Ensure we have a valid empty JSON array if there are no certificates
			businessDetails.CertificateURLs = "[]"
		}

		// Save updates to database
		if err := db.DB.Save(&businessDetails).Error; err != nil {
			result.Err = err
			result.ErrType = "update_business_details_failed"
			result.ErrMessage = "Failed to update business details"
			resultChan <- result
			return
		}

		// Get provider details for email
		var provider models.User
		if err := db.DB.First(&provider, providerID).Error; err != nil {
			result.Err = err
			result.ErrType = "provider_not_found"
			result.ErrMessage = "Provider not found"
			resultChan <- result
			return
		}

		// Send confirmation email
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
			result.Err = err
			result.ErrType = "send_email_failed"
			result.ErrMessage = "Failed to send confirmation email"
			resultChan <- result
			return
		}

		result.BusinessDetails = businessDetails
		result.Provider = provider
		result.CertificateURLs = certificateURLs
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(getStatusCodeForError(result.ErrType)).JSON(utils.ErrorResponse{
			Message: result.ErrMessage,
			Error:   result.Err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Media uploaded successfully",
		"profile_picture":  result.BusinessDetails.ProfilePictureURL,
		"certificate_urls": result.CertificateURLs,
	})
}

// Helper function to determine status code based on error type
func getStatusCodeForError(errType string) int {
	switch errType {
	case "business_details_not_found", "provider_not_found":
		return fiber.StatusNotFound
	case "save_profile_picture_failed", "save_certificate_failed", "upload_profile_picture_failed",
		"upload_certificate_failed", "serialize_urls_failed", "update_business_details_failed", "send_email_failed":
		return fiber.StatusInternalServerError
	default:
		return fiber.StatusInternalServerError
	}
}

// GetProviderSettings retrieves the provider's settings
// @Summary Get provider settings
// @Description Retrieve the authenticated provider's settings; returns default settings if none exist
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{settings=models.ProviderSettings} "Provider settings"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Router /provider/profile/settings [get]
func GetProviderSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type settingsResult struct {
		Settings models.ProviderSettings
		Found    bool
		Err      error
	}

	resultChan := make(chan settingsResult)

	go func() {
		var result settingsResult

		var settings models.ProviderSettings
		if err := db.DB.Where("provider_id = ?", userID).First(&settings).Error; err != nil {
			// If settings not found, prepare default settings
			result.Settings = models.ProviderSettings{
				ProviderID:           userID,
				NotificationsEnabled: true,
				AutoConfirmBookings:  false,
				AdvanceBookingDays:   30,
			}
			result.Found = false
		} else {
			result.Settings = settings
			result.Found = true
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve provider settings",
		})
	}

	return c.JSON(fiber.Map{
		"settings": result.Settings,
	})
}

// UpdateProviderSettings updates the provider's settings
// @Summary Update provider settings
// @Description Update or create the authenticated provider's settings (e.g., notifications, auto-confirm, advance booking days)
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.ProviderSettings true "Updated provider settings"
// @Success 200 {object} object{message=string,settings=models.ProviderSettings} "Settings updated successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile/settings [patch]
func UpdateProviderSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse update data
	updatedSettings := new(models.ProviderSettings)
	if err := c.BodyParser(updatedSettings); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Ensure provider ID is set correctly
	updatedSettings.ProviderID = userID

	type updateSettingsResult struct {
		Settings models.ProviderSettings
		Err      error
		ErrType  string
	}

	resultChan := make(chan updateSettingsResult)

	go func() {
		var result updateSettingsResult

		var settings models.ProviderSettings
		// Try to find existing settings
		dbResult := db.DB.Where("provider_id = ?", userID).First(&settings)

		// If settings exist, update them
		if dbResult.RowsAffected > 0 {
			if err := db.DB.Model(&settings).Updates(updatedSettings).Error; err != nil {
				result.Err = err
				result.ErrType = "update_failed"
				resultChan <- result
				return
			}
		} else {
			// If not exists, create new settings
			if err := db.DB.Create(updatedSettings).Error; err != nil {
				result.Err = err
				result.ErrType = "create_failed"
				resultChan <- result
				return
			}
		}

		// Get the updated/created settings
		if err := db.DB.Where("provider_id = ?", userID).First(&result.Settings).Error; err != nil {
			result.Err = err
			result.ErrType = "retrieve_failed"
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "update_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update settings",
			})
		case "create_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create settings",
			})
		case "retrieve_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve updated settings",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":  "Settings updated successfully",
		"settings": result.Settings,
	})
}

// GetWorkingHours retrieves the provider's working hours
// @Summary Get working hours
// @Description Retrieve the authenticated provider's working hours
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{working_hours=[]models.WorkingHours} "List of working hours"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "No working hours found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile/working-hours [get]
func GetWorkingHours(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type workingHoursResult struct {
		Hours []models.WorkingHours
		Err   error
	}

	resultChan := make(chan workingHoursResult)

	go func() {
		var result workingHoursResult

		var workingHours []models.WorkingHours
		if err := db.DB.Where("provider_id = ?", userID).Find(&workingHours).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		result.Hours = workingHours
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve working hours",
		})
	}

	// If no working hours are found, return default template
	if len(result.Hours) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No working hours found",
		})
	}

	return c.JSON(fiber.Map{
		"working_hours": result.Hours,
	})
}

// CreateWorkingHours creates new working hours for the provider
// @Summary Create working hours
// @Description Create new working hours entries for the authenticated provider
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body []models.WorkingHours true "Array of working hours entries"
// @Success 200 {object} object{message=string,working_hours=[]models.WorkingHours} "Working hours created successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input, duplicate days, or invalid times"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile/working-hours [post]
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

	type createHoursResult struct {
		CreatedHours []models.WorkingHours
		Err          error
		ErrType      string
	}

	resultChan := make(chan createHoursResult)

	go func() {
		var result createHoursResult

		// Check if working hours already exist
		var existingHours []models.WorkingHours
		if err := db.DB.Where("provider_id = ?", userID).Find(&existingHours).Error; err != nil {
			result.Err = err
			result.ErrType = "check_failed"
			resultChan <- result
			return
		}
		if len(existingHours) > 0 {
			result.Err = fmt.Errorf("working hours already exist")
			result.ErrType = "already_exists"
			resultChan <- result
			return
		}

		// Create working hours
		if err := db.DB.Create(&inputHours).Error; err != nil {
			result.Err = err
			result.ErrType = "create_failed"
			resultChan <- result
			return
		}

		// Retrieve created working hours
		var createdHours []models.WorkingHours
		if err := db.DB.Where("provider_id = ?", userID).Find(&createdHours).Error; err != nil {
			result.Err = err
			result.ErrType = "retrieve_failed"
			resultChan <- result
			return
		}

		result.CreatedHours = createdHours
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "check_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check existing working hours",
			})
		case "already_exists":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Working hours already exist; use update endpoint to modify",
			})
		case "create_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create working hours: " + result.Err.Error(),
			})
		case "retrieve_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve created working hours",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":       "Working hours created successfully",
		"working_hours": result.CreatedHours,
	})
}

// UpdateWorkingHours updates the provider's working hours
// @Summary Update working hours
// @Description Update the authenticated provider's working hours, replacing existing entries
// @Tags provider-profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body []models.WorkingHours true "Array of working hours entries"
// @Success 200 {object} object{message=string,working_hours=[]models.WorkingHours} "Working hours updated successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input, duplicate days, or invalid times"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/profile/working-hours [patch]
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

	// Set provider ID for all input hours
	for i := range inputHours {
		inputHours[i].ProviderID = userID
	}

	type updateHoursResult struct {
		UpdatedHours []models.WorkingHours
		Err          error
	}

	resultChan := make(chan updateHoursResult)

	go func() {
		var result updateHoursResult

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
			result.Err = err
			resultChan <- result
			return
		}

		// Retrieve updated working hours
		var workingHours []models.WorkingHours
		if err := db.DB.Where("provider_id = ?", userID).Find(&workingHours).Error; err != nil {
			result.Err = fmt.Errorf("failed to retrieve updated working hours: %v", err)
			resultChan <- result
			return
		}

		result.UpdatedHours = workingHours
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update working hours: " + result.Err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":       "Working hours updated successfully",
		"working_hours": result.UpdatedHours,
	})
}

// CreateReceptionist creates a new receptionist for the provider
// @Summary Create receptionist
// @Description Create a new receptionist user and link them to the authenticated provider, requires 'services' create permission
// @Tags provider-receptionist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.User true "Receptionist user details (e.g., name, email, password)"
// @Success 200 {object} models.User "Created receptionist"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - lacks 'services' create permission"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/receptionist [post]
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

	type createReceptionistResult struct {
		Receptionist *models.User
		Err          error
		ErrType      string
	}

	resultChan := make(chan createReceptionistResult)

	go func() {
		var result createReceptionistResult

		// Create the receptionist user
		if err := db.DB.Create(receptionist).Error; err != nil {
			result.Err = err
			result.ErrType = "create_failed"
			resultChan <- result
			return
		}

		// Find the provider (assumed to be the authenticated user)
		var provider models.User
		if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
			result.Err = err
			result.ErrType = "provider_not_found"
			resultChan <- result
			return
		}

		// Create receptionist settings entry
		receptionistSettings := models.ReceptionistSettings{
			ReceptionistID: receptionist.ID,
			ProviderID:     provider.ID,
		}
		if err := db.DB.Create(&receptionistSettings).Error; err != nil {
			result.Err = err
			result.ErrType = "settings_failed"
			resultChan <- result
			return
		}

		result.Receptionist = receptionist
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "create_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create receptionist: " + result.Err.Error(),
			})
		case "provider_not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		case "settings_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create receptionist settings: " + result.Err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(result.Receptionist)
}

// GetReceptionistList retrieves the list of receptionists for the provider
// @Summary Get receptionist list
// @Description Retrieve a list of receptionists associated with the authenticated provider
// @Tags provider-receptionist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{receptionists=[]models.User} "List of receptionists"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/receptionist [get]
func GetReceptionistList(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	type receptionistListResult struct {
		Receptionists []models.User
		Err           error
		ErrType       string
	}

	resultChan := make(chan receptionistListResult)

	go func() {
		var result receptionistListResult

		// Find the provider
		var provider models.User
		if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
			result.Err = err
			result.ErrType = "provider_not_found"
			resultChan <- result
			return
		}

		var receptionistSettings []models.ReceptionistSettings
		if err := db.DB.Preload("Receptionist").Preload("Provider").Find(&receptionistSettings, "provider_id = ?", provider.ID).Error; err != nil {
			result.Err = err
			result.ErrType = "settings_fetch_failed"
			resultChan <- result
			return
		}

		// If no receptionists are found, return an empty list
		if len(receptionistSettings) == 0 {
			result.Receptionists = []models.User{}
			resultChan <- result
			return
		}

		//find receptionist by receptionist id
		var receptionistIDs []uint
		for _, setting := range receptionistSettings {
			receptionistIDs = append(receptionistIDs, setting.ReceptionistID)
		}

		var receptionists []models.User
		if err := db.DB.Where("id IN ?", receptionistIDs).Find(&receptionists).Error; err != nil {
			result.Err = err
			result.ErrType = "receptionists_fetch_failed"
			resultChan <- result
			return
		}

		result.Receptionists = receptionists
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "provider_not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		case "settings_fetch_failed", "receptionists_fetch_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch receptionists: " + result.Err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(result.Receptionists)
}

// GetReceptionistByID retrieves a specific receptionist by ID
// @Summary Get receptionist by ID
// @Description Retrieve details of a specific receptionist associated with the authenticated provider
// @Tags provider-receptionist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Receptionist ID"
// @Success 200 {object} models.User "Receptionist details"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid receptionist ID"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} fiber.Map{error=string} "Provider or receptionist not found"
// @Router /provider/receptionist/{id} [get]
func GetReceptionistByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse receptionist ID from URL parameter
	receptionistID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid receptionist ID",
		})
	}

	type receptionistResult struct {
		Receptionist models.User
		Err          error
		ErrType      string
	}

	resultChan := make(chan receptionistResult)

	go func() {
		var result receptionistResult

		// Find the provider
		var provider models.User
		if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
			result.Err = err
			result.ErrType = "provider_not_found"
			resultChan <- result
			return
		}

		var receptionist models.User
		if err := db.DB.Where("id = ? AND role_id = 4", receptionistID).First(&receptionist).Error; err != nil {
			result.Err = err
			result.ErrType = "receptionist_not_found"
			resultChan <- result
			return
		}

		result.Receptionist = receptionist
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "provider_not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		case "receptionist_not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Receptionist not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(result.Receptionist)
}

// DeleteReceptionist deletes a receptionist
// @Summary Delete receptionist
// @Description Delete a receptionist and their settings associated with the authenticated provider, requires 'services' delete permission
// @Tags provider-receptionist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Receptionist ID"
// @Success 200 {object} object{message=string} "Receptionist deleted successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid receptionist ID"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - lacks 'services' delete permission"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/receptionist/{id} [delete]
func DeleteReceptionist(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse receptionist ID from URL parameter
	receptionistID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid receptionist ID",
		})
	}

	type deleteResult struct {
		Err     error
		ErrType string
	}

	resultChan := make(chan deleteResult)

	go func() {
		var result deleteResult

		// Find the provider
		var provider models.User
		if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
			result.Err = err
			result.ErrType = "provider_not_found"
			resultChan <- result
			return
		}

		// Delete the receptionist settings
		if err := db.DB.Where("receptionist_id = ? AND provider_id = ?", receptionistID, provider.ID).Delete(&models.ReceptionistSettings{}).Error; err != nil {
			result.Err = err
			result.ErrType = "settings_delete_failed"
			resultChan <- result
			return
		}

		// Delete the receptionist user
		if err := db.DB.Delete(&models.User{}, receptionistID).Error; err != nil {
			result.Err = err
			result.ErrType = "receptionist_delete_failed"
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		switch result.ErrType {
		case "provider_not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Provider not found",
			})
		case "settings_delete_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete receptionist settings: " + result.Err.Error(),
			})
		case "receptionist_delete_failed":
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to delete receptionist: " + result.Err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Receptionist deleted successfully",
	})
}
