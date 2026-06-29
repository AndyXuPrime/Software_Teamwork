-- AI Gateway model profile queries.
-- The S-02 repository adapter keeps hand-written SQL in Go while preserving
-- service-local query files for sqlc adoption.

-- name: ListModelProfiles :many
SELECT
  id,
  name,
  purpose,
  provider,
  base_url,
  model,
  enabled,
  is_default,
  timeout_ms,
  api_key_configured,
  supports_streaming,
  dimensions,
  top_n,
  default_parameters_json,
  credential_id,
  created_by_user_id,
  updated_by_user_id,
  created_at,
  updated_at,
  deleted_at
FROM model_profiles
WHERE deleted_at IS NULL
ORDER BY purpose, is_default DESC, created_at DESC;
