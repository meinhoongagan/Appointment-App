package service

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// GetDashboardOverview returns overview statistics for the dashboard
// @Summary Get dashboard overview
// @Description Retrieve overview statistics for the authenticated user's dashboard, including total appointments, status counts, services, and revenue
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{total_appointments=int64,pending_count=int64,confirmed_count=int64,completed_count=int64,canceled_count=int64,total_services=int64,total_revenue=float64,last_updated=string} "Dashboard overview statistics"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/dashboard/overview [get]
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

	type Stats struct {
		TotalAppointments         int64     `json:"total_appointments"`
		TotalAppointmentsPrevious int64     `json:"total_appointments_previous"`
		PendingCount              int64     `json:"pending_count"`
		PendingCountPrevious      int64     `json:"pending_count_previous"`
		ConfirmedCount            int64     `json:"confirmed_count"`
		ConfirmedCountPrevious    int64     `json:"confirmed_count_previous"`
		CompletedCount            int64     `json:"completed_count"`
		CompletedCountPrevious    int64     `json:"completed_count_previous"`
		CanceledCount             int64     `json:"canceled_count"`
		CanceledCountPrevious     int64     `json:"canceled_count_previous"`
		TotalServices             int64     `json:"total_services"`
		TotalServicesPrevious     int64     `json:"total_services_previous"`
		TotalRevenue              float64   `json:"total_revenue"`
		TotalRevenuePrevious      float64   `json:"total_revenue_previous"`
		LastUpdated               time.Time `json:"last_updated"`
	}

	type statsResult struct {
		Stats Stats
		Err   error
	}

	resultChan := make(chan statsResult)

	go func() {
		var result statsResult
		var statistics Stats

		// Date ranges
		now := time.Now()
		startCurrent := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startPrevious := startCurrent.AddDate(0, -1, 0)
		endPrevious := startCurrent.Add(-time.Nanosecond)

		// Base queries
		appointmentQuery := db.DB.Model(&models.Appointment{})
		serviceQuery := db.DB.Model(&models.Service{}).Where("provider_id = ?", userID)

		if role == "provider" {
			appointmentQuery = appointmentQuery.Where("provider_id = ?", userID)
			serviceQuery = serviceQuery.Where("provider_id = ?", userID)
		} else if role != "admin" {
			appointmentQuery = appointmentQuery.Where("customer_id = ?", userID)
		}

		// --- Current period ---
		appointmentQuery.Where("created_at >= ?", startCurrent).Count(&statistics.TotalAppointments)
		appointmentQuery.Where("status = ? AND created_at >= ?", models.StatusPending, startCurrent).Count(&statistics.PendingCount)
		appointmentQuery.Where("status = ? AND created_at >= ?", models.StatusConfirmed, startCurrent).Count(&statistics.ConfirmedCount)
		appointmentQuery.Where("status = ? AND created_at >= ?", models.StatusCompleted, startCurrent).Count(&statistics.CompletedCount)
		appointmentQuery.Where("status = ? AND created_at >= ?", models.StatusCanceled, startCurrent).Count(&statistics.CanceledCount)
		serviceQuery.Where("created_at >= ?", startCurrent).Count(&statistics.TotalServices)

		// Revenue (current)
		type RevenueResult struct{ TotalRevenue float64 }
		var revenueResult RevenueResult
		revenueQuery := db.DB.Table("appointments").
			Joins("JOIN services ON appointments.service_id = services.id").
			Where("appointments.status = ?", models.StatusCompleted).
			Where("appointments.created_at >= ?", startCurrent)
		if role == "provider" {
			revenueQuery = revenueQuery.Where("appointments.provider_id = ?", userID)
		} else if role != "admin" {
			revenueQuery = revenueQuery.Where("appointments.customer_id = ?", userID)
		}
		revenueQuery.Select("SUM(services.cost) as total_revenue").Scan(&revenueResult)
		statistics.TotalRevenue = revenueResult.TotalRevenue

		// --- Previous period ---
		appointmentQueryPrev := db.DB.Model(&models.Appointment{})
		serviceQueryPrev := db.DB.Model(&models.Service{}).Where("provider_id = ?", userID)
		if role == "provider" {
			appointmentQueryPrev = appointmentQueryPrev.Where("provider_id = ?", userID)
			serviceQueryPrev = serviceQueryPrev.Where("provider_id = ?", userID)
		} else if role != "admin" {
			appointmentQueryPrev = appointmentQueryPrev.Where("customer_id = ?", userID)
		}
		appointmentQueryPrev.Where("created_at >= ? AND created_at <= ?", startPrevious, endPrevious).Count(&statistics.TotalAppointmentsPrevious)
		appointmentQueryPrev.Where("status = ? AND created_at >= ? AND created_at <= ?", models.StatusPending, startPrevious, endPrevious).Count(&statistics.PendingCountPrevious)
		appointmentQueryPrev.Where("status = ? AND created_at >= ? AND created_at <= ?", models.StatusConfirmed, startPrevious, endPrevious).Count(&statistics.ConfirmedCountPrevious)
		appointmentQueryPrev.Where("status = ? AND created_at >= ? AND created_at <= ?", models.StatusCompleted, startPrevious, endPrevious).Count(&statistics.CompletedCountPrevious)
		appointmentQueryPrev.Where("status = ? AND created_at >= ? AND created_at <= ?", models.StatusCanceled, startPrevious, endPrevious).Count(&statistics.CanceledCountPrevious)
		serviceQueryPrev.Where("created_at >= ? AND created_at <= ?", startPrevious, endPrevious).Count(&statistics.TotalServicesPrevious)

		// Revenue (previous)
		var revenueResultPrev RevenueResult
		revenueQueryPrev := db.DB.Table("appointments").
			Joins("JOIN services ON appointments.service_id = services.id").
			Where("appointments.status = ?", models.StatusCompleted).
			Where("appointments.created_at >= ? AND appointments.created_at <= ?", startPrevious, endPrevious)
		if role == "provider" {
			revenueQueryPrev = revenueQueryPrev.Where("appointments.provider_id = ?", userID)
		} else if role != "admin" {
			revenueQueryPrev = revenueQueryPrev.Where("appointments.customer_id = ?", userID)
		}
		revenueQueryPrev.Select("SUM(services.cost) as total_revenue").Scan(&revenueResultPrev)
		statistics.TotalRevenuePrevious = revenueResultPrev.TotalRevenue

		// Set last updated time
		statistics.LastUpdated = time.Now()

		result.Stats = statistics
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(result.Stats)
}

