#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="$ROOT_DIR/deploy/.env"
RUN_DIR="$ROOT_DIR/.local/run"
LOG_DIR="$ROOT_DIR/.local/logs"
CURRENT_STEP="initializing"

GO_SERVICES=(
  "auth|$ROOT_DIR/services/auth|./cmd/server"
  "file|$ROOT_DIR/services/file|./cmd/server"
  "knowledge|$ROOT_DIR/services/knowledge|./cmd/adapter"
  "ai-gateway|$ROOT_DIR/services/ai-gateway|./cmd/server"
  "qa|$ROOT_DIR/services/qa|./cmd/server"
  "document|$ROOT_DIR/services/document|./cmd/server"
  "gateway|$ROOT_DIR/services/gateway|./cmd/server"
)
STARTED_SERVICES=()

on_exit() {
  status=$?
  if (( status == 0 )); then
    echo "local backend startup: completed successfully"
  else
    echo "local backend startup: failed during ${CURRENT_STEP} (exit ${status})" >&2
    echo "Check service logs under .local/logs/ and current process files under .local/run/." >&2
  fi
}
trap on_exit EXIT

echo "local backend startup: starting Go module checks and host services"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing deploy/.env; run: cp deploy/.env.example deploy/.env" >&2
  exit 1
fi

if ! command -v setsid >/dev/null 2>&1; then
  echo "setsid is required to manage host-run service process groups" >&2
  exit 1
fi

# deploy/.env is copied by the user from deploy/.env.example. The script does
# not own defaults; it only exposes that file to host child processes.
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

mkdir -p "$RUN_DIR" "$LOG_DIR"

print_go_module_hint() {
  cat >&2 <<EOF
Go module download failed before backend startup completed.

Current effective Go module settings:
  GOPROXY=${GOPROXY:-<unset>}
  GOSUMDB=${GOSUMDB:-<unset>}

The repository default for mainland China developer networks is:
  GOPROXY=https://goproxy.cn,direct
  GOSUMDB=sum.golang.google.cn

If your deploy/.env was copied before those defaults existed, add the two lines
above or recopy deploy/.env.example and reapply only your local private changes.
EOF
}

print_startup_failure_hint() {
  cat >&2 <<EOF
Backend process startup failed after services were forked.

Use the log tails above first. If a log shows proxy.golang.org, sum.golang.org,
i/o timeout, or go: downloading before exit, check these effective Go module
settings:
  GOPROXY=${GOPROXY:-<unset>}
  GOSUMDB=${GOSUMDB:-<unset>}

For port binding, database, Redis, token, or runtime dependency errors, follow
the specific service log instead of treating it as a Go module mirror issue.
EOF
}

check_go_module_settings() {
  effective_goproxy="${GOPROXY:-$(go env GOPROXY 2>/dev/null || true)}"
  effective_gosumdb="${GOSUMDB:-$(go env GOSUMDB 2>/dev/null || true)}"

  if [[ -z "${GOPROXY:-}" && ( -z "$effective_goproxy" || "$effective_goproxy" == *"proxy.golang.org"* ) ]]; then
    export GOPROXY="https://goproxy.cn,direct"
    echo "deploy/.env did not set GOPROXY; using repository default for this run: $GOPROXY"
  elif [[ -z "${GOPROXY:-}" ]]; then
    export GOPROXY="$effective_goproxy"
    echo "deploy/.env did not set GOPROXY; using global go env value: $GOPROXY"
  fi

  if [[ -z "${GOSUMDB:-}" && ( -z "$effective_gosumdb" || "$effective_gosumdb" == "sum.golang.org" ) ]]; then
    export GOSUMDB="sum.golang.google.cn"
    echo "deploy/.env did not set GOSUMDB; using repository default for this run: $GOSUMDB"
  elif [[ -z "${GOSUMDB:-}" ]]; then
    export GOSUMDB="$effective_gosumdb"
    echo "deploy/.env did not set GOSUMDB; using global go env value: $GOSUMDB"
  fi

  if [[ "$GOPROXY" == *"proxy.golang.org"* ]]; then
    echo "warning: GOPROXY includes proxy.golang.org; this may time out on some developer networks" >&2
    echo "current GOPROXY=$GOPROXY" >&2
  fi
}

