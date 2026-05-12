package httpserver

import (
	"department-api/internal/domain"
	"encoding/json"
	"errors"
	"net/http"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorResponse{Code: code, Message: msg})
}

func mapServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")

	case errors.Is(err, domain.ErrDuplicateDepartmentName):
		writeError(w, http.StatusConflict, "conflict", "department name already exists under parent")

	case errors.Is(err, domain.ErrDepartmentCycle):
		writeError(w, http.StatusConflict, "conflict", "cycle detected in department tree")

	case errors.Is(err, domain.ErrReassignTargetInvalid):
		writeError(w, http.StatusBadRequest, "bad_request", "invalid reassign target department")

	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "bad_request", "invalid input")

	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}

	return true
}
