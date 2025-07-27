package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
)

// GetAllAppointments retrieves all appointments for the authenticated provider
// @Summary Get all provider appointments
// @Description Retrieve a list of all appointments for the authenticated provider, optionally filtered by status
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter appointments by status (e.g., pending, confirmed, completed, canceled)"
// @Success 200 {array} models.Appointment "List of appointments"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not a provider or admin"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments [get]
func GetAllAppointments(c *fiber.Ctx) error {
	appointmentStatus := c.Query("status")
	month := c.Query("month")
	year := c.Query("year")

	//Get appointments for service provider take provider id from c.Locals("userID")
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}
	// Get user role
	role, ok := c.Locals("role").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User role not found in context",
		})
	}
	// Verify that the user is a provider or receptionist
	if role != "provider" && role != "admin" && role != "receptionist" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers and receptionists can access this endpoint.",
		})
	}

	appointmentsChan := make(chan []models.Appointment)
	errorChan := make(chan error)

	go func() {
		var appointments []models.Appointment
		query := db.DB.
			Preload("Service").
			Preload("Provider").
			Preload("Customer").
			Where("provider_id = ?", userID)

		if appointmentStatus != "" {
			query = query.Where("status = ?", appointmentStatus)
		}

		// Filter by month and year if provided
		if month != "" && year != "" {
			// Parse month and year
			m, errM := strconv.Atoi(month)
			y, errY := strconv.Atoi(year)
			if errM == nil && errY == nil {
				start := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
				end := start.AddDate(0, 1, 0)
				query = query.Where("start_time >= ? AND start_time < ?", start, end)
			}
		}

		if err := query.Find(&appointments).Error; err != nil {
			errorChan <- err
			return
		}
		appointmentsChan <- appointments
	}()

	select {
	case appointments := <-appointmentsChan:
		return c.JSON(appointments)
	case err := <-errorChan:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}

// GetAppointmentDetails retrieves details of a specific appointment
// @Summary Get appointment details
// @Description Retrieve details of a specific appointment by ID, including service, provider, and customer information
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Success 200 {object} models.Appointment "Appointment details"
// @Failure 400 {object} utils.ErrorResponse{Message=string,Error=string} "Bad request - invalid appointment ID"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 404 {object} utils.ErrorResponse{Message=string,Error=string} "Appointment not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments/{id} [get]
func GetAppointmentDetails(c *fiber.Ctx) error {
	appointmentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ErrorResponse{
			Message: "Invalid appointment ID",
			Error:   err.Error(),
		})
	}

	appointmentChan := make(chan models.Appointment)
	errorChan := make(chan error)

	go func() {
		var appointment models.Appointment
		if err := db.DB.Preload("Service").Preload("Provider").Preload("Customer").First(&appointment, appointmentID).Error; err != nil {
			errorChan <- err
			return
		}
		appointmentChan <- appointment
	}()

	select {
	case appointment := <-appointmentChan:
		return c.JSON(appointment)
	case err := <-errorChan:
		return c.Status(fiber.StatusNotFound).JSON(utils.ErrorResponse{
			Message: "Appointment not found",
			Error:   err.Error(),
		})
	}
}

