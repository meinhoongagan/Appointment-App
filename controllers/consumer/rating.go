package consumer

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"gorm.io/gorm"
)

// CreateReview creates a new review for a provider
// @Summary Create a review
// @Description Creates a new review for a provider
// @Tags Reviews
// @Accept json
// @Produce json
// @Param review body object{Rating=number,Comment=string,ProviderID=uint,ServiceID=uint,IsAnonymous=bool,AppointmentID=uint} true "Review details"
// @Success 201 {object} object{ID=uint,Rating=number,Comment=string,ProviderID=uint,CustomerID=uint,ServiceID=uint,IsAnonymous=bool,IsVerified=bool,AppointmentID=uint,CreatedAt=string,UpdatedAt=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 409 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security BearerAuth
// @Router /reviews [post]
func CreateReview(c *fiber.Ctx) error {
	reviewChan := make(chan *models.Review)

	go func() {
		userIDVal := c.Locals("userID")
		userID, ok := userIDVal.(uint)
		if !ok {
			reviewChan <- nil
			return
		}

		review := new(models.Review)
		if err := c.BodyParser(review); err != nil {
			reviewChan <- nil
			return
		}

		review.CustomerID = userID

		var provider models.User
		if err := db.DB.First(&provider, review.ProviderID).Error; err != nil {
			reviewChan <- nil
			return
		}

		var service models.Service
		if err := db.DB.Where("id = ? AND provider_id = ?", review.ServiceID, review.ProviderID).First(&service).Error; err != nil {
			reviewChan <- nil
			return
		}

		hasExisting, err := review.HasExistingReview(db.DB)
		if err != nil {
			reviewChan <- nil
			return
		}

		if hasExisting {
			reviewChan <- nil
			return
		}

		if review.AppointmentID != nil && *review.AppointmentID > 0 {
			var appointment models.Appointment
			if err := db.DB.Where("id = ? AND customer_id = ? AND provider_id = ? AND service_id = ?",
				*review.AppointmentID, userID, review.ProviderID, review.ServiceID).
				First(&appointment).Error; err != nil {
				reviewChan <- nil
				return
			}

			review.IsVerified = true
		}

		if err := db.DB.Create(review).Error; err != nil {
			reviewChan <- nil
			return
		}

		reviewChan <- review
	}()

	review := <-reviewChan
	if review == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create review",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(review)
}

// GetProviderReviews retrieves reviews for a provider
// @Summary Get provider reviews
// @Description Retrieves a paginated list of reviews for a specific provider
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {object} object{reviews=[]object{ID=uint,Rating=number,Comment=string,ProviderID=uint,Customer=object{ID=uint,Name=string,CreatedAt=string},Service=object{ID=uint,Name=string},IsAnonymous=bool,IsVerified=bool,AppointmentID=uint,CreatedAt=string,UpdatedAt=string},total=int,page=int,limit=int,pages=int}
// @Failure 500 {object} object{error=string}
// @Router /providers/{id}/reviews [get]
func GetProviderReviews(c *fiber.Ctx) error {
	providerID := c.Params("id")

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	offset := (page - 1) * limit

	var reviews []models.Review
	var count int64

	// Create two goroutines to fetch reviews and count concurrently
	reviewsChan := make(chan []models.Review)
	countChan := make(chan int64)

	go func() {
		if err := db.DB.Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, created_at")
		}).
			Preload("Service", "name").
			Where("provider_id = ?", providerID).
			Order("created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&reviews).Error; err != nil {
			reviewsChan <- []models.Review{}
			return
		}
		reviewsChan <- reviews
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ?", providerID).Count(&count)
		countChan <- count
	}()

	// Wait for both goroutines to finish
	reviews = <-reviewsChan
	count = <-countChan

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
// @Summary Update a review
// @Description Updates a review by its ID
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path string true "Review ID"
// @Param review body object{Rating=number,Comment=string,IsAnonymous=bool} true "Updated review details"
// @Success 200 {object} object{ID=uint,Rating=number,Comment=string,ProviderID=uint,CustomerID=uint,ServiceID=uint,IsAnonymous=bool,IsVerified=bool,AppointmentID=uint,CreatedAt=string,UpdatedAt=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security BearerAuth
// @Router /reviews/{id} [put]
func UpdateReview(c *fiber.Ctx) error {
	userIDVal := c.Locals("userID")
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	reviewID := c.Params("id")

	var existingReview models.Review
	var updateMap map[string]interface{}

	// Create two goroutines to fetch review and parse update data concurrently
	reviewChan := make(chan models.Review)
	updateDataChan := make(chan map[string]interface{})

	go func() {
		if err := db.DB.First(&existingReview, reviewID).Error; err != nil {
			reviewChan <- models.Review{}
			return
		}
		reviewChan <- existingReview
	}()

	go func() {
		updateData := make(map[string]interface{})
		if err := c.BodyParser(&updateData); err != nil {
			updateDataChan <- map[string]interface{}{}
			return
		}

		allowedFields := map[string]bool{
			"rating":       true,
			"comment":      true,
			"is_anonymous": true,
		}

		updateMap = make(map[string]interface{})
		for key, value := range updateData {
			if allowedFields[key] {
				if key == "rating" {
					rating, ok := value.(float64)
					if !ok {
						if strRating, ok := value.(string); ok {
							parsedRating, err := strconv.ParseFloat(strRating, 64)
							if err == nil {
								rating = parsedRating
							}
						}
					}

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

		updateDataChan <- updateMap
	}()

	// Wait for both goroutines to finish
	existingReview = <-reviewChan
	updateMap = <-updateDataChan

	if existingReview.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Review not found",
		})
	}

	if existingReview.CustomerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You don't have permission to update this review",
		})
	}

	if err := db.DB.Model(&existingReview).Updates(updateMap).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update review",
		})
	}

	return c.JSON(existingReview)
}

