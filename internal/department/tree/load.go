package tree

import (
	"context"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
)

const MaxDepth = 5

var ErrNotFound = errors.New("department not found")

func ClampMaxDepth(requested int) int {
	if requested <= 0 {
		return MaxDepth
	}
	if requested > MaxDepth {
		return MaxDepth
	}
	return requested
}

type Node struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uint      `json:"parent_id,omitempty"`
	Depth     int        `json:"depth"`
	Employees []Employee `json:"employees,omitempty"`
	Children  []Node     `json:"children,omitempty"`
}

type Employee struct {
	ID           uint   `json:"id"`
	DepartmentID uint   `json:"department_id"`
	FullName     string `json:"full_name"`
	Position     string `json:"position"`
	HiredAt      string `json:"hired_at"`
}

type deptFlat struct {
	ID       uint   `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	ParentID *uint  `gorm:"column:parent_id"`
	Depth    int    `gorm:"column:depth"`
}

type employeeRow struct {
	ID           uint      `gorm:"column:id"`
	DepartmentID uint      `gorm:"column:department_id"`
	FullName     string    `gorm:"column:full_name"`
	Position     string    `gorm:"column:position"`
	HiredAt      time.Time `gorm:"column:hired_at"`
}

func Load(ctx context.Context, db *gorm.DB, rootID uint, maxDepth int, includeEmployees bool) (*Node, error) {
	md := ClampMaxDepth(maxDepth)

	var flat []deptFlat
	if err := db.WithContext(ctx).Raw(`
WITH RECURSIVE sub AS (
	SELECT id, name, parent_id, 0::int AS depth
	FROM departments
	WHERE id = ?
	UNION ALL
	SELECT d.id, d.name, d.parent_id, s.depth + 1
	FROM departments d
	INNER JOIN sub s ON d.parent_id = s.id
	WHERE s.depth < ?
)
SELECT id, name, parent_id, depth
FROM sub
ORDER BY depth ASC, id ASC
`, rootID, md).Scan(&flat).Error; err != nil {
		return nil, err
	}

	if len(flat) == 0 {
		return nil, ErrNotFound
	}

	var empByDept map[uint][]Employee
	if includeEmployees {
		ids := make([]uint, len(flat))
		for i := range flat {
			ids[i] = flat[i].ID
		}

		m, err := loadEmployeesByDepartmentIDs(ctx, db, ids)
		if err != nil {
			return nil, err
		}
		empByDept = m
	}

	root, ok := buildTreeRecursive(rootID, flat, empByDept, includeEmployees)
	if !ok {
		return nil, ErrNotFound
	}

	return root, nil
}

func loadEmployeesByDepartmentIDs(ctx context.Context, db *gorm.DB, departmentIDs []uint) (map[uint][]Employee, error) {
	out := make(map[uint][]Employee)

	if len(departmentIDs) == 0 {
		return out, nil
	}

	var rows []employeeRow
	err := db.WithContext(ctx).
		Table("employees").
		Select("id", "department_id", "full_name", "position", "hired_at").
		Where("department_id IN ?", departmentIDs).
		Order("full_name ASC, id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		out[r.DepartmentID] = append(out[r.DepartmentID], Employee{
			ID:           r.ID,
			DepartmentID: r.DepartmentID,
			FullName:     r.FullName,
			Position:     r.Position,
			HiredAt:      r.HiredAt.Format("2006-01-02"),
		})
	}

	return out, nil
}

func buildTreeRecursive(rootID uint, flat []deptFlat, emps map[uint][]Employee, includeEmployees bool) (*Node, bool) {
	var rootRow *deptFlat
	childrenMap := make(map[uint][]deptFlat)

	for i := range flat {
		if flat[i].ID == rootID {
			rootRow = &flat[i]
			continue
		}

		if flat[i].ParentID != nil {
			childrenMap[*flat[i].ParentID] = append(childrenMap[*flat[i].ParentID], flat[i])
		}
	}

	if rootRow == nil {
		return nil, false
	}

	for pid := range childrenMap {
		sort.Slice(childrenMap[pid], func(i, j int) bool {
			return childrenMap[pid][i].ID < childrenMap[pid][j].ID
		})
	}

	res := buildNode(*rootRow, childrenMap, emps, includeEmployees)
	return &res, true
}

func buildNode(node deptFlat, children map[uint][]deptFlat, emps map[uint][]Employee, includeEmployees bool) Node {
	out := Node{
		ID:       node.ID,
		Name:     node.Name,
		ParentID: node.ParentID,
		Depth:    node.Depth,
	}

	if includeEmployees {
		if list := emps[node.ID]; len(list) > 0 {
			out.Employees = append([]Employee(nil), list...)
		}
	}

	for _, ch := range children[node.ID] {
		out.Children = append(out.Children, buildNode(ch, children, emps, includeEmployees))
	}

	return out
}
