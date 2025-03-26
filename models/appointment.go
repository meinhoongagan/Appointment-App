package models

import (
	"time"

	"gorm.io/gorm"
)

type AppointmentStatus string

const (
	StatusPending   AppointmentStatus = "pending"
	StatusConfirmed AppointmentStatus = "confirmed"
	StatusCanceled  AppointmentStatus = "canceled"
	StatusCompleted AppointmentStatus = "completed"
)

type Appointment struct {
	gorm.Model
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Status       AppointmentStatus `json:"status"`
	IsRecurring  bool              `json:"is_recurring"`
	RecurPattern string            `json:"recur_pattern"` // E.g., "weekly", "monthly", "daily"
	ServiceID    uint              `json:"service_id"`
	Service      Service           `json:"service" gorm:"foreignKey:ServiceID"`
	ProviderID   uint              `json:"provider_id"`
	Provider     User              `json:"provider" gorm:"foreignKey:ProviderID"`
	CustomerID   uint              `json:"customer_id"`
	Customer     User              `json:"customer" gorm:"foreignKey:CustomerID"`
}

func (a *Appointment) BeforeCreate(tx *gorm.DB) error {
	if a.Status == "" {
		a.Status = StatusPending
	}
	return nil
}
