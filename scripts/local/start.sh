#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG_LOADER="$ROOT_DIR/scripts/config/load-profile.sh"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
RUN_DIR="$ROOT_DIR/.local/run"
LOG_DIR="$ROOT_DIR/.local/logs"
TOOLS_DIR="$ROOT_DIR/.local/tools"
BIN_DIR="$ROOT_DIR/.local/bin"
STAMP_DIR="$ROOT_DIR/.local/stamps"
RUNTIME_DIR="$ROOT_DIR/services/knowledge-runtime"
LOCAL_RUNTIME_DIR="$ROOT_DIR/.local/knowledge-runtime"
RUNTIME_SYNC_STAMP="$RUNTIME_DIR/.venv/.local-start-profile"
LOCAL_LIB_DIR="$ROOT_DIR/scripts/local/lib"
CURRENT_STEP="initializing"
CHINA_MIRRORS=0
INFRA_ONLY=0
BACKEND_ONLY=0
SKIP_INFRA=0
SKIP_MIGRATIONS=0
SKIP_SEED=0
RUNTIME_MODE="full"
SKIP_PREPARE=0
STARTED_SERVICES=()

INFRA_SERVICES=(postgres redis minio elasticsearch)
GOOSE_VERSION="v3.27.0"
MIN_GO_MAJOR=1
MIN_GO_MINOR=25
MIN_GO_PATCH=1
OFFICIAL_POSTGRES_IMAGE="postgres:16-alpine"
OFFICIAL_REDIS_IMAGE="redis:7-alpine"
OFFICIAL_MINIO_IMAGE="minio/minio:RELEASE.2025-09-07T16-13-09Z"
OFFICIAL_MINIO_MC_IMAGE="minio/mc:RELEASE.2025-08-13T08-35-41Z"
OFFICIAL_ELASTICSEARCH_IMAGE="docker.elastic.co/elasticsearch/elasticsearch:8.15.3"
CHINA_POSTGRES_IMAGE="docker.1ms.run/library/postgres:16-alpine"
CHINA_REDIS_IMAGE="docker.1ms.run/library/redis:7-alpine"
CHINA_MINIO_IMAGE="docker.1ms.run/minio/minio:RELEASE.2025-09-07T16-13-09Z"
CHINA_MINIO_MC_IMAGE="docker.1ms.run/minio/mc:RELEASE.2025-08-13T08-35-41Z"
CHINA_ELASTICSEARCH_IMAGE="docker.1ms.run/elasticsearch:8.15.3"

GO_SERVICES=(
  "auth|auth-server|$ROOT_DIR/services/auth|./cmd/server"
  "file|file-server|$ROOT_DIR/services/file|./cmd/server"
  "knowledge|knowledge-adapter|$ROOT_DIR/services/knowledge|./cmd/adapter"
  "ai-gateway|ai-gateway-server|$ROOT_DIR/services/ai-gateway|./cmd/server"
  "qa|qa-server|$ROOT_DIR/services/qa|./cmd/server"
  "document|document-server|$ROOT_DIR/services/document|./cmd/server"
  "gateway|gateway-server|$ROOT_DIR/services/gateway|./cmd/server"
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

start_command_hint() {
  if (( CHINA_MIRRORS )); then
    printf './scripts/local/start.sh --china'
  else
    printf './scripts/local/start.sh'
  fi
}

usage() {
  cat <<'EOF'
Usage: ./scripts/local/start.sh [options]

Prepares missing local tools, host-run binaries, Docker infrastructure images,
and Knowledge runtime dependencies, then starts the local stack. By default it
starts infrastructure, applies migrations and seed data, starts Knowledge
runtime API plus worker, and starts backend host-run services.
Requires an existing .env.local created by the user; the script never creates
or overwrites it.

Options:
  --china             Use mainland China Docker image rewrites and runtime
                      mirrors for this run only. Committed config files and
                      .env.local are not edited.
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
  --skip-prepare      Do not build, download, or pull missing local artifacts;
                      fail if required artifacts are absent.
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
      --skip-prepare)
        SKIP_PREPARE=1
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
    log_hint "Rerun $(start_command_hint) after fixing the issue above."
    log_hint "Use --skip-prepare only when all local tools, images, binaries, and runtime files are already prepared."
    log_hint "Logs: .local/logs/*.log"
  fi
}

