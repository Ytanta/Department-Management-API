package httpserver

import (
	"net/http"

	deptsvc "department-api/internal/department/services"
	employeesvc "department-api/internal/employee/services"
)

func RegisterRoutes(mux *http.ServeMux, dept *deptsvc.DepartmentService, emp *employeesvc.Service) {
	dh := &DepartmentHandler{Svc: dept}
	eh := &EmployeeHandler{Svc: emp}

	mux.HandleFunc("GET /employees/{emp_id}", eh.GetEmployee)
	mux.HandleFunc("PATCH /employees/{emp_id}", eh.PatchEmployee)
	mux.HandleFunc("DELETE /employees/{emp_id}", eh.DeleteEmployee)

	mux.HandleFunc("POST /departments", dh.PostDepartment)
	mux.HandleFunc("GET /departments/{id}", dh.GetDepartment)
	mux.HandleFunc("PATCH /departments/{id}", dh.PatchDepartment)
	mux.HandleFunc("DELETE /departments/{id}", dh.DeleteDepartment)

	mux.HandleFunc("POST /departments/{id}/employees", eh.PostEmployee)
}
