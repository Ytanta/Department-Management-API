package domain

import "errors"

var (
	ErrNotFound                = errors.New("not found")
	ErrDuplicateDepartmentName = errors.New("duplicate department name under parent")
	ErrDepartmentCycle         = errors.New("department move would create a cycle")
	ErrReassignTargetInvalid   = errors.New("invalid reassign target department")
	ErrInvalidInput            = errors.New("invalid input")
)
