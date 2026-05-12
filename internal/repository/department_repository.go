package repository

import (
	"context"
	"errors"

	"department-api/internal/domain"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	baseRepo
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{baseRepo{db: db}}
}

func (r *DepartmentRepository) Create(ctx context.Context, tx *gorm.DB, d *domain.Department) error {
	return r.conn(tx).WithContext(ctx).Create(d).Error
}

func (r *DepartmentRepository) FindByID(ctx context.Context, tx *gorm.DB, id uint) (*domain.Department, error) {
	var d domain.Department
	err := r.conn(tx).WithContext(ctx).Take(&d, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

// ИСПРАВЛЕНО: Этот метод принадлежит DepartmentRepository и ищет дочерние департаменты
func (r *DepartmentRepository) FindChildren(ctx context.Context, tx *gorm.DB, parentID uint) ([]domain.Department, error) {
	var out []domain.Department
	err := r.conn(tx).WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("id ASC").
		Find(&out).Error
	return out, err
}

func (r *DepartmentRepository) FindByParentID(ctx context.Context, tx *gorm.DB, parentID *uint) ([]domain.Department, error) {
	q := r.conn(tx).WithContext(ctx).Model(&domain.Department{})
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	var out []domain.Department
	err := q.Order("id ASC").Find(&out).Error
	return out, err
}

func (r *DepartmentRepository) Update(ctx context.Context, tx *gorm.DB, d *domain.Department) error {
	return r.conn(tx).WithContext(ctx).Save(d).Error
}

func (r *DepartmentRepository) UpdateColumns(ctx context.Context, tx *gorm.DB, id uint, cols map[string]interface{}) error {
	if len(cols) == 0 {
		return nil
	}
	return r.conn(tx).WithContext(ctx).Model(&domain.Department{}).Where("id = ?", id).Updates(cols).Error
}

func (r *DepartmentRepository) DeleteByIDs(ctx context.Context, tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.conn(tx).WithContext(ctx).Delete(&domain.Department{}, ids).Error
}

func (r *DepartmentRepository) ExistsByNameUnderParent(ctx context.Context, tx *gorm.DB, name string, parentID *uint, excludeID *uint) (bool, error) {
	q := r.conn(tx).WithContext(ctx).Model(&domain.Department{}).Where("name = ?", name)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	if excludeID != nil {
		q = q.Where("id <> ?", *excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DepartmentRepository) IsStrictDescendant(ctx context.Context, tx *gorm.DB, ancestorID, targetID uint) (bool, error) {
	if ancestorID == targetID {
		return false, nil
	}
	var ok bool
	err := r.conn(tx).WithContext(ctx).Raw(`
WITH RECURSIVE sub AS (
    SELECT id, 0 AS dist
    FROM departments
    WHERE id = ?
    UNION ALL
    SELECT d.id, s.dist + 1
    FROM departments d
    INNER JOIN sub s ON d.parent_id = s.id
)
SELECT EXISTS(SELECT 1 FROM sub WHERE id = ? AND dist > 0)
`, ancestorID, targetID).Scan(&ok).Error
	return ok, err
}

type SubtreeFlatRow struct {
	ID       uint
	Name     string
	ParentID *uint
	Depth    int
}

func (r *DepartmentRepository) ListEntireSubtreeFlat(ctx context.Context, tx *gorm.DB, rootID uint) ([]SubtreeFlatRow, error) {
	var rows []SubtreeFlatRow
	err := r.conn(tx).WithContext(ctx).Raw(`
WITH RECURSIVE sub AS (
    SELECT id, name, parent_id, 0::int AS depth
    FROM departments
    WHERE id = ?
    UNION ALL
    SELECT d.id, d.name, d.parent_id, s.depth + 1
    FROM departments d
    INNER JOIN sub s ON d.parent_id = s.id
)
SELECT id, name, parent_id, depth
FROM sub
`, rootID).Scan(&rows).Error
	return rows, err
}

func (r *DepartmentRepository) UpdateDirectChildrenParent(ctx context.Context, tx *gorm.DB, oldParentID uint, newParentID *uint) error {
	return r.conn(tx).WithContext(ctx).Model(&domain.Department{}).
		Where("parent_id = ?", oldParentID).
		Updates(map[string]interface{}{"parent_id": newParentID}).Error
}

func (r *DepartmentRepository) ListByIDs(ctx context.Context, tx *gorm.DB, ids []uint) ([]domain.Department, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []domain.Department
	err := r.conn(tx).WithContext(ctx).Where("id IN ?", ids).Find(&out).Error
	return out, err
}

func (r *DepartmentRepository) CountByParent(ctx context.Context, tx *gorm.DB, parentID uint) (int64, error) {
	var n int64
	err := r.conn(tx).WithContext(ctx).Model(&domain.Department{}).Where("parent_id = ?", parentID).Count(&n).Error
	return n, err
}

func (r *DepartmentRepository) FirstByNameUnderParent(ctx context.Context, tx *gorm.DB, name string, parentID *uint) (*domain.Department, error) {
	q := r.conn(tx).WithContext(ctx).Where("name = ?", name)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	var d domain.Department
	err := q.First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}
