package db

import (
	"fmt"
	"log"

	"github.com/meinhoongagan/appointment-app/models"
)

func Migrate() {
	// Initialize DB connection
	Init()

	// Run AutoMigrate only when explicitly called
	err := DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Recurrence{},
		&models.Appointment{},
		&models.Service{},
		&models.WorkingHours{},
	)
	if err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	fmt.Println("✅ Migrations applied successfully!")
}
