package employeeservice

import (
	"context"

	"department-api/internal/domain"

	"gorm.io/gorm"
)

type EmployeeStore interface {
	CreateEmployee(ctx context.Context, db *gorm.DB, e *domain.Employee) error
	GetEmployeeByID(ctx context.Context, db *gorm.DB, id uint) (*domain.Employee, error) // Должно быть так
	UpdateEmployee(ctx context.Context, db *gorm.DB, id uint, updates map[string]interface{}) error
	DeleteEmployee(ctx context.Context, db *gorm.DB, id uint) error
}

type DepartmentGetter interface {
	GetByID(ctx context.Context, db *gorm.DB, id uint) (*domain.Department, error)
}
