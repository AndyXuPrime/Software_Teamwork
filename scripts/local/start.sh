#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG_LOADER="$ROOT_DIR/scripts/config/load-profile.sh"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
RUN_DIR="$ROOT_DIR/.local/run"
LOG_DIR="$ROOT_DIR/.local/logs"
TOOLS_DIR="$ROOT_DIR/.local/tools"
BIN_DIR="$ROOT_DIR/.local/bin"
RUNTIME_DIR="$ROOT_DIR/services/knowledge-runtime"
LOCAL_RUNTIME_DIR="$ROOT_DIR/.local/knowledge-runtime"
LOCAL_LIB_DIR="$ROOT_DIR/scripts/local/lib"
CURRENT_STEP="initializing"
CHINA_MIRRORS=0
INFRA_ONLY=0
BACKEND_ONLY=0
SKIP_INFRA=0
SKIP_MIGRATIONS=0
SKIP_SEED=0
RUNTIME_MODE="full"
STARTED_SERVICES=()

INFRA_SERVICES=(postgres redis minio elasticsearch)
CHINA_POSTGRES_IMAGE="docker.1ms.run/library/postgres:16-alpine"
CHINA_REDIS_IMAGE="docker.1ms.run/library/redis:7-alpine"
CHINA_MINIO_IMAGE="docker.1ms.run/minio/minio:RELEASE.2025-09-07T16-13-09Z"
CHINA_MINIO_MC_IMAGE="docker.1ms.run/minio/mc:RELEASE.2025-08-13T08-35-41Z"
CHINA_ELASTICSEARCH_IMAGE="docker.1ms.run/elasticsearch:8.15.3"

GO_SERVICES=(
  "auth|auth-server|$ROOT_DIR/services/auth"
  "file|file-server|$ROOT_DIR/services/file"
  "knowledge|knowledge-adapter|$ROOT_DIR/services/knowledge"
  "ai-gateway|ai-gateway-server|$ROOT_DIR/services/ai-gateway"
  "qa|qa-server|$ROOT_DIR/services/qa"
  "document|document-server|$ROOT_DIR/services/document"
  "gateway|gateway-server|$ROOT_DIR/services/gateway"
)

# shellcheck source=scripts/local/lib/common.sh
. "$LOCAL_LIB_DIR/common.sh"
# shellcheck source=scripts/local/lib/process.sh
. "$LOCAL_LIB_DIR/process.sh"
# shellcheck source=scripts/local/lib/knowledge-runtime.sh
. "$LOCAL_LIB_DIR/knowledge-runtime.sh"

COLOR_RESET=""
COLOR_BLUE=""
COLOR_GREEN=""
COLOR_YELLOW=""
COLOR_RED=""
COLOR_CYAN=""
if [[ -z "${NO_COLOR:-}" && ( -t 1 || "${FORCE_COLOR:-0}" == "1" || "${CLICOLOR_FORCE:-0}" == "1" ) ]]; then
  COLOR_RESET=$'\033[0m'
  COLOR_BLUE=$'\033[1;34m'
  COLOR_GREEN=$'\033[1;32m'
  COLOR_YELLOW=$'\033[1;33m'
  COLOR_RED=$'\033[1;31m'
  COLOR_CYAN=$'\033[1;36m'
fi

log_info() { printf '%b%s %s%b\n' "$COLOR_BLUE" "[start]" "$*" "$COLOR_RESET"; }
log_ok() { printf '%b%s %s%b\n' "$COLOR_GREEN" "[ok]" "$*" "$COLOR_RESET"; }
log_warn() { printf '%b%s %s%b\n' "$COLOR_YELLOW" "[warn]" "$*" "$COLOR_RESET" >&2; }
log_error() { printf '%b%s %s%b\n' "$COLOR_RED" "[fail]" "$*" "$COLOR_RESET" >&2; }
log_hint() { printf '%b%s %s%b\n' "$COLOR_CYAN" "[hint]" "$*" "$COLOR_RESET" >&2; }

