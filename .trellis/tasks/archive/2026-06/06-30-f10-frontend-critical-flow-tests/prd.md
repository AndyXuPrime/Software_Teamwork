# F-10 Frontend Critical Flow Tests

## Goal

Implement issue #117 by adding repeatable frontend test infrastructure and a first layer of critical-flow coverage for the Vite React app. The work should make PR-before verification explicit without overstating current CI coverage.

## Requirements

- Base work on the latest `upstream/develop`; branch: `Frontend/test/frontend-critical-flows`.
- Keep frontend code and tests under `apps/web/`.
- Use Bun for dependency/script changes and command execution.
- Add Vitest-based unit tests for permission helpers, API envelope/error handling, download helper behavior, and SSE parsing/streaming behavior.
- Add React Testing Library coverage for at least the login flow and one or more critical operational UI states.
- Add Playwright smoke-test scaffolding for login, document upload, chat streaming, and report generation/download flows.
- Mock network at the API boundary and do not test generated OpenAPI internals.
- Document the new test commands and PR-before checklist in the frontend README or testing docs.
- Preserve existing `check` and `build` behavior while adding explicit test commands.

## Acceptance Criteria

- [ ] `bun run --cwd apps/web check` passes.
- [ ] `bun run --cwd apps/web build` passes.
- [ ] `git diff --check` passes.
- [ ] New frontend unit/component test command runs locally.
- [ ] New Playwright smoke command is available and either runs or documents any browser/runtime prerequisite.
- [ ] Tests use gateway-facing mocks and avoid generated OpenAPI implementation assertions.
- [ ] PR description can include concrete commands/results and `Closes #117`.

## Definition of Done

- Focused frontend/testing changes only.
- Required docs/specs read before implementation.
- Test dependencies are added through Bun and lockfile updates are committed.
- Required checks and new test commands are run and reported.
- Trellis task is archived and `.trellis/workspace/AndyXuPrime` is recorded after the work commit.

## Technical Approach

- Add Vitest, jsdom, React Testing Library, user-event, jest-dom, and Playwright as frontend dev dependencies.
- Extend `apps/web/package.json` with `test`, `test:unit`, `test:unit:run`, and `test:e2e` scripts; keep `check` as typecheck/lint/format unless a repo doc requires changing it.
- Configure Vitest in `vite.config.ts` or a colocated test config so aliases and React plugin match the app.
- Add a small `src/test/` setup area for DOM matchers and reusable rendering helpers when needed.
- Prefer colocated `*.test.ts(x)` files near helper/component owners or a consistent `src/**/*.test.ts(x)` pattern included by tsconfig.
- Add Playwright config and smoke specs under `apps/web/e2e/`; use route-level mocks instead of backend dependencies.
- Update `apps/web/README.md` with command usage and `docs/testing/strategy.md` only if current-vs-target testing status needs factual synchronization.

## Out of Scope

- Do not add a required GitHub Actions frontend CI gate unless project docs explicitly require it.
- Do not change backend contracts, generated OpenAPI files, or service docs.
- Do not make tests depend on live backend services.
- Do not broaden into full coverage for every page.

## Technical Notes

- GitHub issue: https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/117
- Authority: `docs/collaboration/frontend-workflow.md`, `docs/testing/strategy.md`, `.trellis/spec/frontend/quality-guidelines.md`, `.trellis/spec/frontend/type-safety.md`.
- `docs/testing/strategy.md` currently says Vitest/RTL/Playwright are frontend gaps, so this task should land executable commands while distinguishing local PR-before checks from required CI.
- Existing frontend scripts: `typecheck`, `lint`, `format:check`, `check`, `build`, `api:generate`.
- Existing key targets: `apps/web/src/lib/permissions.ts`, `apps/web/src/lib/download.ts`, `apps/web/src/api/client.ts`, `apps/web/src/pages/login/page.tsx`, QA/report/knowledge pages and helpers.
