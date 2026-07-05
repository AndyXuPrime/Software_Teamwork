#!/usr/bin/env sh
set -eu

if [ "${DOCKER_MIGRATE_ENABLED:-true}" = "false" ]; then
  echo "docker migrate: skipped"
  exit 0
fi

migrate_service() {
  service="$1"
  database_url="$2"
  if [ -z "$database_url" ]; then
    echo "docker migrate: $service database URL is empty" >&2
    exit 1
  fi
  echo "docker migrate: $service"
  goose -dir "/workspace/services/$service/migrations" postgres "$database_url" up
}

migrate_service auth "$AUTH_DATABASE_URL"
migrate_service file "$FILE_DATABASE_URL"
migrate_service knowledge "$KNOWLEDGE_DATABASE_URL"
migrate_service qa "$QA_DATABASE_URL"
migrate_service document "$DOCUMENT_DATABASE_URL"
migrate_service ai-gateway "$AI_GATEWAY_DATABASE_URL"
