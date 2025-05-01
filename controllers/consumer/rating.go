package consumer

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"gorm.io/gorm"
)

// CreateReview adds a new review for a provider
func CreateReview(c *fiber.Ctx) error {
	// Get the authenticated user ID
	userIDVal := c.Locals("userID")
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Parse the review data
	review := new(models.Review)
	if err := c.BodyParser(review); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid review data",
		})
	}

	// Set the customer ID to the authenticated user
	review.CustomerID = userID

	// Check if the provider exists
	var provider models.User
	if err := db.DB.First(&provider, review.ProviderID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Check if the service exists and belongs to the provider
	var service models.Service
	if err := db.DB.Where("id = ? AND provider_id = ?", review.ServiceID, review.ProviderID).First(&service).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found or does not belong to this provider",
		})
	}

	// Check if the user has already reviewed this provider/service
	hasExisting, err := review.HasExistingReview(db.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing reviews",
		})
	}

	if hasExisting {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "You have already reviewed this service. Please update your existing review.",
		})
	}

	// If appointmentID is provided, verify it exists and belongs to the customer
	if review.AppointmentID != nil && *review.AppointmentID > 0 {
		var appointment models.Appointment
		if err := db.DB.Where("id = ? AND customer_id = ? AND provider_id = ? AND service_id = ?",
			*review.AppointmentID, userID, review.ProviderID, review.ServiceID).
			First(&appointment).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Appointment not found or does not match the review details",
			})
		}

		// Mark as verified since it's linked to a real appointment
		review.IsVerified = true
	}

	// Create the review
	if err := db.DB.Create(review).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create review",
		})
	}

	// Return the created review
	return c.Status(fiber.StatusCreated).JSON(review)
}

// GetProviderReviews retrieves all reviews for a specific provider
func GetProviderReviews(c *fiber.Ctx) error {
	// Get provider ID from URL parameters
	providerID := c.Params("id")

	// Get pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Calculate offset
	offset := (page - 1) * limit

	// Get reviews for the provider
	var reviews []models.Review
	if err := db.DB.Preload("Customer", func(db *gorm.DB) *gorm.DB {
		// Only select non-sensitive fields
		return db.Select("id, name, created_at")
	}).
		Preload("Service", "name").
		Where("provider_id = ?", providerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch reviews",
		})
	}

	// Count total reviews for pagination
	var count int64
	db.DB.Model(&models.Review{}).Where("provider_id = ?", providerID).Count(&count)

	// Handle anonymous reviews - hide customer information
	for i := range reviews {
		if reviews[i].IsAnonymous {
			reviews[i].Customer.Name = "Anonymous User"
		}
	}

	return c.JSON(fiber.Map{
		"reviews": reviews,
		"total":   count,
		"page":    page,
		"limit":   limit,
		"pages":   (int(count) + limit - 1) / limit,
	})
}

// UpdateReview updates an existing review
func UpdateReview(c *fiber.Ctx) error {
	// Get the authenticated user ID
	userIDVal := c.Locals("userID")
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get review ID from URL parameters
	reviewID := c.Params("id")

	// Find the existing review
	var existingReview models.Review
	if err := db.DB.First(&existingReview, reviewID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Review not found",
		})
	}

	// Check if the authenticated user is the review owner
	if existingReview.CustomerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to update this review",
		})
	}

	// Parse the updated review data
	updateData := make(map[string]interface{})
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid review data",
		})
	}

	// Only allow certain fields to be updated
	allowedFields := map[string]bool{
		"rating":       true,
		"comment":      true,
		"is_anonymous": true,
	}

	// Filter out fields that shouldn't be updated
	updateMap := make(map[string]interface{})
	for key, value := range updateData {
		if allowedFields[key] {
			// Ensure rating is between 1.0 and 5.0
			if key == "rating" {
				rating, ok := value.(float64)
				if !ok {
					// Try to convert from JSON number or string
					if strRating, ok := value.(string); ok {
						parsedRating, err := strconv.ParseFloat(strRating, 64)
						if err == nil {
							rating = parsedRating
						}
					}
				}

				// Validate range
				if rating < 1.0 {
					rating = 1.0
				} else if rating > 5.0 {
					rating = 5.0
				}

				updateMap[key] = rating
			} else {
				updateMap[key] = value
			}
		}
	}

	// Perform the update
	if err := db.DB.Model(&existingReview).Updates(updateMap).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update review",
		})
	}

	// Return the updated review
	return c.JSON(existingReview)
}

// DeleteReview removes a review
func DeleteReview(c *fiber.Ctx) error {
	// Get the authenticated user ID
	userIDVal := c.Locals("userID")
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Get review ID from URL parameters
	reviewID := c.Params("id")

	// Find the existing review
	var existingReview models.Review
	if err := db.DB.First(&existingReview, reviewID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Review not found",
		})
	}

	// Check if the authenticated user is the review owner or an admin (check role)
	var user models.User
	if err := db.DB.Preload("Role").First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user details",
		})
	}

	isAdmin := user.Role.Name == "admin"
	if existingReview.CustomerID != userID && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to delete this review",
		})
	}

	// Delete the review (soft delete since using gorm.Model)
	if err := db.DB.Delete(&existingReview).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete review",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetProviderReviewStats retrieves review statistics for a provider
func GetProviderReviewStats(c *fiber.Ctx) error {
	// Get provider ID from URL parameters
	providerID := c.Params("id")

	// Create a struct to hold stats
	type ReviewStats struct {
		ProviderID   uint    `json:"provider_id"`
		TotalReviews int64   `json:"total_reviews"`
		AvgRating    float64 `json:"average_rating"`
		Rating5Count int64   `json:"rating_5_count"`
		Rating4Count int64   `json:"rating_4_count"`
		Rating3Count int64   `json:"rating_3_count"`
		Rating2Count int64   `json:"rating_2_count"`
		Rating1Count int64   `json:"rating_1_count"`
	}

	providerIDUint, _ := strconv.ParseUint(providerID, 10, 32)
	stats := ReviewStats{
		ProviderID: uint(providerIDUint),
	}

	// Count total reviews
	db.DB.Model(&models.Review{}).Where("provider_id = ?", providerID).Count(&stats.TotalReviews)

	// Get average rating
	var avgResult struct {
		AvgRating float64
	}
	db.DB.Model(&models.Review{}).
		Select("COALESCE(AVG(rating), 0) as avg_rating").
		Where("provider_id = ?", providerID).
		Scan(&avgResult)

	stats.AvgRating = avgResult.AvgRating

	// Count reviews by rating
	db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 4.5 AND rating <= 5.0", providerID).Count(&stats.Rating5Count)
	db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 3.5 AND rating < 4.5", providerID).Count(&stats.Rating4Count)
	db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 2.5 AND rating < 3.5", providerID).Count(&stats.Rating3Count)
	db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 1.5 AND rating < 2.5", providerID).Count(&stats.Rating2Count)
	db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 1.0 AND rating < 1.5", providerID).Count(&stats.Rating1Count)

	return c.JSON(stats)
}
