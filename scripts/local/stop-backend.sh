#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_DIR="$ROOT_DIR/.local/run"
CURRENT_STEP="initializing"
STOPPED_COUNT=0

on_exit() {
  status=$?
  if (( status == 0 )); then
    echo "local backend stop: completed successfully; processed ${STOPPED_COUNT} pid file(s)"
  else
    echo "local backend stop: failed during ${CURRENT_STEP} (exit ${status})" >&2
    echo "Check .local/run/*.pid and running service processes manually." >&2
  fi
}
trap on_exit EXIT

echo "local backend stop: starting"

if [[ ! -d "$RUN_DIR" ]]; then
  echo "local backend stop: no .local/run directory; nothing to stop"
  exit 0
fi

shopt -s nullglob
pid_files=("$RUN_DIR"/*.pid)

if (( ${#pid_files[@]} == 0 )); then
  echo "local backend stop: no pid files found; nothing to stop"
  exit 0
fi

for pid_file in "${pid_files[@]}"; do
  pid="$(cat "$pid_file")"
  name="$(basename "$pid_file" .pid)"
  CURRENT_STEP="stopping $name"
  STOPPED_COUNT=$((STOPPED_COUNT + 1))

  if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
    echo "removing invalid pid file for $name"
    rm -f "$pid_file"
    continue
  fi

  echo "stopping $name"
  if kill -0 -- "-$pid" 2>/dev/null; then
    kill -TERM -- "-$pid" 2>/dev/null || true
    for _ in {1..25}; do
      kill -0 -- "-$pid" 2>/dev/null || break
      sleep 0.2
    done
    kill -0 -- "-$pid" 2>/dev/null && kill -KILL -- "-$pid" 2>/dev/null || true
  elif kill -0 "$pid" 2>/dev/null; then
    kill -TERM "$pid" 2>/dev/null || true
  else
    echo "$name was not running"
  fi
  rm -f "$pid_file"
done
