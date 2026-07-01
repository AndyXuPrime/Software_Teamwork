#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ ! -d .venv ]]; then
  echo "Run: uv sync --python 3.13 --frozen" >&2
  exit 1
fi

export PYTHONPATH=.
export RAGFLOW_CONF="${RAGFLOW_CONF:-$ROOT/conf/service_conf.yaml}"

if [[ ! -f "$RAGFLOW_CONF" ]]; then
  if [[ -f "$ROOT/conf/service_conf.compose.yaml" ]]; then
    cp "$ROOT/conf/service_conf.compose.yaml" "$RAGFLOW_CONF"
    echo "Copied conf/service_conf.compose.yaml -> $RAGFLOW_CONF (edit hosts for local)" >&2
  else
    echo "Missing $RAGFLOW_CONF" >&2
    exit 1
  fi
fi

exec uv run python api/ragflow_server.py
