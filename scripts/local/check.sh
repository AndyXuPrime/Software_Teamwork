#!/usr/bin/env bash
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
TOOLS_DIR="$ROOT_DIR/.local/tools"
BIN_DIR="$ROOT_DIR/.local/bin"
RUNTIME_DIR="$ROOT_DIR/services/knowledge-runtime"
LOCAL_LIB_DIR="$ROOT_DIR/scripts/local/lib"
SHOW_CHINA=0
RUNTIME_MODE="full"
ISSUES=0
WARNINGS=0

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
  "auth|auth-server|services/auth|./cmd/server"
  "file|file-server|services/file|./cmd/server"
  "knowledge|knowledge-adapter|services/knowledge|./cmd/adapter"
  "ai-gateway|ai-gateway-server|services/ai-gateway|./cmd/server"
  "qa|qa-server|services/qa|./cmd/server"
  "document|document-server|services/document|./cmd/server"
  "gateway|gateway-server|services/gateway|./cmd/server"
)

# shellcheck source=scripts/local/lib/common.sh
. "$LOCAL_LIB_DIR/common.sh"

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

log_info() { printf '%b%s %s%b\n' "$COLOR_BLUE" "[check]" "$*" "$COLOR_RESET"; }
log_ok() { printf '%b%s %s%b\n' "$COLOR_GREEN" "[ok]" "$*" "$COLOR_RESET"; }
log_warn() { printf '%b%s %s%b\n' "$COLOR_YELLOW" "[warn]" "$*" "$COLOR_RESET"; }
log_error() { printf '%b%s %s%b\n' "$COLOR_RED" "[fail]" "$*" "$COLOR_RESET"; }
log_hint() { printf '%b%s %s%b\n' "$COLOR_CYAN" "[hint]" "$*" "$COLOR_RESET"; }

usage() {
  cat <<'EOF'
Usage: ./scripts/local/check.sh [options]

Checks the local environment and prints setup suggestions. It never downloads,
builds, pulls Docker images, syncs uv environments, or edits .env.local.

Options:
  --china             Show mainland China mirror suggestions first.
  --runtime full      Check Knowledge runtime API and worker files. Default.
  --runtime api       Check only Knowledge runtime API files.
  --runtime none      Skip Knowledge runtime file checks.
  -h, --help          Show this help.
EOF
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      --china)
        SHOW_CHINA=1
        ;;
      --runtime)
        if [[ $# -lt 2 ]]; then
          log_error "--runtime requires full, api, or none"
          exit 2
        fi
        RUNTIME_MODE="$2"
        shift
        ;;
      --runtime=*)
        RUNTIME_MODE="${1#--runtime=}"
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
    full|api|none) ;;
    *)
      log_error "--runtime must be full, api, or none"
      exit 2
      ;;
  esac
}

mark_issue() {
  ISSUES=$((ISSUES + 1))
  log_warn "$*"
}

mark_warning() {
  WARNINGS=$((WARNINGS + 1))
  log_warn "$*"
}

have_command() {
  command -v "$1" >/dev/null 2>&1
}

check_command() {
  local name="$1"
  local install_hint="$2"
  if have_command "$name"; then
    log_ok "$name: $(command -v "$name")"
    return
  fi
  mark_issue "$name missing"
  log_hint "$install_hint"
}

check_file() {
  local label="$1"
  local path="$2"
  local hint="$3"
  if [[ -e "$path" ]]; then
    log_ok "$label: $path"
    return
  fi
  mark_issue "$label missing: $path"
  log_hint "$hint"
}

check_optional_file() {
  local label="$1"
  local path="$2"
  local hint="$3"
  if [[ -e "$path" ]]; then
    log_ok "$label: $path"
    return
  fi
  mark_warning "$label missing: $path"
  log_hint "$hint"
}

section() {
  printf '\n%b%s %s%b\n' "$COLOR_BLUE" "[check]" "$*" "$COLOR_RESET"
}

