package utils

import (
	"fmt"
	"reflect"
	"time"

	"github.com/meinhoongagan/appointment-app/models"

	"github.com/meinhoongagan/appointment-app/db"
)

// Check if the appointment is within the provider's working days and hours (including break handling)
func CheckWorkingDayAndHours(providerID uint, appointmentStart time.Time) (bool, error) {
	var providerWorkingHours []models.WorkingHours
	if err := db.DB.Where("provider_id = ?", providerID).Find(&providerWorkingHours).Error; err != nil {
		return false, fmt.Errorf("provider working hours not found")
	}

	// Get the day of the week for the appointment (0 for Sunday, 1 for Monday, ... 6 - Saturday)
	appointmentDay := int(appointmentStart.Weekday())

	// Check if the appointment falls within the provider's working days
	var workingHoursForTheDay models.WorkingHours
	for _, wh := range providerWorkingHours {
		if int(wh.DayOfWeek) == appointmentDay && wh.IsWorkDay {
			workingHoursForTheDay = wh
			break
		}
	}
	// If no working hours found for the day
	if reflect.DeepEqual(workingHoursForTheDay, models.WorkingHours{}) {
		return false, nil // Appointment is outside working days
	}

	// Convert start and end times to time.Time for comparison
	layout := "15:04"
	startTime, err := time.Parse(layout, workingHoursForTheDay.StartTime)
	if err != nil {
		return false, fmt.Errorf("invalid start time format")
	}

	endTime, err := time.Parse(layout, workingHoursForTheDay.EndTime)
	if err != nil {
		return false, fmt.Errorf("invalid end time format")
	}

	// Check if the appointment start time falls within working hours
	if appointmentStart.Before(startTime) || appointmentStart.After(endTime) {
		return false, nil // Appointment is outside working hours
	}

	// Check for break periods if they exist
	if workingHoursForTheDay.BreakStart != nil && workingHoursForTheDay.BreakEnd != nil {
		breakStart, err := time.Parse(layout, *workingHoursForTheDay.BreakStart)
		if err != nil {
			return false, fmt.Errorf("invalid break start time format")
		}
		breakEnd, err := time.Parse(layout, *workingHoursForTheDay.BreakEnd)
		if err != nil {
			return false, fmt.Errorf("invalid break end time format")
		}

		// If appointment falls during break, it's invalid
		if appointmentStart.After(breakStart) && appointmentStart.Before(breakEnd) {
			return false, nil // Appointment is within break time
		}
	}

	return true, nil
}
