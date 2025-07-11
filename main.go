package main

import (
	"log"

	"github.com/your-username/onboarding/api"
	"github.com/your-username/onboarding/config"
	"github.com/your-username/onboarding/db"
)

func main() {
	// 1. Load Configuration from .env file or environment variables.
	config.LoadConfig()

	// 2. Initialize Database connection.
	db.InitDB()

	// 3. Setup the Gin router with all our defined routes.
	router := api.SetupRouter()

	// 4. Start the server.
	log.Println("Starting server on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
