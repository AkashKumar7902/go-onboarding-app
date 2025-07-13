package services

import (
	"context"
	"errors"

	"github.com/your-username/onboarding/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

// GetEntityByID fetches a single document by its ID, ensuring it belongs to the tenant.
func GetEntityByID[T Entity](ctx context.Context, collectionName, id, tenantID string) (*T, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid id format")
	}

	collection := db.GetCollection(collectionName)
	var result T

	filter := bson.M{"_id": objID, "tenantId": tenantID}
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("entity not found or does not belong to this tenant")
		}
		return nil, err
	}

	return &result, nil
}

// UpdateEntity updates a document in a collection.
// It uses bson.M for the update data to allow for partial updates (PATCH-like behavior).
func UpdateEntity[T Entity](ctx context.Context, collectionName, id, tenantID string, updateData bson.M) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id format")
	}

	collection := db.GetCollection(collectionName)
	filter := bson.M{"_id": objID, "tenantId": tenantID}
	update := bson.M{"$set": updateData}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("entity not found or does not belong to this tenant")
	}

	return nil
}

// DeleteEntity deletes a document from a collection.
func DeleteEntity[T Entity](ctx context.Context, collectionName, id, tenantID string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid id format")
	}

	collection := db.GetCollection(collectionName)
	filter := bson.M{"_id": objID, "tenantId": tenantID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("entity not found or does not belong to this tenant")
	}

	return nil
}
