-- +goose Up
-- +goose StatementBegin

CREATE TABLE employees (
    id             BIGSERIAL PRIMARY KEY,
    department_id  BIGINT NOT NULL
        REFERENCES departments (id)
            ON UPDATE CASCADE
            ON DELETE CASCADE,

    full_name      VARCHAR(200) NOT NULL,
    position       VARCHAR(200) NOT NULL,

    hired_at       DATE NULL,
    
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

CHECK (length(full_name) > 0 AND length(full_name) <= 200),
CHECK (length(position) > 0 AND length(position) <= 200)
);

CREATE INDEX idx_employees_department_id
    ON employees (department_id);

CREATE INDEX idx_employees_department_fullname
    ON employees (department_id, full_name);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS employees;

-- +goose StatementEnd