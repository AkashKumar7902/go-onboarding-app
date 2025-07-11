package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/your-username/onboarding/models"
	"github.com/your-username/onboarding/services"
)

// --- Public Handlers ---

// TenantSignupHandler handles the creation of a new tenant and its first admin user.
func TenantSignupHandler(c *gin.Context) {
	var signupData services.TenantSignupData
	if err := c.ShouldBindJSON(&signupData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, user, err := services.CreateTenantAndAdminUser(&signupData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Account created successfully. Please log in.",
		"tenantId":    tenant.ID,
		"adminUserId": user.ID,
	})
}

// --- Auth Handlers ---

// LoginHandler handles user login and returns a JWT.
func LoginHandler(c *gin.Context) {
	var creds struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := services.LoginUser(creds.Username, creds.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// --- Protected Handlers (Require JWT) ---

// Employee Handlers
func CreateEmployeeHandler(c *gin.Context) {
	var employee models.Employee
	if err := c.ShouldBindJSON(&employee); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set tenantID from the JWT claims, which were placed in the context by the middleware.
	employee.TenantID = c.GetString("tenantId")

	createdEmployee, err := services.CreateEmployee(&employee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
		return
	}
	c.JSON(http.StatusCreated, createdEmployee)
}

func GetEmployeesHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	employees, err := services.GetEmployeesByTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
		return
	}
	c.JSON(http.StatusOK, employees)
}

// --- Dynamic Entity Handlers ---
// The pattern for all dynamic entities is the same. We define a few here as examples.

// Location Handlers
func CreateLocationHandler(c *gin.Context) {
	var entity models.Location
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")

	createdEntity, err := services.CreateEntity(c.Request.Context(), "locations", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create location"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetLocationsHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.Location](c.Request.Context(), "locations", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch locations"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Department Handlers
func CreateDepartmentHandler(c *gin.Context) {
	var entity models.Department
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "departments", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create department"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetDepartmentsHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.Department](c.Request.Context(), "departments", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Managers Handlers
func CreateManagersHandler(c *gin.Context) {
	var entity models.Manager
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "managers", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manager"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetManagersHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.Manager](c.Request.Context(), "managers", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch managers"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// JobRole Handlers
func CreateJobRoleHandler(c *gin.Context) {
	var entity models.JobRole
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "job_roles", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job role"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetJobRolesHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.JobRole](c.Request.Context(), "job_roles", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job roles"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// EmploymentType Handlers
func CreateEmploymentTypeHandler(c *gin.Context) {
	var entity models.EmploymentType
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "employment_types", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employment type"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetEmploymentTypesHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.EmploymentType](c.Request.Context(), "employment_types", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employment types"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Team Handlers
func CreateTeamHandler(c *gin.Context) {
	var entity models.Team
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "teams", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetTeamsHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.Team](c.Request.Context(), "teams", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Cost Center Handlers
func CreateCostCenterHandler(c *gin.Context) {
	var entity models.CostCenter
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "cost_centers", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cost center"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetCostCentersHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.CostCenter](c.Request.Context(), "cost_centers", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cost centers"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Hardware Asset Handlers
func CreateHardwareAssetHandler(c *gin.Context) {
	var entity models.HardwareAsset
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "hardware_assets", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create hardware asset"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetHardwareAssetsHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.HardwareAsset](c.Request.Context(), "hardware_assets", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hardware assets"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Onboarding Buddy Handlers
func CreateOnboardingBuddyHandler(c *gin.Context) {
	var entity models.OnboardingBuddy
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "onboarding_buddies", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create onboarding buddy"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetOnboardingBuddiesHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.OnboardingBuddy](c.Request.Context(), "onboarding_buddies", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch onboarding buddies"})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// Access Level Handlers
func CreateAccessLevelHandler(c *gin.Context) {
	var entity models.AccessLevel
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	entity.TenantID = c.GetString("tenantId")
	createdEntity, err := services.CreateEntity(c.Request.Context(), "access_levels", &entity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access level"})
		return
	}
	c.JSON(http.StatusCreated, createdEntity)
}

func GetAccessLevelsHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	entities, err := services.GetEntitiesByTenant[models.AccessLevel](c.Request.Context(), "access_levels", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch access levels"})
		return
	}
	c.JSON(http.StatusOK, entities)
}
