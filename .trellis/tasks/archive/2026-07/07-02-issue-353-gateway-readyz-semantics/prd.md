# Gateway readyz semantics

## Goal

Resolve issue #353 by making the Gateway `/readyz` contract explicit and
consistent across public Gateway docs, implementation status docs, and local
integration/deploy readiness guidance.

The selected decision is option C from the issue: keep `/readyz` as a lightweight
Gateway readiness signal and use separate smoke/diagnostics for full
cross-service business availability.

## Confirmed Facts

- Issue #353 requires choosing one of:
  - A. `/readyz` only represents Gateway itself plus Auth/Redis.
  - B. `/readyz` probes all configured owner services.
  - C. `/readyz` is separated from heavier dependency diagnostics or smoke.
- Current `develop` implementation in `services/gateway/cmd/server/main.go`
  checks Redis readiness, Auth `/readyz`, and that Knowledge/QA/Document/AI
  Gateway owner base URLs are configured.
- Current implementation does not call Knowledge, QA, Document, or AI Gateway
  `/readyz` from Gateway `/readyz`.
- `docs/runbooks/local-integration.md` already documents env-gated cross-service
  smoke paths and tracks the remaining full E2E gap under #125.
- Issue #352 owns the narrower Auth/Gateway/Redis smoke scripting work.
- Full owner-service business availability can depend on seed data, File,
  Parser, Qdrant, model profiles, provider credentials, and AI Gateway profile
  state; those checks are too heavy and too environment-specific for a top-level
  Gateway readiness gate.

## Requirements

- Document Gateway `/healthz` and `/readyz` semantics consistently:
  `/healthz` is process liveness; `/readyz` proves Gateway can accept normal
  public routing traffic at a lightweight infrastructure/configuration level.
- State that `/readyz` includes Redis, Auth readiness, and configured owner base
  URLs, but does not prove every configured owner service can complete every
  business workflow.
- State that complete cross-service availability is verified by targeted smoke
  checks and diagnostics, especially #125 and #352.
- Preserve current implementation behavior unless stronger evidence proves code
  must change.
- Do not change Auth, Knowledge, Document, QA, AI Gateway, production Compose,
  or owner service business API semantics.

## Acceptance Criteria

- [x] Gateway README explicitly records decision C and the lightweight
      `/readyz` scope.
- [x] Gateway implementation document matches current code and no longer
      implies a future real owner-service probe is undecided.
- [x] Local integration/deploy readiness docs distinguish Gateway `/readyz`
      from full cross-service smoke/diagnostics.
- [x] If no code changes are made, PR notes explain why current behavior matches
      the chosen semantics.
- [x] `git diff --check` passes.
- [x] Documentation-only verification is recorded for the PR, including Gateway
      OpenAPI YAML parsing or an equivalent contract parse check.

## Out Of Scope

- Implementing real owner-service probes inside Gateway `/readyz`.
- Adding a new diagnostics endpoint.
- Implementing #125 or #352 smoke automation in this PR.
- Changing production Compose health check behavior.
