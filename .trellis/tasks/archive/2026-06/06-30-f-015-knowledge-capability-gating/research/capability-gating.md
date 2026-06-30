# Capability Gating Research

Date: 2026-06-30

## Sources Read

- GitHub issue #281: F-015 Knowledge frontend capability gating.
- `docs/architecture/frontend-backend-contract.md`
- `docs/architecture/current-capability-matrix.md`
- `docs/services/gateway/docs/active-api-owner-map.md`
- `docs/services/knowledge/docs/implementation.md`
- `.trellis/spec/backend/error-handling.md`
- `.trellis/spec/backend/api-contracts.md`
- `.trellis/spec/frontend/index.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `.trellis/spec/frontend/type-safety.md`

## Findings

- Gateway active paths include Knowledge base CRUD, document list/upload/detail, document PATCH/DELETE, chunks, content, `knowledge-queries`, and admin parser configs.
- Knowledge implementation docs say CRUD and document list/upload/detail are implemented, while document PATCH/DELETE, chunks, content, `knowledge-queries`, and parser configs may still return `501 not_implemented`.
- Standard Gateway errors use `{ error: { code, message, requestId, fields? } }`.
- Backend error docs define `not_implemented` as `501`, `dependency_error` as `502`, and `forbidden` as `403`.
- Frontend requirements forbid direct browser calls to internal Knowledge/File/Parser/Qdrant/AI Gateway addresses.

## Implementation Implications

- Treat `501` or `not_implemented` as "capability not ready", not empty data.
- Treat `dependency_error` as "backend dependency failed", not no results.
- Treat `forbidden` separately as permission denied.
- Include `requestId` in UI details when present; explicitly say no requestId was returned when absent.
- Keep ready APIs live and classify errors from runtime responses so the UI works as backend capability moves from 501 to real implementation.
