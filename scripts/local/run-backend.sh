#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="$ROOT_DIR/deploy/.env"
RUN_DIR="$ROOT_DIR/.local/run"
LOG_DIR="$ROOT_DIR/.local/logs"

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

start() {
  name="$1"
  dir="$2"
  shift 2

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
}

(cd "$ROOT_DIR/services/parser" && uv sync --frozen --group dev --extra paddleocr)

start auth "$ROOT_DIR/services/auth" go run ./cmd/server
start file "$ROOT_DIR/services/file" go run ./cmd/server
start parser "$ROOT_DIR/services/parser" uv run --frozen parser-service
start knowledge "$ROOT_DIR/services/knowledge" go run ./cmd/server
start ai-gateway "$ROOT_DIR/services/ai-gateway" go run ./cmd/server
start qa "$ROOT_DIR/services/qa" go run ./cmd/server
start document "$ROOT_DIR/services/document" go run ./cmd/server
start gateway "$ROOT_DIR/services/gateway" go run ./cmd/server

echo "backend started; logs: .local/logs/*.log"