usage() {
  cat <<'EOF'
Usage: ./scripts/local/start.sh [options]

Starts local infrastructure from already-present images, applies migrations and
seed data with prepared local tools, starts prepared host-run binaries, and by
default starts the Knowledge runtime API plus worker.
This script does not run dependency downloads, Docker pulls, uv sync,
go mod download, or go run module@version.
Requires prepared .local/tools/config-ctl, .local/tools/goose, .local/bin/, and
runtime .venv when runtime startup is enabled.

Options:
  --china             Use mainland China Docker image rewrites and runtime
                      HuggingFace mirror for this run only. Committed config
                      files and .env.local are not edited.
  --runtime api       Start only Knowledge runtime API from prepared .venv.
  --runtime full      Start Knowledge runtime API and worker. Default.
  --runtime none      Skip Knowledge runtime startup.
  --no-runtime        Same as --runtime none.
  --infra-only        Start infra, migrations, and seed, then stop.
  --backend-only      Skip infra/migrations/seed and start host-run services;
                      runtime still follows --runtime. Use --no-runtime when
                      connecting to an existing external runtime API.
  --skip-infra        Skip Docker infrastructure startup.
  --skip-migrations   Skip service migrations.
  --skip-seed         Skip local seed SQL and local seed overlays.
  -h, --help          Show this help.
EOF
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      --china)
        CHINA_MIRRORS=1
        ;;
      --runtime)
        if [[ $# -lt 2 ]]; then
          log_error "--runtime requires api, full, or none"
          exit 2
        fi
        RUNTIME_MODE="$2"
        shift
        ;;
      --runtime=*)
        RUNTIME_MODE="${1#--runtime=}"
        ;;
      --no-runtime)
        RUNTIME_MODE="none"
        ;;
      --infra-only)
        INFRA_ONLY=1
        RUNTIME_MODE="none"
        ;;
      --backend-only)
        BACKEND_ONLY=1
        SKIP_INFRA=1
        SKIP_MIGRATIONS=1
        SKIP_SEED=1
        ;;
      --skip-infra)
        SKIP_INFRA=1
        ;;
      --skip-migrations)
        SKIP_MIGRATIONS=1
        ;;
      --skip-seed)
        SKIP_SEED=1
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        log_error "unknown argument: $1"
        usage >&2
        exit 2
        ;;
    esac
    shift
  done
  case "$RUNTIME_MODE" in
    none|api|full) ;;
    *)
      log_error "--runtime must be api, full, or none"
      exit 2
      ;;
  esac
}

on_exit() {
  local status=$?
  if (( status == 0 )); then
    log_ok "completed successfully"
  else
    log_error "failed during ${CURRENT_STEP} (exit ${status})"
    log_hint "Run ./scripts/local/check.sh to inspect missing local prerequisites and setup suggestions."
    log_hint "Use ./scripts/local/check.sh --china for mainland China mirror suggestions."
    log_hint "Logs: .local/logs/*.log"
  fi
}

run_step() {
  CURRENT_STEP="$1"
  shift
  log_info "$CURRENT_STEP"
  "$@"
  log_ok "$CURRENT_STEP succeeded"
}

require_file() {
  local path="$1"
  local hint="$2"
  if [[ ! -e "$path" ]]; then
    log_error "missing $path"
    log_hint "$hint"
    return 1
  fi
}

require_command_local() {
  local command_name="$1"
  local hint="$2"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log_error "$command_name is required"
    log_hint "$hint"
    return 1
  fi
}

