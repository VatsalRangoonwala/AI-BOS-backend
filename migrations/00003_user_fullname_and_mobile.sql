-- +goose Up
ALTER TABLE users RENAME COLUMN name TO full_name;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mobile VARCHAR(30);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS mobile;
ALTER TABLE users RENAME COLUMN full_name TO name;
