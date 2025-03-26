package models

import (
	"time"

	"gorm.io/gorm"
)

type Service struct {
	gorm.Model
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Duration    time.Duration `json:"duration"`
	Cost        float64       `json:"cost"`
	BufferTime  time.Duration `json:"buffer_time"` // Time between appointments
	ProviderID  uint          `json:"provider_id"`
	Provider    User         `json:"provider" gorm:"foreignKey:ProviderID"`
}