apply_china_sources() {
  export POSTGRES_IMAGE="$CHINA_POSTGRES_IMAGE"
  export REDIS_IMAGE="$CHINA_REDIS_IMAGE"
  export MINIO_IMAGE="$CHINA_MINIO_IMAGE"
  export MINIO_MC_IMAGE="$CHINA_MINIO_MC_IMAGE"
  export KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE="$CHINA_ELASTICSEARCH_IMAGE"
  write_compose_env_value POSTGRES_IMAGE "$POSTGRES_IMAGE"
  write_compose_env_value REDIS_IMAGE "$REDIS_IMAGE"
  write_compose_env_value MINIO_IMAGE "$MINIO_IMAGE"
  write_compose_env_value MINIO_MC_IMAGE "$MINIO_MC_IMAGE"
  write_compose_env_value KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE "$KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE"
  log_info "using mainland China mirrors for this run; committed config files and .env.local are not modified"
}

write_compose_env_value() {
  local key="$1"
  local value="$2"
  local tmp_file
  [[ -n "${CONFIG_COMPOSE_ENV_FILE:-}" && -f "$CONFIG_COMPOSE_ENV_FILE" ]] || return 0
  tmp_file="$(mktemp "$CONFIG_COMPOSE_ENV_FILE.XXXXXX")"
  awk -v key="$key" -v value="$value" '
    BEGIN { written = 0 }
    index($0, key "=") == 1 {
      print key "=" value
      written = 1
      next
    }
    { print }
    END {
      if (!written) {
        print key "=" value
      }
    }
  ' "$CONFIG_COMPOSE_ENV_FILE" >"$tmp_file"
  mv "$tmp_file" "$CONFIG_COMPOSE_ENV_FILE"
}

load_config() {
  export SOFTWARE_TEAMWORK_ROOT="$ROOT_DIR"
  export CONFIG_CTL_REQUIRE_PREPARED=1
  # shellcheck disable=SC1090
  . "$CONFIG_LOADER"
}

align_host_run_ai_gateway_models() {
  if [[ -z "${AI_GATEWAY_LOCAL_CHAT_MODEL:-}" ]]; then
    return
  fi
  if [[ -z "${MODEL_ID:-}" || "${MODEL_ID:-}" == "local-placeholder-chat" ]]; then
    export MODEL_ID="$AI_GATEWAY_LOCAL_CHAT_MODEL"
    log_info "using AI_GATEWAY_LOCAL_CHAT_MODEL for host-run QA MODEL_ID: $MODEL_ID"
  fi
  if [[ -z "${DOCUMENT_AI_GATEWAY_MODEL:-}" || "${DOCUMENT_AI_GATEWAY_MODEL:-}" == "local-placeholder-chat" ]]; then
    export DOCUMENT_AI_GATEWAY_MODEL="$AI_GATEWAY_LOCAL_CHAT_MODEL"
    log_info "using AI_GATEWAY_LOCAL_CHAT_MODEL for host-run Document model: $DOCUMENT_AI_GATEWAY_MODEL"
  fi
}

compose_cmd() {
  docker compose -f "$COMPOSE_FILE" --env-file "$CONFIG_COMPOSE_ENV_FILE" "$@"
}

check_start_prerequisites() {
  require_file "$TOOLS_DIR/config-ctl" "Run ./scripts/local/check.sh for the manual build command." || return 1
  if (( ! SKIP_MIGRATIONS )); then
    require_file "$TOOLS_DIR/goose" "Run ./scripts/local/check.sh for setup suggestions." || return 1
  fi
  if (( ! INFRA_ONLY )); then
    local item name binary dir
    for item in "${GO_SERVICES[@]}"; do
      IFS='|' read -r name binary dir <<<"$item"
      require_file "$BIN_DIR/$binary" "Run ./scripts/local/check.sh for the manual build command." || return 1
    done
  fi
  if (( ! SKIP_INFRA )); then
    require_command_local docker "Install Docker or run ./scripts/local/check.sh to inspect the environment." || return 1
  fi
  if (( ! SKIP_MIGRATIONS || ! SKIP_SEED )); then
    require_command_local psql "Install the PostgreSQL client; the server runs in Docker." || return 1
  fi
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    require_command_local curl "Install curl for local runtime readiness checks." || return 1
  fi
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    require_command_local uv "Install uv and run ./scripts/local/check.sh for runtime setup suggestions." || return 1
    require_command_local python3 "Install python3 for runtime readiness checks." || return 1
    require_file "$RUNTIME_DIR/.venv" "Run ./scripts/local/check.sh for the manual uv sync command." || return 1
  fi
}

