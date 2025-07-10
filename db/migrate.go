package db

import (
	"fmt"
	"log"
)

func Migrate() {
	// Init()

	err := DB.AutoMigrate(
	// &models.User{},
	// &models.UserDetails{},
	// &models.Role{},
	// &models.Permission{},
	// &models.Recurrence{},
	// &models.Appointment{},
	// &models.Service{},
	// &models.WorkingHours{},
	// &models.BusinessDetails{},
	// &models.ReceptionistSettings{},
	// &models.ProviderSettings{},
	// &models.Review{},
	)
	if err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	fmt.Println("✅ Migrations applied successfully!")
}
