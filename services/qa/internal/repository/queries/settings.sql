-- name: GetActiveQAConfig :one
SELECT
    id::text,
    top_k,
    similarity_threshold,
    use_rerank,
    COALESCE(rerank_threshold, 0.5)::float8 AS rerank_threshold,
    COALESCE(rerank_top_n, 3)::integer AS rerank_top_n
FROM qa_config_versions
WHERE is_active = true
ORDER BY version_no DESC
LIMIT 1;

-- name: ListQAConfigKnowledgeBaseIDs :many
SELECT external_kb_id
FROM qa_config_knowledge_bases
WHERE config_id = sqlc.arg(config_id)::uuid
ORDER BY sort_order, external_kb_id;

-- name: LockQAConfigVersions :exec
LOCK TABLE qa_config_versions IN EXCLUSIVE MODE;

-- name: NextQAConfigVersionNo :one
SELECT COALESCE(MAX(version_no), 0) + 1::integer AS version_no
FROM qa_config_versions;

-- name: DeactivateAllQAConfigs :exec
UPDATE qa_config_versions
SET is_active = false
WHERE is_active = true;

-- name: InsertQAConfigVersion :one
INSERT INTO qa_config_versions (
    version_no,
    top_k,
    similarity_threshold,
    use_rerank,
    rerank_threshold,
    rerank_top_n,
    is_active,
    created_by_user_id
) VALUES (
    sqlc.arg(version_no),
    sqlc.arg(top_k),
    sqlc.arg(similarity_threshold),
    sqlc.arg(use_rerank),
    sqlc.arg(rerank_threshold),
    sqlc.arg(rerank_top_n),
    true,
    sqlc.arg(created_by_user_id)
)
RETURNING id::text;

-- name: InsertQAConfigKnowledgeBase :exec
INSERT INTO qa_config_knowledge_bases (config_id, external_kb_id, sort_order)
VALUES (sqlc.arg(config_id)::uuid, sqlc.arg(external_kb_id), sqlc.arg(sort_order));

-- name: GetActiveLLMConfig :one
SELECT
    id::text,
    provider,
    COALESCE(profile_id, '') AS profile_id,
    COALESCE(api_endpoint, '') AS api_endpoint,
    api_key_encrypted,
    COALESCE(api_key_last4, '') AS api_key_last4,
    token_header,
    model_name,
    timeout_seconds,
    temperature,
    max_tokens
FROM llm_config_versions
WHERE is_active = true
ORDER BY version_no DESC
LIMIT 1;

-- name: LockLLMConfigVersions :exec
LOCK TABLE llm_config_versions IN EXCLUSIVE MODE;

-- name: NextLLMConfigVersionNo :one
SELECT COALESCE(MAX(version_no), 0) + 1::integer AS version_no
FROM llm_config_versions;

-- name: DeactivateAllLLMConfigs :exec
UPDATE llm_config_versions
SET is_active = false
WHERE is_active = true;

-- name: InsertLLMConfigVersion :exec
INSERT INTO llm_config_versions (
    version_no,
    provider,
    profile_id,
    api_endpoint,
    api_key_encrypted,
    api_key_last4,
    token_header,
    model_name,
    timeout_seconds,
    temperature,
    max_tokens,
    is_active,
    created_by_user_id
) VALUES (
    sqlc.arg(version_no),
    'direct',
    NULL,
    sqlc.arg(api_endpoint),
    sqlc.arg(api_key_encrypted),
    sqlc.arg(api_key_last4),
    sqlc.arg(token_header),
    sqlc.arg(model_name),
    sqlc.arg(timeout_seconds),
    sqlc.arg(temperature),
    sqlc.arg(max_tokens),
    true,
    sqlc.arg(created_by_user_id)
);

-- name: GetRuntimeSetting :one
SELECT value
FROM qa_runtime_settings
WHERE key = sqlc.arg(key);

