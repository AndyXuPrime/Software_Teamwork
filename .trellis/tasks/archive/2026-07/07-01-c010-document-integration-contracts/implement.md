# Implementation Plan

1. Read required backend specs and relevant service docs.
2. Inspect existing Document tests for fileclient, report files, templates,
   materials, generation, operation logs, and route coverage.
3. Add targeted contract tests:
   - File Service client path/header/error behavior.
   - Report file content binary success and JSON error envelope.
   - Public template/material/report-file response safety.
   - Fake dependency failure mapping for file/content or generation paths.
4. Update Document README or implementation docs with C-010 local integration
   guidance.
5. Run quality checks:
   - `cd services/document && go test ./...`
   - `cd services/document && go build ./cmd/server`
   - `git diff --check`
6. Fix any failures and re-run impacted checks.

## Risky Files

- `services/document/internal/http/*_test.go`
- `services/document/internal/platform/fileclient/client_test.go`
- `services/document/internal/service/*_test.go`
- `docs/services/document/README.md`

## Notes

Do not commit generated files unless a contract/code change genuinely requires
regeneration. Do not start external services as part of the normal test path.
