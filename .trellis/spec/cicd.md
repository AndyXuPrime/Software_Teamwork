# CI/CD Guidelines

> GitHub Actions and Docker Compose delivery rules for this monorepo.

---

## Overview

This repository uses GitHub Actions for pull request checks and deployment
automation. Existing collaboration workflows protect contribution rules. Product
CI/CD should be added around the confirmed monorepo layout:

```text
apps/web/
services/gateway/
services/auth/
services/file/
services/qa/
services/knowledge/
services/document/
services/ai-gateway/
deploy/docker-compose.yml
```

Deployment target: single-machine Docker Compose.

---

## Existing Guard Workflows

These workflows already exist and must remain separate from product build jobs:

| Workflow | File | Purpose |
|----------|------|---------|
| Auto Label | `.github/workflows/auto-label.yml` | Applies team/path labels and syncs PR `blocked` label from linked issues |
| PR Guard | `.github/workflows/pr-guard.yml` | Enforces fork + PR collaboration rules and allowed base branches |
| Commitlint | `.github/workflows/commitlint.yml` | Enforces Conventional Commits on PR commits |

Do not weaken collaboration checks when adding product CI.

## Current CI Status

Use `docs/testing/strategy.md` as the authority for current CI coverage and
required-check candidates. As of the current docs baseline:

- Current CI covers collaboration guardrails, Go service tests/builds, goose
  migration apply, Gateway contract drift, and frontend Gateway API type drift.
- The best required-check candidates are Go service tests, goose migration
  apply, Gateway contract/API drift, and API type drift.
- Full frontend `check/build` CI, Vitest/React Testing Library/Playwright,
  path-filter matrices, Docker image builds, full DB integration test jobs, and
  cross-service E2E smoke are gaps until stable workflows and dependencies land.
- Open PRs, draft issues, and unmerged capabilities must be documented as
  pending/follow-up, not as current `develop` behavior.

When editing workflow specs, keep the current-vs-target distinction explicit:
current CI coverage, PR-before local recommendations, and future gaps are not
interchangeable.

---

## Auto Label Service Path Contract

### 1. Scope / Trigger

Update this contract when changing `.github/labeler.json` service labels,
service directory layout, or service documentation layout.

### 2. Signatures

- Workflow file: `.github/workflows/auto-label.yml`
- Config file: `.github/labeler.json`
- Rule section: `pathLabels[]`
- Rule shape: `{ "paths": string[], "labels": string[] }`

### 3. Contracts

Each service label must cover both implementation and documentation paths:

| Label | Required paths |
|-------|----------------|
| `service:gateway` | `services/gateway/**`, `docs/services/gateway/**` |
| `service:auth` | `services/auth/**`, `docs/services/auth/**` |
| `service:file` | `services/file/**`, `docs/services/file/**` |
| `service:qa` | `services/qa/**`, `docs/services/qa/**` |
| `service:knowledge` | `services/knowledge/**`, `docs/services/knowledge/**` |
| `service:document` | `services/document/**`, `docs/services/document/**` |
| `service:ai-gateway` | `services/ai-gateway/**`, `docs/services/ai-gateway/**` |

All labels referenced by `.github/labeler.json` must exist in the GitHub
repository. The workflow skips missing labels rather than failing the PR, so
local changes must verify remote label existence when adding a new label name.

### 4. Validation & Error Matrix

| Condition | Required handling |
|-----------|-------------------|
| `.github/labeler.json` is invalid JSON | Fix before commit; Auto Label would fail at runtime. |
| Referenced label does not exist remotely | Create the label or remove the rule before PR. |
| Service implementation path changes | Update the matching docs path rule in the same PR. |
| Service documentation path changes | Update the matching implementation path rule in the same PR. |

### 5. Good/Base/Bad Cases

- Good: `docs/services/knowledge/README.md` matches `documentation` and
  `service:knowledge`.
- Base: `services/knowledge/internal/service/service.go` matches `backend` and
  `service:knowledge`.
