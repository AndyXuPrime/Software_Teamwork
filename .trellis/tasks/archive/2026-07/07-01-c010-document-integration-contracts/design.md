# Design

## Approach

Keep the C-010 implementation as a test and documentation slice. The safest
place to enforce the boundary is close to the existing Document handlers,
service-layer fakes, and platform clients:

- `internal/platform/fileclient`: verifies outbound File Service paths, headers,
  binary reads, and error mapping.
- `internal/http`: verifies public HTTP response envelopes, content endpoint
  behavior, request ID propagation, and safe public JSON.
- `internal/service`: verifies generation/report-file orchestration maps fake
  dependency failures to stable service errors and does not expose internals.
- `docs/services/document/README.md` or implementation docs: records local
  validation commands and smoke prerequisites.

## Boundaries

- Document owns report templates, materials, reports, outlines, sections, report
  jobs, report files, statistics, settings, and operation logs.
- File Service owns raw object bytes and storage internals. Document may persist
  internal file references but must not expose them publicly.
- Knowledge owns retrieval and Qdrant interactions. Document only calls its
  internal API through a client.
- AI Gateway owns provider profiles, credentials, provider base URLs, and
  OpenAI-compatible model calls. Document references profiles and calls the
  internal model API through clients.

## Test Strategy

1. Prefer existing fake repositories and fake service clients to avoid fragile
   external dependencies.
2. Use handler-level tests for public contract behavior because they catch JSON
   envelope and binary response regressions.
3. Use platform-client tests for cross-service path and header contracts.
4. Add a small reusable assertion helper for forbidden public substrings only if
   local test style supports it; otherwise keep assertions in targeted tests.

## Compatibility

No database migration, API contract, or runtime behavior change should be
needed. If a test reveals an existing contract leak, fix the smallest mapping or
response serialization issue without changing public OpenAPI semantics unless
the contract is demonstrably wrong.

## Validation

Run from repository root unless noted:

- `cd services/document && go test ./...`
- `cd services/document && go build ./cmd/server`
- `git diff --check`
