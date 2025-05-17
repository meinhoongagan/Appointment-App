package service

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
)

// GetProviderUpcomingAppointments returns upcoming appointments for the logged-in provider
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

	// Verify that the user is a provider
	if role != "provider" && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers can access this endpoint.",
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

	var appointments []models.Appointment

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
	if err := query.Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"appointments": appointments,
		"count":        len(appointments),
		"filter":       dateFilter,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
	})
}

// GetProviderAppointmentHistory returns past appointments for the logged-in provider
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

	var appointments []models.Appointment
	var total int64

	// Count total matching appointments
	countQuery := db.DB.Model(&models.Appointment{}).
		Where("provider_id = ?", userID).
		Where("status IN ?", statuses)

	// Apply date filter if not "all"
	if dateRange != "all" {
		countQuery = countQuery.Where("end_time >= ? AND end_time <= ?", startDate, endDate)
	}

	countQuery.Count(&total)

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
		Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"appointments": appointments,
		"total":        total,
		"page":         page,
		"limit":        limit,
		"pages":        (total + int64(limit) - 1) / int64(limit), // Ceiling division
		"range":        dateRange,
		"status":       status,
	})
}

// UpdateAppointmentStatus updates the status of an appointment (accept/reject)
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

	// Verify that the user is a provider
	if role != "provider" && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied. Only providers can update appointment status.",
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

	// Find the appointment
	var appointment models.Appointment
	if err := db.DB.First(&appointment, appointmentID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Appointment not found",
		})
	}

	// Check if the provider owns this appointment
	if appointment.ProviderID != userID && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You can only update your own appointments",
		})
	}

	// Update the status
	if err := appointment.UpdateStatus(db.DB, newStatus); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// find provider and customer and send email
	var provider models.User
	if err := db.DB.First(&provider, appointment.ProviderID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}
	var customer models.User
	if err := db.DB.First(&customer, appointment.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
	}
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
	emailBody = fmt.Sprintf(emailBody, customer.Name, provider.Name, newStatus, appointment.Service.Name, appointment.StartTime.Format("2006-01-02 15:04"), appointment.EndTime.Format("2006-01-02 15:04"), newStatus)
	if err := utils.SendEmail(customer.Email, "Appointment Status Update", emailBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send email notification",
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Appointment status updated successfully",
		"appointment": appointment,
	})
}

// RescheduleAppointment reschedules an existing appointment
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
	if role != "provider" && role != "admin" {
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

	if err := c.BodyParser(&rescheduleData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
	if startTime.Before(now) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot schedule an appointment in the past",
		})
	}

	// Find the appointment
	var appointment models.Appointment
	if err := db.DB.First(&appointment, appointmentID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Appointment not found",
		})
	}

	// Find service duration to calculate end time
	var service models.Service
	if err := db.DB.First(&service, appointment.ServiceID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
		})
	}
	endTime := startTime.Add(time.Duration(service.Duration) * time.Minute)

	// Check if the provider owns this appointment
	if appointment.ProviderID != userID && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You can only reschedule your own appointments",
		})
	}

	// Check if appointment is in a status that can be rescheduled
	if appointment.Status != models.StatusPending && appointment.Status != models.StatusConfirmed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only pending or confirmed appointments can be rescheduled",
		})
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
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "The requested time slot conflicts with existing appointments",
		})
	}

	// Update the appointment times
	appointment.StartTime = startTime
	appointment.EndTime = endTime

	if err := db.DB.Save(&appointment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reschedule appointment",
		})
	}
	// find provider and customer and send email
	var provider models.User
	if err := db.DB.First(&provider, appointment.ProviderID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}
	var customer models.User
	if err := db.DB.First(&customer, appointment.CustomerID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Customer not found",
		})
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
			<p>Dear ` + customer.Name + `,</p>
			<p>Your appointment with ` + provider.Name + ` has been rescheduled to the following times:</p>
			<p>Start Time: ` + appointment.StartTime.Format("2006-01-02 15:04:05") + `</p>
			<p>End Time: ` + appointment.EndTime.Format("2006-01-02 15:04:05") + `</p>
			<p>Best regards,<br>` + provider.Name + `</p>
		</body>
		</html>
	`
	err = utils.SendEmail(customer.Email, "Appointment Rescheduled", emailBody)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send email",
		})
	}

	fmt.Println("Email sent to:", customer.Email)

	return c.JSON(fiber.Map{
		"message":     "Appointment rescheduled successfully",
		"appointment": appointment,
	})
}
