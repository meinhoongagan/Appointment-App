package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/meinhoongagan/appointment-app/controllers"
	"github.com/meinhoongagan/appointment-app/middleware"
)

// SetupRBACRoutes configures all RBAC related routes
func SetupRBACRoutes(app *fiber.App) {
	rbac := app.Group("/rbac", middleware.Protected())

	// Roles
	rbac.Post("/roles", middleware.RequireRole("admin"), controllers.CreateRole)
	rbac.Get("/roles", middleware.RequirePermission("roles", "read"), controllers.GetRoles)

	// Permissions
	rbac.Post("/permissions", middleware.RequireRole("admin"), controllers.CreatePermission)
	rbac.Get("/permissions", middleware.RequirePermission("permissions", "read"), controllers.GetPermissions)

	// Assign role to user
	rbac.Post("/users/role", middleware.RequireRole("admin"), controllers.AssignRoleToUser)

	// Assign permission to role
	rbac.Post("/roles/permission", middleware.RequireRole("admin"), controllers.AssignPermissionToRole)
}
