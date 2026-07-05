#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.cloud.yml"
ENV_FILE="${DOCKER_CLOUD_ENV_FILE:-$ROOT_DIR/.env.docker.cloud}"

usage() {
  cat <<'EOF'
Usage: ./scripts/docker/start.sh [--env-file PATH] [compose up args...]

Build and start the cloud-backed Docker app stack. Copy the template once:
  cp deploy/docker/cloud.env.example .env.docker.cloud
Then fill cloud PostgreSQL, Redis, object storage, Knowledge runtime, and service secrets.
OCR/model provider seed credentials are required only when DOCKER_SEED_ENABLED=true.
EOF
}

log_info() {
  printf '[docker] %s\n' "$*"
}

log_error() {
  printf '[docker] ERROR: %s\n' "$*" >&2
}

read_env_value() {
  local key="$1"
  grep -E "^[[:space:]]*${key}=" "$ENV_FILE" 2>/dev/null | tail -n 1 | sed -E 's/^[^=]+=//; s/^[[:space:]]+//; s/[[:space:]]+$//; s/^["'\'']//; s/["'\'']$//'
}

is_false_value() {
  case "$(printf '%s' "${1:-}" | tr '[:upper:]' '[:lower:]')" in
    0|false|no|off) return 0 ;;
    *) return 1 ;;
  esac
}

append_missing_values() {
  local key value
  for key in "$@"; do
    value="$(read_env_value "$key" || true)"
    if [[ -z "$value" || "$value" == *"<"* || "$value" == *">"* ]]; then
      missing+=("$key")
    fi
  done
}

is_missing_key() {
  local key="$1"
  local missing_key
  for missing_key in "${missing[@]}"; do
    if [[ "$missing_key" == "$key" ]]; then
      return 0
    fi
  done
  return 1
}

is_unsafe_cloud_value() {
  local value lower
  value="${1:-}"
  lower="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]')"
  if [[ "$value" == *"<"* || "$value" == *">"* ]]; then
    return 0
  fi
  case "$lower" in
    *local-dev*|*local-demo*|*change-me*) return 0 ;;
    sha256:26c6719c056dabe8530ea09f1e8f7593cbcf98a060731c0fc786a5eb48e71ce7) return 0 ;;
    *) return 1 ;;
  esac
}

append_unsafe_values() {
  local key value
  for key in "$@"; do
    value="$(read_env_value "$key" || true)"
    if [[ -n "$value" ]] && is_unsafe_cloud_value "$value"; then
      if [[ "$value" == *"<"* || "$value" == *">"* ]] && is_missing_key "$key"; then
        continue
      fi
      unsafe+=("$key")
    fi
  done
}

validate_cloud_env() {
  local required=(
    INTERNAL_SERVICE_TOKEN
    AUTH_GATEWAY_ADMIN_SERVICE_TOKEN
    GATEWAY_AUTH_ADMIN_SERVICE_TOKEN
    TOKEN_HASH_SECRET
    AI_GATEWAY_SERVICE_TOKEN_HASHES
    AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY
    AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF
    AUTH_DATABASE_URL
    FILE_DATABASE_URL
    KNOWLEDGE_DATABASE_URL
    QA_DATABASE_URL
    DOCUMENT_DATABASE_URL
    AI_GATEWAY_DATABASE_URL
    GATEWAY_REDIS_ADDR
    DOCUMENT_REDIS_ADDR
    FILE_MINIO_ENDPOINT
    FILE_MINIO_ACCESS_KEY
    FILE_MINIO_SECRET_KEY
    FILE_MINIO_BUCKET
    VENDOR_RUNTIME_URL
    VENDOR_RUNTIME_SERVICE_TOKEN
    MCP_SERVER_TOKEN
    KNOWLEDGE_MCP_TOKEN
  )
  local optional_secret_values=(
    GATEWAY_REDIS_PASSWORD
    DOCUMENT_REDIS_PASSWORD
  )
  local seed_required=(
    POSTGRES_ADMIN_URL
    PADDLEOCR_ACCESS_TOKEN
  )
  local provider_seed_required=(
    AI_GATEWAY_LOCAL_PROVIDER_BASE_URL
    AI_GATEWAY_LOCAL_PROVIDER_API_KEY
    AI_GATEWAY_LOCAL_CHAT_MODEL
  )
  local missing=()
  local unsafe=()
  append_missing_values "${required[@]}"
  append_unsafe_values "${required[@]}" "${optional_secret_values[@]}"

  local docker_seed_enabled ai_gateway_seed_enabled
  docker_seed_enabled="$(read_env_value DOCKER_SEED_ENABLED || true)"
  if ! is_false_value "$docker_seed_enabled"; then
    append_missing_values "${seed_required[@]}"
    append_unsafe_values "${seed_required[@]}"
    ai_gateway_seed_enabled="$(read_env_value AI_GATEWAY_LOCAL_SEED_ENABLED || true)"
    if ! is_false_value "$ai_gateway_seed_enabled"; then
      append_missing_values "${provider_seed_required[@]}"
      append_unsafe_values "${provider_seed_required[@]}"
    fi
  fi

  local failed=0
  if (( ${#missing[@]} > 0 )); then
    log_error "fill these cloud values in $ENV_FILE before starting: ${missing[*]}"
    failed=1
  fi
  if (( ${#unsafe[@]} > 0 )); then
    log_error "replace local/demo placeholder values in $ENV_FILE before starting: ${unsafe[*]}"
    failed=1
  fi
  return "$failed"
}

args=()
while (($#)); do
  case "$1" in
    --env-file)
      if [[ $# -lt 2 ]]; then
        log_error "--env-file requires a path"
        exit 2
      fi
      ENV_FILE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      args+=("$1")
      shift
      ;;
  esac
done

if [[ ! -f "$ENV_FILE" ]]; then
  log_error "$ENV_FILE does not exist"
  log_info "create it with: cp deploy/docker/cloud.env.example .env.docker.cloud"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  log_error "docker is required"
  exit 1
fi
if ! docker info >/dev/null 2>&1; then
  log_error "Docker daemon is not reachable"
  exit 1
fi

validate_cloud_env

frontend_port="$(read_env_value DOCKER_FRONTEND_PORT || true)"
frontend_port="${frontend_port:-18080}"

log_info "validating cloud compose config"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" config --quiet

log_info "building and starting cloud-backed Docker stack"
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --build "${args[@]}"

log_info "web: http://localhost:${frontend_port}"
log_info "logs: docker compose -f deploy/docker-compose.cloud.yml --env-file $ENV_FILE logs -f"
