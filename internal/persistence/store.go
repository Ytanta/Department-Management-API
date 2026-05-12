package persistence

import (
	"context"
	"errors"

	"department-api/internal/department/services"
	"department-api/internal/domain"
	"department-api/internal/repository"

	"gorm.io/gorm"
)

type Store struct {
	dept *repository.DepartmentRepository
	emp  *repository.EmployeeRepository
}

func NewStore(db *gorm.DB) *Store {
	return &Store{
		dept: repository.NewDepartmentRepository(db),
		emp:  repository.NewEmployeeRepository(db),
	}
}

func (s *Store) GetByID(ctx context.Context, db *gorm.DB, id uint) (*domain.Department, error) {
	d, err := s.dept.FindByID(ctx, db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return d, err
}

func (s *Store) ExistsDepartmentWithNameUnderParent(ctx context.Context, db *gorm.DB, name string, parentID *uint, excludeID *uint) (bool, error) {
	return s.dept.ExistsByNameUnderParent(ctx, db, name, parentID, excludeID)
}

func (s *Store) Create(ctx context.Context, db *gorm.DB, d *domain.Department) error {
	return s.dept.Create(ctx, db, d)
}

func (s *Store) UpdateParentID(ctx context.Context, db *gorm.DB, id uint, parentID *uint) error {
	return s.dept.UpdateColumns(ctx, db, id, map[string]interface{}{"parent_id": parentID})
}

func (s *Store) UpdateDepartmentName(ctx context.Context, db *gorm.DB, id uint, name string) error {
	return s.dept.UpdateColumns(ctx, db, id, map[string]interface{}{"name": name})
}

func (s *Store) IsStrictDescendant(ctx context.Context, db *gorm.DB, ancestorID, targetID uint) (bool, error) {
	return s.dept.IsStrictDescendant(ctx, db, ancestorID, targetID)
}

func (s *Store) ListEntireSubtreeFlat(ctx context.Context, db *gorm.DB, rootID uint) ([]services.DepartmentFlat, error) {
	rows, err := s.dept.ListEntireSubtreeFlat(ctx, db, rootID)
	if err != nil {
		return nil, err
	}

	out := make([]services.DepartmentFlat, len(rows))
	for i, r := range rows {
		out[i] = services.DepartmentFlat{
			ID:       r.ID,
			Name:     r.Name,
			ParentID: r.ParentID,
			Depth:    r.Depth,
		}
	}
	return out, nil
}

func (s *Store) DeleteEmployeesByDepartmentIDs(ctx context.Context, db *gorm.DB, departmentIDs []uint) error {
	return s.emp.DeleteByDepartmentIDs(ctx, db, departmentIDs)
}

func (s *Store) DeleteDepartmentsByIDs(ctx context.Context, db *gorm.DB, departmentIDs []uint) error {
	return s.dept.DeleteByIDs(ctx, db, departmentIDs)
}

func (s *Store) UpdateDirectChildrenParent(ctx context.Context, db *gorm.DB, oldParentID uint, newParentID *uint) error {
	return s.dept.UpdateDirectChildrenParent(ctx, db, oldParentID, newParentID)
}

func (s *Store) UpdateEmployeesDepartment(ctx context.Context, db *gorm.DB, fromDepartmentIDs []uint, toDepartmentID uint) error {
	return s.emp.ReassignDepartment(ctx, db, fromDepartmentIDs, toDepartmentID)
}

func (s *Store) CreateEmployee(ctx context.Context, db *gorm.DB, e *domain.Employee) error {
	return s.emp.Create(ctx, db, e)
}

func (s *Store) GetEmployeeByID(ctx context.Context, db *gorm.DB, id uint) (*domain.Employee, error) {
	e, err := s.emp.FindByID(ctx, db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return e, err
}

func (s *Store) UpdateEmployee(ctx context.Context, db *gorm.DB, id uint, updates map[string]interface{}) error {
	return s.emp.UpdateColumns(ctx, db, id, updates)
}

func (s *Store) DeleteEmployee(ctx context.Context, db *gorm.DB, id uint) error {
	return s.emp.Delete(ctx, db, id)
}
