package service

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetAllServices returns all services
func GetAllServices(c *fiber.Ctx) error {
	var services []models.Service

	// Preload Provider and its Role properly
	if err := db.DB.Debug().
		Preload("Provider.Role"). // Preload Role nested inside Provider
		Find(&services).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(services)
}

func GetService(c *fiber.Ctx) error {
	id := c.Params("id")
	var service models.Service
	db.DB.Find(&service, id)
	return c.JSON(service)
}

func CreateService(c *fiber.Ctx) error {
	service := new(models.Service)
	if err := c.BodyParser(service); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Find the provider first
	var provider models.User
	if err := db.DB.Where("id = ?", service.ProviderID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Set the provider
	service.Provider = provider

	// Create the service
	if err := db.DB.Create(service).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create service",
		})
	}

	return c.JSON(service)
}

// UpdateService updates a service
func UpdateService(c *fiber.Ctx) error {
	// Parse service ID from URL parameter
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service ID",
		})
	}

	// Find existing service
	var existingService models.Service
	if err := db.DB.First(&existingService, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Create a map to store update data
	updateData := make(map[string]interface{})
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Remove restricted fields
	fieldsToIgnore := []string{"id", "ID", "provider", "Provider", "ProviderID", "provider_id"}
	for _, field := range fieldsToIgnore {
		delete(updateData, field)
	}

	// Special handling for specific fields that might need type conversion
	specialFields := map[string]func(interface{}) interface{}{
		"duration": func(v interface{}) interface{} {
			switch val := v.(type) {
			case float64:
				return time.Duration(val)
			case string:
				duration, _ := time.ParseDuration(val)
				return duration
			default:
				return v
			}
		},
		"buffer_time": func(v interface{}) interface{} {
			switch val := v.(type) {
			case float64:
				return time.Duration(val)
			case string:
				duration, _ := time.ParseDuration(val)
				return duration
			default:
				return v
			}
		},
		"cost": func(v interface{}) interface{} {
			switch val := v.(type) {
			case float64:
				return val
			case string:
				cost, _ := strconv.ParseFloat(val, 64)
				return cost
			default:
				return v
			}
		},
	}

	// Prepare update map
	updateMap := make(map[string]interface{})
	for key, value := range updateData {
		// Apply special field handling if exists
		if converter, exists := specialFields[key]; exists {
			updateMap[key] = converter(value)
		} else {
			updateMap[key] = value
		}
	}

	// Perform update only on provided fields
	if len(updateMap) > 0 {
		if err := db.DB.Model(&existingService).Updates(updateMap).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update service",
			})
		}
	}

	// Retrieve the updated service
	if err := db.DB.First(&existingService, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated service",
		})
	}

	return c.JSON(existingService)
}

// DeleteService deletes a service
func DeleteService(c *fiber.Ctx) error {
	id := c.Params("id")
	var service models.Service
	if db.DB.First(&service, id).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}
	db.DB.Delete(&service)
	return c.SendStatus(fiber.StatusNoContent)
}
