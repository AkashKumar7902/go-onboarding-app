package services

import (
	"context"
	"errors"

	"github.com/your-username/onboarding/auth"
	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"github.com/your-username/onboarding/utils"
	"go.mongodb.org/mongo-driver/bson"
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
