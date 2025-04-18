package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
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

// GetProviderSettings retrieves the provider's settings
func GetProviderSettings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var settings models.ProviderSettings
	if err := db.DB.Where("provider_id = ?", userID).First(&settings).Error; err != nil {
		// If settings not found, return default settings
		return c.JSON(fiber.Map{
			"settings": models.ProviderSettings{
				ProviderID:              userID,
				NotificationsEnabled:    true,
				AutoConfirmBookings:     false,
				AdvanceBookingDays:      30,
				CancellationPeriodHours: 24,
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

// GetTimeOff retrieves the provider's time off periods
func GetTimeOff(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var timeOffPeriods []models.TimeOff
	if err := db.DB.Where("provider_id = ?", userID).Find(&timeOffPeriods).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve time off periods",
		})
	}

	return c.JSON(fiber.Map{
		"time_off": timeOffPeriods,
	})
}

// AddTimeOff adds a new time off period for the provider
func AddTimeOff(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	// Parse time off data
	timeOff := new(models.TimeOff)
	if err := c.BodyParser(timeOff); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate time off period
	if timeOff.StartTime.After(timeOff.EndTime) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Start time must be before end time",
		})
	}

	// Set provider ID
	timeOff.ProviderID = userID

	// Create new time off period
	if err := db.DB.Create(timeOff).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create time off period",
		})
	}

	return c.JSON(fiber.Map{
		"message":  "Time off period added successfully",
		"time_off": timeOff,
	})
}

// DeleteTimeOff deletes a time off period
func DeleteTimeOff(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	timeOffID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid time off ID",
		})
	}

	// Check if time off period exists and belongs to the provider
	var timeOff models.TimeOff
	if db.DB.Where("id = ? AND provider_id = ?", timeOffID, userID).First(&timeOff).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Time off period not found",
		})
	}

	// Delete time off period
	if err := db.DB.Delete(&timeOff).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete time off period",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
