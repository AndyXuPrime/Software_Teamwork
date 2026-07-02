# S-031 Production Compose Baseline Design

## File Boundaries

- Keep `deploy/docker-compose.yml` and `deploy/.env.example` as the local/demo integration baseline.
- Add production/staging-specific files under `deploy/`:
  - `deploy/docker-compose.production.yml` for the single-machine production/staging Compose shape.
  - `deploy/.env.production.example` for required variable names and safe placeholders.
  - `deploy/production-baseline.md` for operator-facing deployment, persistence, secret, upgrade, and rollback guidance.
- Update existing entry-point docs to link the new baseline:
  - `deploy/README.md`
  - `README.md`
  - `docs/runbooks/local-integration.md`
  - `docs/architecture/technology-decisions.md` or current capability matrix if status language needs synchronization.

## Compose Strategy

The production/staging Compose file should be runnable for config validation without requiring real secrets. Runtime startup with placeholders is expected to fail before serving production traffic; the document must say operators must replace placeholders before use.

Use service images instead of source builds as the primary baseline:

- `FRONTEND_IMAGE`
- `GATEWAY_IMAGE`
- `AUTH_IMAGE`
- `FILE_IMAGE`
- `PARSER_IMAGE`
- `KNOWLEDGE_IMAGE`
- `QA_IMAGE`
- `DOCUMENT_IMAGE`
- `AI_GATEWAY_IMAGE`

Each image variable gets a non-`latest` placeholder tag such as `registry.example.com/software-teamwork/<service>:REPLACE_WITH_TAG`. This keeps production deployment separated from local source builds and supports rollback by pinning previous tags.

Infrastructure images keep pinned defaults aligned with the technology baseline:

- PostgreSQL: `postgres:16-alpine`
- Redis: `redis:7-alpine`
- Qdrant: `qdrant/qdrant:v1.18.2`
- MinIO server: `minio/minio:RELEASE.2025-09-07T16-13-09Z`
- MinIO mc: `minio/mc:RELEASE.2025-08-13T08-35-41Z`

## Runtime Defaults

Production/staging must prefer persistent service configuration:

- File Service uses PostgreSQL metadata and `FILE_STORAGE_BACKEND=minio`.
- Knowledge uses PostgreSQL, Redis/asynq, Parser, File, Qdrant, and explicit service token wiring.
- QA uses PostgreSQL, Knowledge, File, Parser, AI Gateway, and a persistent `qa_workspace` volume.
- Document uses PostgreSQL, Redis/asynq, File Service, AI Gateway, and optional Knowledge URL.
- AI Gateway uses PostgreSQL, encrypted-column credential storage, service token hashes, and externally injected credential encryption material.
- Parser uses a service token and resource-bounded defaults. Model files are runtime/environment concerns; the baseline documents how to build/pull the parser image and prepare OCR model configuration when needed.

## Migrations And Seed

Production/staging should run service migrations before starting services. Local demo seed scripts are not part of the production baseline.

Use short-lived migration services built from the same application image family where possible, or keep the existing `deploy/Dockerfile.migrate` shape for config validation if service images do not yet embed migrations. The production baseline must not run `seed-local` or `seed-local-ai`.

## Secrets

`deploy/.env.production.example` uses placeholder values only. Any value that would be sensitive in production should be a replacement token such as `REPLACE_WITH_STRONG_SECRET` or `REPLACE_WITH_SECRET_REF`.

Secrets that must never be real in the repository include database passwords, service tokens, token hash secrets, MinIO root credentials, AI Gateway encryption key material, provider API keys, SSH keys, and cloud credentials.

## Validation

Static validation should prove:

- Local Compose still parses.
- Production Compose parses with `deploy/.env.production.example`.
- Docker policy checker still passes.
- Production template has no obvious local demo values or `latest` image defaults.
- Markdown and YAML have no conflict markers or trailing whitespace.
