# Implementation Plan

## Steps

1. Read backend and frontend Trellis specs before editing.
2. Add Knowledge adapter helpers for:
   - MIME field normalization
   - small common extension-to-MIME mapping
   - filename/suffix extraction
3. Change `documentFromVendor` to use normalized content type instead of `optionalStringField(raw, "type")`.
4. Preserve or infer upload file part content type before forwarding to Knowledge runtime.
5. Add backend tests:
   - raw runtime `type: "doc"` plus common filenames `.txt`, `.md`, `.docx`, `.xlsx`, `.csv`, `.pptx`, image files map to MIME values
   - document list contract does not expose `doc` as `contentType`
   - multipart upload preserves file part content type
6. Add frontend display helper and page tests covering legacy `contentType: "doc"` with distinct filenames.
7. Run focused checks, then broader frontend checks:
   - `cd services/knowledge && go test ./internal/adapter ./internal/vendorclient`
   - `bun run --cwd apps/web test:unit -- src/pages/knowledge/documents/page.test.tsx`
   - `bun run --cwd apps/web check`
   - `bun run --cwd apps/web build`
   - `git diff --check`

## Risk Points

- Multipart file part header formatting must remain accepted by Knowledge runtime.
- Some frontend tests mock documents with minimal fields; updates should keep existing tests stable.
- Go toolchain may require the local Go 1.25 path if the default environment has mixed `GOROOT`.

## Review Gate

Before starting implementation, confirm:

- `prd.md`, `design.md`, and `implement.md` exist.
- Applicable backend and frontend specs have been read.
- Task status is moved to `in_progress` with `task.py start`.
