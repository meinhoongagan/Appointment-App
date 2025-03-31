package db

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func GetDB() *gorm.DB {
	return DB
}

// Init establishes the DB connection without running migrations
func Init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file. Using environment variables directly.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	
	// Initialize default roles and permissions
	// initDefaultRolesAndPermissions()

	DB = db
	log.Println("✅ Database connection established successfully!")
}



// func initDefaultRolesAndPermissions() {
// 	// Default roles
// 	roles := []models.Role{
// 		{Name: "admin", Description: "Administrator with full access"},
// 		{Name: "provider", Description: "Service provider who can manage appointments"},
// 		{Name: "client", Description: "Client who can book appointments"},
// 		{Name: "receptionist", Description: "Front desk staff who can manage schedule"},
// 	}

// 	// Create roles if they don't exist
// 	for _, role := range roles {
// 		var existingRole models.Role
// 		if DB.Where("name = ?", role.Name).First(&existingRole).RowsAffected == 0 {
// 			DB.Create(&role)
// 		}
// 	}

// 	// Default permissions
// 	permissions := []models.Permission{
// 		// User management
// 		{Name: "create_user", Description: "Create new users", Resource: "users", Action: "create"},
// 		{Name: "read_users", Description: "View user list", Resource: "users", Action: "read"},
// 		{Name: "update_user", Description: "Update user details", Resource: "users", Action: "update"},
// 		{Name: "delete_user", Description: "Delete users", Resource: "users", Action: "delete"},

// 		// Appointment management
// 		{Name: "create_appointment", Description: "Create appointments", Resource: "appointments", Action: "create"},
// 		{Name: "read_appointments", Description: "View appointments", Resource: "appointments", Action: "read"},
// 		{Name: "update_appointment", Description: "Update appointments", Resource: "appointments", Action: "update"},
// 		{Name: "delete_appointment", Description: "Cancel appointments", Resource: "appointments", Action: "delete"},

// 		// Service management
// 		{Name: "create_service", Description: "Create services", Resource: "services", Action: "create"},
// 		{Name: "read_services", Description: "View services", Resource: "services", Action: "read"},
// 		{Name: "update_service", Description: "Update services", Resource: "services", Action: "update"},
// 		{Name: "delete_service", Description: "Delete services", Resource: "services", Action: "delete"},

// 		// Role management
// 		{Name: "create_role", Description: "Create roles", Resource: "roles", Action: "create"},
// 		{Name: "read_roles", Description: "View roles", Resource: "roles", Action: "read"},
// 		{Name: "update_role", Description: "Update roles", Resource: "roles", Action: "update"},
// 		{Name: "delete_role", Description: "Delete roles", Resource: "roles", Action: "delete"},

// 		// Permission management
// 		{Name: "create_permission", Description: "Create permissions", Resource: "permissions", Action: "create"},
// 		{Name: "read_permissions", Description: "View permissions", Resource: "permissions", Action: "read"},
// 		{Name: "update_permission", Description: "Update permissions", Resource: "permissions", Action: "update"},
// 		{Name: "delete_permission", Description: "Delete permissions", Resource: "permissions", Action: "delete"},
// 	}

// 	// Create permissions if they don't exist
// 	for _, permission := range permissions {
// 		var existingPermission models.Permission
// 		if DB.Where("name = ?", permission.Name).First(&existingPermission).RowsAffected == 0 {
// 			DB.Create(&permission)
// 		}
// 	}

// 	// Assign permissions to admin role
// 	var adminRole models.Role
// 	if DB.Where("name = ?", "admin").First(&adminRole).RowsAffected > 0 {
// 		var allPermissions []models.Permission
// 		DB.Find(&allPermissions)

// 		DB.Model(&adminRole).Association("Permissions").Clear()
// 		DB.Model(&adminRole).Association("Permissions").Append(allPermissions)
// 	}

// 	// Assign permissions to provider role
// 	var providerRole models.Role
// 	if DB.Where("name = ?", "provider").First(&providerRole).RowsAffected > 0 {
// 		var providerPermissions []models.Permission
// 		DB.Where("resource = ? OR resource = ?", "appointments", "services").
// 			Where("action IN (?)", []string{"read", "create", "update"}).
// 			Find(&providerPermissions)

// 		var userReadPermission models.Permission
// 		DB.Where("name = ?", "read_users").First(&userReadPermission)
// 		providerPermissions = append(providerPermissions, userReadPermission)

// 		DB.Model(&providerRole).Association("Permissions").Clear()
// 		DB.Model(&providerRole).Association("Permissions").Append(providerPermissions)
// 	}

// 	// Assign permissions to client role
// 	var clientRole models.Role
// 	if DB.Where("name = ?", "client").First(&clientRole).RowsAffected > 0 {
// 		var clientPermissions []models.Permission
// 		DB.Where("name IN (?)", []string{
// 			"create_appointment",
// 			"read_appointments",
// 			"update_appointment",
// 			"delete_appointment",
// 			"read_services",
// 		}).Find(&clientPermissions)

// 		DB.Model(&clientRole).Association("Permissions").Clear()
// 		DB.Model(&clientRole).Association("Permissions").Append(clientPermissions)
// 	}

// 	// Assign permissions to receptionist role
// 	var receptionistRole models.Role
// 	if DB.Where("name = ?", "receptionist").First(&receptionistRole).RowsAffected > 0 {
// 		var receptionistPermissions []models.Permission
// 		DB.Where("resource IN (?)", []string{"appointments", "services", "users"}).
// 			Where("action IN (?)", []string{"read", "create", "update"}).
// 			Find(&receptionistPermissions)

// 		DB.Model(&receptionistRole).Association("Permissions").Clear()
// 		DB.Model(&receptionistRole).Association("Permissions").Append(receptionistPermissions)
// 	}
// }
