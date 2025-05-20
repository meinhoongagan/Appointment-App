package utils

import (
	"time"

	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"gorm.io/gorm"
)

// CheckAvailability checks if a provider is available for a given time slot, including buffer time
func CheckAvailability(providerID uint, startTime time.Time, totalDuration time.Duration) (bool, error) {
	// Convert startTime and endTime to IST before checking
	startTimeIST := ToIST(startTime)
	endTimeIST := ToIST(startTime.Add(totalDuration)) // totalDuration includes Duration + BufferTime

	// Check if any conflicting appointments exist and lock them
	var existingAppointment models.Appointment
	err := db.DB.Raw(`
		SELECT *
		FROM appointments
		WHERE provider_id = ? AND status != ? AND (
			(start_time < ? AND end_time > ?) OR
			(start_time >= ? AND start_time < ?)
		)
		FOR UPDATE
	`, providerID, models.StatusCompleted, endTimeIST, startTimeIST, startTimeIST, endTimeIST).
		First(&existingAppointment).Error

	// If there is a conflicting appointment (excluding completed), return false
	if err == nil && existingAppointment.ID != 0 {
		return false, nil
	}

	// Handle database errors
	if err != nil && err != gorm.ErrRecordNotFound {
		return false, err
	}

	// No conflict, slot is available
	return true, nil
}
