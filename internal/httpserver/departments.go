package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	deptsvc "department-api/internal/department/services"
	"department-api/internal/department/tree"
	"department-api/internal/domain"
)

type DepartmentHandler struct {
	Svc *deptsvc.DepartmentService
}

type createDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

type departmentResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	ParentID  *uint     `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *DepartmentHandler) PostDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req createDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	if req.ParentID != nil && *req.ParentID == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "parent_id must be positive or null")
		return
	}

	d, err := h.Svc.CreateDepartment(r.Context(), req.Name, req.ParentID)
	if mapServiceError(w, err) {
		return
	}

	writeJSON(w, http.StatusCreated, toDepartmentResponse(d))
}

func (h *DepartmentHandler) GetDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	id, err := parseUintPathValue(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	q := r.URL.Query()

	depth, err := parseDepthQuery(q.Get("depth"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	includeEmployees, err := parseBoolQuery(q.Get("include_employees"), true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	sortBy := strings.TrimSpace(strings.ToLower(q.Get("sort_by")))
	if sortBy == "" {
		sortBy = "full_name"
	}
	if sortBy != "full_name" && sortBy != "created_at" {
		writeError(w, http.StatusBadRequest, "bad_request", "sort_by must be full_name or created_at")
		return
	}

	node, err := h.Svc.GetDepartmentTree(r.Context(), id, includeEmployees, depth, sortBy)
	if mapServiceError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, node)
}

func (h *DepartmentHandler) PatchDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	id, err := parseUintPathValue(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	in, err := decodePatchDepartment(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if in.Name == nil && !in.HasParent {
		writeError(w, http.StatusBadRequest, "bad_request", "empty patch")
		return
	}

	d, err := h.Svc.PatchDepartment(r.Context(), id, in)
	if mapServiceError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, toDepartmentResponse(d))
}

func (h *DepartmentHandler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	id, err := parseUintPathValue(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	q := r.URL.Query()
	mode := deptsvc.DeleteMode(strings.ToLower(strings.TrimSpace(q.Get("mode"))))
	
	opts := deptsvc.DeleteDepartmentOptions{
		Mode: mode,
	}

	switch mode {
	case deptsvc.DeleteModeCascade:

	case deptsvc.DeleteModeReassign:
		reassignTo, err := parseUintQuery(q.Get("reassign_to_department_id"))
		if err != nil || reassignTo == 0 {
			writeError(w, http.StatusBadRequest, "bad_request", "reassign_to_department_id is required and must be > 0")
			return
		}
		opts.ReassignEmployeesTo = reassignTo

		if raw := strings.TrimSpace(q.Get("promote_children_parent_id")); raw != "" {
			v, err := parseUintQuery(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "invalid promote_children_parent_id")
				return
			}
			opts.PromoteChildrenParentID = &v
		}

	default:
		writeError(w, http.StatusBadRequest, "bad_request", "mode must be cascade or reassign")
		return
	}

	if err := h.Svc.DeleteDepartment(r.Context(), id, opts); mapServiceError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toDepartmentResponse(d *domain.Department) departmentResponse {
	return departmentResponse{
		ID:        d.ID,
		Name:      d.Name,
		ParentID:  d.ParentID,
		CreatedAt: d.CreatedAt,
	}
}

func decodePatchDepartment(r *http.Request) (deptsvc.PatchDepartmentInput, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return deptsvc.PatchDepartmentInput{}, errors.New("invalid json")
	}

	var in deptsvc.PatchDepartmentInput

	if v, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(v, &name); err != nil {
			return deptsvc.PatchDepartmentInput{}, errors.New("invalid name")
		}

		name = strings.TrimSpace(name)
		if name == "" {
			return deptsvc.PatchDepartmentInput{}, errors.New("name cannot be empty")
		}

		in.Name = &name
	}

	if v, ok := raw["parent_id"]; ok {
		in.HasParent = true
		trimmed := bytes.TrimSpace(v)

		if bytes.Equal(trimmed, []byte("null")) {
			in.ParentID = nil
		} else {
			var pid uint
			if err := json.Unmarshal(v, &pid); err != nil {
				return deptsvc.PatchDepartmentInput{}, errors.New("invalid parent_id")
			}
			if pid == 0 {
				return deptsvc.PatchDepartmentInput{}, errors.New("parent_id must be positive or null")
			}
			in.ParentID = &pid
		}
	}

	return in, nil
}

func parseDepthQuery(v string) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 1, nil
	}

	d, err := strconv.Atoi(v)
	if err != nil || d < 0 {
		return 0, errors.New("invalid depth")
	}

	if d > tree.MaxDepth {
		return 0, errors.New("depth exceeds server limit")
	}

	return d, nil
}

func parseBoolQuery(v string, defaultValue bool) (bool, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return defaultValue, nil
	}

	switch v {
	case "false", "0":
		return false, nil
	case "true", "1":
		return true, nil
	default:
		return false, errors.New("invalid boolean value")
	}
}

func parseUintQuery(v string) (uint, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, errors.New("empty")
	}

	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(n), nil
}

func parseUintPathValue(r *http.Request, key string) (uint, error) {
	s := r.PathValue(key)
	if s == "" {
		return 0, errors.New("missing path value")
	}

	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}

	if n == 0 {
		return 0, errors.New("id must be positive")
	}

	return uint(n), nil
}
