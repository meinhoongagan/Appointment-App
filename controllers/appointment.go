package controllers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
	"gorm.io/gorm"
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
// CreateAppointment godoc
func CreateAppointment(c *fiber.Ctx) error {
	var appointment models.Appointment

	// Parse request body
	if err := c.BodyParser(&appointment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Failed to parse request body",
			Error:   err.Error(),
		})
	}

	// Get the service to calculate duration
	var service models.Service
	if err := db.DB.First(&service, appointment.ServiceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Service not found",
			Error:   err.Error(),
		})
	}

	// Get duration directly from service
	duration := service.Duration

	// Convert StartTime to IST before checking availability
	appointment.StartTime = utils.ToIST(appointment.StartTime)

	// Check for availability
	available, err := utils.CheckAvailability(appointment.ProviderID, appointment.StartTime, duration)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Error checking availability",
			Error:   err.Error(),
		})
	}
	if !available {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Message: "Time slot not available",
		})
	}

	// Set end time and convert to IST
	appointment.EndTime = utils.ToIST(appointment.StartTime.Add(duration))

	// Create appointment within transaction
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		available, err := utils.CheckAvailability(appointment.ProviderID, appointment.StartTime, duration)
		if err != nil {
			return err
		}
		if !available {
			return fmt.Errorf("time slot not available")
		}

		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Message: "Time slot not available or failed to create appointment",
			Error:   err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(appointment)
}

// UpdateAppointment godoc
func UpdateAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var updatedAppointment models.Appointment

	// Parse incoming request
	if err := c.BodyParser(&updatedAppointment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Failed to parse request body",
			Error:   err.Error(),
		})
	}

	var existingAppointment models.Appointment
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the appointment row to prevent race conditions
		if err := tx.Raw(`
			SELECT * 
			FROM appointments
			WHERE id = ? FOR UPDATE
		`, id).Scan(&existingAppointment).Error; err != nil {
			return err
		}

		if existingAppointment.ID == 0 {
			return fmt.Errorf("appointment not found")
		}

		// Check if start_time or provider_id is being modified
		isTimeUpdated := updatedAppointment.StartTime != (time.Time{}) && updatedAppointment.StartTime != existingAppointment.StartTime
		isProviderUpdated := updatedAppointment.ProviderID != 0 && updatedAppointment.ProviderID != existingAppointment.ProviderID

		// If start_time or provider_id is updated, recheck availability
		if isTimeUpdated || isProviderUpdated {
			var service models.Service
			if err := tx.First(&service, updatedAppointment.ServiceID).Error; err != nil {
				return fmt.Errorf("service not found")
			}

			duration := service.Duration

			// Convert StartTime to IST
			updatedAppointment.StartTime = utils.ToIST(updatedAppointment.StartTime)

			// Check availability in IST
			available, err := utils.CheckAvailability(updatedAppointment.ProviderID, updatedAppointment.StartTime, duration)
			if err != nil {
				return err
			}
			if !available {
				return fmt.Errorf("time slot not available")
			}

			// Set updated end_time if start_time is modified
			if isTimeUpdated {
				updatedAppointment.EndTime = utils.ToIST(updatedAppointment.StartTime.Add(duration))
			}
		}

		// Preserve existing values if fields are not updated
		if updatedAppointment.Title == "" {
			updatedAppointment.Title = existingAppointment.Title
		}
		if updatedAppointment.Description == "" {
			updatedAppointment.Description = existingAppointment.Description
		}
		if updatedAppointment.Status == "" {
			updatedAppointment.Status = existingAppointment.Status
		}
		if updatedAppointment.ServiceID == 0 {
			updatedAppointment.ServiceID = existingAppointment.ServiceID
		}
		if updatedAppointment.ProviderID == 0 {
			updatedAppointment.ProviderID = existingAppointment.ProviderID
		}
		if updatedAppointment.CustomerID == 0 {
			updatedAppointment.CustomerID = existingAppointment.CustomerID
		}

		// Perform the update
		if err := tx.Model(&existingAppointment).Where("id = ?", id).Updates(updatedAppointment).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Message: "Failed to update appointment or time slot not available",
			Error:   err.Error(),
		})
	}

	return c.JSON(updatedAppointment)
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
