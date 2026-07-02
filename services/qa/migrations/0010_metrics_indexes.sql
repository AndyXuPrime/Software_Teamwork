-- +goose NO TRANSACTION
-- +goose Up
-- Index for QA metrics aggregation queries.  The combination (role,
-- created_at) speeds up trend, top-queries, and intent-distribution
-- which all filter on role='user' and a date range.
-- Note: started_at is already covered by idx_response_runs_started_at
-- from migration 0001.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_role_created
    ON messages (role, created_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_role_created;