check_core_commands() {
  section "core commands"
  check_command docker "Install Docker Desktop/Engine; official docs: https://docs.docker.com/engine/install/"
  check_command go "Install Go 1.25.x; official downloads: https://go.dev/dl/"
  check_command psql "Install PostgreSQL client tools; Ubuntu example: sudo apt-get install postgresql-client"
  check_command bun "Install Bun; official docs: https://bun.sh/docs/installation"
  check_command curl "Install curl from your OS package manager."
  check_command python3 "Install Python 3.10+ from your OS package manager or https://www.python.org/downloads/"
  if [[ "$RUNTIME_MODE" != "none" ]]; then
    check_command uv "Install uv; official docs: https://docs.astral.sh/uv/getting-started/installation/"
  fi
}

check_local_env() {
  section "local env"
  if [[ -f "$ROOT_DIR/.env.local" ]]; then
    log_ok "local env: $ROOT_DIR/.env.local"
  else
    mark_issue "local env missing: $ROOT_DIR/.env.local"
    log_hint "Create it from the template: cp .env.example .env.local"
  fi
}

check_local_tools() {
  section "local tools and service binaries"
  check_file "config renderer" "$TOOLS_DIR/config-ctl" "Build once: (cd config/ctl && go build -o ../../.local/tools/config-ctl .)"
  check_file "goose" "$TOOLS_DIR/goose" "Install once: mkdir -p .local/tools && GOBIN=\"$ROOT_DIR/.local/tools\" go install github.com/pressly/goose/v3/cmd/goose@v3.27.1"
  check_file "AI Gateway local seed helper" "$TOOLS_DIR/render-ai-gateway-local-seed" "Build once: go build -o .local/tools/render-ai-gateway-local-seed ./scripts/local/render_ai_gateway_local_seed.go"

  local item service binary dir target
  for item in "${GO_SERVICES[@]}"; do
    IFS='|' read -r service binary dir target <<<"$item"
    check_file "$service binary" "$BIN_DIR/$binary" "Build once: mkdir -p .local/bin && (cd $dir && go build -o ../../.local/bin/$binary $target)"
  done
}

check_config_render() {
  section "rendered config"
  if [[ ! -x "$TOOLS_DIR/config-ctl" ]]; then
    mark_warning "Compose config not rendered because .local/tools/config-ctl is missing"
    log_hint "Build config-ctl first, then rerun ./scripts/local/check.sh"
    return
  fi
  if [[ ! -f "$ROOT_DIR/.env.local" ]]; then
    mark_warning "Compose config not rendered because .env.local is missing"
    return
  fi
  (
    cd "$ROOT_DIR"
    CONFIG_CTL_REQUIRE_PREPARED=1 CONFIG_SECRET_FILE=.env.local ./scripts/config/load-profile.sh --print-compose-env >/dev/null
  )
  local status=$?
  if (( status == 0 )); then
    log_ok "rendered config: $ROOT_DIR/.local/config/dev.env"
  else
    mark_issue "config render failed"
    log_hint "Check .env.local values against .env.example, then rerun ./scripts/local/check.sh"
  fi
}

check_compose_config() {
  section "Docker Compose config"
  if [[ ! -f "$ROOT_DIR/.local/config/dev.env" ]]; then
    mark_warning "Compose config skipped because .local/config/dev.env does not exist yet"
    log_hint "Build .local/tools/config-ctl and keep .env.local present, then rerun ./scripts/local/check.sh"
    return
  fi
  if ! have_command docker; then
    mark_warning "Compose config skipped because docker is missing"
    return
  fi
  if docker compose -f "$COMPOSE_FILE" --env-file "$ROOT_DIR/.local/config/dev.env" config --quiet; then
    log_ok "Compose config parses"
  else
    mark_issue "Compose config failed to parse"
    log_hint "Run: docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config"
  fi
}

