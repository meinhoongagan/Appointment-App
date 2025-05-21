package models

import (
	"gorm.io/gorm"
)

type UserDetails struct {
	gorm.Model
	User             User      `json:"user" gorm:"foreignKey:UserID"`
	UserID           uint      `json:"user_id"`
	ProfilePicture   string    `json:"profile_picture"`
	FavoriteServices []Service `json:"favorite_services" gorm:"many2many:user_favorite_services;"`
}
