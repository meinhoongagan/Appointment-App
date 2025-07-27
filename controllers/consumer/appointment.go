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
// @Summary Get all appointments
// @Description Retrieve a list of all appointments with associated service, provider, and customer details
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Appointment "List of appointments"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments [get]
func GetAllAppointments(c *fiber.Ctx) error {
	appointmentChan := make(chan []models.Appointment)
	go func() {
		var appointments []models.Appointment
		if err := db.DB.Preload("Service").Preload("Provider").Preload("Customer").Find(&appointments).Error; err != nil {
			appointmentChan <- nil
			return
		}
		appointmentChan <- appointments
	}()

	appointments := <-appointmentChan
	if appointments == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to fetch appointments",
			Error:   "Database error",
		})
	}
	return c.JSON(appointments)
}

// GetAppointment godoc
// @Summary Get appointment by ID
// @Description Retrieve a specific appointment by its ID with associated service, provider, and customer details
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Success 200 {object} models.Appointment "Appointment details"
// @Failure 404 {object} utils.ErrorResponse "Appointment not found"
// @Router /appointments/{id} [get]
func GetAppointment(c *fiber.Ctx) error {
	appointmentChain := make(chan models.Appointment)
	go func() {
		id := c.Params("id")
		var appointment models.Appointment
		if err := db.DB.Preload("Service").Preload("Provider").Preload("Customer").First(&appointment, id).Error; err != nil {
			appointmentChain <- models.Appointment{}
			return
		}
		appointmentChain <- appointment
	}()

	appointment := <-appointmentChain
	if appointment.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Appointment not found",
			Error:   "Database error",
		})
	}
	return c.JSON(appointment)
}

// GetServiceDetails godoc
// @Summary Get service details by ID
// @Description Retrieve detailed information about a specific service including provider and category
// @Tags services
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Service ID"
// @Success 200 {object} models.Service "Service details"
// @Failure 404 {object} fiber.Map "Service not found"
// @Router /appointments/service/{id} [get]
func GetServiceDetails(c *fiber.Ctx) error {
	serviceChan := make(chan models.Service)

	go func() {
		id := c.Params("id")
		var service models.Service
		if err := db.DB.First(&service, id).Error; err != nil {
			service = models.Service{}
		} else {
			// Optionally preload provider or category
			if err := db.DB.Preload("Provider").Preload("Category").First(&service, id).Error; err != nil {
				service = models.Service{}
			}
		}
		serviceChan <- service
	}()

	service := <-serviceChan
	if service.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}
	return c.JSON(service)
}