- Bad: `docs/services/knowledge/README.md` matches only `documentation`.

### 6. Tests Required

- Parse `.github/labeler.json` as JSON.
- Run a local matcher using the same glob conversion as `auto-label.yml` for at
  least one implementation path and one docs path per service label.
- Check all configured labels exist with `gh label list` before adding a new
  label reference.

### 7. Wrong vs Correct

#### Wrong

```json
{
  "paths": ["services/knowledge/**"],
  "labels": ["service:knowledge"]
}
```

#### Correct

```json
{
  "paths": ["services/knowledge/**", "docs/services/knowledge/**"],
  "labels": ["service:knowledge"]
}
```

---

## Auto Label Blocked PR Contract

### 1. Scope / Trigger

Update this contract when changing PR issue-link requirements, task issue
blocked semantics, or `.github/workflows/auto-label.yml` blocked-label logic.

### 2. Signatures

- Workflow file: `.github/workflows/auto-label.yml`
- PR events: `pull_request_target` opened, edited, synchronize, reopened,
  ready_for_review, labeled, unlabeled
- Issue events: `issues` edited, labeled, unlabeled, closed, reopened
- Primary PR link source: GitHub `closingIssuesReferences`
- Fallback PR link syntax: GitHub closing keywords in the `关联 Issue` section,
  for example `Closes #118`, `Fixes #119`, or `Resolves #120`
- Synced label: `blocked`

### 3. Contracts

- The workflow only treats GitHub closing issue references as linked issues.
- A PR receives `blocked` only when it has at least one linked issue and every
  linked issue is blocked.
- A managed task issue with body fields is blocked only when it is open and has
  task body field `状态：Blocked` or `Risk：Blocked`.
- A non-task linked issue without those body fields may use issue label
  `blocked` as the blocked signal.
- Closed issues, pull request pseudo-issues, unreadable issues, and issues
  without blocked state count as not blocked.
- On issue changes, the workflow finds open pull requests that reference that
  issue through timeline cross-references and PR search, then recomputes the PR
  `blocked` label.

### 4. Validation & Error Matrix

| Condition | Required handling |
|-----------|-------------------|
| PR has no linked issues | Remove `blocked` from the PR if present. |
| PR has mixed blocked and not-blocked linked issues | Remove `blocked` from the PR. |
| All linked issues are blocked | Add `blocked` to the PR when the label exists. |
| Linked issue changes from blocked to not blocked | Recompute open linked PRs and remove `blocked` where needed. |
| A linked issue cannot be read | Treat it as not blocked and log a warning. |
| `blocked` label does not exist remotely | Skip adding it and log a warning rather than failing unrelated PR labeling. |

### 5. Good/Base/Bad Cases

- Good: PR body contains `Closes #118` and `Fixes #119`; both issues are open
  with `Risk：Blocked`; PR gets `blocked`.
- Base: PR body contains `Closes #118`; issue #118 is open with
  `状态：In Progress`; PR does not get `blocked`.
- Bad: PR body says `关联 Issue: #118` without a closing keyword and expects
  blocked sync.

### 6. Tests Required

- Parse `.github/workflows/auto-label.yml` as YAML.
- Run `actionlint`.
- Extract the embedded `github-script` body and run `node --check` inside an
  async wrapper.
- Before relying on a new synced label name, verify it exists with
  `gh label list`.

### 7. Wrong vs Correct

#### Wrong

```markdown
## 关联 Issue

- #118
```

#### Correct

```markdown
## 关联 Issue

- Closes #118
```

---

## Target Product Workflows

Recommended workflow files after the corresponding implementation issues land:

| Workflow | Suggested File | Trigger |
|----------|----------------|---------|
| Frontend CI | `.github/workflows/frontend-ci.yml` | `apps/web/**` |
| Go Services CI | `.github/workflows/go-services-ci.yml` | `services/**` |
| Docker Build | `.github/workflows/docker-build.yml` | service Dockerfiles, service code, `deploy/**` |
| Deploy | `.github/workflows/deploy.yml` | protected branch or manual dispatch |

