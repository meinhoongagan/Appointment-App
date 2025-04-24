package consumer

import (
	"fmt"
	"strconv"

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
		"total": count,
		"page": page,
		"limit": limit,
		"pages": (int(count) + limit - 1) / limit,
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
		"provider": provider,
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