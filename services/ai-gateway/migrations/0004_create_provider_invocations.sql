-- +goose Up
CREATE TABLE IF NOT EXISTS provider_invocations (
    id TEXT PRIMARY KEY,
    request_id TEXT,
    caller_service TEXT,
    external_user_id TEXT,
    operation TEXT NOT NULL CHECK (operation IN ('chat_completion', 'embedding', 'reranking')),
    profile_id TEXT NOT NULL REFERENCES model_profiles(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL CHECK (provider IN ('openai_compatible', 'siliconflow', 'local_compatible')),
    model TEXT NOT NULL,
    stream BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed', 'cancelled', 'timeout')),
    provider_status_code INTEGER,
    prompt_tokens INTEGER CHECK (prompt_tokens IS NULL OR prompt_tokens >= 0),
    completion_tokens INTEGER CHECK (completion_tokens IS NULL OR completion_tokens >= 0),
    total_tokens INTEGER CHECK (total_tokens IS NULL OR total_tokens >= 0),
    input_count INTEGER CHECK (input_count IS NULL OR input_count >= 0),
    embedding_dimensions INTEGER CHECK (embedding_dimensions IS NULL OR embedding_dimensions > 0),
    rerank_top_n INTEGER CHECK (rerank_top_n IS NULL OR rerank_top_n > 0),
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    attempt_count INTEGER NOT NULL CHECK (attempt_count >= 0),
    normalized_error_code TEXT,
    normalized_error_type TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_provider_invocations_created
    ON provider_invocations (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_provider_invocations_request_id
    ON provider_invocations (request_id);

CREATE INDEX IF NOT EXISTS idx_provider_invocations_caller_created
    ON provider_invocations (caller_service, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_provider_invocations_profile_created
    ON provider_invocations (profile_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_provider_invocations_operation_status_created
    ON provider_invocations (operation, status, created_at DESC);
