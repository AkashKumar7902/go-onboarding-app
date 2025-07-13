package db

import (
	"context"
	"log"
	"time"

	"github.com/your-username/onboarding/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient is a global variable to hold the database client instance.
var MongoClient *mongo.Client

// InitDB initializes the database connection.
func InitDB() {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the URI from the loaded configuration.
	clientOptions := options.Client().ApplyURI(config.AppConfig.MongoURI)
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v", err)
	}

	// Ping the primary to verify that a connection was established.
	err = MongoClient.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Could not ping MongoDB: %v", err)
	}

	log.Println("Successfully connected to MongoDB!")
}

// GetCollection is a helper function to get a handle for a collection from the database.
func GetCollection(collectionName string) *mongo.Collection {
	return MongoClient.Database(config.AppConfig.DatabaseName).Collection(collectionName)
}
