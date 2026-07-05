#!/usr/bin/env sh
set -eu

is_false_value() {
  case "$1" in
    false|FALSE|False|0|no|NO|No|off|OFF|Off) return 0 ;;
    *) return 1 ;;
  esac
}

case "${DOCKER_SEED_ENABLED:-false}" in
  false|FALSE|False|0|no|NO|No|off|OFF|Off)
  echo "docker seed: skipped"
  exit 0
  ;;
esac

require_env() {
  name="$1"
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "docker seed: $name is required when DOCKER_SEED_ENABLED is not false" >&2
    exit 1
  fi
}

is_unsafe_cloud_value() {
  value="${1:-}"
  lower="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]')"
  case "$value" in
    *"<"*|*">"*) return 0 ;;
  esac
  case "$lower" in
    *local-dev*|*local-demo*|*change-me*) return 0 ;;
    sha256:26c6719c056dabe8530ea09f1e8f7593cbcf98a060731c0fc786a5eb48e71ce7) return 0 ;;
    *) return 1 ;;
  esac
}

require_cloud_value() {
  name="$1"
  require_env "$name"
  eval "value=\${$name:-}"
  if is_unsafe_cloud_value "$value"; then
    echo "docker seed: $name must be replaced before seeding a cloud database" >&2
    exit 1
  fi
}

validate_seed_inputs() {
  require_cloud_value INTERNAL_SERVICE_TOKEN
  require_cloud_value AUTH_GATEWAY_ADMIN_SERVICE_TOKEN
  require_cloud_value GATEWAY_AUTH_ADMIN_SERVICE_TOKEN
  require_cloud_value TOKEN_HASH_SECRET
  require_cloud_value AI_GATEWAY_SERVICE_TOKEN_HASHES
  require_cloud_value AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY
  require_cloud_value AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF
  require_cloud_value POSTGRES_ADMIN_URL
  require_cloud_value PADDLEOCR_ACCESS_TOKEN

  if ! is_false_value "${AI_GATEWAY_LOCAL_SEED_ENABLED:-true}"; then
    require_cloud_value AI_GATEWAY_LOCAL_PROVIDER_BASE_URL
    require_cloud_value AI_GATEWAY_LOCAL_PROVIDER_API_KEY
    require_cloud_value AI_GATEWAY_LOCAL_CHAT_MODEL
  fi
}

sql_literal() {
  printf "'%s'" "$(printf '%s' "${1:-}" | sed "s/'/''/g")"
}

apply_static_seed() {
  require_env POSTGRES_ADMIN_URL
  echo "docker seed: static demo data"
  psql "$POSTGRES_ADMIN_URL" \
    -v ON_ERROR_STOP=1 \
    -f /workspace/deploy/seeds/001-local-demo-seed.sql \
    -f /workspace/deploy/seeds/002-ai-gateway-model-profiles.sql \
    -f /workspace/deploy/seeds/003-qa-document-mcp.sql \
    -f /workspace/deploy/seeds/004-qa-default-knowledge-base.sql
}

apply_ai_gateway_seed() {
  echo "docker seed: ai gateway cloud provider"
  ai-gateway-local-seed
}

apply_paddleocr_cloud_parser_overlay() {
  if [ -z "${PADDLEOCR_ACCESS_TOKEN:-}" ]; then
    echo "docker seed: PADDLEOCR_ACCESS_TOKEN not set; keeping existing parser config"
    return
  fi

  base_url="${PADDLEOCR_BASE_URL:-https://paddleocr.aistudio-app.com}"
  algorithm="${PADDLEOCR_ALGORITHM:-PP-StructureV3}"
  parser_id="parser_config_paddleocr_cloud_default"

  echo "docker seed: paddleocr cloud parser"
  psql "$KNOWLEDGE_DATABASE_URL" -v ON_ERROR_STOP=1 <<SQL
BEGIN;
UPDATE parser_configs
SET is_default = false, updated_at = now()
WHERE is_default
  AND deleted_at IS NULL
  AND id <> $(sql_literal "$parser_id");

INSERT INTO parser_configs (
  id, name, backend, enabled, is_default, concurrency,
  supported_content_types, endpoint_url, default_parameters, created_at, updated_at
) VALUES (
  $(sql_literal "$parser_id"),
  'Default PaddleOCR cloud parser',
  'paddleocr_cloud',
  true,
  true,
  4,
  ARRAY['application/pdf'],
  NULL,
  jsonb_build_object(
    'paddleocr_base_url', $(sql_literal "$base_url"),
    'paddleocr_access_token', $(sql_literal "$PADDLEOCR_ACCESS_TOKEN"),
    'paddleocr_algorithm', $(sql_literal "$algorithm")
  ),
  now(),
  now()
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    backend = EXCLUDED.backend,
    enabled = true,
    is_default = true,
    concurrency = EXCLUDED.concurrency,
    supported_content_types = EXCLUDED.supported_content_types,
    endpoint_url = NULL,
    default_parameters = EXCLUDED.default_parameters,
    updated_at = now(),
    deleted_at = NULL;
COMMIT;
SQL
}

validate_seed_inputs
apply_static_seed
apply_ai_gateway_seed
apply_paddleocr_cloud_parser_overlay
