package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/your-username/onboarding/auth"
)

func Default() gin.HandlerFunc {
	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	config.AllowAllOrigins = true
	return cors.New(config)
}

// SetupRouter configures the routes for the application.
func SetupRouter() *gin.Engine {
	router := gin.Default()

	router.Use(Default())

	// --- Public Routes ---
	// No authentication required for these.f
	public := router.Group("/public")
	{
		public.POST("/signup", TenantSignupHandler)
	}

	// --- Authentication Routes ---
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/login", LoginHandler)
	}

	// --- Protected API Routes ---
	// All routes in this group will be protected by the JWT AuthMiddleware.
	api := router.Group("/api/v1")
	api.Use(auth.AuthMiddleware())
	{
		// Employee CRUD
		employees := api.Group("/employees")
		employees.Use(auth.RequireEntityAccess("employees"))
		{
			employees.POST("", CreateEmployeeHandler)
			employees.GET("", GetEmployeesHandler)
			employees.GET("/:id", GetEmployeeByIDHandler)
			employees.PUT("/:id", UpdateEmployeeHandler)
			employees.DELETE("/:id", DeleteEmployeeHandler)
		}

		users := api.Group("/users")
		{
			users.POST("", CreateUserHandler)
			users.GET("", GetUsersHandler)
		}

		// --- CRUD routes for all dynamic entities ---
		// Using a helper function to keep this section clean.
		createEntityRoutes(api, "locations", CreateLocationHandler, GetLocationsHandler, GetLocationByIDHandler, UpdateLocationHandler, DeleteLocationHandler)
		createEntityRoutes(api, "departments", CreateDepartmentHandler, GetDepartmentsHandler, GetDepartmentByIDHandler, UpdateDepartmentHandler, DeleteDepartmentHandler)
		createEntityRoutes(api, "managers", CreateManagersHandler, GetManagersHandler, GetManagerByIDHandler, UpdateManagerHandler, DeleteManagerHandler)
		createEntityRoutes(api, "job-roles", CreateJobRoleHandler, GetJobRolesHandler, GetJobRoleByIDHandler, UpdateJobRoleHandler, DeleteJobRoleHandler)
		createEntityRoutes(api, "employement-types", CreateEmploymentTypeHandler, GetEmploymentTypesHandler, GetEmploymentTypeByIDHandler, UpdateEmploymentTypeHandler, DeleteEmploymentTypeHandler)
		createEntityRoutes(api, "teams", CreateTeamHandler, GetTeamsHandler, GetTeamByIDHandler, UpdateTeamHandler, DeleteTeamHandler)
		createEntityRoutes(api, "costs", CreateCostCenterHandler, GetCostCentersHandler, GetCostCenterByIDHandler, UpdateCostCenterHandler, DeleteCostCenterHandler)
		createEntityRoutes(api, "hardware-assets", CreateHardwareAssetHandler, GetHardwareAssetsHandler, GetHardwareAssetByIDHandler, UpdateHardwareAssetHandler, DeleteAccessLevelHandler)
		createEntityRoutes(api, "onboarding-buddy", CreateOnboardingBuddyHandler, GetOnboardingBuddiesHandler, GetOnboardingBuddyByIDHandler, UpdateOnboardingBuddyHandler, DeleteOnboardingBuddyHandler)
		createEntityRoutes(api, "access-levels", CreateAccessLevelHandler, GetAccessLevelsHandler, GetAccessLevelByIDHandler, UpdateAccessLevelHandler, DeleteAccessLevelHandler)
	}
	return router
}

// createEntityRoutes is a helper function to reduce route definition boilerplate.
func createEntityRoutes(group *gin.RouterGroup, resource string, create, getAll, getByID, update, del gin.HandlerFunc) {
	entityGroup := group.Group(resource)
	entityGroup.Use(auth.RequireEntityAccess(resource))
	{
		entityGroup.POST("", create)
		entityGroup.GET("", getAll)
		entityGroup.GET("/:id", getByID)
		entityGroup.PUT("/:id", update)
		entityGroup.DELETE("/:id", del)
	}
}