// DeleteReview deletes a review
// @Summary Delete a review
// @Description Deletes a review by its ID
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path string true "Review ID"
// @Success 204
// @Failure 401 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Security BearerAuth
// @Router /reviews/{id} [delete]
func DeleteReview(c *fiber.Ctx) error {
	userIDVal := c.Locals("userID")
	userID, ok := userIDVal.(uint)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	reviewID := c.Params("id")

	var existingReview models.Review
	if err := db.DB.First(&existingReview, reviewID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Review not found",
		})
	}

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

	if err := db.DB.Delete(&existingReview).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete review",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetProviderReviewStats retrieves review statistics for a provider
// @Summary Get provider review statistics
// @Description Retrieves review statistics for a specific provider
// @Tags Reviews
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} object{ProviderID=uint,TotalReviews=int,AvgRating=number,Rating5Count=int,Rating4Count=int,Rating3Count=int,Rating2Count=int,Rating1Count=int}
// @Failure 500 {object} object{error=string}
// @Router /providers/{id}/review-stats [get]
func GetProviderReviewStats(c *fiber.Ctx) error {
	providerID := c.Params("id")

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

	// Create 6 goroutines to fetch different stats concurrently
	totalReviewsChan := make(chan int64)
	avgRatingChan := make(chan float64)
	rating5CountChan := make(chan int64)
	rating4CountChan := make(chan int64)
	rating3CountChan := make(chan int64)
	rating2CountChan := make(chan int64)
	rating1CountChan := make(chan int64)

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ?", providerID).Count(&stats.TotalReviews)
		totalReviewsChan <- stats.TotalReviews
	}()

	go func() {
		var avgResult struct {
			AvgRating float64
		}
		db.DB.Model(&models.Review{}).
			Select("COALESCE(AVG(rating), 0) as avg_rating").
			Where("provider_id = ?", providerID).
			Scan(&avgResult)
		avgRatingChan <- avgResult.AvgRating
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 4.5 AND rating <= 5.0", providerID).Count(&stats.Rating5Count)
		rating5CountChan <- stats.Rating5Count
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 3.5 AND rating < 4.5", providerID).Count(&stats.Rating4Count)
		rating4CountChan <- stats.Rating4Count
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 2.5 AND rating < 3.5", providerID).Count(&stats.Rating3Count)
		rating3CountChan <- stats.Rating3Count
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 1.5 AND rating < 2.5", providerID).Count(&stats.Rating2Count)
		rating2CountChan <- stats.Rating2Count
	}()

	go func() {
		db.DB.Model(&models.Review{}).Where("provider_id = ? AND rating >= 1.0 AND rating < 1.5", providerID).Count(&stats.Rating1Count)
		rating1CountChan <- stats.Rating1Count
	}()

	// Wait for all goroutines to finish
	stats.TotalReviews = <-totalReviewsChan
	stats.AvgRating = <-avgRatingChan
	stats.Rating5Count = <-rating5CountChan
	stats.Rating4Count = <-rating4CountChan
	stats.Rating3Count = <-rating3CountChan
	stats.Rating2Count = <-rating2CountChan
	stats.Rating1Count = <-rating1CountChan

	return c.JSON(stats)
}
