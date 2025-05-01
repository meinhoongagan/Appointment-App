package models

import (
	"gorm.io/gorm"
)

type Review struct {
	gorm.Model
	Rating        float64 `json:"rating" gorm:"type:decimal(2,1);not null"`
	Comment       string  `json:"comment"`
	ProviderID    uint    `json:"provider_id"`
	Provider      User    `json:"provider" gorm:"foreignKey:ProviderID"`
	CustomerID    uint    `json:"customer_id"`
	Customer      User    `json:"customer" gorm:"foreignKey:CustomerID"`
	ServiceID     uint    `json:"service_id"`
	Service       Service `json:"service" gorm:"foreignKey:ServiceID"`
	IsAnonymous   bool    `json:"is_anonymous" gorm:"default:false"`
	IsVerified    bool    `json:"is_verified" gorm:"default:false"` // Indicates if this review is from a verified appointment
	AppointmentID *uint   `json:"appointment_id"`                   // Optional link to appointment
}

// BeforeCreate hook to validate rating
func (r *Review) BeforeCreate(tx *gorm.DB) error {
	// Ensure rating is between 1.0 and 5.0
	if r.Rating < 1.0 {
		r.Rating = 1.0
	} else if r.Rating > 5.0 {
		r.Rating = 5.0
	}

	return nil
}

// Check if customer has already reviewed this provider
func (r *Review) HasExistingReview(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Model(&Review{}).
		Where("customer_id = ? AND provider_id = ? AND service_id = ? AND deleted_at IS NULL",
			r.CustomerID, r.ProviderID, r.ServiceID).
		Count(&count).Error

	return count > 0, err
}
