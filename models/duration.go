package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DurationSwagger stores duration as hours and minutes
type Duration struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

// Value implements the driver.Valuer interface
func (d Duration) Value() (driver.Value, error) {
	jsonData, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return string(jsonData), nil // Return as string for JSONB type
}

// Scan implements the sql.Scanner interface
func (d *Duration) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("failed to unmarshal DurationSwagger: unsupported type %T", value)
	}

	return json.Unmarshal(data, d)
}

// ToDuration converts DurationSwagger to time.Duration
func (d Duration) ToDuration() time.Duration {
	hours := time.Duration(d.Hours) * time.Hour
	minutes := time.Duration(d.Minutes) * time.Minute
	return hours + minutes
}
