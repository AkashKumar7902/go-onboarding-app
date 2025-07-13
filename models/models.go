package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Tenant represents a customer organization, the top-level entity in our multitenant design.
type Tenant struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`           // e.g., "Acme Corporation"
	Status    string             `bson:"status" json:"status"`       // e.g., "active", "suspended", "trial"
	CreatedAt primitive.DateTime `bson:"createdAt" json:"createdAt"`
	EnabledEntities []string           `bson:"enabledEntities" json:"enabledEntities"` // Stores slugs like "locations", "departments", "costs"
}

// User represents a user who can log in and perform actions within a specific tenant.
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"-"` // Omit password from JSON responses for security.
	TenantID string             `bson:"tenantId" json:"tenantId"`
}

// --- Base and Specific Entity Structs ---

// BaseEntity contains fields common to all dynamic entities.
// It is embedded in specific entity structs to keep our code DRY (Don't Repeat Yourself).
// The `bson:",inline"` tag tells the MongoDB driver to flatten the embedded struct's fields
// into the parent struct's document.
type BaseEntity struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name     string             `bson:"name" json:"name"` // e.g., "New York", "Engineering", "Software Engineer"
	TenantID string             `bson:"tenantId" json:"tenantId"`
}

// 1. Location specifies a physical work location.
type Location struct {
	BaseEntity `bson:",inline"` // Embeds ID, Name, TenantID
	Address    string           `bson:"address" json:"address"`
	PostalCode string           `bson:"postalCode" json:"postalCode"`
}

// 2. Department represents a company department.
type Department struct {
	BaseEntity `bson:",inline"`
	Head       string `bson:"head" json:"head"` // Name of the department head
}

// 3. Manager represents a person to whom an employee reports.
type Manager struct {
	BaseEntity `bson:",inline"` // Name here would be the Manager's full name
	Email      string           `bson:"email" json:"email"`
}

// 4. JobRole defines a specific position title.
type JobRole struct {
	BaseEntity  `bson:",inline"`
	Description string `bson:"description" json:"description"`
}

// 5. EmploymentType defines the work arrangement (e.g., "Full-Time", "Part-Time").
type EmploymentType struct {
	BaseEntity `bson:",inline"`
}

// 6. Team represents a specific group within a department (e.g., "Frontend", "Platform").
type Team struct {
	BaseEntity `bson:",inline"`
}

// 7. CostCenter is an accounting entity for tracking expenses.
type CostCenter struct {
	BaseEntity `bson:",inline"`
	Code       string `bson:"code" json:"code"` // e.g., "FIN-404", "ENG-101"
}

// 8. HardwareAsset represents a piece of company equipment assigned to an employee.
type HardwareAsset struct {
	BaseEntity  `bson:",inline"` // Name here would be "MacBook Pro 16 Inch"
	ModelNumber string           `bson:"modelNumber" json:"modelNumber"`
}

// 9. OnboardingBuddy is a peer assigned to help a new hire.
type OnboardingBuddy struct {
	BaseEntity `bson:",inline"` // Name is the buddy's full name
	TeamID     primitive.ObjectID `bson:"teamId,omitempty" json:"teamId,omitempty"` // Optional reference to the buddy's team
}

// 10. AccessLevel defines a permissions or security clearance level.
type AccessLevel struct {
	BaseEntity `bson:",inline"` // e.g., "Standard User", "Admin", "Restricted"
}


// --- The Core Employee Struct ---

// Employee is the main entity for onboarding.
// It links together all the other entities via their ObjectIDs.
type Employee struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName      string             `bson:"firstName" json:"firstName"`
	LastName       string             `bson:"lastName" json:"lastName"`
	Email          string             `bson:"email" json:"email"`
	PhoneNumber    string             `bson:"phoneNumber" json:"phoneNumber"`
	OnboardingDate primitive.DateTime `bson:"onboardingDate" json:"onboardingDate"`
	TenantID       string             `bson:"tenantId" json:"tenantId"`

	// --- Dynamic Field References ---
	// These fields store the `_id` of a document from their respective collections.
	LocationID        primitive.ObjectID `bson:"locationId" json:"locationId"`
	DepartmentID      primitive.ObjectID `bson:"departmentId" json:"departmentId"`
	ManagerID         primitive.ObjectID `bson:"managerId" json:"managerId"`
	JobRoleID         primitive.ObjectID `bson:"jobRoleId" json:"jobRoleId"`
	EmploymentTypeID  primitive.ObjectID `bson:"employmentTypeId" json:"employmentTypeId"`
	TeamID            primitive.ObjectID `bson:"teamId" json:"teamId"`
	CostCenterID      primitive.ObjectID `bson:"costCenterId" json:"costCenterId"`
	HardwareAssetID   primitive.ObjectID `bson:"hardwareAssetId" json:"hardwareAssetId"`
	OnboardingBuddyID primitive.ObjectID `bson:"onboardingBuddyId" json:"onboardingBuddyId"`
	AccessLevelID     primitive.ObjectID `bson:"accessLevelId" json:"accessLevelId"`
}

