package service

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

func GetDashboardOverview(c *fiber.Ctx) error {
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

	var statistics struct {
		TotalAppointments int64     `json:"total_appointments"`
		PendingCount      int64     `json:"pending_count"`
		ConfirmedCount    int64     `json:"confirmed_count"`
		CompletedCount    int64     `json:"completed_count"`
		CanceledCount     int64     `json:"canceled_count"`
		TotalServices     int64     `json:"total_services"`
		TotalRevenue      float64   `json:"total_revenue"`
		LastUpdated       time.Time `json:"last_updated"`
	}

	// Base query - will be modified based on role
	appointmentQuery := db.DB.Model(&models.Appointment{})
	serviceQuery := db.DB.Model(&models.Service{})

	// Filter queries based on user role
	if role == "provider" {
		// If provider, only show their appointments and services
		appointmentQuery = appointmentQuery.Where("provider_id = ?", userID)
		serviceQuery = serviceQuery.Where("provider_id = ?", userID)
	} else if role != "admin" {
		// If client, only show their appointments
		appointmentQuery = appointmentQuery.Where("customer_id = ?", userID)
	}
	// Admin sees all data, so no additional filtering needed

	// Get total appointments
	appointmentQuery.Count(&statistics.TotalAppointments)

	// Get appointments by status
	appointmentQuery.Where("status = ?", models.StatusPending).Count(&statistics.PendingCount)
	appointmentQuery.Where("status = ?", models.StatusConfirmed).Count(&statistics.ConfirmedCount)
	appointmentQuery.Where("status = ?", models.StatusCompleted).Count(&statistics.CompletedCount)
	appointmentQuery.Where("status = ?", models.StatusCanceled).Count(&statistics.CanceledCount)

	// Get total services
	serviceQuery.Count(&statistics.TotalServices)

	// Calculate total revenue (from completed appointments)
	type RevenueResult struct {
		TotalRevenue float64
	}
	var revenueResult RevenueResult

	// Revenue query
	revenueQuery := db.DB.Table("appointments").
		Joins("JOIN services ON appointments.service_id = services.id").
		Where("appointments.status = ?", models.StatusCompleted)

	// Filter by provider if needed
	if role == "provider" {
		revenueQuery = revenueQuery.Where("appointments.provider_id = ?", userID)
	} else if role != "admin" {
		revenueQuery = revenueQuery.Where("appointments.customer_id = ?", userID)
	}

	revenueQuery.Select("SUM(services.cost) as total_revenue").Scan(&revenueResult)
	statistics.TotalRevenue = revenueResult.TotalRevenue

	// Set last updated time
	statistics.LastUpdated = time.Now()

	return c.JSON(statistics)
}

