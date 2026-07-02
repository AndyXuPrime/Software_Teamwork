# Implementation Plan

1. Update `deploy/docker-compose.yml` to remove all local business service, migration, and seed services; keep only PostgreSQL, Redis, Qdrant, MinIO, and minio-init.
2. Update `.env.example` comments and pinned image overrides so default local Docker is pull-only infrastructure with mainland-China-friendly registry rewrite; keep host-run service defaults centralized in `deploy/.env.example`.
3. Remove service Dockerfiles / service-level Compose build paths from the default repository workflow, then update references.
4. Update Docker policy checker and tests to enforce infra-only root Compose and no root Compose `build:` entries.
5. Update seed contract checker/tests or document that local seed validation changed because seed jobs no longer live in Compose.
6. Update CI Docker/deploy workflow so it no longer runs business-service Dockerfile builds; keep Compose config and policy checks.
7. Rewrite README, deploy README, local integration runbook, Docker image pull runbook, AGENTS.md, backend spec, and CICD spec around infra-only Docker plus host-run scripts.
8. Search for old default `docker compose ... --build`, `docker build`, `Dockerfile`, or business-service Docker startup references and remove them from local/default docs.
9. Run required validation and fix failures.
10. Commit, push, and prepare a new PR/issue flow for the infra-only direction.
