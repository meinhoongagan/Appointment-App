package consumer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetAllProviders returns all service providers
func GetAllProviders(c *fiber.Ctx) error {
	var providers []models.User

	// Get pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Calculate offset
	offset := (page - 1) * limit

	// Only return users with the provider role
	if err := db.DB.Preload("Role").
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "provider").
		Limit(limit).
		Offset(offset).
		Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch providers",
		})
	}

	// Count total records for pagination
	var count int64
	db.DB.Model(&models.User{}).
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "provider").
		Count(&count)

	return c.JSON(fiber.Map{
		"providers": providers,
		"total":     count,
		"page":      page,
		"limit":     limit,
		"pages":     (int(count) + limit - 1) / limit,
	})
}

// GetProviderDetails returns details for a specific provider
func GetProviderDetails(c *fiber.Ctx) error {
	id := c.Params("id")

	var provider models.User
	if err := db.DB.Preload("Role").
		Preload("WorkingHours").
		First(&provider, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Check if the user is a provider
	if provider.Role.Name != "provider" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User is not a service provider",
		})
	}

	// Get business details separately
	var businessDetails models.BusinessDetails
	db.DB.Where("provider_id = ?", id).First(&businessDetails)

	// Remove sensitive information
	provider.Password = ""

	return c.JSON(fiber.Map{
		"provider":         provider,
		"business_details": businessDetails,
	})
}

// GetProviderServices returns services offered by a specific provider
func GetProviderServices(c *fiber.Ctx) error {
	id := c.Params("id")

	// Check if the provider exists
	var provider models.User
	if err := db.DB.First(&provider, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Get provider's services
	var services []models.Service
	if err := db.DB.Where("provider_id = ?", id).Find(&services).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch provider services",
		})
	}

	return c.JSON(services)
}

// SearchProviders searches for providers by name, business name, or service
func SearchProviders(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	var providers []models.User

	// Search in providers and their business details
	searchQuery := fmt.Sprintf("%%%s%%", query)

	if err := db.DB.Preload("Role").
		Joins("JOIN roles ON users.role_id = roles.id").
		Joins("LEFT JOIN business_details ON users.id = business_details.provider_id").
		Where("roles.name = ? AND (users.name LIKE ? OR business_details.business_name LIKE ?)",
			"provider", searchQuery, searchQuery).
		Group("users.id").
		Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search providers",
		})
	}

	// Clean sensitive data
	for i := range providers {
		providers[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetProvidersByCategory returns providers that offer services in a specific category
func GetProvidersByCategory(c *fiber.Ctx) error {
	categoryId := c.Params("categoryId")

	var providers []models.User

	// Get providers that offer services in the specified category
	if err := db.DB.Preload("Role").
		Joins("JOIN roles ON users.role_id = roles.id").
		Joins("JOIN services ON users.id = services.provider_id").
		Joins("JOIN service_categories ON services.category_id = service_categories.id").
		Where("roles.name = ? AND service_categories.id = ?", "provider", categoryId).
		Group("users.id").
		Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch providers by category",
		})
	}

	// Clean sensitive data
	for i := range providers {
		providers[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetFeaturedProviders returns featured or top-rated providers
func GetFeaturedProviders(c *fiber.Ctx) error {
	var providers []models.User

	// Get top-rated providers (this would typically include rating logic)
	if err := db.DB.Preload("Role").
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("roles.name = ?", "provider").
		Limit(10).
		Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch featured providers",
		})
	}

	// Clean sensitive data
	for i := range providers {
		providers[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// GetNearbyProviders returns providers near the user's location
func GetNearbyProviders(c *fiber.Ctx) error {
	// Get location from query parameters
	latitude := c.Query("lat")
	longitude := c.Query("lng")
	radius := c.Query("radius", "10") // Default radius: 10km

	if latitude == "" || longitude == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Location parameters (lat, lng) are required",
		})
	}

	// In a real implementation, you would use a geospatial query
	// For now, we'll just return providers as if they were nearby
	var providers []models.User

	if err := db.DB.Preload("Role").
		Joins("JOIN roles ON users.role_id = roles.id").
		Joins("JOIN business_details ON users.id = business_details.provider_id").
		Where("roles.name = ?", "provider").
		Limit(20).
		Find(&providers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch nearby providers",
		})
	}

	// Clean sensitive data
	for i := range providers {
		providers[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"providers": providers,
		"radius":    radius,
		"lat":       latitude,
		"lng":       longitude,
	})
}

