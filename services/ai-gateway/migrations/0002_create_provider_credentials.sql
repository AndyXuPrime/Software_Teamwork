-- +goose Up
CREATE TABLE IF NOT EXISTS provider_credentials (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES model_profiles(id) ON DELETE RESTRICT,
    storage_mode TEXT NOT NULL CHECK (storage_mode = 'encrypted_column'),
    ciphertext BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    encryption_key_version TEXT NOT NULL,
    fingerprint_sha256 TEXT NOT NULL,
    key_last4 TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'rotated', 'disabled', 'deleted')),
    created_by_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_provider_credentials_active_profile
    ON provider_credentials (profile_id)
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_provider_credentials_profile_status
    ON provider_credentials (profile_id, status);
