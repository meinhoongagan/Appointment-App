package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// CreateRole creates a new role
func CreateRole(c *fiber.Ctx) error {
	role := new(models.Role)

	if err := c.BodyParser(role); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	if role.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role name is required",
		})
	}

	// Check if role already exists
	var existingRole models.Role
	if db.DB.Where("name = ?", role.Name).First(&existingRole).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Role with this name already exists",
		})
	}

	// Create role
	if err := db.DB.Create(&role).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create role",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(role)
}

// GetRoles returns all roles
func GetRoles(c *fiber.Ctx) error {
	var roles []models.Role

	if err := db.DB.Find(&roles).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get roles",
		})
	}

	return c.JSON(roles)
}

// CreatePermission creates a new permission
func CreatePermission(c *fiber.Ctx) error {
	permission := new(models.Permission)

	if err := c.BodyParser(permission); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	if permission.Name == "" || permission.Resource == "" || permission.Action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name, resource, and action are required",
		})
	}

	// Check if permission already exists
	var existingPermission models.Permission
	if db.DB.Where("name = ?", permission.Name).First(&existingPermission).RowsAffected > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Permission with this name already exists",
		})
	}

	// Create permission
	if err := db.DB.Create(&permission).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create permission",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(permission)
}

// GetPermissions returns all permissions
func GetPermissions(c *fiber.Ctx) error {
	var permissions []models.Permission

	if err := db.DB.Find(&permissions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get permissions",
		})
	}

	return c.JSON(permissions)
}

// AssignRoleToUser assigns a role to a user
func AssignRoleToUser(c *fiber.Ctx) error {
	type AssignRoleInput struct {
		UserID uint `json:"user_id"`
		RoleID uint `json:"role_id"`
	}

	input := new(AssignRoleInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Check if user exists
	var user models.User
	if db.DB.First(&user, input.UserID).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Check if role exists
	var role models.Role
	if db.DB.First(&role, input.RoleID).RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	// Assign role to user
	user.RoleID = input.RoleID

	if err := db.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to assign role to user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Role assigned successfully",
	})
}

// AssignPermissionToRole assigns a permission to a role
func AssignPermissionToRole(c *fiber.Ctx) error {
	type AssignPermissionInput struct {
		RoleID       uint `json:"role_id"`
		PermissionID uint `json:"permission_id"`
	}

	input := new(AssignPermissionInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	// Check if role exists
	var role models.Role
	if err := db.DB.Preload("Permissions").First(&role, input.RoleID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Role not found",
		})
	}

	// Check if permission exists
	var permission models.Permission
	if err := db.DB.First(&permission, input.PermissionID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Permission not found",
		})
	}

	// Check if permission is already assigned to role
	for _, p := range role.Permissions {
		if p.ID == permission.ID {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Permission already assigned to role",
			})
		}
	}

	// Assign permission to role
	if err := db.DB.Model(&role).Association("Permissions").Append(&permission); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to assign permission to role",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Permission assigned successfully",
	})
}
