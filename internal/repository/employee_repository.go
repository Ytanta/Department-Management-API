package repository

import (
	"context"
	"errors"

	"department-api/internal/domain"

	"gorm.io/gorm"
)

type EmployeeRepository struct {
	baseRepo
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{baseRepo{db: db}}
}

func (r *EmployeeRepository) Create(ctx context.Context, tx *gorm.DB, e *domain.Employee) error {
	return r.conn(tx).WithContext(ctx).Create(e).Error
}

func (r *EmployeeRepository) FindByID(ctx context.Context, tx *gorm.DB, id uint) (*domain.Employee, error) {
	var e domain.Employee
	err := r.conn(tx).WithContext(ctx).Take(&e, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

func (r *EmployeeRepository) FindChildren(ctx context.Context, tx *gorm.DB, departmentID uint) ([]domain.Employee, error) {
	return r.FindByDepartmentID(ctx, tx, departmentID, "full_name")
}

func (r *EmployeeRepository) FindByDepartmentID(ctx context.Context, tx *gorm.DB, departmentID uint, sortBy string) ([]domain.Employee, error) {
	var out []domain.Employee

	orderClause := "full_name ASC, id ASC"
	if sortBy == "created_at" {
		orderClause = "created_at DESC, id ASC"
	}

	err := r.conn(tx).WithContext(ctx).
		Where("department_id = ?", departmentID).
		Order(orderClause).
		Find(&out).Error
	return out, err
}

func (r *EmployeeRepository) Update(ctx context.Context, tx *gorm.DB, e *domain.Employee) error {
	return r.conn(tx).WithContext(ctx).Save(e).Error
}

func (r *EmployeeRepository) UpdateColumns(ctx context.Context, tx *gorm.DB, id uint, cols map[string]interface{}) error {
	if len(cols) == 0 {
		return nil
	}
	return r.conn(tx).WithContext(ctx).Model(&domain.Employee{}).Where("id = ?", id).Updates(cols).Error
}

func (r *EmployeeRepository) Delete(ctx context.Context, tx *gorm.DB, id uint) error {
	return r.conn(tx).WithContext(ctx).Delete(&domain.Employee{}, id).Error
}

func (r *EmployeeRepository) ListByDepartmentIDs(ctx context.Context, tx *gorm.DB, departmentIDs []uint, sortBy string) ([]domain.Employee, error) {
	if len(departmentIDs) == 0 {
		return nil, nil
	}

	orderClause := "department_id ASC, full_name ASC, id ASC"
	if sortBy == "created_at" {
		orderClause = "department_id ASC, created_at DESC, id ASC"
	}

	var out []domain.Employee
	err := r.conn(tx).WithContext(ctx).
		Where("department_id IN ?", departmentIDs).
		Order(orderClause).
		Find(&out).Error
	return out, err
}

func (r *EmployeeRepository) DeleteByDepartmentIDs(ctx context.Context, tx *gorm.DB, departmentIDs []uint) error {
	if len(departmentIDs) == 0 {
		return nil
	}
	return r.conn(tx).WithContext(ctx).
		Where("department_id IN ?", departmentIDs).
		Delete(&domain.Employee{}).Error
}

func (r *EmployeeRepository) ReassignDepartment(ctx context.Context, tx *gorm.DB, fromDepartmentIDs []uint, toDepartmentID uint) error {
	if len(fromDepartmentIDs) == 0 {
		return nil
	}
	return r.conn(tx).WithContext(ctx).Model(&domain.Employee{}).
		Where("department_id IN ?", fromDepartmentIDs).
		Update("department_id", toDepartmentID).Error
}

func (r *EmployeeRepository) CountByDepartment(ctx context.Context, tx *gorm.DB, departmentID uint) (int64, error) {
	var n int64
	err := r.conn(tx).WithContext(ctx).Model(&domain.Employee{}).
		Where("department_id = ?", departmentID).
		Count(&n).Error
	return n, err
}

func (r *EmployeeRepository) ExistsByID(ctx context.Context, tx *gorm.DB, id uint) (bool, error) {
	var count int64
	err := r.conn(tx).WithContext(ctx).Model(&domain.Employee{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *EmployeeRepository) FindFirstByFullNameInDepartment(ctx context.Context, tx *gorm.DB, departmentID uint, fullName string) (*domain.Employee, error) {
	var e domain.Employee
	err := r.conn(tx).WithContext(ctx).
		Where("department_id = ? AND full_name = ?", departmentID, fullName).
		First(&e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
