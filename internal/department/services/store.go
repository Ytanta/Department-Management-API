package services

import (
	"context"

	"department-api/internal/domain"

	"gorm.io/gorm"
)

type DepartmentStore interface {
	GetByID(ctx context.Context, db *gorm.DB, id uint) (*domain.Department, error)

	ExistsDepartmentWithNameUnderParent(
		ctx context.Context,
		db *gorm.DB,
		name string,
		parentID *uint,
		excludeID *uint,
	) (bool, error)

	Create(ctx context.Context, db *gorm.DB, d *domain.Department) error

	UpdateParentID(ctx context.Context, db *gorm.DB, id uint, parentID *uint) error

	UpdateDepartmentName(ctx context.Context, db *gorm.DB, id uint, name string) error

	IsStrictDescendant(ctx context.Context, db *gorm.DB, ancestorID, targetID uint) (bool, error)

	ListEntireSubtreeFlat(ctx context.Context, db *gorm.DB, rootID uint) ([]DepartmentFlat, error)

	DeleteEmployeesByDepartmentIDs(ctx context.Context, db *gorm.DB, departmentIDs []uint) error

	DeleteDepartmentsByIDs(ctx context.Context, db *gorm.DB, departmentIDs []uint) error

	UpdateDirectChildrenParent(ctx context.Context, db *gorm.DB, oldParentID uint, newParentID *uint) error

	UpdateEmployeesDepartment(ctx context.Context, db *gorm.DB, fromDepartmentIDs []uint, toDepartmentID uint) error
}