// GetProviderUpcomingAppointments returns upcoming appointments for the logged-in provider
// @Summary Get upcoming appointments
// @Description Retrieve a list of upcoming appointments (pending or confirmed) for the authenticated provider, filtered by date range
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of appointments to return (default 10)"
// @Param filter query string false "Date filter (today, tomorrow, week, month; default month)"
// @Success 200 {object} object{appointments=[]models.Appointment,count=int,filter=string,start_date=string,end_date=string} "List of upcoming appointments"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not a provider or admin"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments/upcoming [get]
func GetProviderUpcomingAppointments(c *fiber.Ctx) error {
	// Get the authenticated user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	// Get user role
	role, ok := c.Locals("role").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User role not found in context",
		})
	}

	// Verify that the user is a provider or receptionist
	if role != "provider" && role != "admin" && role != "receptionist" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers and receptionists can access this endpoint.",
		})
	}

	// Get optional query parameters
	limit := 10 // Default limit
	if c.Query("limit") != "" {
		if parsedLimit := c.QueryInt("limit"); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Parse date filter if provided
	var startDate time.Time
	var endDate time.Time
	now := time.Now()

	// Default: from now to 30 days in the future
	startDate = now
	endDate = now.AddDate(0, 0, 30)

	// Override date range if filter is provided
	dateFilter := c.Query("filter", "month")
	switch dateFilter {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1)
		startDate = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, now.Location())
	case "week":
		startDate = now
		endDate = now.AddDate(0, 0, 7)
	case "month":
		startDate = now
		endDate = now.AddDate(0, 1, 0)
	}

	type appointmentResult struct {
		Appointments []models.Appointment
		Err          error
	}

	resultChan := make(chan appointmentResult)

	go func() {
		var result appointmentResult

		// Query for upcoming appointments
		query := db.DB.
			Preload("Service").
			Preload("Customer").
			Where("provider_id = ?", userID).
			Where("start_time >= ?", startDate).
			Where("start_time <= ?", endDate).
			Where("status IN ?", []models.AppointmentStatus{models.StatusPending, models.StatusConfirmed})

		// Sort by start time
		query = query.Order("start_time asc")

		// Apply limit
		if limit > 0 {
			query = query.Limit(limit)
		}

		// Execute the query
		if err := query.Find(&result.Appointments).Error; err != nil {
			result.Err = err
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"appointments": result.Appointments,
		"count":        len(result.Appointments),
		"filter":       dateFilter,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
	})
}

// GetProviderAppointmentHistory returns past appointments for the logged-in provider
// @Summary Get appointment history
// @Description Retrieve a paginated list of past appointments (completed or canceled) for the authenticated provider, filtered by status and date range
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Number of appointments per page (default 10)"
// @Param status query string false "Filter by status (completed, canceled)"
// @Param range query string false "Date range (week, month, year, all; default month)"
// @Success 200 {object} object{appointments=[]models.Appointment,total=int64,page=int,limit=int,pages=int64,range=string,status=string} "Appointment history"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not a provider or admin"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments/history [get]
func GetProviderAppointmentHistory(c *fiber.Ctx) error {
	// Get the authenticated user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	// Get user role
	role, ok := c.Locals("role").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User role not found in context",
		})
	}

	// Verify that the user is a provider
	if role != "provider" && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers can access this endpoint.",
		})
	}

	// Get pagination parameters
	page := 1
	limit := 10

	if c.Query("page") != "" {
		if parsedPage := c.QueryInt("page"); parsedPage > 0 {
			page = parsedPage
		}
	}

	if c.Query("limit") != "" {
		if parsedLimit := c.QueryInt("limit"); parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Parse optional status filter
	var statuses []models.AppointmentStatus
	status := c.Query("status")
	if status != "" {
		switch models.AppointmentStatus(status) {
		case models.StatusCompleted:
			statuses = []models.AppointmentStatus{models.StatusCompleted}
		case models.StatusCanceled:
			statuses = []models.AppointmentStatus{models.StatusCanceled}
		default:
			statuses = []models.AppointmentStatus{models.StatusCompleted, models.StatusCanceled}
		}
	} else {
		// Default: show both completed and canceled
		statuses = []models.AppointmentStatus{models.StatusCompleted, models.StatusCanceled}
	}

	// Parse optional date range
	var startDate, endDate time.Time
	now := time.Now()

	// Default: last 30 days
	startDate = now.AddDate(0, 0, -30)
	endDate = now

	// Override if specific range provided
	dateRange := c.Query("range", "month")
	switch dateRange {
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	case "all":
		startDate = time.Time{} // Beginning of time
	}

	type appointmentResult struct {
		Appointments []models.Appointment
		Total        int64
		Err          error
	}

	resultChan := make(chan appointmentResult)

	go func() {
		var result appointmentResult

		// Count total matching appointments
		countQuery := db.DB.Model(&models.Appointment{}).
			Where("provider_id = ?", userID).
			Where("status IN ?", statuses)

		// Apply date filter if not "all"
		if dateRange != "all" {
			countQuery = countQuery.Where("end_time >= ? AND end_time <= ?", startDate, endDate)
		}

		if err := countQuery.Count(&result.Total).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		// Query for appointment history
		query := db.DB.
			Preload("Service").
			Preload("Customer").
			Where("provider_id = ?", userID).
			Where("status IN ?", statuses)

		// Apply date filter if not "all"
		if dateRange != "all" {
			query = query.Where("end_time >= ? AND end_time <= ?", startDate, endDate)
		}

		// Apply ordering, pagination
		if err := query.
			Order("end_time desc").
			Offset(offset).
			Limit(limit).
			Find(&result.Appointments).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"appointments": result.Appointments,
		"total":        result.Total,
		"page":         page,
		"limit":        limit,
		"pages":        (result.Total + int64(limit) - 1) / int64(limit), // Ceiling division
		"range":        dateRange,
		"status":       status,
	})
}

