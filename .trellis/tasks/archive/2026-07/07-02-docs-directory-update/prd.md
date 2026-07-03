# Update architecture current-state documentation

## Goal

Audit and update the repository architecture documentation under
`docs/architecture/` so current capability summaries, system link coverage, and
gap references are easier to follow from the current `develop` baseline.

## Confirmed Facts

- User asked to start from `docs/` and create a Trellis task.
- User clarified that the target content should be in `docs/architecture/`.
- User selected the "current state / gap" document class, not contract semantic
  changes.
- Current implementation work is isolated in branch
  `SakayoriTeam/docs/update-docs`.
- On 2026-07-02, the working branch was rebased onto latest
  `fork/develop` / `origin/develop` at `0615ccb8`; `aktnl/develop` was older
  and was not used as the current baseline.
- `docs/architecture/` currently contains:
  `current-capability-matrix.md`, `frontend-backend-contract.md`,
  `service-boundaries.md`, `system-link-condition-coverage.md`, and
  `technology-decisions.md`.
- `current-capability-matrix.md` and
  `system-link-condition-coverage.md` contain many current-state and gap claims.
- `frontend-backend-contract.md`, `service-boundaries.md`, and
  `technology-decisions.md` are contract/decision documents where semantic
  changes need tighter review.
- The first implementation scope is:
  `docs/architecture/current-capability-matrix.md` and
  `docs/architecture/system-link-condition-coverage.md`.
- `docs/README.md` is the documented entry point for project requirements,
  architecture contracts, runbooks, testing, and collaboration docs; it may need
  a small navigation update only if architecture doc changes require it.
- `docs/collaboration/documentation-workflow.md` defines where documentation
  content belongs and says service README files must not duplicate repository
  workflow, full technology baselines, or implementation gap lists.
- `.trellis/spec/guides/cross-layer-thinking-guide.md` states that `docs/` is
  the repository contract source of truth and points to
  `docs/collaboration/documentation-workflow.md` for hierarchy.
- This task is scoped as a lightweight documentation task; PRD-only is
  sufficient unless the scope expands into contract semantics or multiple
  independently verifiable deliverables.

## Requirements

- Start with `docs/architecture/current-capability-matrix.md` and
  `docs/architecture/system-link-condition-coverage.md`, not service
  implementation code.
- Preserve the existing documentation hierarchy: architecture docs own
  cross-service contracts, current capability summaries, system link coverage,
  and technology decisions; service-specific implementation state remains in
  `docs/services/<service>/docs/implementation.md`.
- Keep `docs/README.md`, collaboration docs, and other architecture docs
  untouched unless a narrow link/navigation consistency fix is needed.
- Do not turn open PRs, unmerged issues, draft plans, or desired future behavior
  into current `develop` facts.
- Do not make semantic contract changes to Gateway OpenAPI, service boundaries,
  data models, or confirmed requirement semantics without an explicit decision.
- Do not modify `docs/architecture/frontend-backend-contract.md`,
  `docs/architecture/service-boundaries.md`, or
  `docs/architecture/technology-decisions.md` for this pass unless only fixing a
  broken link or obvious reference drift discovered while validating the two
  target docs.
- Keep updates reviewable and focused. If the architecture audit finds unrelated
  or large service-specific cleanup, record it as follow-up instead of folding
  everything into one broad edit.
- Use existing docs as the authority before asking scope questions that can be
  answered from the repository.

## Acceptance Criteria

- [ ] Updates are limited to architecture current-state/gap docs:
      `current-capability-matrix.md` and
      `system-link-condition-coverage.md`, plus only incidental navigation/link
      fixes if required.
- [ ] Updated docs remain consistent with
      `docs/collaboration/documentation-workflow.md`.
- [ ] Touched architecture docs link to the correct authoritative sources and do
      not duplicate state that belongs in service implementation docs.
- [ ] Any current-state claims added or changed are backed by the current
      `develop` baseline or clearly labeled as pending/follow-up.
- [ ] The two target docs do not claim open PRs, unmerged issues, future targets,
      or env-gated smoke samples as fully implemented `develop` capabilities.
- [ ] Documentation-only validation is run for the touched files, including at
      least link/path checks or targeted repository searches appropriate to the
      edits.
- [ ] No code, Docker, Compose, OpenAPI contract, or frontend behavior changes
      are made unless the task scope is revised first.

## Out of Scope

- Implementing backend, frontend, Docker, or CI behavior.
- Changing public API semantics, authentication rules, service ownership, or
  data model contracts without separate approval.
- Updating every architecture and service document in one pass.
- Rewriting service implementation status docs except to cross-check facts.

## Open Questions

- None. Planning is ready for implementation.
