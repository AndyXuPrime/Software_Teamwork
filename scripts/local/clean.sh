#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CONFIG_LOADER="$ROOT_DIR/scripts/config/load-profile.sh"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"
CURRENT_STEP="initializing"
YES=0
STOP_PROCESSES=1
REMOVE_ORPHANS=1

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

log_info() { printf '%b%s %s%b\n' "$COLOR_BLUE" "[clean]" "$*" "$COLOR_RESET"; }
log_ok() { printf '%b%s %s%b\n' "$COLOR_GREEN" "[ok]" "$*" "$COLOR_RESET"; }
log_warn() { printf '%b%s %s%b\n' "$COLOR_YELLOW" "[warn]" "$*" "$COLOR_RESET" >&2; }
log_error() { printf '%b%s %s%b\n' "$COLOR_RED" "[fail]" "$*" "$COLOR_RESET" >&2; }
log_hint() { printf '%b%s %s%b\n' "$COLOR_CYAN" "[hint]" "$*" "$COLOR_RESET" >&2; }

usage() {
  cat <<'EOF'
Usage: ./scripts/local/clean.sh [options]

Stops host-run processes and removes local Compose-managed data volumes.
Images, source files, .env.local, .local/tools, and .local/bin are not removed.
Requires prepared .local/tools/config-ctl to render the Compose env file.

Options:
  -y, --yes          Skip the interactive confirmation prompt.
  --skip-stop        Do not run ./scripts/local/stop.sh first.
  --keep-orphans     Do not pass --remove-orphans to docker compose down.
  --profile <name>   Set CONFIG_PROFILE for scripts/config/load-profile.sh.
  --secret-file <p>  Set CONFIG_SECRET_FILE for scripts/config/load-profile.sh.
  -h, --help         Show this help.
EOF
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      -y|--yes|--force)
        YES=1
        ;;
      --skip-stop)
        STOP_PROCESSES=0
        ;;
      --keep-orphans)
        REMOVE_ORPHANS=0
        ;;
      --profile)
        [[ $# -ge 2 ]] || { log_error "--profile requires a value"; exit 2; }
        export CONFIG_PROFILE="$2"
        shift
        ;;
      --secret-file)
        [[ $# -ge 2 ]] || { log_error "--secret-file requires a value"; exit 2; }
        export CONFIG_SECRET_FILE="$2"
        shift
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
}

on_exit() {
  local status=$?
  if (( status == 0 )); then
    log_ok "completed successfully"
  else
    log_error "failed during ${CURRENT_STEP} (exit ${status})"
    log_hint "Run ./scripts/local/start.sh to prepare missing local prerequisites."
  fi
}

run_step() {
  CURRENT_STEP="$1"
  shift
  log_info "$CURRENT_STEP"
  "$@"
  log_ok "$CURRENT_STEP succeeded"
}

confirm_clean() {
  if (( YES )); then
    return
  fi
  log_warn "This deletes local PostgreSQL, MinIO, and Elasticsearch data volumes."
  log_warn "Docker images and prepared local binaries are kept."
  if [[ ! -t 0 ]]; then
    log_error "refusing to clean without --yes in a non-interactive shell"
    return 2
  fi
  printf 'Type "clean" to continue: ' >&2
  local answer
  read -r answer
  [[ "$answer" == "clean" ]] || { log_error "aborted"; return 2; }
}

stop_processes() {
  if (( ! STOP_PROCESSES )); then
    log_info "skipping host-run process stop"
    return
  fi
  "$ROOT_DIR/scripts/local/stop.sh"
}

load_config() {
  export SOFTWARE_TEAMWORK_ROOT="$ROOT_DIR"
  export CONFIG_CTL_REQUIRE_PREPARED=1
  # shellcheck disable=SC1090
  . "$CONFIG_LOADER"
}

compose_down() {
  local args=(down -v)
  if (( REMOVE_ORPHANS )); then
    args+=(--remove-orphans)
  fi
  docker compose -f "$COMPOSE_FILE" --env-file "$CONFIG_COMPOSE_ENV_FILE" "${args[@]}"
}

parse_args "$@"
trap on_exit EXIT

run_step "confirming local data cleanup" confirm_clean
run_step "stopping host-run processes" stop_processes
run_step "loading local config" load_config
run_step "validating Docker Compose config" docker compose -f "$COMPOSE_FILE" --env-file "$CONFIG_COMPOSE_ENV_FILE" config --quiet
run_step "removing local development data volumes" compose_down
log_ok "local development data cleared; run ./scripts/local/start.sh before starting again"
