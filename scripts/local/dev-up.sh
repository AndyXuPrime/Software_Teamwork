#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="$ROOT_DIR/deploy/.env"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.yml"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing deploy/.env; run: cp deploy/.env.example deploy/.env" >&2
  exit 1
fi

# deploy/.env is copied by the user from deploy/.env.example. The script does
# not own defaults; it only exposes that file to host migration/seed commands.
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

compose=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")

initialize_qdrant_collection() {
  qdrant_url="${QDRANT_URL:-}"

  if [[ -z "$qdrant_url" ]]; then
    echo "QDRANT_URL is empty; skipping Qdrant collection initialization"
    return
  fi
  qdrant_collection="${QDRANT_COLLECTION:?QDRANT_COLLECTION must be set in deploy/.env}"
  embedding_dimension="${EMBEDDING_DIMENSION:?EMBEDDING_DIMENSION must be set in deploy/.env}"
  if [[ ! "$qdrant_collection" =~ ^[A-Za-z0-9_.-]+$ ]]; then
    echo "QDRANT_COLLECTION must contain only letters, numbers, dots, underscores, or hyphens" >&2
    exit 1
  fi
  if [[ ! "$embedding_dimension" =~ ^[1-9][0-9]*$ ]]; then
    echo "EMBEDDING_DIMENSION must be a positive integer" >&2
    exit 1
  fi

  qdrant_url="${qdrant_url%/}"
  response_file="$(mktemp)"
  status="$(
    curl --noproxy '*' -sS -o "$response_file" -w '%{http_code}' \
      "$qdrant_url/collections/$qdrant_collection" || true
  )"

  case "$status" in
    200)
      compact_response="$(tr -d '[:space:]' <"$response_file")"
      rm -f "$response_file"
      if [[ "$compact_response" != *"\"vectors\":{\"size\":$embedding_dimension"* ]] ||
        [[ "$compact_response" != *"\"distance\":\"Cosine\""* ]]; then
        echo "Qdrant collection $qdrant_collection exists but does not match EMBEDDING_DIMENSION=$embedding_dimension" >&2
        exit 1
      fi
      echo "Qdrant collection $qdrant_collection is ready"
      ;;
    404)
      rm -f "$response_file"
      echo "creating Qdrant collection $qdrant_collection"
      curl --noproxy '*' -fsS -X PUT "$qdrant_url/collections/$qdrant_collection" \
        -H 'Content-Type: application/json' \
        --data "{\"vectors\":{\"size\":$embedding_dimension,\"distance\":\"Cosine\"}}" >/dev/null
      ;;
    *)
      echo "could not inspect Qdrant collection $qdrant_collection at $qdrant_url (HTTP $status)" >&2
      rm -f "$response_file"
      exit 1
      ;;
  esac
}

"${compose[@]}" config --quiet
"${compose[@]}" pull
"${compose[@]}" up -d --wait --wait-timeout "${LOCAL_INFRA_WAIT_TIMEOUT_SECONDS:-180}"

initialize_qdrant_collection

for item in \
  "auth:$AUTH_DATABASE_URL" \
  "file:$FILE_DATABASE_URL" \
  "knowledge:$KNOWLEDGE_DATABASE_URL" \
  "qa:$QA_DATABASE_URL" \
  "document:$DOCUMENT_DATABASE_URL" \
  "ai-gateway:$AI_GATEWAY_DATABASE_URL"; do
  service="${item%%:*}"
  database_url="${item#*:}"
  echo "migrating $service"
  (
    cd "$ROOT_DIR/services/$service"
    go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$database_url" up
  )
done

psql "$POSTGRES_ADMIN_URL" \
  -v ON_ERROR_STOP=1 \
  -f "$ROOT_DIR/deploy/seeds/001-local-demo-seed.sql" \
  -f "$ROOT_DIR/deploy/seeds/002-ai-gateway-model-profiles.sql"

echo "infra, migrations, and seed are ready"
