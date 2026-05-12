package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"department-api/internal/domain"
	employeesvc "department-api/internal/employee/services"
)

type EmployeeHandler struct {
	Svc *employeesvc.Service
}

type createEmployeeRequest struct {
	FullName string `json:"full_name"`
	Position string `json:"position"`
	HiredAt  string `json:"hired_at"`
}

type employeeResponse struct {
	ID           uint      `json:"id"`
	DepartmentID uint      `json:"department_id"`
	FullName     string    `json:"full_name"`
	Position     string    `json:"position"`
	HiredAt      string    `json:"hired_at"`
	CreatedAt    time.Time `json:"created_at"`
}

func (h *EmployeeHandler) PostEmployee(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	deptID, err := parseUintPathValue(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid department id")
		return
	}

	var req createEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	hiredAt, err := time.Parse("2006-01-02", strings.TrimSpace(req.HiredAt))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "hired_at must be YYYY-MM-DD")
		return
	}

	e, err := h.Svc.CreateEmployee(r.Context(), deptID, req.FullName, req.Position, hiredAt)
	if mapServiceError(w, err) {
		return
	}

	writeJSON(w, http.StatusCreated, toEmployeeResponse(e))
}

func toEmployeeResponse(e *domain.Employee) employeeResponse {
	return employeeResponse{
		ID:           e.ID,
		DepartmentID: e.DepartmentID,
		FullName:     e.FullName,
		Position:     e.Position,
		HiredAt:      e.HiredAt.Format("2006-01-02"),
		CreatedAt:    e.CreatedAt,
	}
}

func (h *EmployeeHandler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintPathValue(r, "emp_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	e, err := h.Svc.GetEmployee(r.Context(), id)
	if mapServiceError(w, err) {
		return
	}

	writeJSON(w, http.StatusOK, toEmployeeResponse(e))
}

func (h *EmployeeHandler) PatchEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintPathValue(r, "emp_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid employee id")
		return
	}

	var req struct {
		FullName string `json:"full_name"`
		Position string `json:"position"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "could not decode request body")
		return
	}

	err = h.Svc.UpdateEmployee(r.Context(), id, req.FullName, req.Position)
	if mapServiceError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EmployeeHandler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := parseUintPathValue(r, "emp_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	err = h.Svc.DeleteEmployee(r.Context(), id)
	if mapServiceError(w, err) {
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
