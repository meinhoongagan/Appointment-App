package consumer

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

// GetServiceDetails returns details for a specific service
func GetServiceDetails(c *fiber.Ctx) error {
	id := c.Params("id")

	var service models.Service
	if err := db.DB.First(&service, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	// Optionally preload provider or category
	if err := db.DB.Preload("Provider").Preload("Category").First(&service, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}

	return c.JSON(service)
}

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
	fmt.Println("Parsed appointment:", appointment)
	// Get the service to calculate duration
	var service models.Service
	if err := db.DB.First(&service, appointment.ServiceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Service not found",
			Error:   err.Error(),
		})
	}
	fmt.Println("Fetched service:", service)
	// Get duration directly from service
	duration := service.Duration

	// Convert StartTime to IST before checking availability
	appointment.StartTime = utils.ToIST(appointment.StartTime)
	fmt.Println("Converted StartTime to IST:", appointment.StartTime)
	// Check if the appointment falls within the provider's working hours
	isWorkingHour, err := utils.CheckWorkingDayAndHours(appointment.ProviderID, appointment.StartTime)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Error checking working hours",
			Error:   err.Error(),
		})
	}
	// Check if the appointment is during break time
	fmt.Println("Checking break time...")
	if !isWorkingHour {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Message: "Appointment is outside working hours or during break",
		})
	}
	fmt.Println("Checked break time successfully")

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

	// Set status to pending by default
	appointment.Status = models.StatusPending

	fmt.Println("Setting status to pending:", appointment.Status)

	// Create appointment and recurrence in a transaction
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// Check availability again to prevent conflicts
		available, err := utils.CheckAvailability(appointment.ProviderID, appointment.StartTime, duration)
		if err != nil {
			return err
		}
		if !available {
			return fmt.Errorf("time slot not available")
		}

		// Create the appointment
		if err := tx.Create(&appointment).Error; err != nil {
			return err
		}

		// Handle Recurrence if `is_recurring` is true
		if appointment.IsRecurring {
			recurrence := models.Recurrence{
				AppointmentID: appointment.ID,
				NextRun:       appointment.StartTime,
				Frequency:     appointment.RecurPattern.Frequency,
				EndAfter:      appointment.RecurPattern.EndAfter,
			}

			// Create the recurrence
			if err := tx.Create(&recurrence).Error; err != nil {
				return fmt.Errorf("failed to create recurrence: %v", err)
			}

			// Link recurrence to the appointment
			if err := tx.Model(&appointment).Update("recurrence_id", recurrence.ID).Error; err != nil {
				return fmt.Errorf("failed to update appointment with recurrence_id: %v", err)
			}
		}

		return nil
	})
	fmt.Println("Transaction completed successfully")
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(utils.ErrorResponse{
			Message: "Time slot not available or failed to create appointment",
			Error:   err.Error(),
		})
	}
	// Find the customer and provider to send emails
	var customer models.User
	if err := db.DB.First(&customer, appointment.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Customer not found",
			Error:   err.Error(),
		})
	}
	var provider models.User
	if err := db.DB.First(&provider, appointment.ProviderID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Provider not found",
			Error:   err.Error(),
		})
	}
	fmt.Println("provider :", provider.Email)
	fmt.Println("customer :", customer.Email)
	fmt.Println("Appointment created successfully:", appointment.CustomerID)
	fmt.Println("Appointment ID:", appointment.ProviderID)
	// Send confirmation email
	emailBody := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your appointment has been successfully created.</p>
		<p><strong>Details:</strong></p>
		<ul>
			<li><strong>Service:</strong> %s</li>
			<li><strong>Provider:</strong> %s</li>
			<li><strong>Start Time:</strong> %s</li>
			<li><strong>End Time:</strong> %s</li>
			<li><strong>Status:</strong> %s</li>
		</ul>
		<p>Thank you for choosing our service!</p>
		<p>Best regards,</p>
		<p>Your Appointment Team</p>
	`, customer.Name, service.Name, provider.Name,
		appointment.StartTime.Format("2006-01-02 15:04:05"), appointment.EndTime.Format("2006-01-02 15:04:05"),
		appointment.Status)
	if err := utils.SendEmail(customer.Email, "Appointment Confirmation", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to send confirmation email",
			Error:   err.Error(),
		})
	}
	fmt.Println("Confirmation email sent successfully")
	//mail to service provider
	emailBody = fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>You have a new appointment scheduled.</p>
		<p><strong>Details:</strong></p>
		<ul>
			<li><strong>Service:</strong> %s</li>
			<li><strong>Customer:</strong> %s</li>
			<li><strong>Start Time:</strong> %s</li>
			<li><strong>End Time:</strong> %s</li>
			<li><strong>Status:</strong> %s</li>
		</ul>
		<p>Thank you for choosing our service!</p>
		<p>Best regards,</p>
		<p>Your Appointment Team</p>
	`, provider.Name, service.Name, customer.Name,
		appointment.StartTime.Format("2006-01-02 15:04:05"), appointment.EndTime.Format("2006-01-02 15:04:05"),
		appointment.Status)
	if err := utils.SendEmail(provider.Email, "New Appointment Scheduled", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to send confirmation email to provider",
			Error:   err.Error(),
		})
	}
	fmt.Println("Confirmation email to provider sent successfully")

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
		// Do Not Change Status
		updatedAppointment.Status = existingAppointment.Status

		// Perform the update
		if err := tx.Model(&existingAppointment).Where("id = ?", id).Updates(updatedAppointment).Error; err != nil {
			return err
		}
		return nil
	})
	// find consumer and provider to send emails
	var customer models.User
	if err := db.DB.First(&customer, existingAppointment.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Customer not found",
			Error:   err.Error(),
		})
	}
	var provider models.User
	if err := db.DB.First(&provider, existingAppointment.ProviderID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Provider not found",
			Error:   err.Error(),
		})
	}
	// find service to send emails
	var service models.Service
	if err := db.DB.First(&service, existingAppointment.ServiceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Service not found",
			Error:   err.Error(),
		})
	}
	// Send confirmation email
	emailBody := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your appointment has been successfully updated.</p>	
		<ul>
			<li>Title: %s</li>	
			<li>Description: %s</li>
			<li>Start Time: %s</li>
			<li>End Time: %s</li>
			<li>Service: %s</li>
			<li>Provider: %s</li>
		</ul>
		<p>Best regards,<br>
		Your Appointment Management System</p>
	`, customer.Name, updatedAppointment.Title, updatedAppointment.Description, updatedAppointment.StartTime, updatedAppointment.EndTime, service.Name, provider.Name)
	if err := utils.SendEmail(customer.Email, "Appointment Updated", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to send confirmation email",
			Error:   err.Error(),
		})
	}
	//mail to service provider
	emailBody = fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>Your appointment has been successfully updated.</p>
		<ul>
			<li>Title: %s</li>
			<li>Description: %s</li>
			<li>Start Time: %s</li>
			<li>End Time: %s</li>
			<li>Service: %s</li>
			<li>Customer: %s</li>
		</ul>
		<p>Best regards,<br>
		Your Appointment Management System</p>
	`, provider.Name, updatedAppointment.Title, updatedAppointment.Description, updatedAppointment.StartTime, updatedAppointment.EndTime, service.Name, customer.Name)
	if err := utils.SendEmail(provider.Email, "Appointment Updated", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to send confirmation email to provider",
			Error:   err.Error(),
		})
	}
	fmt.Println("Confirmation email to provider sent successfully")

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
	var appointment models.Appointment
	if err := db.DB.First(&appointment, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Appointment not found",
			Error:   err.Error(),
		})
	}

	// Prevent deletion of completed or canceled appointments
	if appointment.Status == models.StatusCompleted || appointment.Status == models.StatusCanceled {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Message: "Cannot delete a completed or canceled appointment",
		})
	}

	// Delete the appointment
	if err := db.DB.Delete(&appointment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to delete appointment",
			Error:   err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
