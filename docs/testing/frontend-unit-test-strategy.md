# Frontend and Unit Test Strategy

Date: 2026-07-02

This document defines the frontend and unit-test baseline for test tasks that focus on static checks, frontend build health, and mock-backed unit or component coverage. It does not replace integration, E2E, permission, security, file-boundary, migration, or manual acceptance test reports.

## Scope

This baseline applies to changes under `apps/web/**` and to test tasks that need to verify the frontend automation surface.

In scope:

- TypeScript checks for app code and test code.
- ESLint and Prettier checks.
- Production build verification.
- Vitest unit and component tests.
- Frontend test coverage inventory.
- Lightweight execution evidence for pure static, unit, or component test tasks.

Out of scope:

- Real backend service integration.
- Full E2E or cross-service smoke.
- Real AI provider quality evaluation.
- Permission, security, file-boundary, migration, or deployment acceptance.
- Business feature fixes unrelated to the current test task.

## Command Matrix

Run default commands from the repository root. Windows local verification may use the fallback command from `apps/web` when Bun cannot write its temp directory or native dependency cache. The evidence must state which runner was used.

| Check           | Default Command                         | Windows Fallback             | Tool                           | What It Proves                                                                                   | Evidence Required                                                         |
| --------------- | --------------------------------------- | ---------------------------- | ------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------- |
| App type check  | `bun run --cwd apps/web typecheck`      | `npm.cmd run typecheck`      | TypeScript                     | Frontend source files have no TypeScript errors.                                                 | Pass/fail and error summary.                                              |
| Test type check | `bun run --cwd apps/web typecheck:test` | `npm.cmd run typecheck:test` | TypeScript                     | Test files and test config type-check.                                                           | Pass/fail and error summary.                                              |
| Lint            | `bun run --cwd apps/web lint`           | `npm.cmd run lint`           | ESLint                         | Code follows configured lint rules.                                                              | Pass/fail and warning summary.                                            |
| Format check    | `bun run --cwd apps/web format:check`   | `npm.cmd run format:check`   | Prettier                       | Files match repository formatting rules.                                                         | Pass/fail and list of unformatted files.                                  |
| Build           | `bun run --cwd apps/web build`          | `npm.cmd run build`          | TypeScript + Vite              | Frontend can be bundled for production.                                                          | Pass/fail and build warnings/errors.                                      |
| Unit tests      | `bun run --cwd apps/web test:unit`      | `npm.cmd run test:unit`      | Vitest + React Testing Library | API clients, hooks, UI components, and pages behave as expected in mocked/local test conditions. | Test count, pass/fail count, failed test names.                           |
| Dev smoke       | `bun run --cwd apps/web dev`            | `npm.cmd run dev`            | Vite + browser                 | App can start locally and render basic pages.                                                    | URL, screenshots, and whether backend-dependent calls failed as expected. |
| Frontend check  | `bun run --cwd apps/web check`          | run fallback checks above    | Bun + project scripts          | Runs typecheck, test typecheck, lint, and format check as the repository baseline.               | Pass/fail and failed subcommand.                                          |

## Current Unit Test Coverage

Current frontend unit and component tests cover these areas:

| Area                            | Test Files                                                                                                                                                               |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| API client and Gateway wrappers | `src/api/chat.test.ts`, `src/api/client.test.ts`, `src/api/knowledge.test.ts`                                                                                            |
| Chat UI                         | `src/components/chat/chat-input.test.tsx`                                                                                                                                |
| Shared UI                       | `src/components/ui/button.test.tsx`                                                                                                                                      |
| Knowledge features              | `src/features/knowledge/capability.test.ts`, `src/features/knowledge/hooks/use-documents.test.tsx`                                                                       |
| QA capability                   | `src/features/qa/capability.test.ts`                                                                                                                                     |
| Reports features                | `src/features/reports/report-generation.api.test.ts`, `src/features/reports/report-generation.errors.test.ts`, `src/features/reports/report-generation.queries.test.tsx` |
| Shared utilities                | `src/lib/download.test.ts`, `src/lib/permissions.test.ts`                                                                                                                |
| Knowledge pages                 | `src/pages/knowledge/documents/page.test.tsx`, `src/pages/knowledge/search/page.test.tsx`                                                                                |
| Login page                      | `src/pages/login/page.test.tsx`                                                                                                                                          |
| Report pages                    | `src/pages/reports/generate/page.test.tsx`, `src/pages/reports/records/page.test.tsx`, `src/pages/reports/templates/page.test.tsx`                                       |

## CI and Manual Boundary

Can run as default automation:

- TypeScript checks.
- ESLint.
- Prettier check.
- Build.
- Vitest unit and component tests.
- Mock-backed Playwright smoke when stable in CI.

Needs separate report or explicit environment evidence:

- Real Gateway/Auth/session login.
- Real Knowledge, QA, File, Parser, Document, or AI Gateway integration.
- Full E2E flows.
- Permission or security regression.
- File upload/download boundary checks.
- Migration or deploy validation.
- Manual visual acceptance.
- Defect reproduction.

## Known Risk Handling

If `format:check` fails on files outside the current task, record the failing file list and do not reformat unrelated files unless the task explicitly includes formatting cleanup.

If Vite, Vitest, Tailwind, or Rolldown native dependencies fail on Windows with `spawn EPERM` or an unloadable native `.node` package, record it as a local environment blocker and rerun in the user's normal PowerShell or CI before treating it as a product failure.

If a unit test times out, record the exact test file and test name. If it is outside the current change scope, classify it as a known unstable or pre-existing failure and create or link an owner issue when required by the test task rules.

## PR Test Comment Template

```text
Frontend/unit test evidence:

Base: develop @ <commit>
Branch: <branch>
Environment: Windows PowerShell, Node <version>, npm <version>

Passed:
- bun run --cwd apps/web check
- bun run --cwd apps/web build
- bun run --cwd apps/web test:unit

Failed / blocked:
- bun run --cwd apps/web <script>: <summary>
- Windows fallback, if used: npm.cmd run <script>: <reason and result>

Not run:
- <item>: <reason and residual risk>

Known issues:
- <test/file/issue>: <summary>

Conclusion:
- 测试通过 / 测试失败且已修复 / 测试失败已转 issue / 因环境缺失未运行
```
