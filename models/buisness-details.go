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
	ProviderID              uint    `json:"provider_id"`
	NotificationsEnabled    bool    `json:"notifications_enabled"`
	EmailNotifications      bool    `json:"email_notifications"`
	SMSNotifications        bool    `json:"sms_notifications"`
	AutoConfirmBookings     bool    `json:"auto_confirm_bookings"`
	RequirePayment          bool    `json:"require_payment"`
	AdvanceBookingDays      int     `json:"advance_booking_days"`
	CancellationPeriodHours int     `json:"cancellation_period_hours"`
	CancellationFeePercent  float64 `json:"cancellation_fee_percent"`
	TaxRate                 float64 `json:"tax_rate"`
	Currency                string  `json:"currency"`
	TimeZone                string  `json:"time_zone"`
	Language                string  `json:"language"`
}

type TimeOff struct {
	gorm.Model
	ProviderID uint      `json:"provider_id"`
	Title      string    `json:"title"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	AllDay     bool      `json:"all_day"`
	Notes      string    `json:"notes"`
}
