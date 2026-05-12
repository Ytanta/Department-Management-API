-- +goose Up
-- +goose StatementBegin
CREATE TABLE employees (
    id             BIGSERIAL PRIMARY KEY,
    department_id  BIGINT NOT NULL REFERENCES departments (id) ON UPDATE CASCADE ON DELETE CASCADE,
    full_name      VARCHAR(255) NOT NULL,
    position       VARCHAR(255) NOT NULL,
    hired_at       DATE NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_employees_department_id ON employees (department_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS employees;
-- +goose StatementEnd