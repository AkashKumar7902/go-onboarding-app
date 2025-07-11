package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/your-username/onboarding/config"
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