// CreateAppointment godoc
// @Summary Create a new appointment
// @Description Create a new appointment with availability checking, working hours validation, and email notifications
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param appointment body models.Appointment true "Appointment details"
// @Success 201 {object} models.Appointment "Created appointment"
// @Failure 400 {object} utils.ErrorResponse "Bad request - invalid input"
// @Failure 404 {object} utils.ErrorResponse "Service/Customer/Provider not found"
// @Failure 409 {object} utils.ErrorResponse "Time slot conflict or outside working hours"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments [post]
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
	totalDuration := duration + service.BufferTime

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
	available, err := utils.CheckAvailability(appointment.ProviderID, appointment.StartTime, totalDuration)
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
		available, err = utils.CheckAvailability(appointment.ProviderID, appointment.StartTime, duration)
		if err != nil {
			return err
		}
		if !available {
			return fmt.Errorf("time slot not available")
		}

		// Create the appointment
		if createErr := tx.Create(&appointment).Error; createErr != nil {
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
			if createRecErr := tx.Create(&recurrence).Error; createRecErr != nil {
				return fmt.Errorf("failed to create recurrence: %v", err)
			}

			// Link recurrence to the appointment
			if updateErr := tx.Model(&appointment).Update("recurrence_id", recurrence.ID).Error; updateErr != nil {
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
// @Summary Update an existing appointment
// @Description Update appointment details with availability checking and email notifications
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Param appointment body models.Appointment true "Updated appointment details"
// @Success 200 {object} models.Appointment "Updated appointment"
// @Failure 400 {object} utils.ErrorResponse "Bad request - invalid input"
// @Failure 404 {object} utils.ErrorResponse "Appointment/Service/Customer/Provider not found"
// @Failure 409 {object} utils.ErrorResponse "Time slot conflict"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments/{id} [patch]
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
	if dbErr := db.DB.First(&customer, existingAppointment.CustomerID).Error; dbErr != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Customer not found",
			Error:   err.Error(),
		})
	}
	var provider models.User
	if dbErr := db.DB.First(&provider, existingAppointment.ProviderID).Error; dbErr != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Provider not found",
			Error:   err.Error(),
		})
	}
	// find service to send emails
	var service models.Service
	if dbErr := db.DB.First(&service, existingAppointment.ServiceID).Error; dbErr != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Service not found",
			Error:   dbErr.Error(),
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
	if emailErr := utils.SendEmail(customer.Email, "Appointment Updated", emailBody); emailErr != nil {
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
	if emailErr := utils.SendEmail(provider.Email, "Appointment Updated", emailBody); emailErr != nil {
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

// GetUpcomingAppointments godoc
// @Summary Get upcoming appointments or appointment history
// @Description Retrieve upcoming appointments or appointment history for the authenticated user based on status query parameter
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string true "Status filter" Enums(upcoming, history) "Filter by 'upcoming' for future appointments or 'history' for past appointments"
// @Success 200 {array} models.Appointment "List of appointments"
// @Failure 400 {object} utils.ErrorResponse "Bad request - invalid or missing status parameter"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized - invalid user token"
// @Failure 404 {object} utils.ErrorResponse "No appointments found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments [get]
func GetUpcomingAppointments(c *fiber.Ctx) error {
	appointmentsChan := make(chan []models.Appointment)

	go func() {
		var appointments []models.Appointment
		status := c.Query("status")
		if status == "" {
			appointments = []models.Appointment{}
		} else if status != "upcoming" && status != "history" {
			appointments = []models.Appointment{}
		} else {
			// Get the user ID from the JWT token
			userID, ok := c.Locals("userID").(uint)
			if !ok {
				appointments = []models.Appointment{}
			} else {
				// Get the current time in UTC
				currentTime := time.Now().UTC()
				// Query the database based on the status
				if status == "upcoming" {
					if err := db.DB.Where("customer_id = ? AND start_time > ?", userID, currentTime).Preload("Service").Preload("Provider").Find(&appointments).Error; err != nil {
						appointments = []models.Appointment{}
					}
				}
				if status == "history" {
					if err := db.DB.Where("customer_id = ? AND start_time < ?", userID, currentTime).Preload("Service").Preload("Provider").Find(&appointments).Error; err != nil {
						appointments = []models.Appointment{}
					}
				}
			}
		}
		appointmentsChan <- appointments
	}()

	appointments := <-appointmentsChan
	if len(appointments) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "No appointments found",
		})
	}
	return c.JSON(appointments)
}

// CancelAppointment godoc
// @Summary Cancel an appointment
// @Description Cancel an existing appointment by changing its status to canceled
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Success 200 {object} models.Appointment "Canceled appointment"
// @Failure 403 {object} utils.ErrorResponse "Forbidden - cannot cancel completed or already canceled appointment"
// @Failure 404 {object} utils.ErrorResponse "Appointment not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments/{id}/cancel [patch]
func CancelAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var appointment models.Appointment
	if err := db.DB.First(&appointment, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Appointment not found",
			Error:   err.Error(),
		})
	}

	// Prevent cancellation of completed or canceled appointments
	if appointment.Status == models.StatusCompleted || appointment.Status == models.StatusCanceled {
		return c.Status(fiber.StatusForbidden).JSON(utils.ErrorResponse{
			Message: "Cannot cancel a completed or already canceled appointment",
		})
	}

	// Update the status to canceled
	appointment.Status = models.StatusCanceled
	if err := db.DB.Save(&appointment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ErrorResponse{
			Message: "Failed to cancel appointment",
			Error:   err.Error(),
		})
	}
	return c.JSON(appointment)
}

// DeleteAppointment godoc
// @Summary Delete an appointment
// @Description Permanently delete an appointment from the system
// @Tags appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Success 204 "No content - appointment successfully deleted"
// @Failure 403 {object} utils.ErrorResponse "Forbidden - cannot delete completed or canceled appointment"
// @Failure 404 {object} utils.ErrorResponse "Appointment not found"
// @Failure 500 {object} utils.ErrorResponse "Internal server error"
// @Router /appointments/{id} [delete]
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
