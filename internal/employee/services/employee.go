package employeeservice

import (
	"context"
	"strings"
	"time"

	"department-api/internal/domain"

	"gorm.io/gorm"
)

type Service struct {
	db   *gorm.DB
	emp  EmployeeStore
	dept DepartmentGetter
}

func New(db *gorm.DB, emp EmployeeStore, dept DepartmentGetter) *Service {
	return &Service{db: db, emp: emp, dept: dept}
}

func (s *Service) CreateEmployee(
	ctx context.Context,
	departmentID uint,
	fullName, position string,
	hiredAt time.Time,
) (*domain.Employee, error) {

	fullName = strings.TrimSpace(fullName)
	position = strings.TrimSpace(position)

	if fullName == "" || position == "" {
		return nil, domain.ErrInvalidInput
	}

	if hiredAt.IsZero() {
		return nil, domain.ErrInvalidInput
	}

	var created *domain.Employee

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d, err := s.dept.GetByID(ctx, tx, departmentID)
		if err != nil {
			return err
		}
		if d == nil {
			return domain.ErrNotFound
		}

		e := &domain.Employee{
			DepartmentID: departmentID,
			FullName:     fullName,
			Position:     position,
			HiredAt:      hiredAt,
			CreatedAt:    time.Now().UTC(),
		}

		if err := s.emp.CreateEmployee(ctx, tx, e); err != nil {
			return err
		}

		created = e
		return nil
	})

	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetEmployee(ctx context.Context, id uint) (*domain.Employee, error) {
	emp, err := s.emp.GetEmployeeByID(ctx, s.db, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	if emp == nil {
		return nil, domain.ErrNotFound
	}

	return emp, nil
}

func (s *Service) UpdateEmployee(ctx context.Context, id uint, fullName, position string) error {
	updates := make(map[string]interface{})

	if fullName != "" {
		updates["full_name"] = strings.TrimSpace(fullName)
	}
	if position != "" {
		updates["position"] = strings.TrimSpace(position)
	}

	return s.emp.UpdateEmployee(ctx, s.db, id, updates)
}

func (s *Service) DeleteEmployee(ctx context.Context, id uint) error {
	return s.emp.DeleteEmployee(ctx, s.db, id)
}
