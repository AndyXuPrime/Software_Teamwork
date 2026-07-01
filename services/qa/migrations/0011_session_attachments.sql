-- +goose Up
CREATE TABLE IF NOT EXISTS session_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    file_ref TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    status TEXT NOT NULL CHECK (status IN ('uploaded', 'parsing', 'ready', 'failed')),
    error_summary TEXT NOT NULL DEFAULT '',
    page_count INTEGER NOT NULL DEFAULT 0,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_session_attachments_session ON session_attachments(conversation_id, external_user_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_session_attachments_expiry ON session_attachments(expires_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS session_attachment_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attachment_id UUID NOT NULL REFERENCES session_attachments(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL,
    page_number INTEGER NOT NULL DEFAULT 0,
    section_path TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    content_preview TEXT NOT NULL DEFAULT '',
    token_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (attachment_id, chunk_index)
);
CREATE INDEX IF NOT EXISTS idx_session_attachment_chunks_lookup ON session_attachment_chunks(conversation_id, attachment_id, chunk_index);

CREATE TABLE IF NOT EXISTS message_attachments (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES session_attachments(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, attachment_id)
);
