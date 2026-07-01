# S-036 AI Gateway DB and cross-service smoke

## Goal

Complete issue S-036 by closing the remaining AI Gateway verification gaps around PostgreSQL persistence, env-gated provider execution, cross-service model-call samples, and operations guidance for profile seeding, service token hashes, and credential rotation.

## Background

- Authoritative docs: `docs/services/ai-gateway/docs/implementation.md`, `docs/services/ai-gateway/README.md`, `docs/services/ai-gateway/docs/provider-adapters.md`, `docs/services/ai-gateway/api/internal.openapi.yaml`, `docs/architecture/service-boundaries.md`, and `docs/architecture/technology-decisions.md`.
- AI Gateway already owns model profiles, encrypted provider credentials, OpenAI-compatible chat/embedding/rerank calls, invocation records, and usage aggregates.
- Existing provider smoke tests cover controlled fake providers and have an explicit real-provider entry point. The current missing test evidence is a real PostgreSQL DB smoke path and clearer runbook evidence for DB/cross-service execution.
- `docs/` wins over old local drafts or code behavior when a conflict is found.

## Requirements

- Add an env-gated AI Gateway PostgreSQL smoke that applies service migrations to an isolated test schema and verifies profile create/update/delete, credential encryption/rotation, provider invocation insert, and usage aggregate update.
- Keep ordinary `go test ./...` runnable without PostgreSQL or real provider credentials; missing environment must produce a clear skip, not repeated setup attempts.
- Preserve existing fake provider regression coverage for chat, streaming chat, embedding, and rerank. Real provider smoke remains explicitly gated by environment variables and must document the required env and skip conditions.
- Document auditable Knowledge/QA/Document calls to AI Gateway, including `X-Service-Token`, `X-Caller-Service`, `X-Request-Id`, profile id behavior, normalized errors, and sensitive-field redaction.
- Supplement profile seed, service token hash, credential rotation, DB smoke, and real provider smoke run instructions in AI Gateway-owned docs/runbooks only.
- Do not claim RAG, Report, Document generation, quota, rate limiting, Prometheus metrics, or OpenTelemetry tracing are complete.
- Do not expose API keys, service tokens, full prompts, document bodies, object keys, provider raw errors, or embedding payloads in logs, responses, docs examples, or test output.

## Acceptance Criteria

- [ ] `cd services/ai-gateway && go test ./...` passes with default local env; DB and real provider smoke tests skip clearly when their env gates are absent.
- [ ] With `AI_GATEWAY_TEST_DATABASE_URL` set, the DB smoke applies migrations and verifies profile CRUD, credential encryption/rotation, provider invocation records, and usage aggregate writes.
- [ ] Real provider smoke documentation covers chat, streaming chat, embedding, and rerank env requirements or explicitly records the skip conditions and fake-provider replacement coverage.
- [ ] Cross-service examples cover Knowledge embedding/rerank, QA chat, and Document chat/profile validation with caller service, request id, profile id, service token, normalized errors, and redaction expectations.
- [ ] Profile seed, token hash, and credential rotation guidance is reusable by deploy/local integration/security tasks.

## Notes

- PR target branch: upstream `develop`.
- Working branch: `Special/test/ai-gateway-db-cross-service-smoke`.
