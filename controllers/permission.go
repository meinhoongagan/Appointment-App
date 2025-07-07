package controllers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/db"
	"github.com/meinhoongagan/appointment-app/models"
)

// CreateRole creates a new role
// @Summary Create a new role
// @Description Create a new role in the system, accessible only to admin users
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role body models.Role true "Role details"
// @Success 201 {object} models.Role "Created role"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input or missing role name"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not an admin"
// @Failure 409 {object} fiber.Map{error=string} "Conflict - role with this name already exists"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/roles [post]
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
// @Summary Get all roles
// @Description Retrieve a list of all roles in the system, requires 'roles' read permission
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Role "List of roles"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - missing 'roles' read permission"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/roles [get]
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
// @Summary Create a new permission
// @Description Create a new permission in the system, accessible only to admin users
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param permission body models.Permission true "Permission details"
// @Success 201 {object} models.Permission "Created permission"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input or missing name/resource/action"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not an admin"
// @Failure 409 {object} fiber.Map{error=string} "Conflict - permission with this name already exists"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/permissions [post]
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
// @Summary Get all permissions
// @Description Retrieve a list of all permissions in the system, requires 'permissions' read permission
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Permission "List of permissions"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - missing 'permissions' read permission"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/permissions [get]
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
// @Summary Assign role to user
// @Description Assign a role to a user by their IDs, accessible only to admin users
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body object{user_id=integer,role_id=integer} true "Role assignment input"
// @Success 200 {object} fiber.Map{message=string} "Role assigned successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not an admin"
// @Failure 404 {object} fiber.Map{error=string} "User or role not found"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/users/role [post]
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
// @Summary Assign permission to role
// @Description Assign a permission to a role by their IDs, accessible only to admin users
// @Tags rbac
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param input body object{role_id=integer,permission_id=integer} true "Permission assignment input"
// @Success 200 {object} fiber.Map{message=string} "Permission assigned successfully"
// @Failure 400 {object} fiber.Map{error=string} "Bad request - invalid input"
// @Failure 401 {object} fiber.Map{error=string} "Unauthorized - invalid or missing token"
// @Failure 403 {object} fiber.Map{error=string} "Forbidden - user is not an admin"
// @Failure 404 {object} fiber.Map{error=string} "Role or permission not found"
// @Failure 409 {object} fiber.Map{error=string} "Conflict - permission already assigned to role"
// @Failure 500 {object} fiber.Map{error=string} "Internal server error"
// @Router /rbac/roles/permission [post]
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
