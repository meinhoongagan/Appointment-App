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
// @Summary Get all providers
// @Description Retrieve a paginated list of all service providers
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {object} fiber.Map{providers=[]models.User,total=int,page=int,limit=int,pages=int} "List of providers with pagination"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers [get]
func GetAllProviders(c *fiber.Ctx) error {
	providersChan := make(chan []models.User)
	countChan := make(chan int64)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset := (page - 1) * limit

	go func() {
		var providers []models.User
		if err := db.DB.Preload("Role").
			Joins("JOIN roles ON users.role_id = roles.id").
			Where("roles.name = ?", "provider").
			Limit(limit).
			Offset(offset).
			Find(&providers).Error; err != nil {
			providersChan <- []models.User{}
			return
		}
		providersChan <- providers
	}()

	go func() {
		var count int64
		db.DB.Model(&models.User{}).
			Joins("JOIN roles ON users.role_id = roles.id").
			Where("roles.name = ?", "provider").
			Count(&count)
		countChan <- count
	}()

	providers := <-providersChan
	count := <-countChan
	return c.JSON(fiber.Map{
		"providers": providers,
		"total":     count,
		"page":      page,
		"limit":     limit,
		"pages":     (int(count) + limit - 1) / limit,
	})
}

// GetProviderDetails returns details for a specific provider
// @Summary Get provider details
// @Description Retrieve details for a specific provider including working hours and business details
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Provider ID"
// @Success 200 {object} fiber.Map{provider=models.User,business_details=models.BusinessDetails} "Provider details"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/{id} [get]
func GetProviderDetails(c *fiber.Ctx) error {
	providerChan := make(chan models.User)
	businessDetailsChan := make(chan models.BusinessDetails)

	go func() {
		id := c.Params("id")
		var provider models.User
		if err := db.DB.Preload("Role").
			Preload("WorkingHours").
			First(&provider, id).Error; err != nil {
			providerChan <- models.User{}
			return
		}
		providerChan <- provider
	}()

	go func() {
		id := c.Params("id")
		var businessDetails models.BusinessDetails
		db.DB.Where("provider_id = ?", id).First(&businessDetails)
		businessDetailsChan <- businessDetails
	}()

	provider := <-providerChan
	businessDetails := <-businessDetailsChan
	if provider.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}
	if provider.Role.Name != "provider" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User is not a service provider",
		})
	}
	provider.Password = ""
	return c.JSON(fiber.Map{
		"provider":         provider,
		"business_details": businessDetails,
	})
}

// GetProviderServices returns services offered by a specific provider
// @Summary Get provider services
// @Description Retrieve services offered by a specific provider
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Provider ID"
// @Success 200 {array} models.Service "List of services"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/{id}/services [get]
func GetProviderServices(c *fiber.Ctx) error {
	providerChan := make(chan models.User)
	servicesChan := make(chan []models.Service)

	go func() {
		id := c.Params("id")
		var provider models.User
		if err := db.DB.First(&provider, id).Error; err != nil {
			providerChan <- models.User{}
			return
		}
		providerChan <- provider
	}()

	go func() {
		id := c.Params("id")
		var services []models.Service
		if err := db.DB.Where("provider_id = ?", id).Find(&services).Error; err != nil {
			servicesChan <- []models.Service{}
			return
		}
		servicesChan <- services
	}()

	provider := <-providerChan
	services := <-servicesChan
	if provider.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}
	return c.JSON(services)
}

