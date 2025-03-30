package utils

import (
	"time"

	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// CheckAvailability checks if a provider is available for a given time slot
func CheckAvailability(providerID uint, startTime time.Time, duration time.Duration) (bool, error) {
	endTime := startTime.Add(duration)

	// Check if any conflicting appointments exist and lock them
	var existingAppointment models.Appointment
	err := db.DB.Raw(`
		SELECT * 
		FROM appointments
		WHERE provider_id = ? AND (
			(start_time < ? AND end_time > ?) OR
			(start_time >= ? AND start_time < ?)
		) FOR UPDATE
		LIMIT 1
	`, providerID, endTime, startTime, startTime, endTime).
		Scan(&existingAppointment).Error

	// If there is any conflicting appointment, return false
	if err == nil && existingAppointment.ID != 0 {
		return false, nil
	}

	// No conflict, slot is available
	return true, nil
}
