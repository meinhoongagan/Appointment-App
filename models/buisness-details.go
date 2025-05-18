package models

import (
	"time"

	"gorm.io/gorm"
)

// BusinessDetails contains information about the provider's business
type BusinessDetails struct {
	gorm.Model
	ProviderID      uint   `json:"provider_id"`
	BusinessName    string `json:"business_name"`
	Description     string `json:"description"`
	Address         string `json:"address"`
	City            string `json:"city"`
	State           string `json:"state"`
	ZipCode         string `json:"zip_code"`
	Country         string `json:"country"`
	PhoneNumber     string `json:"phone_number"`
	Email           string `json:"email"`
	Website         string `json:"website"`
	LogoURL         string `json:"logo_url"`
	BusinessHours   string `json:"business_hours"`
	TaxNumber       string `json:"tax_number"`
	BusinessLicense string `json:"business_license"`
}

// ProviderSettings contains settings for the provider
type ProviderSettings struct {
	gorm.Model
	ProviderID           uint      `json:"provider_id"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	ClosingStartDate     time.Time `json:"closing_start_date"`
	ClosingEndDate       time.Time `json:"closing_end_date"`
	ClosingRemarks       string    `json:"closing_remarks"`
	AutoConfirmBookings  bool      `json:"auto_confirm_bookings"`
	AdvanceBookingDays   int       `json:"advance_booking_days"`
	Currency             string    `json:"currency"`
	TimeZone             string    `json:"time_zone"`
	Language             string    `json:"language"`
}

type ReceptionistSettings struct {
	gorm.Model
	Provider       User `json:"provider" gorm:"foreignKey:ProviderID"`
	ProviderID     uint `json:"provider_id"`
	Receptionist   User `json:"receptionist" gorm:"foreignKey:ReceptionistID"`
	ReceptionistID uint `json:"receptionist_id"`
}