// GetRecentAppointments returns the most recent appointments
// @Summary Get recent appointments
// @Description Retrieve a list of recent appointments for the authenticated user, filtered by role
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of appointments to return (default 5)"
// @Success 200 {array} models.Appointment "List of recent appointments"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/dashboard/recent-appointments [get]
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

	limit := 5 // Default limit

	// Check if limit is provided in query params
	if c.Query("limit") != "" {
		parsedLimit := c.QueryInt("limit")
		if parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	type appointmentsResult struct {
		Appointments []models.Appointment
		Err          error
	}

	resultChan := make(chan appointmentsResult)

	go func() {
		var result appointmentsResult
		var appointments []models.Appointment

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
			result.Err = err
			resultChan <- result
			return
		}

		result.Appointments = appointments
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(result.Appointments)
}

// GetRevenueSummary returns revenue statistics
// @Summary Get revenue summary
// @Description Retrieve revenue statistics for the authenticated user, filtered by role and time range
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param range query string false "Time range for revenue data (day, week, month, year; default week)"
// @Success 200 {object} object{data=[]object{date=string,revenue=float64,count=int,services=int},summary=object{total_revenue=float64,total_appointments=int,time_range=string,start_date=string,end_date=string}} "Revenue summary"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /provider/dashboard/revenue [get]
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

	type revenueResult struct {
		Data     []RevenueData
		Summary  fiber.Map
		Err      error
	}

	resultChan := make(chan revenueResult)

	go func() {
		var result revenueResult

		var queryResult []struct {
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
		if err := db.DB.Raw(query, params...).Scan(&queryResult).Error; err != nil {
			result.Err = err
			resultChan <- result
			return
		}

		// Calculate totals
		var totalRevenue float64
		var totalAppointments int
		revenueData := make([]RevenueData, 0)

		for _, r := range queryResult {
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
		result.Data = revenueData
		result.Summary = fiber.Map{
			"total_revenue":      totalRevenue,
			"total_appointments": totalAppointments,
			"time_range":         timeRange,
			"start_date":         startDate.Format("2006-01-02"),
			"end_date":           endDate.Format("2006-01-02"),
		}

		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	response := fiber.Map{
		"data":    result.Data,
		"summary": result.Summary,
	}

	return c.JSON(response)
}

// GetQuickActions returns available quick actions for the dashboard
// @Summary Get quick actions
// @Description Retrieve a list of quick actions available for the authenticated user's dashboard, based on their role
// @Tags dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{quick_actions=[]object{id=string,title=string,description=string,icon=string,url=string,color=string},user_id=uint,role=string} "Quick actions list"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token or role"
// @Router /provider/dashboard/quick-actions [get]
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

	type actionsResult struct {
		QuickActions []map[string]interface{}
		Err          error
	}

	resultChan := make(chan actionsResult)

	go func() {
		var result actionsResult
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
					"id":          "receptionists",
					"title":       "Manage receptionists",
					"description": "View and edit receptionists accounts",
					"icon":        "users",
					"url":         "/service-dashboard/receptionists",
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
				map[string]interface{}{
					"id":          "manage_receptionist",
					"title":       "Manage Receptionist",
					"description": "Add or manage your receptionists",
					"icon":        "users",
					"url":         "/service-dashboard/receptionists",
					"color":       "purple",
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

		result.QuickActions = quickActions
		resultChan <- result
	}()

	result := <-resultChan
	if result.Err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"quick_actions": result.QuickActions,
		"user_id":       userID,
		"role":          role,
	})
}
