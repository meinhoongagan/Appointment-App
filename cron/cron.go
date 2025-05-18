package cron

import (
	"fmt"
	"log"
	"time"

	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
	"github.com/meinhoongagan/appointment-app/utils"
	"github.com/robfig/cron/v3"
)

// StartCronJobs initializes and starts the cron scheduler for appointment reminders
func StartCronJobs() {
	fmt.Println("Starting cron job scheduler...")
	c := cron.New()
	// Run every minute to check for appointments in the next hour
	_, err := c.AddFunc("* * * * *", sendAppointmentReminders)
	if err != nil {
		log.Fatalf("Failed to add cron job: %v", err)
	}
	c.Start()
	log.Println("Cron job scheduler started for appointment reminders")
}

// sendAppointmentReminders checks for appointments and sends reminders
func sendAppointmentReminders() {
	var appointments []models.Appointment
	now := time.Now()
	// Look for appointments starting in the next hour
	startWindow := now.Add(55 * time.Minute)
	endWindow := now.Add(65 * time.Minute)

	// Query appointments that are confirmed and within the time window
	err := db.DB.Preload("Customer").Preload("Service").Preload("Provider").
		Where("status = ? AND start_time BETWEEN ? AND ?", models.StatusConfirmed, startWindow, endWindow).
		Find(&appointments).Error
	if err != nil {
		log.Printf("Error fetching appointments for reminders: %v", err)
		return
	}

	fmt.Printf("Found %d appointments for reminders\n", len(appointments))

	for _, appointment := range appointments {
		// Send reminder email to customer
		err := sendReminderEmail(&appointment)
		if err != nil {
			log.Printf("Failed to send reminder for appointment %d: %v", appointment.ID, err)
			continue
		}
		log.Printf("Sent reminder for appointment %d to %s", appointment.ID, appointment.Customer.Email)
	}
}

// sendReminderEmail constructs and sends the reminder email
func sendReminderEmail(appointment *models.Appointment) error {
	subject := fmt.Sprintf("Reminder: Upcoming Appointment - %s", appointment.Title)
	body := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>This is a reminder for your upcoming appointment scheduled in one hour.</p>
		<p><strong>Details:</strong></p>
		<ul>
			<li><strong>Service:</strong> %s</li>
			<li><strong>Provider:</strong> %s</li>
			<li><strong>Start Time:</strong> %s</li>
			<li><strong>End Time:</strong> %s</li>
			<li><strong>Status:</strong> %s</li>
		</ul>
		<p>Please arrive on time. If you need to reschedule or cancel, contact us as soon as possible.</p>
		<p>Best regards,</p>
		<p>Your Appointment Team</p>
	`, appointment.Customer.Name, appointment.Service.Name, appointment.Provider.Name,
		appointment.StartTime.Format("2006-01-02 15:04:05"),
		appointment.EndTime.Format("2006-01-02 15:04:05"),
		appointment.Status)

	return utils.SendEmail(appointment.Customer.Email, subject, body)
}
