-- +goose Up
CREATE TABLE IF NOT EXISTS model_profile_revisions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES model_profiles(id) ON DELETE RESTRICT,
    revision_no INTEGER NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('created', 'updated', 'credential_rotated', 'disabled', 'deleted', 'default_changed')),
    changed_fields_json JSONB,
    before_snapshot_json JSONB,
    after_snapshot_json JSONB,
    changed_by_user_id TEXT,
    caller_service TEXT,
    request_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_profile_revisions_profile_revision
    ON model_profile_revisions (profile_id, revision_no);

CREATE INDEX IF NOT EXISTS idx_model_profile_revisions_profile_created
    ON model_profile_revisions (profile_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_model_profile_revisions_request_id
    ON model_profile_revisions (request_id);