-- name: UpsertRuntimeSetting :exec
INSERT INTO qa_runtime_settings (key, value, updated_at)
VALUES (sqlc.arg(key), sqlc.arg(value), now())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = now();

-- name: ListMCPServers :many
SELECT
    id::text,
    alias,
    display_name,
    transport,
    COALESCE(command, '') AS command,
    args_json,
    COALESCE(endpoint_url, '') AS endpoint_url,
    token_encrypted,
    COALESCE(token_last4, '') AS token_last4,
    token_header,
    tool_timeout_seconds,
    enabled,
    sort_order,
    tool_count,
    last_connected_at,
    COALESCE(last_error, '') AS last_error,
    created_by_user_id,
    created_at,
    updated_at
FROM mcp_servers
ORDER BY sort_order, alias;

-- name: GetMCPServer :one
SELECT
    id::text,
    alias,
    display_name,
    transport,
    COALESCE(command, '') AS command,
    args_json,
    COALESCE(endpoint_url, '') AS endpoint_url,
    token_encrypted,
    COALESCE(token_last4, '') AS token_last4,
    token_header,
    tool_timeout_seconds,
    enabled,
    sort_order,
    tool_count,
    last_connected_at,
    COALESCE(last_error, '') AS last_error,
    created_by_user_id,
    created_at,
    updated_at
FROM mcp_servers
WHERE id = sqlc.arg(id)::uuid;

-- name: InsertMCPServer :one
INSERT INTO mcp_servers (
    alias,
    display_name,
    transport,
    command,
    args_json,
    endpoint_url,
    token_encrypted,
    token_last4,
    token_header,
    tool_timeout_seconds,
    enabled,
    sort_order,
    created_by_user_id
) VALUES (
    sqlc.arg(alias),
    sqlc.arg(display_name),
    sqlc.arg(transport),
    NULLIF(sqlc.arg(command), ''),
    sqlc.arg(args_json),
    NULLIF(sqlc.arg(endpoint_url), ''),
    sqlc.arg(token_encrypted),
    NULLIF(sqlc.arg(token_last4), ''),
    sqlc.arg(token_header),
    sqlc.arg(tool_timeout_seconds),
    sqlc.arg(enabled),
    sqlc.arg(sort_order),
    sqlc.arg(created_by_user_id)
)
RETURNING id::text, created_at, updated_at;

-- name: UpdateMCPServer :one
UPDATE mcp_servers
SET
    display_name = sqlc.arg(display_name),
    transport = sqlc.arg(transport),
    command = NULLIF(sqlc.arg(command), ''),
    args_json = sqlc.arg(args_json),
    endpoint_url = NULLIF(sqlc.arg(endpoint_url), ''),
    token_encrypted = sqlc.arg(token_encrypted),
    token_last4 = NULLIF(sqlc.arg(token_last4), ''),
    token_header = sqlc.arg(token_header),
    tool_timeout_seconds = sqlc.arg(tool_timeout_seconds),
    enabled = sqlc.arg(enabled),
    sort_order = sqlc.arg(sort_order),
    updated_at = now()
WHERE id = sqlc.arg(id)::uuid
RETURNING updated_at;

-- name: DeleteMCPServer :execrows
DELETE FROM mcp_servers
WHERE id = sqlc.arg(id)::uuid;

-- name: UpdateMCPConnectionStatus :exec
UPDATE mcp_servers
SET
    tool_count = sqlc.arg(tool_count),
    last_connected_at = sqlc.arg(last_connected_at),
    last_error = NULLIF(sqlc.arg(last_error), ''),
    updated_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: InsertAuditLog :exec
INSERT INTO admin_audit_logs (
    external_user_id,
    action,
    target_type,
    target_id,
    before_data,
    after_data,
    request_id
) VALUES (
    sqlc.arg(external_user_id),
    sqlc.arg(action),
    sqlc.arg(target_type),
    NULLIF(sqlc.arg(target_id), ''),
    sqlc.arg(before_data),
    sqlc.arg(after_data),
    NULLIF(sqlc.arg(request_id), '')
);
