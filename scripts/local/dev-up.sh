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

"${compose[@]}" config --quiet
"${compose[@]}" pull
"${compose[@]}" up -d

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