run_step() {
  local step="$1"
  shift
  CURRENT_STEP="$step"
  log_info "$step"
  "$@"
  CURRENT_STEP="$step"
  log_ok "$step succeeded"
}

run_with_heartbeat() {
  local label="$1"
  shift
  local interval="${LOCAL_PREPARE_HEARTBEAT_SECONDS:-15}"
  local start_time pid status now elapsed
  if [[ ! "$interval" =~ ^[0-9]+$ || "$interval" -le 0 ]]; then
    interval=15
  fi
  start_time=$SECONDS
  "$@" &
  pid=$!
  while kill -0 "$pid" 2>/dev/null; do
    sleep "$interval" || true
    if kill -0 "$pid" 2>/dev/null; then
      now=$SECONDS
      elapsed=$((now - start_time))
      log_info "still $label (${elapsed}s elapsed)"
    fi
  done
  set +e
  wait "$pid"
  status=$?
  set -e
  return "$status"
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

preflight_command() {
  local command_name="$1"
  local hint="$2"
  local command_path
  require_command_local "$command_name" "$hint" || return 1
  command_path="$(command -v "$command_name")"
  log_ok "$command_name: $command_path"
}

check_local_env_file() {
  if [[ -f "$ROOT_DIR/.env.local" ]]; then
    log_ok ".env.local present"
    return 0
  fi
  log_error "missing $ROOT_DIR/.env.local"
  log_hint "Create it once: cp .env.example .env.local"
  log_hint "Edit .env.local for local secrets/provider settings, then rerun $(start_command_hint)."
  return 1
}

trim_shell_value() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  if [[ ( "$value" == \"*\" && "$value" == *\" ) || ( "$value" == \'*\' && "$value" == *\' ) ]]; then
    value="${value:1:${#value}-2}"
  fi
  printf '%s\n' "$value"
}

load_go_source_env_from_local_file() {
  local env_file="$ROOT_DIR/.env.local"
  local keys=(GOPROXY GOSUMDB GOPRIVATE GONOPROXY GONOSUMDB GOINSECURE)
  local line key value loaded=()
  [[ -f "$env_file" ]] || return 0
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    if [[ "$line" == export[[:space:]]* ]]; then
      line="${line#export}"
      line="${line#"${line%%[![:space:]]*}"}"
    fi
    for key in "${keys[@]}"; do
      if [[ "$line" == "$key="* ]]; then
        [[ -n "${!key:-}" ]] && continue
        value="$(trim_shell_value "${line#*=}")"
        export "$key=$value"
        loaded+=("$key")
      fi
    done
  done < "$env_file"
  if ((${#loaded[@]} > 0)); then
    log_ok "loaded Go module source settings from .env.local: ${loaded[*]}"
  fi
}

version_number_from_go() {
  local version="$1"
  version="${version#go}"
  printf '%s\n' "$version"
}

check_go_version() {
  local raw version major minor patch
  raw="$(go env GOVERSION 2>/dev/null || true)"
  if [[ -z "$raw" ]]; then
    raw="$(go version | awk '{print $3}')"
  fi
  version="$(version_number_from_go "$raw")"
  if [[ ! "$version" =~ ^([0-9]+)\.([0-9]+)(\.([0-9]+))? ]]; then
    log_error "could not parse Go version from: $raw"
    return 1
  fi
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[4]:-0}"
  if (( major != MIN_GO_MAJOR || minor != MIN_GO_MINOR || patch < MIN_GO_PATCH )); then
    log_error "Go $raw is unsupported; expected go${MIN_GO_MAJOR}.${MIN_GO_MINOR}.${MIN_GO_PATCH} or a later ${MIN_GO_MAJOR}.${MIN_GO_MINOR}.x patch"
    log_hint "Install Go ${MIN_GO_MAJOR}.${MIN_GO_MINOR}.x and ensure go is on PATH."
    return 1
  fi
  log_ok "go version: $raw"
}

check_python_version() {
  python3 -c 'import sys; raise SystemExit(0 if sys.version_info >= (3, 10) else 1)' || {
    log_error "python3 >= 3.10 is required"
    return 1
  }
  log_ok "python3 version: $(python3 -c 'import platform; print(platform.python_version())')"
}

preflight_host_environment() {
  check_local_env_file || return 1
  load_go_source_env_from_local_file
  preflight_command go "Install Go 1.25.x and ensure go is on PATH." || return 1
  check_go_version || return 1
  log_go_module_source_settings
  if (( ! SKIP_INFRA )); then
    preflight_command docker "Install Docker Desktop/Engine and start the Docker daemon." || return 1
    if ! docker info >/dev/null 2>&1; then
      log_error "Docker CLI is installed but the Docker daemon is not reachable"
      log_hint "Start Docker, then rerun $(start_command_hint)."
      return 1
    fi
    log_ok "docker daemon is reachable"
  fi
  if (( ! SKIP_MIGRATIONS || ! SKIP_SEED )); then
    preflight_command psql "Install the PostgreSQL client; the server runs in Docker." || return 1
  fi
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    preflight_command curl "Install curl for local runtime readiness checks." || return 1
    preflight_command python3 "Install python3 >= 3.10 for Knowledge runtime preparation and readiness checks." || return 1
    check_python_version || return 1
    preflight_command uv "Install uv for Knowledge runtime dependency preparation." || return 1
  fi
}

go_env_args() {
  printf '%s\0' "GOTOOLCHAIN=local"
  if (( CHINA_MIRRORS )); then
    printf '%s\0' "GOPROXY=https://goproxy.cn,direct" "GOSUMDB=sum.golang.google.cn"
  fi
}

log_go_module_source_settings() {
  local goproxy gosumdb
  goproxy="$(run_go env GOPROXY 2>/dev/null || true)"
  gosumdb="$(run_go env GOSUMDB 2>/dev/null || true)"
  log_info "Go module source: GOPROXY=${goproxy:-<unset>} GOSUMDB=${gosumdb:-<unset>}"
}

run_go() {
  local env_args=()
  while IFS= read -r -d '' arg; do
    env_args+=("$arg")
  done < <(go_env_args)
  env "${env_args[@]}" go "$@"
}

source_fingerprint() {
  local path file
  {
    for path in "$@"; do
      if [[ -d "$path" ]]; then
        (
          cd "$path"
          while IFS= read -r -d '' file; do
            printf '%s  ' "$path/${file#./}"
            sha256sum "$file"
          done < <(find . -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -print0 | sort -z)
        )
      elif [[ -f "$path" ]]; then
        printf '%s  ' "$path"
        sha256sum "$path"
      else
        printf 'missing  %s\n' "$path"
      fi
    done
  } | sha256sum | awk '{print $1}'
}

source_inputs_newer_than() {
  local artifact="$1"
  shift
  local path newer
  for path in "$@"; do
    if [[ -f "$path" ]]; then
      [[ "$path" -nt "$artifact" ]] && return 0
    elif [[ -d "$path" ]]; then
      newer="$(find "$path" -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer "$artifact" -print -quit)"
      [[ -n "$newer" ]] && return 0
    fi
  done
  return 1
}

artifact_stamp_path() {
  local name="$1"
  printf '%s/%s.sha256\n' "$STAMP_DIR" "$name"
}

artifact_current_for_sources() {
  local artifact="$1"
  local stamp="$2"
  local expected="$3"
  shift 3
  [[ -x "$artifact" ]] || return 1
  if [[ -f "$stamp" ]]; then
    [[ "$(cat "$stamp")" == "$expected" ]]
    return
  fi
  ! source_inputs_newer_than "$artifact" "$@"
}

mark_artifact_current_for_sources() {
  local stamp="$1"
  local fingerprint="$2"
  mkdir -p "$STAMP_DIR"
  printf '%s\n' "$fingerprint" > "$stamp"
}

log_artifact_prepare_reason() {
  local label="$1"
  local artifact="$2"
  local stamp="$3"
  if [[ ! -x "$artifact" ]]; then
    log_info "building $label"
  elif [[ ! -f "$stamp" ]]; then
    log_info "rebuilding $label (missing local build stamp or source is newer)"
  else
    log_info "rebuilding $label (source changed)"
  fi
}

require_current_artifact() {
  local label="$1"
  local artifact="$2"
  local stamp="$3"
  local fingerprint="$4"
  local hint="$5"
  shift 5
  require_file "$artifact" "$hint" || return 1
  if ! artifact_current_for_sources "$artifact" "$stamp" "$fingerprint" "$@"; then
    log_error "$label is stale for the current source tree: $artifact"
    log_hint "$hint"
    return 1
  fi
}

run_go_with_heartbeat() {
  local label="$1"
  shift
  local env_args=()
  while IFS= read -r -d '' arg; do
    env_args+=("$arg")
  done < <(go_env_args)
  run_with_heartbeat "$label" env "${env_args[@]}" go "$@"
}

run_go_install() {
  local env_args=()
  while IFS= read -r -d '' arg; do
    env_args+=("$arg")
  done < <(go_env_args)
  run_with_heartbeat "installing goose $GOOSE_VERSION" env "${env_args[@]}" GOBIN="$TOOLS_DIR" go install "$@"
}

prepare_config_renderer() {
  mkdir -p "$TOOLS_DIR"
  local artifact="$TOOLS_DIR/config-ctl"
  local stamp fingerprint
  stamp="$(artifact_stamp_path config-ctl)"
  fingerprint="$(source_fingerprint "$ROOT_DIR/config/ctl")"
  if artifact_current_for_sources "$artifact" "$stamp" "$fingerprint" "$ROOT_DIR/config/ctl"; then
    log_ok "config renderer already prepared for current source: $artifact"
    return 0
  fi
  log_artifact_prepare_reason "config renderer" "$artifact" "$stamp"
  run_go_with_heartbeat "building config renderer" -C "$ROOT_DIR/config/ctl" build -o "$artifact" .
  mark_artifact_current_for_sources "$stamp" "$fingerprint"
}

prepare_goose() {
  (( SKIP_MIGRATIONS )) && return 0
  mkdir -p "$TOOLS_DIR"
  local version_output
  version_output="$("$TOOLS_DIR/goose" -version 2>/dev/null || true)"
  if [[ "$version_output" == *"$GOOSE_VERSION"* ]]; then
    log_ok "goose already prepared: $version_output"
    return 0
  fi
  if [[ -n "$version_output" ]]; then
    log_info "reinstalling goose $GOOSE_VERSION (found: $version_output)"
  else
    log_info "installing goose $GOOSE_VERSION"
  fi
  run_go_install "github.com/pressly/goose/v3/cmd/goose@$GOOSE_VERSION"
}

prepare_ai_gateway_seed_helper() {
  (( SKIP_SEED )) && return 0
  mkdir -p "$TOOLS_DIR"
  local artifact="$TOOLS_DIR/render-ai-gateway-local-seed"
  local source="$ROOT_DIR/scripts/local/render_ai_gateway_local_seed.go"
  local stamp fingerprint
  stamp="$(artifact_stamp_path render-ai-gateway-local-seed)"
  fingerprint="$(source_fingerprint "$source")"
  if artifact_current_for_sources "$artifact" "$stamp" "$fingerprint" "$source"; then
    log_ok "AI Gateway local seed helper already prepared for current source: $artifact"
    return 0
  fi
  log_artifact_prepare_reason "AI Gateway local seed helper" "$artifact" "$stamp"
  run_go_with_heartbeat "building AI Gateway local seed helper" build -o "$artifact" "$source"
  mark_artifact_current_for_sources "$stamp" "$fingerprint"
}

prepare_local_tools() {
  prepare_config_renderer
  prepare_goose
  prepare_ai_gateway_seed_helper
}

prepare_backend_binaries() {
  (( INFRA_ONLY )) && return 0
  mkdir -p "$BIN_DIR"
  local item name binary dir target artifact stamp fingerprint
  for item in "${GO_SERVICES[@]}"; do
    IFS='|' read -r name binary dir target <<<"$item"
    artifact="$BIN_DIR/$binary"
    stamp="$(artifact_stamp_path "$binary")"
    fingerprint="$(source_fingerprint "$dir")"
    if artifact_current_for_sources "$artifact" "$stamp" "$fingerprint" "$dir"; then
      log_ok "$name binary already prepared for current source: $artifact"
      continue
    fi
    log_artifact_prepare_reason "$name binary" "$artifact" "$stamp"
    run_go_with_heartbeat "building $name binary" -C "$dir" build -o "$artifact" "$target"
    mark_artifact_current_for_sources "$stamp" "$fingerprint"
  done
}

runtime_sync_profile() {
  case "$RUNTIME_MODE" in
    api) printf 'api\n' ;;
    full) printf 'worker\n' ;;
    *) printf 'all\n' ;;
  esac
}

runtime_china_arg() {
  if (( CHINA_MIRRORS )); then
    printf '%s\n' "--china"
  fi
}

runtime_profile_satisfies() {
  local prepared="$1"
  local required="$2"
  [[ "$prepared" == "$required" ]] && return 0
  [[ "$prepared" == "worker" && "$required" == "api" ]] && return 0
  [[ "$prepared" == "all" ]] && return 0
  return 1
}

runtime_dependency_inputs() {
  printf '%s\n' \
    "$RUNTIME_DIR/pyproject.toml" \
    "$RUNTIME_DIR/uv.lock" \
    "$RUNTIME_DIR/ragflow_deps/download_deps.py"
}

runtime_dependency_fingerprint() {
  local inputs=()
  while IFS= read -r path; do
    inputs+=("$path")
  done < <(runtime_dependency_inputs)
  source_fingerprint "${inputs[@]}"
}

runtime_synced_profile() {
  if [[ -f "$RUNTIME_SYNC_STAMP" ]]; then
    head -n 1 "$RUNTIME_SYNC_STAMP" | sed 's/^profile=//'
  fi
}

runtime_synced_fingerprint() {
  if [[ -f "$RUNTIME_SYNC_STAMP" ]]; then
    sed -n 's/^fingerprint=//p' "$RUNTIME_SYNC_STAMP" | head -n 1
  fi
}

runtime_dependencies_synced() {
  local required="$1"
  local prepared expected actual
  local inputs=()
  [[ -d "$RUNTIME_DIR/.venv" ]] || return 1
  [[ -f "$RUNTIME_SYNC_STAMP" ]] || return 1
  prepared="$(runtime_synced_profile)"
  [[ -n "$prepared" ]] || return 1
  runtime_profile_satisfies "$prepared" "$required" || return 1
  expected="$(runtime_dependency_fingerprint)"
  actual="$(runtime_synced_fingerprint)"
  if [[ -n "$actual" ]]; then
    [[ "$actual" == "$expected" ]]
    return
  fi
  while IFS= read -r path; do
    inputs+=("$path")
  done < <(runtime_dependency_inputs)
  ! source_inputs_newer_than "$RUNTIME_SYNC_STAMP" "${inputs[@]}"
}

mark_runtime_dependencies_synced() {
  local profile="$1"
  if [[ ! -d "$RUNTIME_DIR/.venv" ]]; then
    log_error "Knowledge runtime sync completed but $RUNTIME_DIR/.venv is missing"
    return 1
  fi
  {
    printf '%s\n' "$profile"
    printf 'fingerprint=%s\n' "$(runtime_dependency_fingerprint)"
  } > "$RUNTIME_SYNC_STAMP"
}

runtime_artifacts_ready() {
  local required=(
    "$RUNTIME_DIR/ragflow_deps/tika-server-standard-3.3.0.jar"
    "$RUNTIME_DIR/ragflow_deps/cl100k_base.tiktoken"
  )
  if [[ "$RUNTIME_MODE" == "full" ]]; then
    required+=(
      "$RUNTIME_DIR/ragflow_deps/nltk_data"
      "$RUNTIME_DIR/rag/res/deepdoc/det.onnx"
      "$RUNTIME_DIR/rag/res/deepdoc/rec.onnx"
      "$RUNTIME_DIR/rag/res/deepdoc/tsr.onnx"
      "$RUNTIME_DIR/rag/res/deepdoc/layout.onnx"
      "$RUNTIME_DIR/rag/res/deepdoc/updown_concat_xgb.model"
    )
  fi
  local path
  for path in "${required[@]}"; do
    [[ -e "$path" ]] || return 1
  done
}

prepare_runtime_dependencies() {
  [[ "$RUNTIME_MODE" == "none" ]] && return 0
  local china_args=() profile
  profile="$(runtime_sync_profile)"
  if (( CHINA_MIRRORS )); then
    china_args+=(--china)
  fi
  if runtime_dependencies_synced "$profile"; then
    log_ok "Knowledge runtime .venv already prepared for $profile: $RUNTIME_DIR/.venv"
  else
    log_info "syncing Knowledge runtime Python dependencies ($profile)"
    run_with_heartbeat "syncing Knowledge runtime Python dependencies" \
      bash -c 'cd "$1" && shift && python3 ragflow_deps/download_deps.py "$@"' \
      _ "$RUNTIME_DIR" --sync-only --profile "$profile" "${china_args[@]}"
    mark_runtime_dependencies_synced "$profile"
  fi
  if runtime_artifacts_ready; then
    log_ok "Knowledge runtime artifacts already prepared"
  else
    log_info "downloading Knowledge runtime artifacts"
    run_with_heartbeat "downloading Knowledge runtime artifacts" \
      bash -c 'cd "$1" && shift && uv run --no-project --with "nltk>=3.9.4" --with "huggingface-hub>=1.3.1" ragflow_deps/download_deps.py "$@"' \
      _ "$RUNTIME_DIR" --skip-uv-sync "${china_args[@]}"
  fi
}

prepare_infra_images() {
  (( SKIP_INFRA )) && return 0
  local images=(
    "$POSTGRES_IMAGE"
    "$REDIS_IMAGE"
    "$MINIO_IMAGE"
    "$MINIO_MC_IMAGE"
    "$KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE"
  )
  local image
  for image in "${images[@]}"; do
    if docker image inspect "$image" >/dev/null 2>&1; then
      log_ok "Docker image already present: $image"
      continue
    fi
    log_info "pulling Docker image: $image"
    run_with_heartbeat "pulling Docker image $image" docker pull "$image"
  done
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
  export POSTGRES_IMAGE="${POSTGRES_IMAGE:-$OFFICIAL_POSTGRES_IMAGE}"
  export REDIS_IMAGE="${REDIS_IMAGE:-$OFFICIAL_REDIS_IMAGE}"
  export MINIO_IMAGE="${MINIO_IMAGE:-$OFFICIAL_MINIO_IMAGE}"
  export MINIO_MC_IMAGE="${MINIO_MC_IMAGE:-$OFFICIAL_MINIO_MC_IMAGE}"
  export KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE="${KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE:-$OFFICIAL_ELASTICSEARCH_IMAGE}"
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

check_prepared_config_renderer() {
  local config_artifact="$TOOLS_DIR/config-ctl"
  local config_stamp config_fingerprint
  config_stamp="$(artifact_stamp_path config-ctl)"
  config_fingerprint="$(source_fingerprint "$ROOT_DIR/config/ctl")"
  require_current_artifact \
    "config renderer" \
    "$config_artifact" \
    "$config_stamp" \
    "$config_fingerprint" \
    "Rerun ./scripts/local/start.sh without --skip-prepare to build the config renderer for the current source." \
    "$ROOT_DIR/config/ctl" || return 1
}

ai_gateway_local_seed_enabled() {
  case "${AI_GATEWAY_LOCAL_SEED_ENABLED:-}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

check_prepared_local_tools() {
  check_prepared_config_renderer || return 1
  if (( ! SKIP_MIGRATIONS )); then
    require_file "$TOOLS_DIR/goose" "Rerun ./scripts/local/start.sh to install goose $GOOSE_VERSION." || return 1
  fi
  if (( ! SKIP_SEED )) && ai_gateway_local_seed_enabled; then
    local seed_artifact="$TOOLS_DIR/render-ai-gateway-local-seed"
    local seed_source="$ROOT_DIR/scripts/local/render_ai_gateway_local_seed.go"
    local seed_stamp seed_fingerprint
    seed_stamp="$(artifact_stamp_path render-ai-gateway-local-seed)"
    seed_fingerprint="$(source_fingerprint "$seed_source")"
    require_current_artifact \
      "AI Gateway local seed helper" \
      "$seed_artifact" \
      "$seed_stamp" \
      "$seed_fingerprint" \
      "Rerun ./scripts/local/start.sh without --skip-prepare to build the AI Gateway local seed helper for the current source." \
      "$seed_source" || return 1
  fi
}

check_start_prerequisites() {
  check_prepared_local_tools || return 1
  if (( ! INFRA_ONLY )); then
    local item name binary dir target artifact stamp fingerprint
    for item in "${GO_SERVICES[@]}"; do
      IFS='|' read -r name binary dir target <<<"$item"
      artifact="$BIN_DIR/$binary"
      stamp="$(artifact_stamp_path "$binary")"
      fingerprint="$(source_fingerprint "$dir")"
      require_current_artifact \
        "$name binary" \
        "$artifact" \
        "$stamp" \
        "$fingerprint" \
        "Rerun ./scripts/local/start.sh without --skip-prepare to build host-run service binaries for the current source." \
        "$dir" || return 1
    done
  fi
  if (( ! SKIP_INFRA )); then
    require_command_local docker "Install Docker and start the Docker daemon." || return 1
  fi
  if (( ! SKIP_MIGRATIONS || ! SKIP_SEED )); then
    require_command_local psql "Install the PostgreSQL client; the server runs in Docker." || return 1
  fi
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    require_command_local curl "Install curl for local runtime readiness checks." || return 1
  fi
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    local profile
    profile="$(runtime_sync_profile)"
    require_command_local uv "Install uv for Knowledge runtime preparation." || return 1
    require_command_local python3 "Install python3 for runtime readiness checks." || return 1
    require_file "$RUNTIME_DIR/.venv" "Rerun ./scripts/local/start.sh to prepare Knowledge runtime .venv." || return 1
    if ! runtime_dependencies_synced "$profile"; then
      log_error "Knowledge runtime .venv is not prepared for $profile dependencies"
      log_hint "Rerun ./scripts/local/start.sh to sync Knowledge runtime dependencies for the selected runtime mode."
      return 1
    fi
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
  if ! ai_gateway_local_seed_enabled; then
    log_info "AI_GATEWAY_LOCAL_SEED_ENABLED is not true; keeping seeded placeholder model profiles"
    return
  fi
  require_file "$TOOLS_DIR/render-ai-gateway-local-seed" "Rerun ./scripts/local/start.sh to build the AI Gateway local seed helper." || return 1
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
  local item name binary dir target
  for item in "${GO_SERVICES[@]}"; do
    IFS='|' read -r name binary dir target <<<"$item"
    start_process "$name" "$dir" "$BIN_DIR/$binary"
  done
}

parse_args "$@"
trap on_exit EXIT

log_info "starting local stack; missing local artifacts will be prepared"
run_step "preflighting host environment" preflight_host_environment
if (( ! SKIP_PREPARE )); then
  run_step "preparing local Go tools" prepare_local_tools
else
  run_step "checking prepared config renderer" check_prepared_config_renderer
fi
run_step "loading local config" load_config
if (( CHINA_MIRRORS )); then
  run_step "applying mainland China image rewrites" apply_china_sources
fi
if (( ! SKIP_PREPARE )); then
  run_step "preparing Docker infrastructure images" prepare_infra_images
  run_step "preparing Knowledge runtime dependencies" prepare_runtime_dependencies
  run_step "preparing backend service binaries" prepare_backend_binaries
fi
run_step "checking prepared artifacts" check_start_prerequisites
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
