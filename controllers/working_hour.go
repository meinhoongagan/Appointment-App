package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetAllWorkingHours retrieves all working hours from the database

func GetAllWorkingHours(c *fiber.Ctx) error {
	var workingHours []models.WorkingHours
	if err := db.DB.Find(&workingHours).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get working hours",
		})
	}
	return c.JSON(workingHours)
}

// GetWorkingHour retrieves a specific working hour by ID
func GetWorkingHour(c *fiber.Ctx) error {
	id := c.Params("id")
	var workingHour models.WorkingHours
	if err := db.DB.First(&workingHour, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get working hour",
		})
	}
	return c.JSON(workingHour)
}

// CreateWorkingHour creates a new working hour
func CreateWorkingHour(c *fiber.Ctx) error {
	workingHour := new(models.WorkingHours)
	if err := c.BodyParser(workingHour); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}
	if err := db.DB.Create(workingHour).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create working hour",
		})
	}
	return c.JSON(workingHour)
}

// UpdateWorkingHour updates an existing working hour
func UpdateWorkingHour(c *fiber.Ctx) error {
	id := c.Params("id")
	var workingHour models.WorkingHours
	if err := db.DB.First(&workingHour, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Working hour not found",
		})
	}
	if err := c.BodyParser(&workingHour); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}
	if err := db.DB.Save(&workingHour).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update working hour",
		})
	}
	return c.JSON(workingHour)
}

// DeleteWorkingHour deletes a working hour by ID
func DeleteWorkingHour(c *fiber.Ctx) error {
	id := c.Params("id")
	var workingHour models.WorkingHours
	if err := db.DB.First(&workingHour, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Working hour not found",
		})
	}
	if err := db.DB.Delete(&workingHour).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete working hour",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
