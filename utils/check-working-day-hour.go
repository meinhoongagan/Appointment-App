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
	fmt.Println("Provider working hours:", providerWorkingHours)
	// Map Go's Weekday (Sunday=0) to your DB's format (Monday=0)
	dbDayOfWeek := (int(appointmentStart.Weekday()) + 6) % 7

	var workingHoursForTheDay models.WorkingHours
	for _, wh := range providerWorkingHours {
		if int(wh.DayOfWeek) == dbDayOfWeek {
			workingHoursForTheDay = wh
			break
		}
	}
	fmt.Println("Working hours for the day:", workingHoursForTheDay)
	if reflect.DeepEqual(workingHoursForTheDay, models.WorkingHours{}) {
		return false, nil
	}

	// Convert DB time strings to time.Time on the same date as the appointment
	location := appointmentStart.Location()
	layout := "15:04"
	fmt.Println("Location:", location)
	buildTime := func(t string) (time.Time, error) {
		parsed, err := time.ParseInLocation(layout, t, location)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(appointmentStart.Year(), appointmentStart.Month(), appointmentStart.Day(),
			parsed.Hour(), parsed.Minute(), 0, 0, location), nil
	}
	fmt.Println("Layout:", layout)
	fmt.Println("Appointment start:", appointmentStart)
	fmt.Println("Working hours start time:", workingHoursForTheDay.StartTime)
	fmt.Println("Working hours end time:", workingHoursForTheDay.EndTime)
	// Parse start and end times
	startTime, err := buildTime(workingHoursForTheDay.StartTime)
	if err != nil {
		return false, fmt.Errorf("invalid start time format")
	}
	endTime, err := buildTime(workingHoursForTheDay.EndTime)
	if err != nil {
		return false, fmt.Errorf("invalid end time format")
	}

	// Check if the appointment time is within working hours
	if appointmentStart.Before(startTime) || appointmentStart.After(endTime) {
		return false, nil
	}

	// Handle break times
	if workingHoursForTheDay.BreakStart != nil && workingHoursForTheDay.BreakEnd != nil {
		breakStart, err := buildTime(*workingHoursForTheDay.BreakStart)
		if err != nil {
			return false, fmt.Errorf("invalid break start time format")
		}
		breakEnd, err := buildTime(*workingHoursForTheDay.BreakEnd)
		if err != nil {
			return false, fmt.Errorf("invalid break end time format")
		}
		if appointmentStart.After(breakStart) && appointmentStart.Before(breakEnd) {
			return false, nil
		}
	}

	return true, nil
}
