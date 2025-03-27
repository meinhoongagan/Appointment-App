package controllers

import (
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

// CreateService creates a new service
func CreateService(c *fiber.Ctx) error {
	service := new(models.Service)
	if err := c.BodyParser(service); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	new_service :=models.Service{
		Name: service.Name,
		Description: service.Description,
		Duration: service.Duration,
		Cost: service.Cost,
		BufferTime: service.BufferTime,
		ProviderID: c.Locals("userID").(uint),
		Provider: models.User{ID: c.Locals("userID").(uint)},
	}
	db.DB.Create(&new_service)
	return c.JSON(new_service)
}

// UpdateService updates a service
func UpdateService(c *fiber.Ctx) error {
	id := c.Params("id")
	service := new(models.Service)
	if err := c.BodyParser(service); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	var existingService models.Service
	if db.DB.First(&existingService, id).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}
	service.ID = existingService.ID
	db.DB.Save(&service)
	return c.JSON(service)
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