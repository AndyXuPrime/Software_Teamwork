#!/usr/bin/env bash
# run_local_smoke.sh — Run all cross-service end-to-end smoke tests.
#
# Prerequisites:
#   docker compose up -d && docker compose --profile ai up -d
#   docker compose run --rm migrate-all
#   docker compose run --rm seed-all
#
# Usage:
#   bash services/deploy/smoke/run_local_smoke.sh          # run all
#   FILE_OWNER_E2E_SMOKE=1 bash ...                         # run specific smoke
#
# Required environment variables are set below with defaults
# matching the local docker-compose topology.  Override as needed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# ---- defaults matching docker-compose.yml ----
export GATEWAY_BASE_URL="${GATEWAY_BASE_URL:-http://localhost:8080}"
export FILE_SERVICE_BASE_URL="${FILE_SERVICE_BASE_URL:-http://localhost:8082}"
export PARSER_SERVICE_BASE_URL="${PARSER_SERVICE_BASE_URL:-http://localhost:8087}"
export KNOWLEDGE_SERVICE_BASE_URL="${KNOWLEDGE_SERVICE_BASE_URL:-http://localhost:8083}"
export QA_SERVICE_BASE_URL="${QA_SERVICE_BASE_URL:-http://localhost:8084}"
export DOCUMENT_SERVICE_BASE_URL="${DOCUMENT_SERVICE_BASE_URL:-http://localhost:8085}"

# ---- credentials from seed data ----
export LOCAL_ADMIN_USERNAME="${LOCAL_ADMIN_USERNAME:-admin}"
export LOCAL_ADMIN_PASSWORD="${LOCAL_ADMIN_PASSWORD:-LocalDemoAdmin#12345}"

# ---- simple arg parsing ----
SMOKE="${1:-all}"

cd "$REPO_ROOT/services/deploy/smoke"

case "$SMOKE" in
  file|all)
    echo "=== File Owner-Service E2E Smoke ==="
    FILE_OWNER_E2E_SMOKE=1 go test -v -count=1 -timeout=120s ./... -run TestFileOwnerE2ESmoke
    ;;
esac

case "$SMOKE" in
  qa|all)
    echo "=== QA MCP RAG Smoke ==="
    QA_MCP_RAG_SMOKE=1 go test -v -count=1 -timeout=180s ./... -run TestQAMCPRAGSmoke
    ;;
esac

case "$SMOKE" in
  document|all)
    echo "=== Document MCP Tool Smoke ==="
    DOCUMENT_MCP_SMOKE=1 go test -v -count=1 -timeout=120s ./... -run TestDocumentMCPToolSmoke
    ;;
esac

echo ""
echo "Smoke run complete."