Use path filters so unrelated documentation or service changes do not run every
job. A workflow may still run a cheap detection job to decide which service jobs
are needed.

## Scenario: Gateway Active API Contract Workflow

### 1. Scope / Trigger

- Trigger: changing the public gateway OpenAPI, gateway active owner map,
  frontend OpenAPI generation command, or the gateway contract verifier.
- Applies to `docs/services/gateway/api/openapi.yaml`,
  `docs/services/gateway/docs/active-api-owner-map.md`, `apps/web/package.json`,
  `package.json`, `scripts/verify_gateway_active_api.py`, `scripts/tests/**`,
  and `.github/workflows/gateway-contract.yml`.

### 2. Signatures

Local commands:

```bash
python scripts/verify_gateway_active_api.py
bun run check:gateway-contract
python -m unittest scripts.tests.test_verify_gateway_active_api
```

Workflow file:

```text
.github/workflows/gateway-contract.yml
```

### 3. Contracts

The verifier is the CI gate for these executable contracts:

- Active `/api/v1/**` operations must include `operationId`, non-empty `tags`,
  `x-owner-service`, effective `security`, at least one `2XX` response, and at
  least one `4XX` response.
- `/healthz` and `/readyz` are operational exceptions owned by `gateway` and may
  use `security: []`.
- Stable active public paths must not use action-style segments such as
  `login`, `logout`, `register`, `download`, `search`, `generate`, `export`,
  `retry`, or `revoke`.
- `x-missing-contracts.placeholderOperations` must not overlap active OpenAPI
  paths.
- `apps/web` API type generation must use
  `../../docs/services/gateway/api/openapi.yaml`.
- `docs/services/gateway/docs/active-api-owner-map.md` must match the active
  operations, owner summary, and missing contract placeholders derived from
  OpenAPI.

### 4. Validation & Error Matrix

| Condition | Required handling |
| --- | --- |
| OpenAPI metadata is missing on an active `/api/v1/**` operation | Verifier exits non-zero and names the method/path and missing field. |
| Owner map table or summary drifts from OpenAPI | Verifier exits non-zero and reports owner-map drift. |
| Missing-contract placeholder overlaps an active operation | Verifier exits non-zero and names the overlapping placeholder. |
| Frontend generation source changes away from gateway OpenAPI | Verifier exits non-zero and prints the expected source path. |
| PyYAML is unavailable in CI | Workflow installs `pyyaml` before running verifier commands. |

### 5. Good/Base/Bad Cases

- Good: update OpenAPI and owner map together, then run
  `bun run check:gateway-contract`.
- Base: update only verifier tests or workflow wiring; CI still runs the
  verifier unit tests and real-contract check.
- Bad: add `GET /api/v1/search` or an active operation without a `4XX` response
  and rely on manual review to catch it.

### 6. Tests Required

- Unit tests must cover missing required metadata, missing `4XX`, action-style
  path segments, missing-contract overlap, frontend generation source drift,
  and owner-map drift.
- Local verification before PR must run:

```bash
python -m unittest scripts.tests.test_verify_gateway_active_api
python scripts/verify_gateway_active_api.py
```

### 7. Wrong vs Correct

#### Wrong

```text
Change docs/services/gateway/api/openapi.yaml
Skip docs/services/gateway/docs/active-api-owner-map.md
Open PR without running the verifier
```

#### Correct

```text
Change docs/services/gateway/api/openapi.yaml
Update docs/services/gateway/docs/active-api-owner-map.md
Run bun run check:gateway-contract
Let .github/workflows/gateway-contract.yml enforce the same gate in PR
```

---

## Frontend CI Target

Frontend CI is not yet a landed required workflow. Until it exists, frontend
`check` and `build` remain PR-before local checks that should be reported in
the PR body when frontend files change.

When the workflow is added, it should run only when frontend files or
frontend-related workflow files change.

Target steps:

```bash
bun install --frozen-lockfile
bun run --cwd apps/web check
bun run --cwd apps/web build
```