// UpdateAppointmentStatus updates the status of an appointment (accept/reject)
// @Summary Update appointment status
// @Description Update the status of an appointment (confirmed, canceled, completed) for the authenticated provider, requires 'services' update permission
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Param status body object{status=string} true "New status (confirmed, canceled, completed)"
// @Success 200 {object} object{message=string,appointment=models.Appointment} "Appointment status updated"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid ID, status, or input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user does not own the appointment or lacks 'services' update permission"
// @Failure 404 {object} fiber.Map{error=string} "Appointment, provider, or customer not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments/{id}/status [patch]
func UpdateAppointmentStatus(c *fiber.Ctx) error {
	// Get the authenticated user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	// Get user role
	role, ok := c.Locals("role").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User role not found in context",
		})
	}

	// Get appointment ID from URL
	appointmentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid appointment ID",
		})
	}

	// Parse request body
	var updateData struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate status value
	newStatus := models.AppointmentStatus(updateData.Status)
	if newStatus != models.StatusConfirmed &&
		newStatus != models.StatusCanceled &&
		newStatus != models.StatusCompleted {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status. Must be 'confirmed', 'canceled', or 'completed'.",
		})
	}

	type updateResult struct {
		Appointment models.Appointment
		Provider    models.User
		Customer    models.User
		Err         error
	}

	resultChan := make(chan updateResult)

	go func() {
		var result updateResult

		// Find the appointment
		if err := db.DB.Preload("Service").First(&result.Appointment, appointmentID).Error; err != nil {
			result.Err = fmt.Errorf("appointment not found: %v", err)
			resultChan <- result
			return
		}

		// Check if the provider owns this appointment
		if result.Appointment.ProviderID != userID && role != "admin" {
			//Get Provider ID by receptionistID
			var provider models.ReceptionistSettings
			if err := db.DB.First(&provider, "receptionist_id = ?", userID).Error; err != nil {
				result.Err = fmt.Errorf("provider not found: %v", err)
				resultChan <- result
				return
			}
			if result.Appointment.ProviderID != provider.ProviderID {
				// Check if the appointment belongs to the provider
				result.Err = fmt.Errorf("you can only update your own appointments")
				resultChan <- result
				return
			}
		}

		// Update the status
		if err := result.Appointment.UpdateStatus(db.DB, newStatus); err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		// find provider and customer
		if err := db.DB.First(&result.Provider, result.Appointment.ProviderID).Error; err != nil {
			result.Err = fmt.Errorf("provider not found: %v", err)
			resultChan <- result
			return
		}

		if err := db.DB.First(&result.Customer, result.Appointment.CustomerID).Error; err != nil {
			result.Err = fmt.Errorf("customer not found: %v", err)
			resultChan <- result
			return
		}

		// Send email notification
		emailBody := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<title>Appointment Status Update</title>
		</head>
		<body>
			<h1>Appointment Status Update</h1>
			<p>Dear %s,</p>
			<p>Your appointment with %s has been %s.</p>
			<p>Appointment Details:</p>
			<ul>
				<li>Service: %s</li>
				<li>Start Time: %s</li>
				<li>End Time: %s</li>
				<li>Status: %s</li>
			</ul>
			<p>Thank you for using our service!</p>
			<p>Best regards,</p>
			<p>Your Appointment App Team</p>
		</body>
		</html>
			`
		emailBody = fmt.Sprintf(emailBody, result.Customer.Name, result.Provider.Name, newStatus,
			result.Appointment.Service.Name, result.Appointment.StartTime.Format("2006-01-02 15:04"),
			result.Appointment.EndTime.Format("2006-01-02 15:04"), newStatus)

		if err := utils.SendEmail(result.Customer.Email, "Appointment Status Update", emailBody); err != nil {
			result.Err = fmt.Errorf("failed to send email notification: %v", err)
			resultChan <- result
			return
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		if strings.Contains(result.Err.Error(), "Appointment not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		} else if strings.Contains(result.Err.Error(), "Provider not found") ||
			strings.Contains(result.Err.Error(), "Customer not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		} else if strings.Contains(result.Err.Error(), "You can only update your own appointments") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		} else if strings.Contains(result.Err.Error(), "Failed to send email") {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": result.Err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":     "Appointment status updated successfully",
		"appointment": result.Appointment,
	})
}

// RescheduleAppointment reschedules an existing appointment
// @Summary Reschedule appointment
// @Description Reschedule an existing appointment to a new start time, requires 'services' update permission
// @Tags provider-appointments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Appointment ID"
// @Param start_time body object{start_time=string} true "New start time in RFC3339 format"
// @Success 200 {object} object{message=string,appointment=models.Appointment} "Appointment rescheduled"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid ID, start time, or appointment status"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user does not own the appointment or lacks 'services' update permission"
// @Failure 404 {object} fiber.Map{error=string} "Appointment, provider, service, or customer not found"
// @Failure 409 {object} fiber.Map{error=string} "Conflict - time slot conflicts or outside working hours"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/appointments/{id}/reschedule [patch]
func RescheduleAppointment(c *fiber.Ctx) error {
	// Get the authenticated user ID from context
	userID, ok := c.Locals("userID").(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User ID not found in context",
		})
	}

	// Get user role
	role, ok := c.Locals("role").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User role not found in context",
		})
	}

	// Verify that the user is a provider
	if role != "provider" && role != "admin" && role != "receptionist" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers can reschedule appointments.",
		})
	}

	// Get appointment ID from URL
	appointmentID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid appointment ID",
		})
	}

	// Parse request body
	var rescheduleData struct {
		StartTime string `json:"start_time"`
	}

	if parseErr := c.BodyParser(&rescheduleData); parseErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}

	// Parse new times
	startTime, err := time.Parse(time.RFC3339, rescheduleData.StartTime)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid start time format. Please use RFC3339 format.",
		})
	}

	now := time.Now()
	fmt.Println("Current time:", startTime)
	if startTime.Before(now) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot schedule an appointment in the past",
		})
	}

	type rescheduleResult struct {
		Appointment models.Appointment
		Service     models.Service
		Provider    models.User
		Customer    models.User
		Err         error
		StatusCode  int
	}

	resultChan := make(chan rescheduleResult)

	go func() {
		var result rescheduleResult

		// Find the appointment
		if err := db.DB.First(&result.Appointment, appointmentID).Error; err != nil {
			result.Err = fmt.Errorf("appointment not found")
			result.StatusCode = fiber.StatusNotFound
			resultChan <- result
			return
		}

		// Find service duration to calculate end time
		if err := db.DB.First(&result.Service, result.Appointment.ServiceID).Error; err != nil {
			result.Err = fmt.Errorf("service not found")
			result.StatusCode = fiber.StatusNotFound
			resultChan <- result
			return
		}
		endTime := startTime.Add(time.Duration(result.Service.Duration) * time.Minute)

		// Check if the provider owns this appointment
		if result.Appointment.ProviderID != userID && role != "admin" {
			var receptionistSettings models.ReceptionistSettings
			if err := db.DB.First(&receptionistSettings, "receptionist_id = ?", userID).Error; err != nil {
				result.Err = fmt.Errorf("provider not found")
				result.StatusCode = fiber.StatusNotFound
				resultChan <- result
				return
			}
			if result.Appointment.ProviderID != receptionistSettings.ProviderID {
				// Check if the appointment belongs to the provider
				result.Err = fmt.Errorf("you can only update your own appointments")
				result.StatusCode = fiber.StatusForbidden
				resultChan <- result
				return
			}
		}

		// Check if appointment is in a status that can be rescheduled
		if result.Appointment.Status != models.StatusPending && result.Appointment.Status != models.StatusConfirmed {
			result.Err = fmt.Errorf("only pending or confirmed appointments can be rescheduled")
			result.StatusCode = fiber.StatusBadRequest
			resultChan <- result
			return
		}

		// Check for scheduling conflicts
		var conflictCount int64
		db.DB.Model(&models.Appointment{}).
			Where("provider_id = ? AND id != ?", userID, appointmentID).
			Where("status IN ?", []models.AppointmentStatus{models.StatusPending, models.StatusConfirmed}).
			Where("(start_time < ? AND end_time > ?) OR (start_time >= ? AND start_time < ?)",
				endTime, startTime, startTime, endTime).
			Count(&conflictCount)

		if conflictCount > 0 {
			result.Err = fmt.Errorf("the requested time slot conflicts with existing appointments")
			result.StatusCode = fiber.StatusConflict
			resultChan <- result
			return
		}

		// Update the appointment times
		result.Appointment.StartTime = startTime
		duration := result.Service.Duration
		total_duration := duration + result.Service.BufferTime
		isWorkingHour, err := utils.CheckWorkingDayAndHours(result.Appointment.ProviderID, result.Appointment.StartTime)
		if err != nil {
			result.Err = fmt.Errorf("error checking working hours: %v", err)
			result.StatusCode = fiber.StatusInternalServerError
			resultChan <- result
			return
		}
		// Check if the appointment is during break time
		fmt.Println("Checking break time...")
		if !isWorkingHour {
			result.Err = fmt.Errorf("appointment is outside working hours or during break")
			result.StatusCode = fiber.StatusConflict
			resultChan <- result
			return
		}
		available, err := utils.CheckAvailability(result.Appointment.ProviderID, result.Appointment.StartTime, total_duration)
		if err != nil {
			result.Err = fmt.Errorf("failed to check availability: %v", err)
			result.StatusCode = fiber.StatusInternalServerError
			resultChan <- result
			return
		}
		if !available {
			result.Err = fmt.Errorf("the requested time slot conflicts with existing appointments")
			result.StatusCode = fiber.StatusConflict
			resultChan <- result
			return
		}
		result.Appointment.EndTime = startTime.Add(time.Duration(result.Service.Duration) * time.Minute)
		result.Appointment.Status = models.StatusPending
		if dbErr := db.DB.Save(&result.Appointment).Error; dbErr != nil {
			result.Err = fmt.Errorf("failed to reschedule appointment: %v", err)
			result.StatusCode = fiber.StatusInternalServerError
			resultChan <- result
			return
		}
		// find provider and customer and send email
		if dbErr := db.DB.First(&result.Provider, result.Appointment.ProviderID).Error; dbErr != nil {
			result.Err = fmt.Errorf("provider not found")
			result.StatusCode = fiber.StatusNotFound
			resultChan <- result
			return
		}
		if dbErr := db.DB.First(&result.Customer, result.Appointment.CustomerID).Error; dbErr != nil {
			result.Err = fmt.Errorf("customer not found")
			result.StatusCode = fiber.StatusNotFound
			resultChan <- result
			return
		}
		emailBody := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">	
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Appointment Rescheduled</title>
		</head>
		<body>
			<h1>Appointment Rescheduled</h1>
			<p>Dear ` + result.Customer.Name + `,</p>
			<p>Your appointment with ` + result.Provider.Name + ` has been rescheduled to the following times:</p>
			<p>Start Time: ` + result.Appointment.StartTime.Format("2006-01-02 15:04:05") + `</p>
			<p>End Time: ` + result.Appointment.EndTime.Format("2006-01-02 15:04:05") + `</p>
			<p>Best regards,<br>` + result.Provider.Name + `</p>
		</body>
		</html>
	`
		err = utils.SendEmail(result.Customer.Email, "Appointment Rescheduled", emailBody)
		if err != nil {
			result.Err = fmt.Errorf("failed to send email: %v", err)
			result.StatusCode = fiber.StatusInternalServerError
			resultChan <- result
			return
		}

		fmt.Println("Email sent to:", result.Customer.Email)
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(result.StatusCode).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Appointment rescheduled successfully",
		"appointment": result.Appointment,
	})
}
