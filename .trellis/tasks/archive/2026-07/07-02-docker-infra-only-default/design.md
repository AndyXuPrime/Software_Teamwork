# Design: Infra-only local Docker default

## Architecture Decision

Local Docker is only for shared infrastructure image pulls. Business services
are developer-owned host processes. This removes Docker build speed and
mirror/proxy issues from the normal startup path.

## Local Docker Boundary

Allowed services in `deploy/docker-compose.yml`:

- `postgres`
- `redis`
- `qdrant`
- `minio`
- `minio-init`

Disallowed in root local Compose:

- migration jobs built from repository Dockerfiles
- seed jobs tied to business migrations
- `auth`, `file`, `knowledge`, `qa`, `document`, `gateway`, `ai-gateway`, `parser`
- any `build:` entry
- any service Dockerfile, service-level Compose, or business packaging workflow

## Host-run Boundary

Go services use `go run ./cmd/server` from each service directory after applying
migrations with pinned goose commands. Parser uses `uv` from `services/parser`.
Frontend uses Bun from `apps/web`.

`deploy/.env.example` is the single default local configuration source. Users
copy it to `deploy/.env`; startup scripts load that file but do not duplicate
service env defaults or create env files for the user.

## Migrations And Seeds

Because migration Docker jobs are removed from default Compose, local migrations
run from host using pinned goose commands inside `scripts/local/dev-up.sh`.
Seed scripts are applied from host with `psql` after migrations.

## CI

Docker/deploy CI should validate Docker policy, environment helper tests, and
Compose config. It must not build business-service Dockerfiles. Service quality
remains covered by Go service CI, Parser CI, frontend CI, migration CI, Gateway
contract, and API type drift checks.

## Compatibility

Published business-service images and production/staging Compose are outside
this repository baseline.
