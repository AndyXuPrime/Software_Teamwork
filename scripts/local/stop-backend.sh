#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUN_DIR="$ROOT_DIR/.local/run"

[[ -d "$RUN_DIR" ]] || exit 0

for pid_file in "$RUN_DIR"/*.pid; do
  [[ -e "$pid_file" ]] || exit 0
  pid="$(cat "$pid_file")"
  name="$(basename "$pid_file" .pid)"
  echo "stopping $name"
  kill "$pid" 2>/dev/null || true
  rm -f "$pid_file"
done
