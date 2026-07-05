#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.cloud.yml"
ENV_FILE="${DOCKER_CLOUD_ENV_FILE:-$ROOT_DIR/.env.docker.cloud}"
YES=0

while (($#)); do
  case "$1" in
    --yes|-y)
      YES=1
      shift
      ;;
    --env-file)
      if [[ $# -lt 2 ]]; then
        printf '[docker] ERROR: --env-file requires a path\n' >&2
        exit 2
      fi
      ENV_FILE="$2"
      shift 2
      ;;
    *)
      printf '[docker] ERROR: unknown argument: %s\n' "$1" >&2
      exit 2
      ;;
  esac
done

if [[ ! -f "$ENV_FILE" ]]; then
  ENV_FILE="$ROOT_DIR/deploy/docker/cloud.env.example"
fi

if (( ! YES )); then
  printf 'This stops the cloud Docker app stack and removes local compose volumes. Continue? [y/N] '
  read -r answer
  case "$answer" in
    y|Y|yes|YES) ;;
    *) exit 0 ;;
  esac
fi

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" down -v --remove-orphans
