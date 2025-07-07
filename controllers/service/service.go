package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetAllServices returns all services
// @Summary Get all services
// @Description Retrieve a list of all services with their provider details
// @Tags services
// @Accept json
// @Produce json
// @Success 200 {array} models.Service "List of services"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services [get]
func GetAllServices(c *fiber.Ctx) error {
	var services []models.Service

	// Preload Provider and its Role properly
	if err := db.DB.Debug().
		Preload("Provider.Role").
		Find(&services).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Log services for debugging
	fmt.Printf("Fetched all services: %+v\n", services)

	return c.JSON(services)
}

// GetService retrieves a specific service by ID
// @Summary Get service by ID
// @Description Retrieve a specific service by its ID with provider details
// @Tags services
// @Accept json
// @Produce json
// @Param id path int true "Service ID"
// @Success 200 {object} models.Service "Service details"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid service ID"
// @Failure 404 {object} fiber.Map{error=string} "Service not found"
// @Router /provider/services/{id} [get]
func GetService(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service ID",
		})
	}
	var service models.Service
	if err := db.DB.Preload("Provider.Role").First(&service, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}
	return c.JSON(service)
}

// GetMyServices returns all services of the authenticated provider
// @Summary Get provider's services
// @Description Retrieve all services created by the authenticated provider or their receptionist
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Service "List of provider's services"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not a provider or receptionist"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services/get-provider/service [get]
func GetMyServices(c *fiber.Ctx) error {
	userIDVal := c.Locals("userID")
	fmt.Println("User ID from locals:", userIDVal)
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	if c.Locals("role") == "receptionist" {
		var receptionist models.ReceptionistSettings
		if err := db.DB.Where("receptionist_id = ?", userID).First(&receptionist).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only receptionists can access this endpoint",
			})
		}
		fmt.Println("Receptionist settings found:", receptionist)
		userID = receptionist.ProviderID
		fmt.Println("User ID from receptionist settings:", userID)
	}

	var services []models.Service
	if err := db.DB.Preload("Provider.Role").
		Where("provider_id = ?", userID).
		Find(&services).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch services",
		})
	}

	// Log services for debugging
	fmt.Printf("Fetched services for user %d: %+v\n", userID, services)

	return c.JSON(services)
}

// SearchServiceNames returns a list of service names matching the search query
// @Summary Search service names
// @Description Retrieve a list of service names that match the search query
// @Tags services
// @Accept json
// @Produce json
// @Param search query string true "Search query for service names"
// @Success 200 {array} string "List of matching service names"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - missing search query"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services/names/search/names [get]
func SearchServiceNames(c *fiber.Ctx) error {
	search := c.Query("search")
	if search == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query parameter is required",
		})
	}
	search = strings.ToLower(search)
	var serviceNames []string
	if err := db.DB.Debug().
		Model(&models.Service{}). // Specify the Service model
		Where("LOWER(name) LIKE ?", "%"+search+"%").
		Pluck("name", &serviceNames).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(serviceNames)
}

// SearchServiceByName searches services by name
// @Summary Search services by name
// @Description Retrieve a list of services whose names match the search query
// @Tags services
// @Accept json
// @Produce json
// @Param name query string true "Service name to search for"
// @Success 200 {array} models.Service "List of matching services"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - missing name query"
// @Failure 404 {object} fiber.Map{error=string} "No services found with the given name"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services/search/name [get]
func SearchServiceByName(c *fiber.Ctx) error {
	// We just need to return List Of Services Names and Should Compare with the lowerCase name
	// and return the list of services that match the name
	name := c.Query("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name query parameter is required",
		})
	}
	name = strings.ToLower(name)
	var services []models.Service
	if err := db.DB.Debug().
		Where("LOWER(name) LIKE ?", "%"+name+"%").
		Find(&services).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if len(services) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No services found with the given name",
		})
	} else {
		// Log services for debugging
		fmt.Printf("Fetched services matching name '%s': %+v\n", name, services)
		return c.JSON(services)
	}
}

