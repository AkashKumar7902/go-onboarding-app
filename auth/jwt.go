package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/your-username/onboarding/config"
	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Claims defines the structure of the data we'll store in the JWT payload.
type Claims struct {
	UserID   string `json:"userId"`
	TenantID string `json:"tenantId"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT for a given user and tenant.
func GenerateToken(userID, tenantID string) (string, error) {
	// Set the token's expiration time. Here, we'll use 24 hours.
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt is a NumericDate type, so we need to convert the time.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Create the token with the HS256 signing algorithm and our claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key from the configuration.
	return token.SignedString([]byte(config.AppConfig.JwtSecretKey))
}

// AuthMiddleware is the Gin middleware for authenticating requests.
// It will be applied to all routes that require a user to be logged in.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the token from the Authorization header.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		// The header should be in the format "Bearer <token>".
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format, must be 'Bearer <token>'"})
			return
		}

		// 2. Parse and validate the token.
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// This function provides the key for validation.
			return []byte(config.AppConfig.JwtSecretKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// 3. If the token is valid, set the user and tenant info in the Gin context.
		// This makes the tenantId and userId available to the actual handlers.
		c.Set("tenantId", claims.TenantID)
		c.Set("userId", claims.UserID)

		// 4. Call the next handler in the chain.
		c.Next()
	}
}

// RequireEntityAccess is a middleware factory. It returns a Gin handler that
// checks if the current tenant has permission to access the specified entity.
func RequireEntityAccess(entitySlug string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the tenantId from the context (set by the AuthMiddleware).
		tenantIDStr := c.GetString("tenantId")
		tenantID, _ := primitive.ObjectIDFromHex(tenantIDStr)

		// 2. Fetch the tenant's data from the database.
		// NOTE: In a high-performance production system, you would cache this information
		// after login instead of querying the DB on every request.
		var tenant models.Tenant
		tenantCollection := db.GetCollection("tenants")
		err := tenantCollection.FindOne(c.Request.Context(), bson.M{"_id": tenantID}).Decode(&tenant)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not verify tenant permissions"})
			return
		}

		// 3. Check if the required entitySlug is in the tenant's list.
		isAllowed := false
		for _, enabledEntity := range tenant.EnabledEntities {
			if enabledEntity == entitySlug {
				isAllowed = true
				break
			}
		}

		// 4. If not allowed, block the request with a 403 Forbidden error.
		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Access to this feature is not enabled for your account.",
			})
			return
		}

		// 5. If allowed, proceed to the actual handler.
		c.Next()
	}
}
