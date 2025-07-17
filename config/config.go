package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	MongoURI     string
	DatabaseName string
	JwtSecretKey string
}

// AppConfig is a global variable that holds the loaded configuration.
var AppConfig Config

// LoadConfig loads config from .env file or environment variables.
func LoadConfig() {
	// Attempt to load .env file. If it doesn't exist, we'll rely on system env vars.
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, will use environment variables if available")
	}

	AppConfig = Config{
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName: getEnv("DATABASE_NAME", "onboarding_db"),
		JwtSecretKey: getEnv("JWT_SECRET_KEY", "default_secret"),
	}
}

// getEnv is a helper function to read an environment variable or return a fallback value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
