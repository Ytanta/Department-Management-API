package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"department-api/internal/department/tree"
	"department-api/internal/domain"

	"gorm.io/gorm"
)

type DeleteMode string

const (
	DeleteModeCascade  DeleteMode = "cascade"
	DeleteModeReassign DeleteMode = "reassign"
)

type DeleteDepartmentOptions struct {
	Mode                    DeleteMode
	ReassignEmployeesTo     uint
	PromoteChildrenParentID *uint
}

type DepartmentService struct {
	db    *gorm.DB
	store DepartmentStore
}

func New(db *gorm.DB, store DepartmentStore) *DepartmentService {
	return &DepartmentService{db: db, store: store}
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, name string, parentID *uint) (*domain.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	var created *domain.Department

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if parentID != nil {
			parent, err := s.store.GetByID(ctx, tx, *parentID)
			if err != nil {
				return err
			}
			if parent == nil {
				return domain.ErrNotFound
			}
		}

		dup, err := s.store.ExistsDepartmentWithNameUnderParent(ctx, tx, name, parentID, nil)
		if err != nil {
			return err
		}
		if dup {
			return domain.ErrDuplicateDepartmentName
		}

		now := time.Now().UTC()

		d := &domain.Department{
			Name:      name,
			ParentID:  parentID,
			CreatedAt: now,
		}

		if err := s.store.Create(ctx, tx, d); err != nil {
			return err
		}

		created = d
		return nil
	})

	return created, err
}

func (s *DepartmentService) MoveDepartment(ctx context.Context, id uint, newParentID *uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		node, err := s.store.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return domain.ErrNotFound
		}

		if newParentID != nil {
			if *newParentID == id {
				return domain.ErrDepartmentCycle
			}

			parent, err := s.store.GetByID(ctx, tx, *newParentID)
			if err != nil {
				return err
			}
			if parent == nil {
				return domain.ErrNotFound
			}

			desc, err := s.store.IsStrictDescendant(ctx, tx, id, *newParentID)
			if err != nil {
				return err
			}
			if desc {
				return domain.ErrDepartmentCycle
			}
		}

		return s.store.UpdateParentID(ctx, tx, id, newParentID)
	})
}

func (s *DepartmentService) GetDepartmentTree(ctx context.Context, rootID uint, includeEmployees bool, maxDepth int, sortBy string) (*tree.Node, error) {
	node, err := tree.Load(ctx, s.db, rootID, maxDepth, includeEmployees, sortBy)
	if err != nil {
		if errors.Is(err, tree.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return node, nil
}

type PatchDepartmentInput struct {
	Name      *string
	HasParent bool
	ParentID  *uint
}

func (s *DepartmentService) PatchDepartment(ctx context.Context, id uint, in PatchDepartmentInput) (*domain.Department, error) {
	var out *domain.Department

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		node, err := s.store.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return domain.ErrNotFound
		}

		finalName := node.Name
		if in.Name != nil {
			finalName = strings.TrimSpace(*in.Name)
			if finalName == "" {
				return domain.ErrInvalidInput
			}
		}

		finalParent := node.ParentID

		if in.HasParent {
			finalParent = in.ParentID

			if in.ParentID != nil {
				if *in.ParentID == id {
					return domain.ErrDepartmentCycle
				}

				desc, err := s.store.IsStrictDescendant(ctx, tx, id, *in.ParentID)
				if err != nil {
					return err
				}
				if desc {
					return domain.ErrDepartmentCycle
				}
			}
		}

		nameChanged := in.Name != nil && finalName != node.Name
		parentChanged := in.HasParent

		if nameChanged || parentChanged {
			dup, err := s.store.ExistsDepartmentWithNameUnderParent(ctx, tx, finalName, finalParent, &id)
			if err != nil {
				return err
			}
			if dup {
				return domain.ErrDuplicateDepartmentName
			}
		}

		if in.HasParent {
			if err := s.store.UpdateParentID(ctx, tx, id, finalParent); err != nil {
				return err
			}
		}

		if in.Name != nil {
			if err := s.store.UpdateDepartmentName(ctx, tx, id, finalName); err != nil {
				return err
			}
		}

		updated, err := s.store.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}

		out = updated
		return nil
	})

	return out, err
}

func (s *DepartmentService) DeleteDepartment(ctx context.Context, id uint, opts DeleteDepartmentOptions) error {
	switch opts.Mode {
	case DeleteModeCascade:
		res := s.db.WithContext(ctx).Delete(&domain.Department{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil

	case DeleteModeReassign:
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if opts.ReassignEmployeesTo == 0 {
				return errors.New("reassign requires ReassignEmployeesTo")
			}

			if opts.ReassignEmployeesTo == id {
				return domain.ErrReassignTargetInvalid
			}

			var target domain.Department
			if err := tx.First(&target, opts.ReassignEmployeesTo).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}

			var currentDept domain.Department
			if err := tx.First(&currentDept, id).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}

			if err := tx.Model(&domain.Employee{}).
				Where("department_id = ?", id).
				Update("department_id", opts.ReassignEmployeesTo).Error; err != nil {
				return err
			}

			var newParent *uint
			if opts.PromoteChildrenParentID != nil {
				newParent = opts.PromoteChildrenParentID
			} else {
				newParent = currentDept.ParentID
			}

			if newParent != nil && *newParent == id {
				return domain.ErrReassignTargetInvalid
			}

			if err := tx.Model(&domain.Department{}).
				Where("parent_id = ?", id).
				Update("parent_id", newParent).Error; err != nil {
				return err
			}

			return tx.Delete(&domain.Department{}, id).Error
		})

	default:
		return errors.New("unknown delete mode")
	}
}
