-- +goose Up
-- +goose StatementBegin

CREATE TABLE departments (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    parent_id   BIGINT REFERENCES departments (id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_departments_parent_name
    ON departments (COALESCE(parent_id, 0), name);

CREATE INDEX idx_departments_parent_id
    ON departments(parent_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS departments;

-- +goose StatementEnd