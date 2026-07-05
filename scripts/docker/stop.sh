#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.cloud.yml"
ENV_FILE="${DOCKER_CLOUD_ENV_FILE:-$ROOT_DIR/.env.docker.cloud}"

if [[ "${1:-}" == "--env-file" ]]; then
  if [[ $# -lt 2 ]]; then
    printf '[docker] ERROR: --env-file requires a path\n' >&2
    exit 2
  fi
  ENV_FILE="$2"
  shift 2
fi

if [[ ! -f "$ENV_FILE" ]]; then
  ENV_FILE="$ROOT_DIR/deploy/docker/cloud.env.example"
fi

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" down --remove-orphans "$@"
