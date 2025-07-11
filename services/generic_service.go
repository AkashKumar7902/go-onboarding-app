package services

import (
	"context"

	"github.com/your-username/onboarding/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Entity is a constraint that serves as a marker for our generic functions.
// It indicates that the type T is one of our model structs.
type Entity interface {
	// This interface is currently empty but is used as a type constraint.
	// We could enforce methods here if needed, e.g., GetID() string
}

// CreateEntity creates a new document in the specified collection for a given tenant.
// It uses generics to work with any of our specific entity types (e.g., models.Location).
func CreateEntity[T Entity](ctx context.Context, collectionName string, entity *T) (*T, error) {
	collection := db.GetCollection(collectionName)

	// To inject a new ObjectID, we marshal the struct to a BSON map, add the _id, and insert.
	// This avoids complex reflection to set the ID field on a generic type.
	data, err := bson.Marshal(entity)
	if err != nil {
		return nil, err
	}
	var docMap bson.M
	err = bson.Unmarshal(data, &docMap)
	if err != nil {
		return nil, err
	}

	docMap["_id"] = primitive.NewObjectID()

	_, err = collection.InsertOne(ctx, docMap)
	if err != nil {
		return nil, err
	}

	// Unmarshal the map back into the original struct type to return it with the new ID.
	data, err = bson.Marshal(docMap)
	if err != nil {
		return nil, err
	}
	err = bson.Unmarshal(data, entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

// GetEntitiesByTenant fetches all documents from a collection for a specific tenant.
func GetEntitiesByTenant[T Entity](ctx context.Context, collectionName, tenantID string) ([]T, error) {
	collection := db.GetCollection(collectionName)
	var results []T

	cursor, err := collection.Find(ctx, bson.M{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	// To be safe, return an empty slice instead of nil if no documents are found.
	if results == nil {
		return []T{}, nil
	}

	return results, nil
}
