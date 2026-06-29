-- +goose Up
CREATE TABLE IF NOT EXISTS provider_invocation_attempts (
    id TEXT PRIMARY KEY,
    invocation_id TEXT NOT NULL REFERENCES provider_invocations(id) ON DELETE CASCADE,
    attempt_no INTEGER NOT NULL CHECK (attempt_no >= 1),
    provider TEXT NOT NULL CHECK (provider IN ('openai_compatible', 'siliconflow', 'local_compatible')),
    base_url_host TEXT,
    model TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed', 'cancelled', 'timeout')),
    provider_status_code INTEGER,
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    error_code TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_provider_invocation_attempts_invocation_no
    ON provider_invocation_attempts (invocation_id, attempt_no);

CREATE INDEX IF NOT EXISTS idx_provider_invocation_attempts_started
    ON provider_invocation_attempts (started_at DESC);