// CreateService creates a new service
// @Summary Create a new service
// @Description Create a new service for the authenticated provider, requires 'services' create permission
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param service body models.Service true "Service details"
// @Success 200 {object} models.Service "Created service"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not a provider or lacks 'services' create permission"
// @Failure 404 {object} fiber.Map{error=string} "Provider not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services [post]
func CreateService(c *fiber.Ctx) error {
	// Extract userID from JWT
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID in token",
		})
	}

	// Verify role
	role, ok := c.Locals("role").(string)
	if !ok || role != "provider" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only providers can create services",
		})
	}

	// Parse request body into Service struct
	service := new(models.Service)
	if err := c.BodyParser(service); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Find the provider
	var provider models.User
	if err := db.DB.Where("id = ?", userID).First(&provider).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Set ProviderID and Provider from JWT userID
	service.ProviderID = userID
	service.Provider = provider

	// Create the service
	if err := db.DB.Create(service).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create service: " + err.Error(),
		})
	}

	return c.JSON(service)
}

// UpdateService updates a service
// @Summary Update a service
// @Description Update an existing service by its ID, requires 'services' update permission
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Service ID"
// @Param service body object true "Service update details (partial)"
// @Success 200 {object} models.Service "Updated service"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input or service ID"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user lacks permission or does not own the service"
// @Failure 404 {object} fiber.Map{error=string} "Service not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services/{id} [patch]
func UpdateService(c *fiber.Ctx) error {
	// Parse service ID from URL parameter
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service ID",
		})
	}

	// Extract userID from JWT
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID in token",
		})
	}

	// Find existing service
	var existingService models.Service
	if err := db.DB.First(&existingService, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Verify the service belongs to the authenticated provider
	if existingService.ProviderID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to update this service",
		})
	}

	// Create a map to store update data
	updateData := make(map[string]interface{})
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
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
				// Assume float64 represents minutes, convert to time.Duration
				if val <= 0 {
					return nil
				}
				return time.Duration(val * float64(time.Minute))
			case string:
				duration, err := time.ParseDuration(val)
				if err != nil {
					return nil
				}
				return duration
			default:
				return nil
			}
		},
		"buffer_time": func(v interface{}) interface{} {
			switch val := v.(type) {
			case float64:
				// Assume float64 represents minutes, convert to time.Duration
				if val <= 0 {
					return nil
				}
				return time.Duration(val * float64(time.Minute))
			case string:
				duration, err := time.ParseDuration(val)
				if err != nil {
					return nil
				}
				return duration
			default:
				return nil
			}
		},
		"cost": func(v interface{}) interface{} {
			switch val := v.(type) {
			case float64:
				return val
			case string:
				cost, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return nil
				}
				return cost
			default:
				return nil
			}
		},
	}

	// Prepare update map
	updateMap := make(map[string]interface{})
	for key, value := range updateData {
		if converter, exists := specialFields[key]; exists {
			if converted := converter(value); converted != nil {
				updateMap[key] = converted
			} else {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("Invalid value for %s", key),
				})
			}
		} else {
			updateMap[key] = value
		}
	}

	// Perform update only on provided fields
	if len(updateMap) > 0 {
		if err := db.DB.Model(&existingService).Updates(updateMap).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update service: " + err.Error(),
			})
		}
	}

	// Retrieve the updated service with preloaded Provider
	if err := db.DB.Preload("Provider.Role").First(&existingService, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve updated service: " + err.Error(),
		})
	}

	return c.JSON(existingService)
}

// DeleteService deletes a service
// @Summary Delete a service
// @Description Delete a service by its ID, requires 'services' delete permission
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Service ID"
// @Success 204 "No content - service successfully deleted"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid service ID"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user lacks permission or does not own the service"
// @Failure 404 {object} fiber.Map{error=string} "Service not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/services/{id} [delete]
func DeleteService(c *fiber.Ctx) error {
	// Parse service ID from URL parameter
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid service ID",
		})
	}

	// Extract userID from JWT
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID in token",
		})
	}

	// Find the service
	var service models.Service
	if err := db.DB.First(&service, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Verify the service belongs to the authenticated provider
	if service.ProviderID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to delete this service",
		})
	}

	// Delete the service
	if err := db.DB.Delete(&service).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete service: " + err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
