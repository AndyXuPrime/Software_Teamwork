# Complete issue 108 frontend OpenAPI client

## Goal

Implement issue #108 by making the frontend API layer use the public gateway OpenAPI contract as its type source and by replacing the legacy `{ code, message, data }` transport assumptions with the documented gateway envelope and streaming/upload/download behavior.

## Requirements

- Generate frontend API types from `docs/services/gateway/api/openapi.yaml` into `apps/web/src/api/generated/gateway.ts` with `openapi-typescript`.
- Keep `apps/web/src/api/generated/` generated-only; business helpers and transport code stay outside that folder.
- Refactor `apps/web/src/api/client.ts` around the gateway `/api/v1` base URL, optional Bearer token, `X-Request-Id`, JSON requests, multipart/form-data uploads, binary downloads, and SSE streams.
- Normalize gateway success envelopes `{ data, requestId }`, paginated envelopes `{ data, page, requestId }`, and error envelopes `{ error }`.
- Implement an SSE helper based on `fetch` stream reader plus `AbortController`; do not use `EventSource` as the POST streaming path.
- Ensure frontend mock support is opt-in and constrained to active OpenAPI paths only.
- Do not generate callable frontend methods for top-level OpenAPI `x-missing-contracts` placeholder operations such as `/api/v1/admin-overview` and `/api/v1/admin-metrics`.
- Preserve current page buildability while moving existing QA/admin calls away from inactive or legacy paths where practical.

## Acceptance Criteria

- [ ] `bun run --cwd apps/web check` passes.
- [ ] `bun run --cwd apps/web build` passes.
- [ ] `git diff --check` passes.
- [ ] `apps/web/src/api/generated/gateway.ts` exists and is generated from the gateway OpenAPI source.
- [ ] `apps/web/src/api/generated/` contains generated artifacts only.
- [ ] `apps/web/src/api/client.ts` exposes typed helpers for JSON, form, binary, and SSE gateway requests.
- [ ] Missing contracts listed in OpenAPI `x-missing-contracts` are not exposed as callable frontend methods or mocks.

## Definition of Done

- Frontend type generation and API transport code are implemented.
- Existing API consumers compile against the new client contract.
- Required frontend checks have been run and reported.
- PR description must use Chinese, title must use English, target `develop`, and include `Closes #108`.

## Technical Approach

Use `openapi-typescript` only for generated schema/path types and keep a small hand-written transport wrapper in `apps/web/src/api/client.ts`. The wrapper should provide stable helpers for the rest of the app instead of exposing raw generated internals. Existing feature modules can keep their current exported function names for compatibility, but their implementation should route through the new gateway client and active paths.

## Decision (ADR-lite)

Context: the repository now treats `docs/services/gateway/api/openapi.yaml` as the public frontend contract, and old hand-written API helpers still assume `{ code, message, data }` plus inactive paths.

Decision: generate type-only OpenAPI output into `apps/web/src/api/generated/gateway.ts` and maintain a focused typed fetch wrapper for transport behavior, envelopes, request IDs, upload/download, SSE, and mocks.

Consequences: generated code remains replaceable, feature modules stay readable, and future endpoint wrappers can be added without duplicating transport rules. Some existing page-level DTOs may remain compatibility adapters until business pages are rebuilt against final backend schemas.

## Out of Scope

- Implementing new business pages.
- Implementing backend endpoints or changing gateway OpenAPI semantics.
- Adding callable methods or mocks for `x-missing-contracts` placeholder operations.
- Building a complete generated SDK surface for every OpenAPI operation in this issue.

## Technical Notes

- Issue: https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/108
- Authoritative docs read:
  - `CONTRIBUTING.md`
  - `docs/collaboration/frontend-workflow.md`
  - `.github/pull_request_template.md`
  - `.trellis/spec/frontend/index.md`
  - `.trellis/spec/frontend/directory-structure.md`
  - `.trellis/spec/frontend/quality-guidelines.md`
  - `docs/architecture/frontend-backend-contract.md`
  - `docs/services/gateway/api/openapi.yaml`
- Updated git/PR rules observed:
  - create branches from latest `upstream/develop`
  - PR base must be `develop`
  - PR title must be English and PR body must be Chinese
  - issue-closing PRs must include an auto-close keyword such as `Closes #108`
- Current branch was created from latest fetched `upstream/develop`: `EIRTeam/fix/issue-108`.
