package models

import (
	"gorm.io/gorm"
)

type DayOfWeek int

const (
	Sunday DayOfWeek = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

type WorkingHours struct {
	gorm.Model
	ProviderID uint      `json:"provider_id"`
	Provider   User      `json:"provider" gorm:"foreignKey:ProviderID"`
	DayOfWeek  DayOfWeek `json:"day_of_week"`
	StartTime  string    `json:"start_time"` // Format "HH:MM" in 24h
	EndTime    string    `json:"end_time"`   // Format "HH:MM" in 24h
	IsWorkDay  bool      `json:"is_work_day" gorm:"default:true"`
	BreakStart *string   `json:"break_start"` // Optional break start time
	BreakEnd   *string   `json:"break_end"`   // Optional break end time
}
