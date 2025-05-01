package db

import (
	"fmt"
	"log"

	"github.com/meinhoongagan/appointment-app/models"
)

func Migrate() {
	// Init()

	err := DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Recurrence{},
		&models.Appointment{},
		&models.Service{},
		&models.WorkingHours{},
		&models.TimeOff{},
		&models.BusinessDetails{},
		&models.ProviderSettings{},
		&models.Review{},
	)
	if err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	fmt.Println("✅ Migrations applied successfully!")
}
