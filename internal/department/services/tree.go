package services

import depttree "department-api/internal/department/tree"

type DepartmentFlat struct {
	ID       uint
	Name     string
	ParentID *uint
	Depth    int
}

type DepartmentTreeNode = depttree.Node