check_runtime_files() {
  [[ "$RUNTIME_MODE" == "none" ]] && return

  section "Knowledge runtime"
  check_file "Knowledge runtime .venv" "$RUNTIME_DIR/.venv" "Official setup: cd services/knowledge-runtime && uv sync --python 3.13 --frozen --no-install-project --group worker"
  check_optional_file "Tika jar" "$RUNTIME_DIR/ragflow_deps/tika-server-standard-3.3.0.jar" "Official artifacts: cd services/knowledge-runtime && uv run --no-project --with 'nltk>=3.9.4' --with 'huggingface-hub>=1.3.1' ragflow_deps/download_deps.py --skip-uv-sync"
  check_optional_file "cl100k encoding" "$RUNTIME_DIR/ragflow_deps/cl100k_base.tiktoken" "Official artifacts: cd services/knowledge-runtime && uv run --no-project --with 'nltk>=3.9.4' --with 'huggingface-hub>=1.3.1' ragflow_deps/download_deps.py --skip-uv-sync"
  if [[ "$RUNTIME_MODE" == "full" ]]; then
    check_file "NLTK data" "$RUNTIME_DIR/ragflow_deps/nltk_data" "Official artifacts: cd services/knowledge-runtime && uv run --no-project --with 'nltk>=3.9.4' --with 'huggingface-hub>=1.3.1' ragflow_deps/download_deps.py --skip-uv-sync"
    check_optional_file "deepdoc models" "$RUNTIME_DIR/rag/res/deepdoc" "Official model download is handled by ragflow_deps/download_deps.py; mainland mirror: add --china."
  fi
}

print_download_suggestions() {
  section "setup suggestions"
  if (( SHOW_CHINA )); then
    printf '%s\n' \
      "Mainland China mirrors, run manually only for missing items:" \
      "  docker pull $CHINA_POSTGRES_IMAGE" \
      "  docker pull $CHINA_REDIS_IMAGE" \
      "  docker pull $CHINA_MINIO_IMAGE" \
      "  docker pull $CHINA_MINIO_MC_IMAGE" \
      "  docker pull $CHINA_ELASTICSEARCH_IMAGE" \
      "  GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn go install github.com/pressly/goose/v3/cmd/goose@v3.27.1"
    if [[ "$RUNTIME_MODE" != "none" ]]; then
      printf '%s\n' \
        "  cd services/knowledge-runtime && python3 ragflow_deps/download_deps.py --sync-only --profile worker --china" \
        "  cd services/knowledge-runtime && uv run --no-project --with 'nltk>=3.9.4' --with 'huggingface-hub>=1.3.1' ragflow_deps/download_deps.py --skip-uv-sync --china"
    fi
    printf '%s\n' \
      "" \
      "Official sources:" \
      "  Docker Hub / Elastic registry images from deploy/docker-compose.yml" \
      "  Go modules via GOPROXY=https://proxy.golang.org,direct and GOSUMDB=sum.golang.org" \
      "  PyPI via UV_DEFAULT_INDEX=https://pypi.org/simple"
  else
    printf '%s\n' \
      "Official sources, run manually only for missing items:" \
      "  docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull postgres redis minio minio-init elasticsearch" \
      "  mkdir -p .local/tools .local/bin" \
      "  (cd config/ctl && go build -o ../../.local/tools/config-ctl .)" \
      "  GOBIN=\"$ROOT_DIR/.local/tools\" go install github.com/pressly/goose/v3/cmd/goose@v3.27.1" \
      "  go build -o .local/tools/render-ai-gateway-local-seed ./scripts/local/render_ai_gateway_local_seed.go"
    if [[ "$RUNTIME_MODE" != "none" ]]; then
      printf '%s\n' \
        "  cd services/knowledge-runtime && uv sync --python 3.13 --frozen --no-install-project --group worker" \
        "  cd services/knowledge-runtime && uv run --no-project --with 'nltk>=3.9.4' --with 'huggingface-hub>=1.3.1' ragflow_deps/download_deps.py --skip-uv-sync"
    fi
    printf '%s\n' \
      "" \
      "Mainland China alternatives:" \
      "  ./scripts/local/check.sh --china" \
      "  Use docker.1ms.run image names, GOPROXY=https://goproxy.cn,direct, and runtime helper --china."
  fi
}

parse_args "$@"
log_info "checking local environment; no downloads or builds will run"
check_core_commands
check_local_env
check_local_tools
check_config_render
check_compose_config
check_runtime_files
print_download_suggestions

if (( ISSUES > 0 )); then
  log_error "local environment has ${ISSUES} missing required item(s) and ${WARNINGS} warning(s)"
  log_hint "Fix the missing items above, then rerun ./scripts/local/check.sh"
  exit 1
fi

if (( WARNINGS > 0 )); then
  log_warn "local environment has ${WARNINGS} warning(s); startup may still work if those paths are unused"
else
  log_ok "local environment looks ready"
fi
