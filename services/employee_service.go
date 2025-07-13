package services

import (
	"context"
	"errors"

	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateEmployee creates a new employee record.
func CreateEmployee(employee *models.Employee) (*models.Employee, error) {
	var employeeCollection = db.GetCollection("employees")
	employee.ID = primitive.NewObjectID()
	_, err := employeeCollection.InsertOne(context.Background(), employee)
	if err != nil {
		return nil, err
	}
	return employee, nil
}

// GetEmployeesByTenant fetches all employees associated with a specific tenant.
func GetEmployeesByTenant(tenantID string) ([]models.Employee, error) {
	var employeeCollection = db.GetCollection("employees")
	var employees []models.Employee
	cursor, err := employeeCollection.Find(context.Background(), bson.M{"tenantId": tenantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	if err = cursor.All(context.Background(), &employees); err != nil {
		return nil, err
	}
	return employees, nil
}

// Note: GetEmployeeByID, UpdateEmployee, and DeleteEmployee would be implemented here as well.
// GetEmployeeByID fetches a single employee by ID, scoped to the tenant.
func GetEmployeeByID(id, tenantID string) (*models.Employee, error) {
	var employeeCollection = db.GetCollection("employees")
	objID, _ := primitive.ObjectIDFromHex(id)
	var employee models.Employee
	filter := bson.M{"_id": objID, "tenantId": tenantID}
	err := employeeCollection.FindOne(context.Background(), filter).Decode(&employee)
	return &employee, err
}

// UpdateEmployee updates an existing employee's data.
func UpdateEmployee(id, tenantID string, employeeData bson.M) error {
	var employeeCollection = db.GetCollection("employees")
	objID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objID, "tenantId": tenantID}
	update := bson.M{"$set": employeeData}
	result, err := employeeCollection.UpdateOne(context.Background(), filter, update)
	if result.MatchedCount == 0 {
		return errors.New("employee not found or does not belong to this tenant")
	}
	return err
}

// DeleteEmployee deletes an employee record.
func DeleteEmployee(id, tenantID string) error {
	var employeeCollection = db.GetCollection("employees")
	objID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": objID, "tenantId": tenantID}
	result, err := employeeCollection.DeleteOne(context.Background(), filter)
	if result.DeletedCount == 0 {
		return errors.New("employee not found or does not belong to this tenant")
	}
	return err
}
