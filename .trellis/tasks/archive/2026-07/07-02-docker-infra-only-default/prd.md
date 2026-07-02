# Make local Docker infrastructure only

## Goal

Make repository Docker usage limited to infrastructure image pulls. The only remaining Docker path must pull and start infrastructure containers: PostgreSQL, Redis, Qdrant, MinIO, and minio-init. All business services run on the host with documented toolchain and environment requirements.

## Background

The team decided that Docker builds are too costly and fragile for ordinary development. Previous work in PR #462 optimized Parser Docker builds and moved the root Compose default toward a full business-service Docker stack, but that direction is superseded. PR #462 and issues #458/#461 were closed because the new direction is infra-only Docker by default.

## Requirements

- Repository Docker usage must be limited to pulling and starting required
  infrastructure images from Docker Hub or an explicit registry rewrite:
  `postgres`, `redis`, `qdrant`, `minio`, and `minio-init`.
- Root local Docker Compose must only contain infrastructure containers:
  `postgres`, `redis`, `qdrant`, `minio`, and `minio-init`.
- Root local Docker Compose must not build or start business services:
  `auth`, `file`, `knowledge`, `qa`, `document`, `gateway`, `ai-gateway`, or
  `parser`.
- Service-level Docker and service-level Compose paths are removed from the
  normal repository workflow; developers configure and run business services on
  the host.
- Local development docs must make host-run services the normal path, with
  `deploy/.env.example` as the default configuration source and short script
  entrypoints for infra/migration/seed and backend startup. Required local tools
  are Docker, Go `1.25.x`, uv, Bun, `psql` client, and curl.
- Business-service Docker commands must be removed from default user/developer startup docs.
- CI must not run business-service Docker builds for local/deploy checks. It
  may validate infrastructure Compose config and Docker pull policy only.
- Docker policy must enforce the new default: root local Compose cannot contain `build:` entries or business-service containers.
- Production/staging Compose and business-service packaging docs are removed from this repository baseline.
- AGENTS.md and Trellis backend/CICD specs must be updated so future agents and developers follow infra-only Docker defaults.
- If old docs conflict with this task, this task wins.

## Acceptance Criteria

- [ ] `deploy/docker-compose.yml` config services are exactly the allowed local Docker services plus declared volumes, with no `build:` path.
- [ ] `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --services` lists only `postgres`, `redis`, `qdrant`, `minio`, and `minio-init`.
- [ ] README and deploy docs start local infrastructure with `docker compose up -d` and start business services on the host.
- [ ] Docs no longer recommend Docker for business services in the default local workflow.
- [ ] CI Docker/deploy checks no longer handle business services.
- [ ] `scripts/check_docker_policy.py` and its tests enforce infra-only root Compose defaults.
- [ ] Validation commands run: Docker policy tests, Docker environment tests, root Compose config, root Compose service list, seed contract checks if still applicable or intentionally updated, workflow YAML syntax checks when workflow files change, and `git diff --check`.

## Out Of Scope

- Designing future business-service packaging or publishing.
- Making every service host-run command perfect E2E smoke in one pass; the required deliverable is clear startup guidance and removal of all non-infrastructure Docker paths from the repository baseline.
