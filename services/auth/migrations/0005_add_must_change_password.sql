-- +goose Up
ALTER TABLE auth_credentials
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE auth_credentials
    DROP COLUMN IF EXISTS must_change_password;
