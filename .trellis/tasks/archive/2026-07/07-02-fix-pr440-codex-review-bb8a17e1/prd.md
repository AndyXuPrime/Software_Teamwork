# Fix PR #440 codex review round bb8a17e1 findings

## Goal

Resolve all 4 findings from the latest Codex PR review (comment id 4866941607, head `bb8a17e154d0`) on upstream PR
<https://github.com/Sakayori-Iroha-168/Software_Teamwork/pull/440>, so that a clean environment can run the
Knowledge adapter + RAGFlow runtime end to end, the runtime is not exposed with forgeable-header auth, and CI
meaningfully covers the vendored runtime.

## Background

- Branch `L1nggTeam/feat/ragflow-runtime-vendor` vendors RAGFlow as `services/knowledge-runtime` behind the Go
  adapter `services/knowledge` (contract boundary for the Gateway).
- Latest review round matches local HEAD `bb8a17e1` exactly; all findings apply to current code.
- Adapter-side `X-Service-Token` enforcement already landed in `bb8a17e1`; the runtime side has no equivalent.

## Requirements

### F1 (blocker) — Clean runtime rejects all Gateway users with 401

- `api/apps/__init__.py` resolves `X-Tenant-Id`/`X-User-Id` against the runtime's own `user` table;
  `api/db/init_data.py` never creates `user`/`tenant`/`user_tenant` rows for Gateway users, and the adapter only
  forwards the Gateway user ID (`vendorclient` sets both headers to the Gateway user ID).
- Requirement: a Gateway-authenticated request arriving via the adapter (trusted per F3 token) MUST be able to use
  the runtime on a clean database without manual provisioning. Runtime must provision (or otherwise resolve)
  tenant/user/user_tenant rows automatically and idempotently.
- Constraint: runtime schema uses `varchar(32)` for `user.id` / `tenant.id` / `*.tenant_id` / `*.created_by`,
  while real auth user IDs are `usr_` + 32 hex = 36 chars. Provisioning must normalize IDs deterministically so
  36-char Gateway IDs work; IDs already ≤ 32 chars (e.g. seed `usr_local_admin`) must keep their raw value for
  backward compatibility with data created before this change.
- Newly provisioned tenants must receive the same default/env-configured models
  (`KNOWLEDGE_RUNTIME_EMBEDDING_*` / `KNOWLEDGE_RUNTIME_RERANK_*`, `user_default_llm` settings) that startup
  initialization applies, otherwise ingestion/retrieval fails for them.

### F2 (blocker) — MinIO bucket mismatch on the standard startup path

- Runtime `conf/service_conf.yaml` uses bucket `software-teamwork-knowledge`; `deploy/docker-compose.yml`
  `minio-init` only creates `software-teamwork-local`; `rag/utils/minio_conn.py` intentionally does NOT
  auto-create the single-bucket-mode bucket (health only checks existence, `put` requires it to exist).
- Requirement: after `dev-up.sh` (compose minio-init), the runtime bucket exists. Keep the runtime bucket
  separate from `software-teamwork-local` (do not merge them); `minio-init` must create both, idempotently.
- Docs that enumerate buckets / troubleshoot `MinIO bucket missing` must stay accurate.

### F3 (high) — Runtime binds 0.0.0.0 and trusts forgeable tenant headers

- `conf/service_conf.yaml` sets `ragflow.host: 0.0.0.0`; `login_required` trusts `X-Tenant-Id`/`X-User-Id` with
  no service-token or origin check, so anyone reaching `:9380` can act as any tenant, bypassing the adapter.
- Requirements:
  - Default bind changes to `127.0.0.1` (settings fallback is already `127.0.0.1` when key missing).
  - Runtime enforces an internal service token on all tenant-scoped (`login_required`) routes:
    header `X-Service-Token` must match env `KNOWLEDGE_RUNTIME_SERVICE_TOKEN` (constant-time compare).
    Fail closed: when env is unset or header mismatches → 401, never trust tenant headers.
  - Tenant auto-provisioning (F1) MUST only run for token-validated requests.
  - `/api/v1/system/ping`, `/system/version`, `/system/healthz` remain tokenless (loopback-bound by default;
    adapter readiness ping keeps working).
  - Adapter vendor client forwards the token on runtime requests. Token is configured via a dedicated env
    (`VENDOR_RUNTIME_SERVICE_TOKEN`) and is required for the adapter to start, mirroring the existing
    `KNOWLEDGE_SERVICE_TOKEN` fail-closed pattern.
  - `deploy/.env.example` and README document both envs with local dev defaults; secrets must not be logged.

### F4 — CI coverage for the vendored runtime

- `.github/workflows/knowledge-runtime.yml` currently runs only `test_config_utils.py` + `test_route_registry.py`.
- Requirements:
  - Add a syntax-level gate over the whole vendored runtime (`python -m compileall` on `api`, `common`, `rag`,
    `deploy`) that needs no heavy deps.
  - New unit tests for F1/F3 logic (token validation, tenant-ID normalization, provisioning idempotency with
    mocked DB services) run in the same targeted pytest job.
  - Best effort (decision gate in implement.md): a clean-Postgres init job that runs `init_database_tables()` +
    provisioning smoke against a real Postgres service. If the import chain requires the heavy ML dependency
    tree, drop the job and record the residual risk in the PR instead of shipping a flaky/slow gate.
  - Full adapter-to-runtime smoke (ES/MinIO/Redis) stays out of scope; note as residual risk.

## Out of scope

- `DOC_ENGINE=elasticsearch` default vs compose-provided ES (raised in an earlier round, not in round
  `bb8a17e1`; runtime/ES stay host-run per current Docker policy).
- Re-adding runtime containers/profiles to root compose (policy forbids).
- Gateway/auth-service changes.

## Acceptance Criteria

- [ ] F1: with a clean `knowledge_system` DB, a token-authenticated request with a fresh 36-char Gateway user ID
      provisions user/tenant/user_tenant exactly once (idempotent under repeat/concurrency) and the request
      proceeds; IDs ≤ 32 chars resolve to their raw value.
- [ ] F2: `deploy/docker-compose.yml` minio-init creates `software-teamwork-knowledge` and
      `software-teamwork-local`; `docker compose --env-file .env.example config --quiet` passes;
      `python3 scripts/check_docker_policy.py` passes; policy unittests pass.
- [ ] F3: runtime default bind is `127.0.0.1`; tenant-scoped routes return 401 without a valid
      `X-Service-Token` (including when env unset); ping/version/healthz stay tokenless; adapter sends the token
      and fails fast at startup when `VENDOR_RUNTIME_SERVICE_TOKEN` is missing; `.env.example`/README updated.
- [ ] F4: workflow gains compileall step + new unit tests; clean-DB init job included or its exclusion justified
      in the PR body.
- [ ] `cd services/knowledge && go test ./... && go build ./cmd/adapter` passes.
- [ ] Runtime targeted pytest (existing + new files) passes locally with the same `uv run --no-project` recipe
      as CI.
- [ ] No secrets/tokens logged or committed beyond documented local-dev placeholders.
