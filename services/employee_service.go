package services

import (
	"context"

	"github.com/your-username/onboarding/db"
	"github.com/your-username/onboarding/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var employeeCollection = db.GetCollection("employees")

// CreateEmployee creates a new employee record.
func CreateEmployee(employee *models.Employee) (*models.Employee, error) {
	employee.ID = primitive.NewObjectID()
	_, err := employeeCollection.InsertOne(context.Background(), employee)
	if err != nil {
		return nil, err
	}
	return employee, nil
}

// GetEmployeesByTenant fetches all employees associated with a specific tenant.
func GetEmployeesByTenant(tenantID string) ([]models.Employee, error) {
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
