# Implementation Plan: Fix PR 428 review findings

1. Update Gateway's QA attachment envelope constant and add exact boundary tests.
2. Add attachment status filtering across handler, service options/validation, repository count/list queries, and tests.
3. Introduce a public attachment response mapper and contract-shape tests.
4. Make attachment metadata creation transactional with a conversation lock and atomic quota checks; add File cleanup compensation and tests.
5. Extend attachment lifecycle/migrations with `purged`, add migration 0011 Down, and add a reversible system-default tool migration.
6. Update the in-code default tool list and its unit tests.
7. Format and run focused QA/Gateway tests, then full service tests and required policy/migration/Compose checks.
8. Review the final diff against all seven active findings and the public OpenAPI contract before handoff.

## Risk and Rollback Points

- Quota transaction SQL is the highest-risk change; verify lock ownership, `RowsAffected`, and error mapping before broader tests.
- Public DTO mapping must cover upload, get, and paginated list responses consistently.
- Migration Down order must remain `message_attachments -> session_attachment_chunks -> session_attachments`.

## Validation Commands

- `cd services/gateway && go test ./internal/http ./internal/middleware`
- `cd services/qa && go test ./internal/http ./internal/service ./internal/repository`
- `cd services/gateway && go test ./...`
- `cd services/qa && go test ./...`
- `python scripts/check_docker_policy.py` (record pre-existing out-of-scope findings separately)
- `python scripts/check_docker_environment.py --profile all --clean-env`
- `docker compose -f deploy/docker-compose.yml config`