// SearchProviders searches for providers by name, business name, or service
// @Summary Search providers
// @Description Search for providers by name, business name, or service
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Success 200 {object} fiber.Map{providers=[]models.User,count=int} "List of providers"
// @Failure 400 {object} fiber.Map{error=string} "Search query is required"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/search/service [get]
func SearchProviders(c *fiber.Ctx) error {
	providersChan := make(chan []models.User)

	go func() {
		query := c.Query("q")
		if query == "" {
			providersChan <- []models.User{}
			return
		}
		var providers []models.User
		searchQuery := fmt.Sprintf("%%%s%%", query)
		if err := db.DB.Preload("Role").
			Joins("JOIN roles ON users.role_id = roles.id").
			Joins("LEFT JOIN business_details ON users.id = business_details.provider_id").
			Where("roles.name = ? AND (users.name LIKE ? OR business_details.business_name LIKE ?)",
				"provider", searchQuery, searchQuery).
			Group("users.id").
			Find(&providers).Error; err != nil {
			providersChan <- []models.User{}
			return
		}
		providersChan <- providers
	}()

	providers := <-providersChan
	if len(providers) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}
	for i := range providers {
		providers[i].Password = ""
	}
	return c.JSON(fiber.Map{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetProvidersByCategory returns providers by category
// @Summary Get providers by category
// @Description Retrieve providers that offer services in a specific category
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param categoryId path int true "Category ID"
// @Success 200 {object} fiber.Map{providers=[]models.User,count=int} "List of providers"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/category/{categoryId} [get]
func GetProvidersByCategory(c *fiber.Ctx) error {
	providersChan := make(chan []models.User)

	go func() {
		categoryId := c.Params("categoryId")
		var providers []models.User
		if err := db.DB.Preload("Role").
			Joins("JOIN roles ON users.role_id = roles.id").
			Joins("JOIN services ON users.id = services.provider_id").
			Joins("JOIN service_categories ON services.category_id = service_categories.id").
			Where("roles.name = ? AND service_categories.id = ?", "provider", categoryId).
			Group("users.id").
			Find(&providers).Error; err != nil {
			providersChan <- []models.User{}
			return
		}
		providersChan <- providers
	}()

	providers := <-providersChan
	for i := range providers {
		providers[i].Password = ""
	}
	return c.JSON(fiber.Map{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetFeaturedProviders returns featured providers
// @Summary Get featured providers
// @Description Retrieve a list of featured or top-rated providers
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} fiber.Map{providers=[]models.User} "List of providers"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/featured [get]
func GetFeaturedProviders(c *fiber.Ctx) error {
	providersChan := make(chan []models.User)

	go func() {
		var providers []models.User
		if err := db.DB.Preload("Role").
			Joins("JOIN roles ON users.role_id = roles.id").
			Where("roles.name = ?", "provider").
			Limit(10).
			Find(&providers).Error; err != nil {
			providersChan <- []models.User{}
			return
		}
		providersChan <- providers
	}()

	providers := <-providersChan
	for i := range providers {
		providers[i].Password = ""
	}
	return c.JSON(fiber.Map{
		"providers": providers,
	})
}

// GetNearbyProviders returns providers near the user's location
// @Summary Get nearby providers
// @Description Retrieve providers near the user's location based on latitude, longitude, and radius
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lat query string true "Latitude"
// @Param lng query string true "Longitude"
// @Param radius query string false "Radius in kilometers" default(10)
// @Success 200 {object} fiber.Map{providers=[]models.User,radius=string,lat=string,lng=string} "List of nearby providers"
// @Failure 400 {object} fiber.Map{error=string} "Location parameters (lat, lng) are required"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/nearby [get]
func GetNearbyProviders(c *fiber.Ctx) error {
	providersChan := make(chan []models.User)

	go func() {
		latitude := c.Query("lat")
		longitude := c.Query("lng")
		if latitude == "" || longitude == "" {
			providersChan <- []models.User{}
			return
		}
		var providers []models.User
		if err := db.DB.Preload("Role").
			Joins("JOIN roles ON users.role_id = roles.id").
			Joins("JOIN business_details ON users.id = business_details.provider_id").
			Where("roles.name = ?", "provider").
			Limit(20).
			Find(&providers).Error; err != nil {
			providersChan <- []models.User{}
			return
		}
		providersChan <- providers
	}()

	providers := <-providersChan
	for i := range providers {
		providers[i].Password = ""
	}
	return c.JSON(fiber.Map{
		"providers": providers,
		"radius":    c.Query("radius", "10"),
		"lat":       c.Query("lat"),
		"lng":       c.Query("lng"),
	})
}

// GetAvailableSlots returns available appointment slots
// @Summary Get available appointment slots
// @Description Retrieve available appointment slots for a provider on a given date
// @Tags providers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param provider_id path int true "Provider ID"
// @Param date query string true "Date in YYYY-MM-DD format"
// @Param service_id query int true "Service ID"
// @Success 200 {object} fiber.Map{slots=[]string,provider_id=string,date=string,service_id=string} "Available slots"
// @Failure 400 {object} fiber.Map{error=string} "Invalid date format or missing service ID"
// @Failure 404 {object} fiber.Map{error=string} "Service not found or does not belong to provider"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /providers/available-time-slots/{provider_id} [get]
func GetAvailableSlots(c *fiber.Ctx) error {
	slotsChan := make(chan []string)

	go func() {
		ist, err := time.LoadLocation("Asia/Kolkata")
		if err != nil {
			slotsChan <- []string{}
			return
		}
		providerID := c.Params("provider_id")
		dateStr := c.Query("date")
		serviceID := c.Query("service_id")
		date, err := time.ParseInLocation("2006-01-02", dateStr, ist)
		if err != nil {
			slotsChan <- []string{}
			return
		}
		if serviceID == "" {
			slotsChan <- []string{}
			return
		}
		var service models.Service
		if dbErr := db.DB.Where("id = ? AND provider_id = ?", serviceID, providerID).First(&service).Error; dbErr != nil {
			slotsChan <- []string{}
			return
		}
		dayOfWeek := models.DayOfWeek(date.Weekday())
		var workingHours models.WorkingHours
		if dbErr := db.DB.Where("provider_id = ? AND day_of_week = ?", providerID, dayOfWeek).First(&workingHours).Error; dbErr != nil {
			slotsChan <- []string{}
			return
		}
		startTime, err := time.ParseInLocation("15:04", workingHours.StartTime, ist)
		if err != nil {
			slotsChan <- []string{}
			return
		}
		endTime, err := time.ParseInLocation("15:04", workingHours.EndTime, ist)
		if err != nil {
			slotsChan <- []string{}
			return
		}
		startDateTime := time.Date(date.Year(), date.Month(), date.Day(), startTime.Hour(), startTime.Minute(), 0, 0, ist)
		endDateTime := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), 0, 0, ist)
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
		slotDuration := service.Duration + service.BufferTime
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, ist)
		endOfDay := startOfDay.Add(24 * time.Hour)
		var appointments []models.Appointment
		if err := db.DB.Where("provider_id = ? AND start_time >= ? AND start_time < ? AND status != ?",
			providerID, startOfDay, endOfDay, models.StatusCanceled).Find(&appointments).Error; err != nil {
			slotsChan <- []string{}
			return
		}
		var availableSlots []string
		currentSlot := startDateTime
		for currentSlot.Add(slotDuration).Before(endDateTime) || currentSlot.Add(slotDuration).Equal(endDateTime) {
			if breakStart != nil && breakEnd != nil {
				if (currentSlot.Equal(*breakStart) || currentSlot.After(*breakStart)) && currentSlot.Before(*breakEnd) {
					currentSlot = currentSlot.Add(slotDuration)
					continue
				}
			}
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
		slotsChan <- availableSlots
	}()

	availableSlots := <-slotsChan
	return c.JSON(fiber.Map{
		"slots":       availableSlots,
		"provider_id": c.Params("provider_id"),
		"date":        c.Query("date"),
		"service_id":  c.Query("service_id"),
	})
}
