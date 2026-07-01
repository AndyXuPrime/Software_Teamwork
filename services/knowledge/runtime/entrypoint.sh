#!/usr/bin/env bash
set -euo pipefail

mode="${KNOWLEDGE_RUNTIME_MODE:-legacy}"

case "$mode" in
  adapter)
    exec /usr/local/bin/knowledge-adapter
    ;;
  legacy)
    exec /usr/local/bin/knowledge
    ;;
  *)
    echo "unsupported KNOWLEDGE_RUNTIME_MODE: $mode (expected legacy or adapter)" >&2
    exit 1
    ;;
esac
