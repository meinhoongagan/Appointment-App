package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AppointmentStatus string

type Recurrence struct {
	gorm.Model
	AppointmentID uint      `json:"appointment_id"`
	NextRun       time.Time `json:"next_run"`
	Frequency     string    `json:"frequency"` // "daily", "weekly", "monthly"
	EndAfter      uint      `json:"end_after"` // Number of occurrences
}

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
	RecurrenceID uint              `json:"recur_pattern_id"`
	RecurPattern Recurrence        `json:"recur_pattern"` // E.g., "weekly", "monthly", "daily"
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

func (a *Appointment) UpdateStatus(tx *gorm.DB, newStatus AppointmentStatus) error {
	switch a.Status {
	case StatusPending:
		if newStatus != StatusConfirmed && newStatus != StatusCanceled {
			return fmt.Errorf("invalid transition from pending to %s", newStatus)
		}
	case StatusConfirmed:
		if newStatus != StatusCompleted && newStatus != StatusCanceled {
			return fmt.Errorf("invalid transition from confirmed to %s", newStatus)
		}
	case StatusCompleted, StatusCanceled:
		return fmt.Errorf("no transitions allowed from %s", a.Status)
	}

	// Update the status
	a.Status = newStatus
	if err := tx.Save(a).Error; err != nil {
		return err
	}

	// Handle Recurrence after completion
	if newStatus == StatusCompleted && a.IsRecurring {
		fmt.Println("Scheduling next recurrence...", a.RecurPattern)

		// Preload Recurrence Pattern before scheduling next occurrence
		if err := tx.Preload("RecurPattern").First(&a, a.ID).Error; err != nil {
			return fmt.Errorf("failed to load recurrence pattern: %v", err)
		}

		return a.ScheduleNextRecurrence(tx)
	}

	return nil
}

func (a *Appointment) ScheduleNextRecurrence(tx *gorm.DB) error {
	var nextTime time.Time

	// Check if recurrence exists
	if a.RecurPattern.ID == 0 {
		return fmt.Errorf("no recurrence pattern found for appointment")
	}
	fmt.Println("Recurrence pattern found:", a.RecurPattern)
	// Determine next occurrence based on recurrence frequency
	switch a.RecurPattern.Frequency {
	case "daily":
		nextTime = a.StartTime.AddDate(0, 0, 1) // Add 1 day
	case "weekly":
		nextTime = a.StartTime.AddDate(0, 0, 7) // Add 7 days
	case "monthly":
		nextTime = a.StartTime.AddDate(0, 1, 0) // Add 1 month
	default:
		return fmt.Errorf("invalid recurrence frequency: %s", a.RecurPattern.Frequency)
	}

	fmt.Println("Next occurrence time:", nextTime)

	// Decrement remaining occurrences if EndAfter > 0
	if a.RecurPattern.EndAfter > 0 {
		a.RecurPattern.EndAfter--
		if a.RecurPattern.EndAfter == 0 {
			return nil // Stop recurrence if occurrences are exhausted
		}
		// Update the Recurrence with decremented EndAfter
		if err := tx.Save(&a.RecurPattern).Error; err != nil {
			return fmt.Errorf("failed to update recurrence: %v", err)
		}
	}
	fmt.Println("Updated recurrence pattern:", a.RecurPattern)
	// Create the next recurring appointment
	nextAppointment := Appointment{
		Title:        a.Title,
		Description:  a.Description,
		StartTime:    nextTime,
		EndTime:      nextTime.Add(a.EndTime.Sub(a.StartTime)),
		Status:       StatusPending,
		IsRecurring:  true,
		RecurrenceID: a.RecurPattern.ID, // ✅ Set recurrence ID correctly
		ServiceID:    a.ServiceID,
		ProviderID:   a.ProviderID,
		CustomerID:   a.CustomerID,
	}
	fmt.Println("Next appointment to be created:", nextAppointment)
	// Save the new appointment
	if err := tx.Create(&nextAppointment).Error; err != nil {
		return fmt.Errorf("failed to create next recurrence: %v", err)
	}

	return nil
}
