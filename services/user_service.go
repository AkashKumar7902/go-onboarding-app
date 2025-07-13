package services

import (
	"context"
	"errors"

	"github.com/your-username/onboarding/auth"
	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"github.com/your-username/onboarding/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// LoginUser verifies a user's credentials and returns a JWT on success.
func LoginUser(username, password string) (string, error) {
	var usersCollection = db.GetCollection("users")

	var user models.User
	err := usersCollection.FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		// User not found. Return a generic error to prevent username enumeration.
		return "", errors.New("invalid username or password")
	}

	// Check if the provided password matches the stored hash.
	if !utils.CheckPasswordHash(password, user.Password) {
		return "", errors.New("invalid username or password")
	}

	// If credentials are valid, generate a JWT containing the user's ID and their tenant ID.
	return auth.GenerateToken(user.ID.Hex(), user.TenantID)
}

// CreateUserData holds the information needed to create a new user.
type CreateUserData struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// CreateUserForTenant creates a new user associated with an existing tenant.
func CreateUserForTenant(data *CreateUserData, tenantID string) (*models.User, error) {
	var usersCollection = db.GetCollection("users")

	// You might want to add validation for the role type here
	if data.Role != "admin" && data.Role != "member" {
		return nil, errors.New("invalid role specified")
	}

	hashedPassword, err := utils.HashPassword(data.Password)
	if err != nil {
		return nil, errors.New("failed to process user credentials")
	}

	newUser := &models.User{
		ID:       primitive.NewObjectID(),
		Username: data.Username,
		Password: hashedPassword,
		TenantID: tenantID,
		Role:     data.Role,
	}

	_, err = usersCollection.InsertOne(context.Background(), newUser)
	if err != nil {
		// Check for duplicate username error
		if mongoErr, ok := err.(mongo.WriteException); ok && mongoErr.WriteErrors[0].Code == 11000 {
			return nil, errors.New("username already exists")
		}
		return nil, errors.New("failed to create user")
	}

	return newUser, nil
}

// GetUsersByTenant fetches all users associated with a specific tenant.
func GetUsersByTenant(tenantID string) ([]models.User, error) {
	var usersCollection = db.GetCollection("users")

	var users []models.User
	cursor, err := usersCollection.Find(context.Background(), bson.M{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &users); err != nil {
		return nil, err
	}
	return users, nil
}