// GetAvailableSlots returns available appointment slots for a provider on a given date
func GetAvailableSlots(c *fiber.Ctx) error {
	// Define IST location
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load IST timezone",
		})
	}

	// Get query parameters
	providerID := c.Params("provider_id")
	dateStr := c.Query("date")         // Expected format: "YYYY-MM-DD"
	serviceID := c.Query("service_id") // Required

	// Parse the date in IST
	date, err := time.ParseInLocation("2006-01-02", dateStr, ist)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date format, use YYYY-MM-DD",
		})
	}

	// Validate service exists and belongs to provider
	if serviceID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Service ID is required",
		})
	}
	var service models.Service
	if err := db.DB.Where("id = ? AND provider_id = ?", serviceID, providerID).First(&service).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found or does not belong to provider",
		})
	}

	// Get working hours for the day of the week
	dayOfWeek := models.DayOfWeek(date.Weekday())
	var workingHours models.WorkingHours
	if err := db.DB.Where("provider_id = ? AND day_of_week = ?", providerID, dayOfWeek).First(&workingHours).Error; err != nil {
		return c.JSON(fiber.Map{
			"slots":   []string{},
			"message": fmt.Sprintf("No working hours defined for %s", date.Weekday()),
		})
	}

	// Parse working hours in IST
	startTime, err := time.ParseInLocation("15:04", workingHours.StartTime, ist)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid start time format",
		})
	}
	endTime, err := time.ParseInLocation("15:04", workingHours.EndTime, ist)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid end time format",
		})
	}

	// Combine date with working hours
	startDateTime := time.Date(date.Year(), date.Month(), date.Day(), startTime.Hour(), startTime.Minute(), 0, 0, ist)
	endDateTime := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), 0, 0, ist)

	// Handle break times if defined
	var breakStart, breakEnd *time.Time
	if workingHours.BreakStart != nil && workingHours.BreakEnd != nil {
		bs, err := time.ParseInLocation("15:04", *workingHours.BreakStart, ist)
		if err == nil {
			bsTime := time.Date(date.Year(), date.Month(), date.Day(), bs.Hour(), bs.Minute(), 0, 0, ist)
			breakStart = &bsTime
		}
		be, err := time.ParseInLocation("15:04", *workingHours.BreakEnd, ist)
		if err == nil {
			beTime := time.Date(date.Year(), date.Month(), date.Day(), be.Hour(), be.Minute(), 0, 0, ist)
			breakEnd = &beTime
		}
	}

	// Get service duration and buffer time
	slotDuration := service.Duration + service.BufferTime

	// Get existing appointments for the date in IST
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, ist)
	endOfDay := startOfDay.Add(24 * time.Hour)
	var appointments []models.Appointment
	if err := db.DB.Where("provider_id = ? AND start_time >= ? AND start_time < ? AND status != ?",
		providerID, startOfDay, endOfDay, models.StatusCanceled).Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch appointments",
		})
	}

	// Calculate available slots
	var availableSlots []string
	currentSlot := startDateTime
	for currentSlot.Add(slotDuration).Before(endDateTime) || currentSlot.Add(slotDuration).Equal(endDateTime) {
		// Skip if slot is during break
		if breakStart != nil && breakEnd != nil {
			if (currentSlot.Equal(*breakStart) || currentSlot.After(*breakStart)) && currentSlot.Before(*breakEnd) {
				currentSlot = currentSlot.Add(slotDuration)
				continue
			}
		}

		// Check if slot is available (no overlap with appointments)
		isAvailable := true
		slotEnd := currentSlot.Add(slotDuration)
		for _, appt := range appointments {
			if (currentSlot.Before(appt.EndTime) && slotEnd.After(appt.StartTime)) ||
				currentSlot.Equal(appt.StartTime) {
				isAvailable = false
				break
			}
		}

		if isAvailable {
			availableSlots = append(availableSlots, currentSlot.Format("15:04"))
		}
		currentSlot = currentSlot.Add(slotDuration)
	}

	return c.JSON(fiber.Map{
		"slots":       availableSlots,
		"provider_id": providerID,
		"date":        dateStr,
		"service_id":  serviceID,
	})
}