start_infra() {
  compose_cmd config --quiet
  compose_cmd up -d --pull never --wait --wait-timeout "${LOCAL_INFRA_WAIT_TIMEOUT_SECONDS:-180}" "${INFRA_SERVICES[@]}"
  compose_cmd up --no-deps --pull never --exit-code-from minio-init minio-init
}

migrate_service() {
  local service="$1"
  local database_url="$2"
  CURRENT_STEP="migrating $service"
  log_info "$CURRENT_STEP"
  (cd "$ROOT_DIR/services/$service" && "$TOOLS_DIR/goose" -dir migrations postgres "$database_url" up)
  log_ok "$CURRENT_STEP succeeded"
}

run_migrations() {
  local item service database_url
  for item in \
    "auth:$AUTH_DATABASE_URL" \
    "file:$FILE_DATABASE_URL" \
    "knowledge:$KNOWLEDGE_DATABASE_URL" \
    "qa:$QA_DATABASE_URL" \
    "document:$DOCUMENT_DATABASE_URL" \
    "ai-gateway:$AI_GATEWAY_DATABASE_URL"; do
    service="${item%%:*}"
    database_url="${item#*:}"
    migrate_service "$service" "$database_url"
  done
}

sql_literal() {
  local value="${1:-}"
  value="${value//\'/\'\'}"
  printf "'%s'" "$value"
}

apply_ai_gateway_local_seed_overlay() {
  case "${AI_GATEWAY_LOCAL_SEED_ENABLED:-}" in
    ""|0|false|False|FALSE|no|No|NO|off|Off|OFF)
      log_info "AI_GATEWAY_LOCAL_SEED_ENABLED is not true; keeping seeded placeholder model profiles"
      return
      ;;
  esac
  require_file "$TOOLS_DIR/render-ai-gateway-local-seed" "Run ./scripts/local/check.sh for the manual build command." || return 1
  "$TOOLS_DIR/render-ai-gateway-local-seed" | psql "$POSTGRES_ADMIN_URL" -v ON_ERROR_STOP=1
}

