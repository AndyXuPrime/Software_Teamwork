-- +goose Up
CREATE TABLE IF NOT EXISTS model_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('chat', 'embedding', 'rerank')),
    provider TEXT NOT NULL CHECK (provider IN ('openai_compatible', 'siliconflow', 'local_compatible')),
    base_url TEXT NOT NULL,
    model TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    timeout_ms INTEGER NOT NULL CHECK (timeout_ms >= 1000),
    api_key_configured BOOLEAN NOT NULL DEFAULT false,
    supports_streaming BOOLEAN NOT NULL DEFAULT false,
    dimensions INTEGER CHECK (dimensions IS NULL OR dimensions > 0),
    top_n INTEGER CHECK (top_n IS NULL OR top_n > 0),
    default_parameters_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    credential_id TEXT,
    created_by_user_id TEXT,
    updated_by_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_model_profiles_purpose_shape CHECK (
        (purpose = 'chat') OR
        (purpose = 'embedding' AND dimensions IS NOT NULL AND supports_streaming = false) OR
        (purpose = 'rerank' AND top_n IS NOT NULL AND supports_streaming = false)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_profiles_enabled_default
    ON model_profiles (purpose)
    WHERE enabled = true AND is_default = true AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_profiles_purpose_name_active
    ON model_profiles (purpose, lower(name))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_model_profiles_purpose_enabled
    ON model_profiles (purpose, enabled)
    WHERE deleted_at IS NULL;