// GetRecentAppointments returns the most recent appointments
func GetRecentAppointments(c *fiber.Ctx) error {
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

	var appointments []models.Appointment
	limit := 5 // Default limit

	// Check if limit is provided in query params
	if c.Query("limit") != "" {
		parsedLimit := c.QueryInt("limit")
		if parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Base query
	query := db.DB.
		Preload("Service").
		Preload("Provider").
		Preload("Customer")

	// Filter by user role and ID
	if role == "provider" {
		query = query.Where("provider_id = ?", userID)
	} else if role != "admin" { // Assuming non-admin, non-provider is a client
		query = query.Where("customer_id = ?", userID)
	}

	// Get recent appointments with preloaded relations
	if err := query.
		Order("created_at desc").
		Limit(limit).
		Find(&appointments).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(appointments)
}

// GetRevenueSummary returns revenue statistics
func GetRevenueSummary(c *fiber.Ctx) error {
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

	// Get time range from query params with defaults
	timeRange := c.Query("range", "week") // Default to week if not specified
	var startDate, endDate time.Time
	now := time.Now()

	// Set time range based on parameter
	switch timeRange {
	case "day":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = now
	case "week":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "month":
		startDate = now.AddDate(0, -1, 0)
		endDate = now
	case "year":
		startDate = now.AddDate(-1, 0, 0)
		endDate = now
	default:
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	}

	// Structure to hold revenue data
	type RevenueData struct {
		Date     string  `json:"date"`
		Revenue  float64 `json:"revenue"`
		Count    int     `json:"count"`
		Services int     `json:"services"`
	}

	var result []struct {
		Date    time.Time
		Revenue float64
		Count   int
	}

	// Base query
	query := `
		SELECT 
			DATE(appointments.start_time) as date,
			SUM(services.cost) as revenue,
			COUNT(*) as count
		FROM 
			appointments
		JOIN 
			services ON appointments.service_id = services.id
		WHERE 
			appointments.status = 'completed' AND
			appointments.start_time BETWEEN ? AND ?
	`

	// Add role-based filtering
	params := []interface{}{startDate, endDate}

	if role == "provider" {
		query += " AND appointments.provider_id = ?"
		params = append(params, userID)
	} else if role != "admin" {
		query += " AND appointments.customer_id = ?"
		params = append(params, userID)
	}

	// Finish the query
	query += `
		GROUP BY 
			DATE(appointments.start_time)
		ORDER BY 
			date ASC
	`

	// Execute query to get revenue data grouped by day
	if err := db.DB.Raw(query, params...).Scan(&result).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Calculate totals
	var totalRevenue float64
	var totalAppointments int
	revenueData := make([]RevenueData, 0)

	for _, r := range result {
		// Format date as string
		dateStr := r.Date.Format("2006-01-02")

		// Base query for services count
		servicesQuery := `
			SELECT 
				COUNT(DISTINCT service_id) 
			FROM 
				appointments 
			WHERE 
				DATE(start_time) = ? AND
				status = 'completed'
		`

		// Add role-based filtering for services count
		servicesParams := []interface{}{dateStr}

		if role == "provider" {
			servicesQuery += " AND provider_id = ?"
			servicesParams = append(servicesParams, userID)
		} else if role != "admin" {
			servicesQuery += " AND customer_id = ?"
			servicesParams = append(servicesParams, userID)
		}

		// Get services count for this date
		var servicesCount int
		db.DB.Raw(servicesQuery, servicesParams...).Scan(&servicesCount)

		// Add to revenue data array
		revenueData = append(revenueData, RevenueData{
			Date:     dateStr,
			Revenue:  r.Revenue,
			Count:    r.Count,
			Services: servicesCount,
		})

		// Add to totals
		totalRevenue += r.Revenue
		totalAppointments += r.Count
	}

	// Create response structure
	response := fiber.Map{
		"data": revenueData,
		"summary": fiber.Map{
			"total_revenue":      totalRevenue,
			"total_appointments": totalAppointments,
			"time_range":         timeRange,
			"start_date":         startDate.Format("2006-01-02"),
			"end_date":           endDate.Format("2006-01-02"),
		},
	}

	return c.JSON(response)
}

// GetQuickActions returns available quick actions for the dashboard
func GetQuickActions(c *fiber.Ctx) error {
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

	// Define quick actions based on user role
	var quickActions []map[string]interface{}

	// Common actions for all roles
	quickActions = append(quickActions, map[string]interface{}{
		"id":          "view_calendar",
		"title":       "View Calendar",
		"description": "Check upcoming appointments",
		"icon":        "calendar",
		"url":         "/calendar",
		"color":       "blue",
	})

	// Role-specific actions
	switch role {
	case "admin":
		quickActions = append(quickActions,
			map[string]interface{}{
				"id":          "create_service",
				"title":       "Add New Service",
				"description": "Create a new service offering",
				"icon":        "add",
				"url":         "/services/new",
				"color":       "green",
			},
			map[string]interface{}{
				"id":          "manage_users",
				"title":       "Manage Users",
				"description": "View and edit user accounts",
				"icon":        "users",
				"url":         "/users",
				"color":       "purple",
			},
			map[string]interface{}{
				"id":          "view_reports",
				"title":       "View Reports",
				"description": "Access detailed business reports",
				"icon":        "chart",
				"url":         "/reports",
				"color":       "orange",
			},
		)
	case "provider":
		quickActions = append(quickActions,
			map[string]interface{}{
				"id":          "manage_schedule",
				"title":       "Manage Schedule",
				"description": "Update your availability",
				"icon":        "clock",
				"url":         "/schedule",
				"color":       "indigo",
			},
			map[string]interface{}{
				"id":          "upcoming_appointments",
				"title":       "Today's Appointments",
				"description": "View appointments for today",
				"icon":        "list",
				"url":         "/appointments/today",
				"color":       "teal",
			},
		)
	default: // client or other roles
		quickActions = append(quickActions,
			map[string]interface{}{
				"id":          "book_appointment",
				"title":       "Book Appointment",
				"description": "Schedule a new appointment",
				"icon":        "plus",
				"url":         "/appointments/new",
				"color":       "emerald",
			},
			map[string]interface{}{
				"id":          "my_appointments",
				"title":       "My Appointments",
				"description": "View your upcoming appointments",
				"icon":        "clipboard",
				"url":         "/appointments/mine",
				"color":       "amber",
			},
		)
	}

	return c.JSON(fiber.Map{
		"quick_actions": quickActions,
		"user_id":       userID,
		"role":          role,
	})
}
