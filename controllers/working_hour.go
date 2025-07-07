package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetAllWorkingHours retrieves all working hours from the database
// @Summary Get all working hours
// @Description Retrieve a list of all working hours in the system
// @Tags working-hours
// @Accept json
// @Produce json
// @Success 200 {array} models.WorkingHours "List of working hours"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /working-hours [get]
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
// @Summary Get working hour by ID
// @Description Retrieve a specific working hour by its ID
// @Tags working-hours
// @Accept json
// @Produce json
// @Param id path int true "Working Hour ID"
// @Success 200 {object} models.WorkingHours "Working hour details"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /working-hours/{id} [get]
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
// @Summary Create a new working hour
// @Description Create a new working hour entry, requires 'working-hours' create permission
// @Tags working-hours
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param working_hour body models.WorkingHours true "Working hour details"
// @Success 200 {object} models.WorkingHours "Created working hour"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - missing 'working-hours' create permission"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /working-hours [post]
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
// @Summary Update a working hour
// @Description Update an existing working hour by its ID, requires 'working-hours' update permission
// @Tags working-hours
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Working Hour ID"
// @Param working_hour body models.WorkingHours true "Updated working hour details"
// @Success 200 {object} models.WorkingHours "Updated working hour"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - missing 'working-hours' update permission"
// @Failure 404 {object} fiber.Map{error=string} "Working hour not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /working-hours/{id} [patch]
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
// @Summary Delete a working hour
// @Description Delete a working hour by its ID, requires 'working-hours' delete permission
// @Tags working-hours
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Working Hour ID"
// @Success 204 "No content - working hour successfully deleted"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - missing 'working-hours' delete permission"
// @Failure 404 {object} fiber.Map{error=string} "Working hour not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /working-hours/{id} [delete]
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