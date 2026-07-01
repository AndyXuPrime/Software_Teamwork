# C-09 Implementation Plan

## Pre-Coding Checklist

- [x] Sync local `develop` with `upstream/develop`.
- [x] Confirm issue #105 and dependency status.
- [x] Read Document, QA, service-boundary, API contract, logging, and MCP runtime
  specs.
- [x] Confirm current implementation facts in `services/document` and
  `services/qa`.
- [x] Read `trellis-before-dev` before editing code.
- [x] Create/switch to branch `PrimeTeam/feat/document-mcp-tools`.

## Implementation Steps

- [x] Inspect existing Document service interfaces and decide exact package
  placement for the MCP tool adapter.
- [x] Add tool schema definitions and a table-driven registry for the nine C-09
  tools.
- [x] Add safe argument DTOs and validation for required fields.
- [x] Add tool execution methods that call existing Document services.
- [x] Add safe result DTOs and error mapping.
- [x] Add operation-log recording for every tool call with `requestSource=mcp`
  and `toolName`.
- [x] Add tests for schemas, validation, successful job/file/status/result
  paths, forbidden/not-ready/dependency errors, and sanitization.
- [x] Update Document implementation docs/README with current MCP tool support
  and #125 smoke-test limitation.

## Validation Commands

Run before commit:

```powershell
cd services/document
gofmt -w <changed-go-files>
go test ./...
go build ./cmd/server
```

From repository root:

```powershell
git diff --check
```

If QA code changes:

```powershell
cd services/qa
go test ./...
go build ./cmd/server
```

## Risk Points

- Tool output must not leak File Service internal IDs or `file_ref`.
- Requirements/context strings must not be persisted as operation-log summaries.
- `export_report_docx` should expose only a public content resource path, not a
  signed URL or internal object reference.
- #125 remains open, so full cross-service MCP smoke may need follow-up even
  after service-local tests pass.

## Commit / PR Notes

- Conventional Commit candidate: `feat(document): add MCP report tool surface`.
- PR target: upstream `develop`.
- PR body must list completed tools, validation commands, known #125 follow-up,
  and close/reference #105 according to team convention.
