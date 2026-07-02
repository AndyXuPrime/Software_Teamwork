#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_DIR="$ROOT_DIR/.local/run"

[[ -d "$RUN_DIR" ]] || exit 0

shopt -s nullglob

for pid_file in "$RUN_DIR"/*.pid; do
  pid="$(cat "$pid_file")"
  name="$(basename "$pid_file" .pid)"

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
