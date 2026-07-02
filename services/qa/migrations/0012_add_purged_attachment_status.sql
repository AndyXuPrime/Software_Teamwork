-- +goose Up
ALTER TABLE session_attachments
    DROP CONSTRAINT IF EXISTS session_attachments_status_check;
ALTER TABLE session_attachments
    ADD CONSTRAINT session_attachments_status_check
    CHECK (status IN ('uploaded', 'parsing', 'ready', 'failed', 'purged'));

-- +goose Down
UPDATE session_attachments
SET status = 'failed'
WHERE status = 'purged';
ALTER TABLE session_attachments
    DROP CONSTRAINT IF EXISTS session_attachments_status_check;
ALTER TABLE session_attachments
    ADD CONSTRAINT session_attachments_status_check
    CHECK (status IN ('uploaded', 'parsing', 'ready', 'failed'));
