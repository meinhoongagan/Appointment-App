package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
)

// GetAllAppointments godoc
func GetAllAppointments(c *fiber.Ctx) error {
	var appointments []models.Appointment
	if err := db.DB.Preload("Service").Preload("Provider").Preload("Customer").Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to fetch appointments",
			Error:   err.Error(),
		})
	}
	return c.JSON(appointments)
}

// GetAppointment godoc
func GetAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var appointment models.Appointment
	if err := db.DB.Preload("Service").Preload("Provider").Preload("Customer").First(&appointment, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Appointment not found",
			Error:   err.Error(),
		})
	}
	return c.JSON(appointment)
}

// CreateAppointment godoc
func CreateAppointment(c *fiber.Ctx) error {
	var appointment models.Appointment
	if err := c.BodyParser(&appointment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Failed to parse request body",
			Error:   err.Error(),
		})
	}
	if err := db.DB.Create(&appointment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to create appointment",
			Error:   err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(appointment)
}

// UpdateAppointment godoc
func UpdateAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var appointment models.Appointment
	if err := c.BodyParser(&appointment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Failed to parse request body",
			Error:   err.Error(),
		})
	}
	if err := db.DB.Model(&appointment).Where("id = ?", id).Updates(appointment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to update appointment",
			Error:   err.Error(),
		})
	}
	return c.JSON(appointment)
}

// DeleteAppointment godoc
func DeleteAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := db.DB.Where("id = ?", id).Delete(&models.Appointment{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to delete appointment",
			Error:   err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
