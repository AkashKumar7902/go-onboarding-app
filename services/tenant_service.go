package services

import (
	"context"
	"errors"
	"time"

	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"github.com/your-username/onboarding/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TenantSignupData holds the information from the public signup form.
type TenantSignupData struct {
	CompanyName string `json:"companyName" binding:"required"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

// CreateTenantAndAdminUser creates a new tenant and its initial admin user.
// This is an atomic operation: both must succeed, or neither should.
// In a production system, this should be wrapped in a MongoDB Session (transaction).
func CreateTenantAndAdminUser(signupData *TenantSignupData) (*models.Tenant, *models.User, error) {
	var tenantCollection = db.GetCollection("tenants")
	var usersCollection = db.GetCollection("users")
	// 1. Create the Tenant
	defaultEntities := []string{
		"employees",
		"locations",
		"departments",
		"managers",
		"job-roles",
		"employement-types",
		"teams",
		"costs",
		"hardware-assets",
		"onboarding-buddy",
		"access-levels",
	}
	newTenant := &models.Tenant{
		ID:              primitive.NewObjectID(),
		Name:            signupData.CompanyName,
		Status:          "active",
		CreatedAt:       primitive.NewDateTimeFromTime(time.Now()),
		EnabledEntities: defaultEntities,
	}

	_, err := tenantCollection.InsertOne(context.Background(), newTenant)
	if err != nil {
		return nil, nil, errors.New("failed to create tenant")
	}

	// 2. Create the first User (Admin) for this Tenant
	hashedPassword, err := utils.HashPassword(signupData.Password)
	if err != nil {
		// If hashing fails, we must roll back the tenant creation to prevent orphaned data.
		tenantCollection.DeleteOne(context.Background(), bson.M{"_id": newTenant.ID})
		return nil, nil, errors.New("failed to process user credentials")
	}

	adminUser := &models.User{
		ID:       primitive.NewObjectID(),
		Username: signupData.Username,
		Password: hashedPassword,
		// CRUCIAL: Assign the new tenant's ID (as a hex string) to this user.
		TenantID: newTenant.ID.Hex(),
		Role:     "admin",
	}

	_, err = usersCollection.InsertOne(context.Background(), adminUser)
	if err != nil {
		// If user creation fails, roll back the tenant creation.
		tenantCollection.DeleteOne(context.Background(), bson.M{"_id": newTenant.ID})
		// Check for duplicate key error on username
		if mongoErr, ok := err.(mongo.WriteException); ok && mongoErr.WriteErrors[0].Code == 11000 {
			return nil, nil, errors.New("username already exists")
		}
		return nil, nil, errors.New("failed to create admin user")
	}

	return newTenant, adminUser, nil
}