apply_paddleocr_cloud_parser_overlay() {
  if [[ -z "${PADDLEOCR_ACCESS_TOKEN:-}" ]]; then
    log_info "PADDLEOCR_ACCESS_TOKEN is not set; keeping the existing default parser config"
    return
  fi

  local base_url="${PADDLEOCR_BASE_URL:-https://paddleocr.aistudio-app.com}"
  local algorithm="${PADDLEOCR_ALGORITHM:-PP-StructureV3}"
  local parser_id="parser_config_paddleocr_cloud_default"

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

apply_seed() {
  psql "$POSTGRES_ADMIN_URL" \
    -v ON_ERROR_STOP=1 \
    -f "$ROOT_DIR/deploy/seeds/001-local-demo-seed.sql" \
    -f "$ROOT_DIR/deploy/seeds/002-ai-gateway-model-profiles.sql" \
    -f "$ROOT_DIR/deploy/seeds/003-qa-document-mcp.sql" \
    -f "$ROOT_DIR/deploy/seeds/004-qa-default-knowledge-base.sql"
  apply_ai_gateway_local_seed_overlay
  apply_paddleocr_cloud_parser_overlay
}

configure_app_version_current_sha() {
  if [[ -n "${GATEWAY_APP_VERSION_CURRENT_SHA:-}" ]]; then
    return
  fi
  if command -v git >/dev/null 2>&1; then
    local sha
    sha="$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || true)"
    if [[ "$sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
      export GATEWAY_APP_VERSION_CURRENT_SHA="${sha,,}"
      log_info "using repository HEAD for Gateway app-version freshness: ${GATEWAY_APP_VERSION_CURRENT_SHA:0:8}"
    fi
  fi
}

service_group_alive_local() {
  service_group_alive "$1"
}

start_process() {
  local name="$1"
  local dir="$2"
  shift 2
  local pid_file="$RUN_DIR/$name.pid"
  local log_file="$LOG_DIR/$name.log"
  CURRENT_STEP="starting $name"
  if service_group_alive_local "$pid_file"; then
    log_ok "$name already running"
    return
  fi
  rm -f "$pid_file"
  log_info "starting $name"
  launch_process_group "$dir" "$@" >"$log_file" 2>&1 &
  echo "$!" >"$pid_file"
  STARTED_SERVICES+=("$name")
}

check_started_services() {
  CURRENT_STEP="checking started services"
  local wait_seconds="${LOCAL_STARTUP_CHECK_SECONDS:-8}"
  local failed=()
  if [[ "$wait_seconds" =~ ^[0-9]+$ ]] && (( wait_seconds > 0 )) && (( ${#STARTED_SERVICES[@]} > 0 )); then
    log_info "checking process groups for ${wait_seconds}s"
    sleep "$wait_seconds"
  fi
  local name
  for name in "${STARTED_SERVICES[@]}"; do
    if ! service_group_alive_local "$RUN_DIR/$name.pid"; then
      failed+=("$name")
    fi
  done
  if (( ${#failed[@]} == 0 )); then
    return 0
  fi
  log_error "startup failed for: ${failed[*]}"
  for name in "${failed[@]}"; do
    printf '%b%s%b\n' "$COLOR_YELLOW" "----- $LOG_DIR/$name.log (tail) -----" "$COLOR_RESET" >&2
    if [[ -f "$LOG_DIR/$name.log" ]]; then
      tail -n 80 "$LOG_DIR/$name.log" >&2
    else
      log_warn "log file missing"
    fi
  done
  return 1
}

report_startup_status() {
  CURRENT_STEP="reporting startup status"
  log_info "startup status"
  if (( SKIP_INFRA )); then
    log_info "Docker infrastructure: skipped"
  else
    log_info "Docker infrastructure:"
    if ! compose_cmd ps "${INFRA_SERVICES[@]}"; then
      log_warn "could not read Docker infrastructure status"
    fi
  fi

  if (( INFRA_ONLY )); then
    return 0
  fi

  log_info "host process groups:"
  local pid_files=()
  local pid_file name pid
  shopt -s nullglob
  pid_files=("$RUN_DIR"/*.pid)
  shopt -u nullglob
  if (( ${#pid_files[@]} == 0 )); then
    log_warn "no host process pid files found in $RUN_DIR"
  fi
  for pid_file in "${pid_files[@]}"; do
    name="$(basename "$pid_file" .pid)"
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    if service_group_alive_local "$pid_file"; then
      log_ok "$name running (pid $pid)"
    else
      log_warn "$name not running (pid file: $pid_file)"
    fi
  done
  log_info "logs: $LOG_DIR/*.log"
}

wait_for_http_ok() {
  local name="$1"
  local url="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))
  local response_file status
  response_file="$(mktemp)"
  CURRENT_STEP="waiting for $name"
  while (( SECONDS < deadline )); do
    local curl_args=(-sS -o "$response_file" -w '%{http_code}')
    if should_bypass_proxy_for_url "$url"; then
      curl_args=(--noproxy '*' "${curl_args[@]}")
    fi
    status="$(curl "${curl_args[@]}" "$url" 2>/dev/null || true)"
    if [[ "$status" =~ ^2[0-9][0-9]$ ]]; then
      rm -f "$response_file"
      log_ok "$name is ready"
      return 0
    fi
    sleep 2
  done
  log_error "$name did not become ready at $url"
  if [[ -s "$response_file" ]]; then
    tail -n 20 "$response_file" >&2
  fi
  rm -f "$response_file"
  return 1
}

prepare_runtime_env() {
  export DOC_ENGINE="${DOC_ENGINE:-elasticsearch}"
  if [[ "$(to_lower "$DOC_ENGINE")" == "elasticsearch" ]]; then
    KNOWLEDGE_RUNTIME_ES_URL="$(normalize_http_url "${KNOWLEDGE_RUNTIME_ES_URL:-http://127.0.0.1:${KNOWLEDGE_RUNTIME_ELASTICSEARCH_PORT:-9200}}")"
    export KNOWLEDGE_RUNTIME_ES_URL
  fi
  export VENDOR_RUNTIME_URL
  VENDOR_RUNTIME_URL="$(normalize_http_url "${VENDOR_RUNTIME_URL:-http://127.0.0.1:9380}")"
  export PYTHONPATH="."
  export LITELLM_LOCAL_MODEL_COST_MAP="${LITELLM_LOCAL_MODEL_COST_MAP:-True}"
  if (( CHINA_MIRRORS )); then
    enable_china_hf_endpoint 1
  fi
  append_no_proxy "127.0.0.1"
  append_no_proxy "localhost"
  append_no_proxy "::1"
  append_no_proxy_for_url "$VENDOR_RUNTIME_URL"
  if [[ -n "${KNOWLEDGE_RUNTIME_ES_URL:-}" ]]; then
    append_no_proxy_for_url "$KNOWLEDGE_RUNTIME_ES_URL"
  fi
  prepare_knowledge_runtime_config host
}

start_runtime() {
  [[ "$RUNTIME_MODE" == "none" ]] && return
  prepare_runtime_env
  if [[ "$(to_lower "${DOC_ENGINE:-elasticsearch}")" == "elasticsearch" && -n "${KNOWLEDGE_RUNTIME_ES_URL:-}" ]]; then
    wait_for_http_ok "Elasticsearch" "$KNOWLEDGE_RUNTIME_ES_URL/_cluster/health" "${KNOWLEDGE_RUNTIME_ELASTICSEARCH_READY_TIMEOUT_SECONDS:-120}"
  fi
  start_process "knowledge-runtime-api" "$RUNTIME_DIR" ./deploy/api/run-local.sh
  if [[ "$RUNTIME_MODE" == "full" ]]; then
    start_process "knowledge-runtime-worker" "$RUNTIME_DIR" ./deploy/worker/run-local.sh
  fi
  wait_for_http_ok "Knowledge runtime API" "${VENDOR_RUNTIME_URL:-http://127.0.0.1:9380}/api/v1/system/ping" "${KNOWLEDGE_RUNTIME_API_READY_TIMEOUT_SECONDS:-90}"
}

start_backend() {
  configure_app_version_current_sha
  local item name binary dir
  for item in "${GO_SERVICES[@]}"; do
    IFS='|' read -r name binary dir <<<"$item"
    start_process "$name" "$dir" "$BIN_DIR/$binary"
  done
}

parse_args "$@"
trap on_exit EXIT

log_info "starting local stack from prepared local environment"
run_step "checking prerequisites" check_start_prerequisites
run_step "loading local config" load_config
if (( CHINA_MIRRORS )); then
  run_step "applying mainland China image rewrites" apply_china_sources
fi
align_host_run_ai_gateway_models
mkdir -p "$RUN_DIR" "$LOG_DIR"

if (( ! SKIP_INFRA )); then
  run_step "starting infrastructure without pulling images" start_infra
fi
if (( ! SKIP_MIGRATIONS )); then
  run_step "applying service migrations" run_migrations
fi
if (( ! SKIP_SEED )); then
  run_step "applying local seed data" apply_seed
fi
if (( INFRA_ONLY )); then
  report_startup_status
  log_ok "infra, migrations, and seed are ready"
  exit 0
fi

run_step "starting Knowledge runtime" start_runtime
run_step "starting backend services" start_backend
run_step "checking started services" check_started_services
run_step "reporting startup status" report_startup_status
log_ok "local backend and runtime mode '$RUNTIME_MODE' are running; frontend remains: cd apps/web && bun install && bun run dev"