check_go_modules() {
  CURRENT_STEP="checking Go module downloads"
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required for host-run backend services" >&2
    exit 1
  fi

  check_go_module_settings

  for service in "${GO_SERVICES[@]}"; do
    IFS='|' read -r name dir _go_target <<<"$service"
    CURRENT_STEP="checking Go modules for $name"
    echo "checking Go modules for $name"
    if ! run_go_mod_download "$dir"; then
      echo "failed to download Go modules for $name" >&2
      print_go_module_hint
      return 1
    fi
  done
}

run_go_mod_download() {
  local dir="$1"
  local timeout_seconds="${LOCAL_GO_MOD_DOWNLOAD_TIMEOUT_SECONDS:-180}"
  local status

  if command -v timeout >/dev/null 2>&1 && [[ "$timeout_seconds" =~ ^[0-9]+$ ]] && (( timeout_seconds > 0 )); then
    set +e
    (cd "$dir" && timeout "$timeout_seconds" go mod download)
    status=$?
    set -e
    if (( status == 124 )); then
      echo "go mod download timed out after ${timeout_seconds}s in $dir" >&2
    fi
    return "$status"
  fi

  (cd "$dir" && go mod download)
}

start() {
  name="$1"
  dir="$2"
  shift 2
  CURRENT_STEP="starting $name"

  if [[ -f "$RUN_DIR/$name.pid" ]]; then
    pid="$(cat "$RUN_DIR/$name.pid")"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 -- "-$pid" 2>/dev/null; then
      echo "$name already running"
      return
    fi
  fi
  rm -f "$RUN_DIR/$name.pid"

  echo "starting $name"
  (cd "$dir" && exec setsid "$@") >"$LOG_DIR/$name.log" 2>&1 &
  echo "$!" >"$RUN_DIR/$name.pid"
  STARTED_SERVICES+=("$name")
}

check_started_services() {
  CURRENT_STEP="checking backend processes"
  local wait_seconds="${LOCAL_BACKEND_STARTUP_CHECK_SECONDS:-8}"
  local failed=()

  if [[ "$wait_seconds" =~ ^[0-9]+$ ]] && (( wait_seconds > 0 )) && (( ${#STARTED_SERVICES[@]} > 0 )); then
    echo "checking backend processes for ${wait_seconds}s"
    sleep "$wait_seconds"
  fi

  for name in "${STARTED_SERVICES[@]}"; do
    pid_file="$RUN_DIR/$name.pid"
    if [[ ! -f "$pid_file" ]]; then
      failed+=("$name")
      continue
    fi

    pid="$(cat "$pid_file")"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 -- "-$pid" 2>/dev/null; then
      continue
    fi

    failed+=("$name")
  done

  if (( ${#failed[@]} == 0 )); then
    return 0
  fi

  echo "backend startup failed for: ${failed[*]}" >&2
  echo "The failed service log tails are shown below." >&2
  for name in "${failed[@]}"; do
    log_file="$LOG_DIR/$name.log"
    echo "----- $log_file (tail) -----" >&2
    if [[ -f "$log_file" ]]; then
      tail -n 40 "$log_file" >&2
    else
      echo "log file missing" >&2
    fi
  done
  print_startup_failure_hint
  return 1
}

check_go_modules

for service in "${GO_SERVICES[@]}"; do
  IFS='|' read -r name dir go_target <<<"$service"
  start "$name" "$dir" go run "$go_target"
done

check_started_services

echo "backend started; logs: .local/logs/*.log"
