# GitHub service docs auto labels

## Goal

When a PR changes documentation for a service-owned area, Auto Label should add
the corresponding `service:<name>` label that would be expected for the
implementation area. This makes documentation-only PRs visible by service area
instead of only receiving the generic `documentation` label.

## What I already know

- The repository already has an `Auto Label` workflow in
  `.github/workflows/auto-label.yml`.
- The workflow reads `.github/labeler.json` from the base branch and applies
  labels from two config sections:
  - `accountLabels`: GitHub account login or numeric ID to Team label.
  - `pathLabels`: changed file path glob to labels.
- Existing path rules already add:
  - `documentation` for `docs/**`, top-level markdown, PR template, Trellis
    specs, and Trellis tasks.
  - `service:<name>` labels for implementation paths under `services/<name>/**`.
- Existing path rules do not add service labels for service documentation paths
  under `docs/services/<service>/**`.
- The current workflow stores labels in a `Set`, so duplicate labels from
  overlapping path rules are safe.
- The workflow skips labels that do not exist in the GitHub repository and logs
  a warning.

## Assumptions

- This task should preserve the existing workflow shape and only extend config
  unless a code change is needed.
- Documentation changes should continue to receive the generic `documentation`
  label.
- Service documentation changes should also receive the relevant
  `service:<name>` label when a service label already exists.
- Service documentation paths mirror implementation service labels:
  - `docs/services/gateway/**` -> `service:gateway`
  - `docs/services/auth/**` -> `service:auth`
  - `docs/services/file/**` -> `service:file`
  - `docs/services/qa/**` -> `service:qa`
  - `docs/services/knowledge/**` -> `service:knowledge`
  - `docs/services/document/**` -> `service:document`
  - `docs/services/ai-gateway/**` -> `service:ai-gateway`

## Open Questions

- None.

## Requirements

- Add path-based service labeling for service documentation paths whose
  corresponding implementation service label already exists.
- Preserve all existing account label mappings.
- Preserve the existing generic documentation label behavior.
- Preserve label-existence safety: Auto Label should still skip missing labels
  rather than failing PRs.
- Keep maintainer-facing documentation consistent with the updated label rules.

## Acceptance Criteria

- [x] `.github/labeler.json` is valid JSON and has a trailing newline.
- [x] A PR touching `docs/services/gateway/**` matches `documentation` and
      `service:gateway`.
- [x] A PR touching `docs/services/auth/**` matches `documentation` and
      `service:auth`.
- [x] A PR touching `docs/services/file/**` matches `documentation` and
      `service:file`.
- [x] A PR touching `docs/services/qa/**` matches `documentation` and
      `service:qa`.
- [x] A PR touching `docs/services/knowledge/**` matches `documentation` and
      `service:knowledge`.
- [x] A PR touching `docs/services/document/**` matches `documentation` and
      `service:document`.
- [x] A PR touching `docs/services/ai-gateway/**` matches `documentation` and
      `service:ai-gateway`.
- [x] Existing code path labels under `services/**`, `apps/web/**`, `.github/**`,
      docs, and Trellis/agent paths keep working.
- [x] Repository collaboration/maintenance docs describe service documentation
      service labels.

## Definition of Done

- Validate JSON syntax for `.github/labeler.json`.
- Run a lightweight local matcher check for representative service
  documentation paths.
- Inspect `git diff` to confirm only intended files changed.

## Out of Scope

- Creating or renaming GitHub labels remotely unless required by the final rule
  set.
- Changing PR Guard or Commitlint behavior.
- Adding product CI workflows.
- Enforcing label rules as merge gates.

## Technical Notes

- Relevant files inspected:
  - `.github/labeler.json`
  - `.github/workflows/auto-label.yml`
  - `CONTRIBUTING.md`
  - `docs/collaboration/repository-settings.md`
  - `docs/collaboration/documentation-workflow.md`
  - `docs/architecture/service-boundaries.md`
- The simplest implementation is adding more `pathLabels` entries, because the
  workflow already supports path-to-label mappings and de-duplicates matched
  labels.
- `docs/services/ai-gateway/**` exists as service documentation, and CI/CD specs
  already list `services/ai-gateway/` as a target service path. The
  implementation introduces `service:ai-gateway` alongside the other service
  labels so documentation and future implementation paths use the same label.
