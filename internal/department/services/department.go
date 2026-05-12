package services

import (
	"context"
	"errors"
	"sort"
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
	Mode DeleteMode

	ReassignEmployeesTo uint

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

	if err != nil {
		return nil, err
	}
	return created, nil
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

func (s *DepartmentService) GetDepartmentTree(ctx context.Context, rootID uint, includeEmployees bool, maxDepth int) (*DepartmentTreeNode, error) {
	node, err := tree.Load(ctx, s.db, rootID, maxDepth, includeEmployees)
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
		parentChanged := in.HasParent && !uintPtrEqual(in.ParentID, node.ParentID)

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

func uintPtrEqual(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *DepartmentService) DeleteDepartment(ctx context.Context, id uint, opts DeleteDepartmentOptions) error {
	switch opts.Mode {
	case DeleteModeCascade:
		return s.deleteCascade(ctx, id)
	case DeleteModeReassign:
		return s.deleteReassign(ctx, id, opts)
	default:
		return errors.New("unknown delete mode")
	}
}

func (s *DepartmentService) deleteCascade(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		node, err := s.store.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return domain.ErrNotFound
		}

		flat, err := s.store.ListEntireSubtreeFlat(ctx, tx, id)
		if err != nil {
			return err
		}
		if len(flat) == 0 {
			return domain.ErrNotFound
		}

		ids := departmentIDsSortedByDepthDesc(flat)

		if err := s.store.DeleteEmployeesByDepartmentIDs(ctx, tx, ids); err != nil {
			return err
		}
		return s.store.DeleteDepartmentsByIDs(ctx, tx, ids)
	})
}

func (s *DepartmentService) deleteReassign(ctx context.Context, id uint, opts DeleteDepartmentOptions) error {
	if opts.ReassignEmployeesTo == 0 {
		return errors.New("reassign requires ReassignEmployeesTo")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dept, err := s.store.GetByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if dept == nil {
			return domain.ErrNotFound
		}

		targetEmpDept, err := s.store.GetByID(ctx, tx, opts.ReassignEmployeesTo)
		if err != nil {
			return err
		}
		if targetEmpDept == nil {
			return domain.ErrNotFound
		}
		if opts.ReassignEmployeesTo == id {
			return domain.ErrReassignTargetInvalid
		}

		var promoteTo *uint
		if opts.PromoteChildrenParentID != nil {
			p, err := s.store.GetByID(ctx, tx, *opts.PromoteChildrenParentID)
			if err != nil {
				return err
			}
			if p == nil {
				return domain.ErrNotFound
			}
			promoteTo = opts.PromoteChildrenParentID
		} else {
			promoteTo = dept.ParentID
		}

		if promoteTo != nil && *promoteTo == id {
			return domain.ErrReassignTargetInvalid
		}

		if err := s.store.UpdateEmployeesDepartment(ctx, tx, []uint{id}, opts.ReassignEmployeesTo); err != nil {
			return err
		}
		if err := s.store.UpdateDirectChildrenParent(ctx, tx, id, promoteTo); err != nil {
			return err
		}
		return s.store.DeleteDepartmentsByIDs(ctx, tx, []uint{id})
	})
}

func departmentIDsSortedByDepthDesc(flat []DepartmentFlat) []uint {
	seen := make(map[uint]struct{}, len(flat))
	rows := make([]DepartmentFlat, 0, len(flat))

	for _, r := range flat {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}
		rows = append(rows, r)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Depth != rows[j].Depth {
			return rows[i].Depth > rows[j].Depth
		}
		return rows[i].ID > rows[j].ID
	})

	out := make([]uint, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}
