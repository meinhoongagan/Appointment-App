package models

import (
	"time"
)

type User struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	Name                 string         `json:"name"`
	Email                string         `json:"email" gorm:"unique"`
	Password             string         `json:"password,omitempty"`
	IsVerified           bool           `json:"is_verified"`
	OTP                  string         `json:"otp,omitempty"`
	OTPExpiresAt         time.Time      `json:"otp_expires_at,omitempty"`
	RoleID               uint           `json:"role_id"`
	Role                 Role           `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	ProvidedServices     []Service      `json:"provided_services,omitempty" gorm:"foreignKey:ProviderID"`
	Appointments         []Appointment  `json:"appointments,omitempty" gorm:"foreignKey:ProviderID"`
	CustomerAppointments []Appointment  `json:"customer_appointments,omitempty" gorm:"foreignKey:CustomerID"`
	WorkingHours         []WorkingHours `json:"working_hours,omitempty" gorm:"foreignKey:ProviderID"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}
