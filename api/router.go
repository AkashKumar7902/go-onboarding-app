package api

import (
	"github.com/gin-gonic/gin"
	"github.com/your-username/onboarding/auth"
)

// SetupRouter configures the routes for the application.
func SetupRouter() *gin.Engine {
	router := gin.Default()

	// --- Public Routes ---
	// No authentication required for these.
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
		{
			employees.POST("", CreateEmployeeHandler)
			employees.GET("", GetEmployeesHandler)
			// Other employee routes (GET by ID, PUT, DELETE) would go here.
		}

		// --- CRUD routes for all dynamic entities ---
		// Using a helper function to keep this section clean.
		createEntityRoutes(api, "locations", CreateLocationHandler, GetLocationsHandler)
		createEntityRoutes(api, "departments", CreateDepartmentHandler, GetDepartmentsHandler)
		createEntityRoutes(api, "managers", CreateManagersHandler, GetManagersHandler)
		createEntityRoutes(api, "job-roles", CreateJobRoleHandler, GetJobRolesHandler)
		createEntityRoutes(api, "employement-types", CreateEmploymentTypeHandler, GetEmploymentTypesHandler)
		createEntityRoutes(api, "teams", CreateTeamHandler, GetTeamsHandler)
		createEntityRoutes(api, "costs", CreateCostCenterHandler, GetCostCentersHandler)
		createEntityRoutes(api, "hardware-assets", CreateHardwareAssetHandler, GetHardwareAssetsHandler)
		createEntityRoutes(api, "onboarding-buddy", CreateOnboardingBuddyHandler, GetOnboardingBuddiesHandler)
		createEntityRoutes(api, "access-levels", CreateAccessLevelHandler, GetAccessLevelsHandler)
	}
	return router
}

// createEntityRoutes is a helper function to reduce route definition boilerplate.
func createEntityRoutes(group *gin.RouterGroup, resource string, create gin.HandlerFunc, get gin.HandlerFunc) {
	entityGroup := group.Group(resource)
	{
		entityGroup.POST("", create)
		entityGroup.GET("", get)
		// You can add GET by ID, PUT, and DELETE routes/handlers here as well.
	}
}
