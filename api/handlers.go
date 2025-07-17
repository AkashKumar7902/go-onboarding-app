package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"github.com/your-username/onboarding/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// CreateUserHandler handles the creation of a new user by a tenant admin.
func CreateUserHandler(c *gin.Context) {
	// 1. Get the ID of the user MAKING the request from the JWT context.
	creatorUserID := c.GetString("userId")
	creatorTenantID := c.GetString("tenantId")

	// 2. Check if the creator is an admin.
	var creatorUser models.User
	creatorObjID, _ := primitive.ObjectIDFromHex(creatorUserID)
	var usersCollection = db.GetCollection("users")
	err := usersCollection.FindOne(c.Request.Context(), bson.M{"_id": creatorObjID}).Decode(&creatorUser)
	if err != nil || creatorUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create new users."})
		return
	}

	// 3. If they are an admin, process the new user's data.
	var newUserData services.CreateUserData
	if err := c.ShouldBindJSON(&newUserData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Call the service to create the new user within the same tenant.
	createdUser, err := services.CreateUserForTenant(&newUserData, creatorTenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

// GetUsersHandler lists all users for the current tenant.
func GetUsersHandler(c *gin.Context) {
	tenantID := c.GetString("tenantId")
	users, err := services.GetUsersByTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

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

	token, user, err := services.LoginUser(creds.Username, creds.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var tenant models.Tenant
	var tenantsCollection = db.GetCollection("tenants")
	tenantObjID, _ := primitive.ObjectIDFromHex(user.TenantID)
	err = tenantsCollection.FindOne(c.Request.Context(), bson.M{"_id": tenantObjID}).Decode(&tenant)
	if err != nil {
		// This would be a serious internal error if a user exists without a tenant
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve tenant information"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID.Hex(),
			"username": user.Username,
			"role":     user.Role,
		},
		"tenant": gin.H{
			"enabledEntities": tenant.EnabledEntities,
		},
	})
}

// --- Protected Handlers (Require JWT) ---
func CreateEmployeeHandler(c *gin.Context) {
	// --- 1. Fetch Tenant Permissions ---
	tenantID := c.GetString("tenantId")
	tenantObjID, _ := primitive.ObjectIDFromHex(tenantID)
	var tenant models.Tenant
	err := db.GetCollection("tenants").FindOne(c.Request.Context(), bson.M{"_id": tenantObjID}).Decode(&tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify tenant permissions"})
		return
	}

	// --- 2. Bind to a flexible map for validation ---
	var employeeData map[string]interface{}
	if err := c.ShouldBindJSON(&employeeData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// --- 3. Dynamic Validation Logic ---
	// This map defines which employee fields are tied to which tenant entity permissions.
	requiredFieldsMap := map[string]string{
		"locations":          "locationId",
		"departments":        "departmentId",
		"managers":           "managerId",
		"job-roles":          "jobRoleId",
		"employment-types":   "employmentTypeId",
		"teams":              "teamId",
		"cost-centers":       "costCenterId",
		"hardware-assets":    "hardwareAssetId",
		"onboarding-buddies": "onboardingBuddyId",
		"access-levels":      "accessLevelId",
	}

	var missingFields []string
	// Check for fundamental fields first
	if _, ok := employeeData["firstName"]; !ok || employeeData["firstName"] == "" {
		missingFields = append(missingFields, "firstName")
	}
	if _, ok := employeeData["lastName"]; !ok || employeeData["lastName"] == "" {
		missingFields = append(missingFields, "lastName")
	}
	if _, ok := employeeData["email"]; !ok || employeeData["email"] == "" {
		missingFields = append(missingFields, "email")
	}

	// Now, check for dynamic fields based on tenant permissions
	for _, enabledEntity := range tenant.EnabledEntities {
		if fieldName, isRequired := requiredFieldsMap[enabledEntity]; isRequired {
			// Check if the required field is present and not empty in the payload
			if val, ok := employeeData[fieldName]; !ok || val == "" {
				missingFields = append(missingFields, fieldName)
			}
		}
	}

	// --- 4. Return Detailed Errors if Validation Fails ---
	if len(missingFields) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Missing required fields for your plan",
			"missing_fields": missingFields,
		})
		return
	}

	// --- 5. If Validation Passes, Populate the Struct and Create ---
	// We can now safely create the employee struct
	var employee models.Employee
	// This is a simple way to map, for more complex scenarios a library like 'mapstructure' could be used.
	employee.FirstName = employeeData["firstName"].(string)
	employee.LastName = employeeData["lastName"].(string)
	employee.Email = employeeData["email"].(string)

	if val, ok := employeeData["phoneNumber"]; ok {
		employee.PhoneNumber = val.(string)
	} else {
		employee.PhoneNumber = "" // Default to empty if not provided
	}
	if val, ok := employeeData["onboardingDate"]; ok {
		if dateStr, ok := val.(string); ok {
			// Parse the date string into a primitive.DateTime
			parsedTime, err := time.Parse(time.RFC3339, dateStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid onboarding date format. Use RFC3339 format (e.g. 2006-01-02T15:04:05Z)"})
				return
			}
			employee.OnboardingDate = primitive.NewDateTimeFromTime(parsedTime)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "onboardingDate must be a string"})
			return
		}
	} else {
		employee.OnboardingDate = primitive.NewDateTimeFromTime(time.Now()) // Default to now if not provided
	}

	// Assign optional fields if they exist
	if val, ok := employeeData["locationId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for locationId"})
			return
		}
		employee.LocationID = id
	}
	if val, ok := employeeData["departmentId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for departmentId"})
			return
		}
		employee.DepartmentID = id
	}
	if val, ok := employeeData["managerId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for managerId"})
			return
		}
		employee.ManagerID = id
	}
	if val, ok := employeeData["jobRoleId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for jobRoleId"})
			return
		}
		employee.JobRoleID = id
	}
	if val, ok := employeeData["employmentTypeId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for employmentTypeId"})
			return
		}
		employee.EmploymentTypeID = id
	}
	if val, ok := employeeData["teamId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for teamId"})
			return
		}
		employee.TeamID = id
	}
	if val, ok := employeeData["costCenterId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for costCenterId"})
			return
		}
		employee.CostCenterID = id
	}
	if val, ok := employeeData["hardwareAssetId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for hardwareAssetId"})
			return
		}
		employee.HardwareAssetID = id
	}
	if val, ok := employeeData["onboardingBuddyId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for onboardingBuddyId"})
			return
		}
		employee.OnboardingBuddyID = id
	}
	if val, ok := employeeData["accessLevelId"]; ok {
		id, err := primitive.ObjectIDFromHex(val.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format for accessLevelId"})
			return
		}
		employee.AccessLevelID = id
	}

	employee.TenantID = tenantID // Set tenantID from the JWT context

	createdEmployee, err := services.CreateEmployee(&employee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee: " + err.Error()})
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

func GetEmployeeByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	employee, err := services.GetEmployeeByID(id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}
	c.JSON(http.StatusOK, employee)
}

func UpdateEmployeeHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateEmployee(id, tenantID, updateData); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully"})
}

func DeleteEmployeeHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	if err := services.DeleteEmployee(id, tenantID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
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

func GetLocationByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.Location](c.Request.Context(), "locations", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateLocationHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.Location](c.Request.Context(), "locations", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Location updated successfully"})
}

func DeleteLocationHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.Location](c.Request.Context(), "locations", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Location deleted successfully"})
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

func GetDepartmentByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.Department](c.Request.Context(), "departments", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateDepartmentHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.Department](c.Request.Context(), "departments", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Department updated successfully"})
}

func DeleteDepartmentHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.Department](c.Request.Context(), "departments", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Department deleted successfully"})
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

func GetManagerByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.Manager](c.Request.Context(), "managers", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manager not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateManagerHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.Manager](c.Request.Context(), "managers", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Manager updated successfully"})
}

func DeleteManagerHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.Manager](c.Request.Context(), "managers", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Manager deleted successfully"})
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

func GetJobRoleByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.JobRole](c.Request.Context(), "job_roles", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job role not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateJobRoleHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.JobRole](c.Request.Context(), "job_roles", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Job role updated successfully"})
}

func DeleteJobRoleHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.JobRole](c.Request.Context(), "job_roles", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Job role deleted successfully"})
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

func GetEmploymentTypeByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.EmploymentType](c.Request.Context(), "employment_types", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employment type not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateEmploymentTypeHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.EmploymentType](c.Request.Context(), "employment_types", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employment type updated successfully"})
}

func DeleteEmploymentTypeHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.EmploymentType](c.Request.Context(), "employment_types", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Employment type deleted successfully"})
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

func GetTeamByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.Team](c.Request.Context(), "teams", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateTeamHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.Team](c.Request.Context(), "teams", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Team updated successfully"})
}

func DeleteTeamHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.Team](c.Request.Context(), "teams", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Team deleted successfully"})
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

func GetCostCenterByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.CostCenter](c.Request.Context(), "cost_centers", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cost center not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateCostCenterHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.CostCenter](c.Request.Context(), "cost_centers", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cost center updated successfully"})
}

func DeleteCostCenterHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.CostCenter](c.Request.Context(), "cost_centers", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cost center deleted successfully"})
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

func GetHardwareAssetByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.HardwareAsset](c.Request.Context(), "hardware_assets", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hardware asset not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateHardwareAssetHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.HardwareAsset](c.Request.Context(), "hardware_assets", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hardware asset updated successfully"})
}

func DeleteHardwareAssetHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.HardwareAsset](c.Request.Context(), "hardware_assets", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hardware asset deleted successfully"})
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

func GetOnboardingBuddyByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.OnboardingBuddy](c.Request.Context(), "onboarding_buddies", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Onboarding buddy not found"})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateOnboardingBuddyHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.OnboardingBuddy](c.Request.Context(), "onboarding_buddies", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Onboarding buddy updated successfully"})
}

func DeleteOnboardingBuddyHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.OnboardingBuddy](c.Request.Context(), "onboarding_buddies", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Onboarding buddy deleted successfully"})
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

func GetAccessLevelByIDHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	entity, err := services.GetEntityByID[models.AccessLevel](c.Request.Context(), "access_levels", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entity)
}

func UpdateAccessLevelHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	var updateData bson.M
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.UpdateEntity[models.AccessLevel](c.Request.Context(), "access_levels", id, tenantID, updateData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Access level updated successfully"})
}

func DeleteAccessLevelHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	err := services.DeleteEntity[models.AccessLevel](c.Request.Context(), "access_levels", id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Access level deleted successfully"})
}
