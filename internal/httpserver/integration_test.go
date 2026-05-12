//go:build integration

package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	deptsvc "department-api/internal/department/services"
	"department-api/internal/domain"
	employeesvc "department-api/internal/employee/services"
	"department-api/internal/httpserver"
	"department-api/internal/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()

	ctx := context.Background()

	pgC, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("dept_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgC.Terminate(context.WithoutCancel(ctx))
	})

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, db.AutoMigrate(&domain.Department{}, &domain.Employee{}))

	store := persistence.NewStore(db)
	deptSvc := deptsvc.NewDepartmentService(db, store)
	empSvc := employeesvc.New(db, store, store)

	mux := http.NewServeMux()
	httpserver.RegisterRoutes(mux, deptSvc, empSvc)

	return httptest.NewServer(mux)
}

type departmentCreateResp struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	ParentID  *uint  `json:"parent_id"`
	CreatedAt string `json:"created_at"`
}

type errorResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func patchJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func get(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	return resp
}

func deleteURL(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestIntegration_CreateDepartment(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/departments", map[string]any{
		"name": "Engineering",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var out departmentCreateResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.NotZero(t, out.ID)
	assert.Equal(t, "Engineering", out.Name)
	assert.Nil(t, out.ParentID)
}

func TestIntegration_CreateEmployee(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	d := postJSON(t, srv.URL+"/departments", map[string]any{"name": "HR"})
	defer d.Body.Close()
	require.Equal(t, http.StatusCreated, d.StatusCode)
	var dept departmentCreateResp
	require.NoError(t, json.NewDecoder(d.Body).Decode(&dept))

	resp := postJSON(t, srv.URL+"/departments/"+uintToStr(dept.ID)+"/employees", map[string]any{
		"full_name": "Jane Doe",
		"position":  "Manager",
		"hired_at":  "2024-01-15",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var emp struct {
		ID           uint   `json:"id"`
		DepartmentID uint   `json:"department_id"`
		FullName     string `json:"full_name"`
		Position     string `json:"position"`
		HiredAt      string `json:"hired_at"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&emp))
	assert.NotZero(t, emp.ID)
	assert.Equal(t, dept.ID, emp.DepartmentID)
	assert.Equal(t, "Jane Doe", emp.FullName)
}

func TestIntegration_MoveDepartmentCycle(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	a := postJSON(t, srv.URL+"/departments", map[string]any{"name": "A"})
	defer a.Body.Close()
	require.Equal(t, http.StatusCreated, a.StatusCode)
	var deptA departmentCreateResp
	require.NoError(t, json.NewDecoder(a.Body).Decode(&deptA))

	b := postJSON(t, srv.URL+"/departments", map[string]any{
		"name":      "B",
		"parent_id": deptA.ID,
	})
	defer b.Body.Close()
	require.Equal(t, http.StatusCreated, b.StatusCode)
	var deptB departmentCreateResp
	require.NoError(t, json.NewDecoder(b.Body).Decode(&deptB))

	resp := patchJSON(t, srv.URL+"/departments/"+uintToStr(deptA.ID), map[string]any{
		"parent_id": deptB.ID,
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var er errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "conflict", er.Code)
}

func TestIntegration_DeleteDepartmentCascade(t *testing.T) {
	srv := newIntegrationServer(t)
	defer srv.Close()

	root := postJSON(t, srv.URL+"/departments", map[string]any{"name": "Root"})
	defer root.Body.Close()
	require.Equal(t, http.StatusCreated, root.StatusCode)
	var r departmentCreateResp
	require.NoError(t, json.NewDecoder(root.Body).Decode(&r))

	child := postJSON(t, srv.URL+"/departments", map[string]any{
		"name":      "Child",
		"parent_id": r.ID,
	})
	defer child.Body.Close()
	require.Equal(t, http.StatusCreated, child.StatusCode)
	var c departmentCreateResp
	require.NoError(t, json.NewDecoder(child.Body).Decode(&c))

	emp := postJSON(t, srv.URL+"/departments/"+uintToStr(c.ID)+"/employees", map[string]any{
		"full_name": "Worker",
		"position":  "IC",
		"hired_at":  "2023-06-01",
	})
	defer emp.Body.Close()
	require.Equal(t, http.StatusCreated, emp.StatusCode)

	del := deleteURL(t, srv.URL+"/departments/"+uintToStr(r.ID)+"?mode=cascade")
	defer del.Body.Close()
	require.Equal(t, http.StatusNoContent, del.StatusCode)

	getRoot := get(t, srv.URL+"/departments/"+uintToStr(r.ID)+"?include_employees=false")
	defer getRoot.Body.Close()
	require.Equal(t, http.StatusNotFound, getRoot.StatusCode)

	getChild := get(t, srv.URL+"/departments/"+uintToStr(c.ID)+"?include_employees=false")
	defer getChild.Body.Close()
	require.Equal(t, http.StatusNotFound, getChild.StatusCode)
}

func uintToStr(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