Rules:

- Keep CI commands behind package scripts.
- Do not encode a specific build tool in workflow logic unless the frontend tool is selected and documented.
- Cache package-manager dependencies using lockfile-based keys.
- Fail if the lockfile and package manifest are inconsistent.
- Add Vitest, React Testing Library, and Playwright only after their scripts and
  dependencies exist in `apps/web/package.json`; do not mark them required from
  this spec alone.

---

## Go Services CI

Each Go service owns an independent `go.mod`. CI must test and build changed
services independently.

Service paths:

```text
services/gateway/
services/auth/
services/file/
services/qa/
services/knowledge/
services/document/
services/ai-gateway/
```

Required service-local checks:

```bash
go test ./...
go build ./cmd/server
```

Rules:

- Run checks from the changed service directory.
- Do not rely on a root `go.mod`.
- Cache Go modules per service or with keys that include service `go.sum`.
- If shared code is introduced later, update path filters so dependent services run.
- Use a matrix job when multiple services changed.

Example matrix dimensions:

```yaml
service:
  - gateway
  - auth
  - file
  - qa
  - knowledge
  - document
  - ai-gateway
```

---

## Docker Build

Every runtime service should have its own Dockerfile:

```text
apps/web/Dockerfile
services/gateway/Dockerfile
services/auth/Dockerfile
services/file/Dockerfile
services/qa/Dockerfile
services/knowledge/Dockerfile
services/document/Dockerfile
services/ai-gateway/Dockerfile
```

Rules:

- Use multi-stage builds for Go services.
- Produce small runtime images.
- Build images for changed services on PRs.
- Push images only from trusted branches or manual release workflows.
- Tag images with commit SHA and, when applicable, branch or release tags.
- Never build images with secrets baked into layers.

---

## Docker Compose Deployment

Deployment uses `deploy/docker-compose.yml` on a single machine.

Compose must include:

- frontend,
- gateway,
- auth,
- file,
- qa,
- knowledge,
- document,
- ai-gateway,
- postgres,
- redis,
- qdrant,
- minio.

Deployment rules:

- Store runtime secrets outside the repository.
- Use `.env.example` for required variable names only.
- Use named volumes for PostgreSQL, Qdrant, MinIO, and Redis when persistence is required.
- Expose only frontend and gateway publicly by default.
- Keep internal services on the Compose network.
- Add health checks for infrastructure and services before relying on automated deployment.

---

## Secrets and Environments

GitHub Actions secrets should be scoped by environment:

- `staging` for test deployment,
- `production` for release deployment if production is later enabled.

Never commit:

- database passwords,
- session, service-token, or signing secrets,
- MinIO access keys or secret keys,
- API keys,
- SSH private keys,
- cloud credentials.

Deployment workflows should use GitHub Environments and required reviewers for
production-like targets.

---

## Rollback

Every deployment workflow must have a documented rollback path before production
use.

Minimum rollback strategy:

1. Keep previous image tags available.
2. Keep the previous Compose file or release revision identifiable.
3. Re-deploy the last known-good image tags.
4. Do not run irreversible migrations automatically without an explicit release decision.

---

## Checks Before Merge

Current required checks are defined by repository branch protection and
`docs/testing/strategy.md`; do not infer additional required checks from target
workflow sections above. For PRs:

- PR Guard passes.
- Commitlint passes.
- Current product CI passes for touched areas when the workflow exists.
- Frontend changes should report local `bun run --cwd apps/web check` and
  `bun run --cwd apps/web build` until full frontend CI lands.
- Docker build, full DB integration jobs, and cross-service smoke are future
  gates until stable workflows land.
- Documentation changes update README/specs when architecture, commands,
  contracts, or implementation status change.

---

## Common Mistakes

- Running all service builds for every small frontend change.
- Assuming a root Go module exists.
- Pushing Docker images from untrusted pull request contexts.
- Committing production `.env` files.
- Exposing internal services directly to the public network.
- Adding deployment automation before rollback and secret handling are defined.
