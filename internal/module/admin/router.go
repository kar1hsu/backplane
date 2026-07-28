package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/handler"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "admin"
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	authHandler := handler.NewAuthHandler()
	userHandler := handler.NewUserHandler()
	roleHandler := handler.NewRoleHandler()
	menuHandler := handler.NewMenuHandler()
	apiHandler := handler.NewAPIHandler()
	operationLogHandler := handler.NewOperationLogHandler()
	configHandler := handler.NewConfigHandler()

	// Public routes (no auth required)
	rg.POST("/login", authHandler.Login)

	// Authenticated routes (JWT only, no RBAC)
	// For: user profile, option/dropdown data queries
	authenticated := rg.Group("", middleware.AdminAuth())
	{
		authenticated.POST("/logout", authHandler.Logout)
		authenticated.GET("/profile", authHandler.GetProfile)
		authenticated.GET("/permissions", authHandler.GetPermissions)
		authenticated.GET("/menus/user", menuHandler.GetUserMenuTree)
		authenticated.GET("/roles/all", roleHandler.ListAll)
		authenticated.GET("/menus/tree", menuHandler.GetTree)
		authenticated.GET("/apis/all", apiHandler.ListAll)
	}

	// Protected routes (JWT + Casbin RBAC)
	// For: CRUD management operations
	// OperationLog sits between auth and RBAC so denied (403) attempts are still audited.
	auth := rg.Group("", middleware.AdminAuth(), middleware.OperationLog(), middleware.CasbinRBAC())
	{
		// Users
		auth.GET("/users", userHandler.List)
		auth.POST("/users", userHandler.Create)
		auth.GET("/users/:id", userHandler.GetByID)
		auth.PUT("/users/:id", userHandler.Update)
		auth.DELETE("/users/:id", userHandler.Delete)

		// Roles
		auth.GET("/roles", roleHandler.List)
		auth.POST("/roles", roleHandler.Create)
		auth.GET("/roles/:id", roleHandler.GetByID)
		auth.PUT("/roles/:id", roleHandler.Update)
		auth.DELETE("/roles/:id", roleHandler.Delete)
		auth.PUT("/roles/:id/menus", roleHandler.SetMenus)
		auth.PUT("/roles/:id/apis", roleHandler.SetAPIs)
		auth.GET("/roles/:id/apis", roleHandler.GetAPIs)

		// Menus
		auth.POST("/menus", menuHandler.Create)
		auth.GET("/menus/:id", menuHandler.GetByID)
		auth.PUT("/menus/:id", menuHandler.Update)
		auth.DELETE("/menus/:id", menuHandler.Delete)

		// APIs
		auth.GET("/apis", apiHandler.List)
		auth.POST("/apis", apiHandler.Create)
		auth.GET("/apis/:id", apiHandler.GetByID)
		auth.PUT("/apis/:id", apiHandler.Update)
		auth.DELETE("/apis/:id", apiHandler.Delete)

		// Operation logs (audit)
		auth.GET("/operation-logs", operationLogHandler.List)
		auth.GET("/operation-logs/:id", operationLogHandler.GetByID)
		auth.DELETE("/operation-logs/:id", operationLogHandler.Delete)
		auth.DELETE("/operation-logs", operationLogHandler.Clear)

		// System configs
		auth.GET("/configs", configHandler.List)
		auth.POST("/configs", configHandler.Create)
		auth.PUT("/configs", configHandler.BatchUpdate)
		auth.PUT("/configs/:id", configHandler.Update)
		auth.DELETE("/configs/:id", configHandler.Delete)
		auth.POST("/configs/refresh", configHandler.Refresh)
	}
}
