# Fix PR 350 review concurrency guards

## Goal

Close the latest PR #350 review findings by making section-version and manual
section writes concurrency-safe against report soft deletion and section
generation state races.

## Requirements

- `CreateSectionVersion` must reject a report that becomes soft-deleted during
  the write transaction. The transactional re-check must lock the report row so
  a concurrent soft delete cannot commit between the check and the version
  insert/current-section update.
- `UpdateSection` must not overwrite a section that becomes
  `generation_status = running` after the entry check. It must lock the current
  section inside the write transaction, re-check same-report ownership and
  running generation state, and only then create a manual version/current
  section update.
- `SaveSections` must apply the same transactional lock/re-check for every
  current section it mutates so a concurrent generation cannot be overwritten
  by a stale bulk-save snapshot.
- `UpdateSection` and `SaveSections` must also lock and re-check the report row
  inside their write transactions so a report soft-deleted after the entry
  check cannot receive manual section writes or manual version snapshots.
- Generated section success writes that intentionally return `conflict` because
  the current section changed during AI execution must not mark the current job
  as `failed`; the stale AI response should be rejected without mutating the
  current section status.
- Follow the existing Document Service repository/service boundaries and keep
  PostgreSQL transactions short; no external calls may be introduced inside a
  database transaction.

## Acceptance Criteria

- [x] Regression tests cover report deletion after the entry check but before
  `CreateSectionVersion` writes, and assert no version/current-section side
  effect occurs.
- [x] Regression tests cover `UpdateSection` and `SaveSections` when a section
  becomes running after the entry check, and assert the manual write returns a
  conflict without creating a manual version or clearing generation state.
- [x] Regression tests cover `UpdateSection` and `SaveSections` when a report
  becomes soft-deleted after the entry check, and assert the manual write
  returns a conflict without mutating the section or creating a manual version.
- [x] Regression tests cover generated section success-path conflict handling
  and assert a stale AI response does not mark the section/job status failed.
- [x] `cd services/document && go test ./internal/service -run ... -count=1`
  demonstrates RED before implementation and GREEN after implementation for the
  new cases.
- [x] `cd services/document && go test ./... -count=1`, `go build ./cmd/server`,
  `go vet ./...`, `govulncheck ./...`, and `git diff --check` pass before push.

## Constraints

- Do not expose local machine paths in PR descriptions or public comments.
- Keep PR body verification commands generic (`go ...`) even when local commands
  use the local Go executable.
