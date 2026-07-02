# S-031 Production Compose Baseline

## Goal

Deliver an explicit production or staging Docker Compose deployment baseline for issue #306 so operators can distinguish it from the existing `deploy/docker-compose.yml` local/demo stack and prepare a single-machine environment without inheriting local weak secrets, in-memory storage defaults, or demo-only seed behavior.

## Background

- Issue #306 requires an independent production/staging baseline, an environment variable template, persistence guidance, secret handling, startup order, health checks, upgrade, rollback, and documentation sync.
- `deploy/README.md` and `docs/runbooks/local-integration.md` already state that `deploy/docker-compose.yml` is a local/demo integration baseline, not production.
- The local stack currently uses local placeholders such as `local-demo-postgres-password`, local admin seed data, local File storage, optional in-memory Knowledge vector index, and fake AI Gateway seed profiles.
- The CI/CD spec requires production-like deployment docs to keep runtime secrets outside the repository, expose only frontend/gateway publicly by default, use named volumes for PostgreSQL/Redis/Qdrant/MinIO persistence, and document rollback before production use.
- Issue #306 also has a follow-up comment requiring Docker Hub/build reliability and parser image-cache guidance so validation does not depend on a developer machine happening to have a cached image.
- Dependencies #286, #289, and #304 are closed and provide local File MinIO/PostgreSQL, Knowledge real-dependency, and Gateway -> Knowledge -> QA RAG smoke inputs. Issue #125 remains open and is out of scope except as a future full smoke reference.

## Requirements

1. Provide an independent production/staging baseline that is visibly separate from the local/demo Compose file.
2. Provide a production/staging environment template with required variable names for all landed services and infrastructure.
3. Ensure the production template uses persistent defaults:
   - PostgreSQL data in a named volume.
   - Redis append-only persistence in a named volume.
   - Qdrant storage in a named volume and configured Knowledge runtime.
   - MinIO data in a named volume and File Service `minio` storage backend.
   - QA workspace in a named volume.
4. Ensure templates do not contain real credentials, production URLs, API keys, private keys, provider secrets, or reusable weak local/demo secrets.
5. Ensure secret handling is explicit:
   - `.env` files with real values stay outside version control.
   - Production API keys and service credentials are injected by deployment environment or secret references.
   - AI Gateway provider API keys are managed through AI Gateway profile/credential APIs, not committed into env files.
6. Document service image selection and build reliability:
   - Production/staging should use prebuilt image tags or explicitly built immutable tags.
   - Docker Hub or enterprise registry reachability is required for first-time build/pull.
   - Parser image absence must have a clear build/pull recovery command.
   - No validation should rely on cached local images without saying so.
7. Document startup order and readiness checks for infrastructure, migrations, core services, gateway, frontend, and optional AI Gateway/model-dependent flows.
8. Document health-check and troubleshooting commands, including gateway and service `/readyz` checks.
9. Document persistence, backup, restore, upgrade, migration, and rollback considerations for PostgreSQL, Redis, Qdrant, MinIO, File storage, and QA workspace.
10. Keep the existing local/demo `deploy/docker-compose.yml` behavior intact unless documentation links or wording need clarification.
11. Keep `docs/architecture/technology-decisions.md`, `deploy/README.md`, `README.md`, and runbook status consistent with the new baseline.

## Acceptance Criteria

- [ ] A production/staging deployment baseline document exists and is separate from the local/demo Compose README.
- [ ] A production/staging env template exists and covers frontend, gateway, auth, file, parser, knowledge, qa, document, ai-gateway, PostgreSQL, Redis, Qdrant, and MinIO.
- [ ] A production/staging Compose baseline exists or the document clearly names the production Compose file to use; config parsing passes for the chosen files.
- [ ] The baseline does not include local demo credentials, real credentials, production URLs, API keys, private keys, or provider secrets.
- [ ] The baseline documents persistent volumes, backup/restore concerns, secret management, health/readiness checks, startup order, upgrade, and rollback.
- [ ] The baseline documents Docker Hub/registry reachability, prebuilt image strategy, and parser image recovery commands.
- [ ] Documentation states that local/demo Compose remains for local integration only and production/staging must use the new baseline.
- [ ] Validation includes `python3 scripts/check_docker_policy.py`, Compose config parsing for local and production/staging files, and `git diff --check`.

## Out Of Scope

- Deploying to a real cloud or server.
- Adding GitHub Actions deployment automation.
- Implementing #125's full cross-service/MCP smoke.
- Implementing a production secret manager integration beyond documenting required secret injection and AI Gateway secret-ref behavior.
- Changing service business logic or storage adapters unless a template cannot represent existing runtime requirements.
