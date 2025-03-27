package models

import (
	"time"

	"gorm.io/gorm"
)

type Permission struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"unique"`
	Description string         `json:"description"`
	Resource    string         `json:"resource"` // e.g., "appointments", "users", etc.
	Action      string         `json:"action"`   // e.g., "create", "read", "update", "delete"
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	Roles       []Role         `json:"roles,omitempty" gorm:"many2many:role_permissions;foreignKey:ID;joinForeignKey:PermissionID;references:ID;joinReferences:RoleID"`
}
