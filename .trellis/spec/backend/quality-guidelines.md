# Quality Guidelines

> Code quality standards for Go backend services.

---

## Overview

Every backend service must remain independently testable, buildable, and
deployable. Quality checks run from each service directory because every service
owns a separate `go.mod`.

Minimum service-local checks:

```bash
go test ./...
go build ./cmd/server
```

When lint tooling is introduced, CI should run the selected linter for each
changed service.

---

## Go Service CI Baseline

### 1. Scope / Trigger

- Trigger: adding or changing repository CI for landed Go services under `services/*`.

### 2. Signatures

- Workflow: `.github/workflows/go-services.yml`.
- Events: `pull_request` and `push` to `develop` with path filters for `services/**` and the workflow file.
- Matrix key: `service`, with one entry for each landed Go service that owns a `go.mod`.

### 3. Contracts

- Toolchain: `actions/setup-go@v5` with `go-version: '1.25.x'`.
- Working directory: `services/${{ matrix.service }}`.
- Required commands for every matrix service: `go test ./...` and `go build ./cmd/server`.
- QA contract: run `go build ./cmd/agent` when `services/qa/cmd/agent` exists.
- Cache dependency input must exist for every matrix service; use `services/${{ matrix.service }}/go.mod` unless all services have `go.sum`.

### 4. Validation & Error Matrix

| Condition | Required response |
|-----------|-------------------|
| Service directory has `go.mod` but no matrix entry | Add it before merging CI changes. |
| Matrix entry has no `services/<name>/go.mod` | Remove or fix the entry; setup/run will fail. |
| Dockerfile Go image diverges from module baseline | Update module and Go build image together. |
| `services/qa/cmd/agent` exists but CI does not build it | Add or restore the QA agent build step. |

### 5. Good/Base/Bad Cases

- Good: `services/qa` runs tests, server build, and agent build under Go `1.25.x`.
- Base: a service with no `go.sum` still caches against its existing `go.mod`.
- Bad: a root-level Go workflow runs from the repository root and assumes a root `go.mod`.

### 6. Tests Required

- For each changed Go service, run `go test ./...` from the service directory.
- For each changed Go service, run `go build ./cmd/server` from the service directory.
- For QA, also run `go build ./cmd/agent` from `services/qa`.
- Run `git diff --check` before commit.

### 7. Wrong vs Correct

Wrong:

```yaml
with:
  go-version: '1.25.x'
  cache-dependency-path: services/${{ matrix.service }}/go.sum
```

Correct when not every service has `go.sum`:

```yaml
with:
  go-version: '1.25.x'
  cache-dependency-path: services/${{ matrix.service }}/go.mod
```

---

## Go Migration CI Baseline

### 1. Scope / Trigger

- Trigger: adding or changing repository CI for service-owned PostgreSQL migrations under `services/*/migrations`.

### 2. Signatures

- Workflow: `.github/workflows/go-migrations.yml`.
- Events: `pull_request` and `push` to `develop` with path filters for service migrations, service README files, the workflow file, and technology decisions.
- Matrix key: `service`, with one entry for each landed Go service that owns SQL migration files.

### 3. Contracts

- PostgreSQL CI image: `postgres:16-alpine`.
- Goose command: `go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up`.
- Working directory: `services/${{ matrix.service }}`.
- Migration filenames must match ordered snake_case names such as `0001_create_users.sql`.
- SQL migrations must include `-- +goose Up`; `-- +goose Down` is optional only for forward-only slices.

### 4. Validation & Error Matrix

| Condition | Required response |
|-----------|-------------------|
| Service has `migrations/*.sql` but no matrix entry | Add the service to migration CI before merging. |
| SQL migration has no `-- +goose Up` annotation | Add the annotation so goose can parse it. |
| Migration filename lacks an ordered prefix | Rename to `0001_<snake_case_summary>.sql` or the next ordered prefix. |
| README goose command version differs from CI | Update both to `v3.27.1`. |

### 5. Good/Base/Bad Cases

- Good: `services/auth` migration applies against an empty PostgreSQL database with `goose@v3.27.1`.
- Base: a forward-only migration has `-- +goose Up` and no down section.
- Bad: a service relies only on PostgreSQL Docker init scripts, or README says `goose` without a pinned version.

### 6. Tests Required

- Run migration apply validation for every matrix service or rely on the PR workflow when local PostgreSQL is unavailable.
- Run `git diff --check` before commit.
- Run service-local Go tests when migration files or repository code changed.

### 7. Wrong vs Correct

Wrong:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

Correct:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up
```

---

## Forbidden Patterns

- Root-level Go module used to build all microservices together.
- Cross-service imports from `services/<other-service>/internal/...`.
- HTTP handlers that contain business rules, SQL, Qdrant queries, or MinIO object logic.
- Unbounded goroutines without cancellation.
- HTTP clients without timeouts.
- SQL built by concatenating user input.
- Panics for expected business errors.
- Global mutable state for request-scoped data.
- Logging secrets, tokens, raw credentials, or full sensitive payloads.

---

## Required Patterns

- Pass `context.Context` through request, service, repository, and infrastructure calls.
- Use graceful shutdown for HTTP servers.
- Validate environment configuration at startup.
- Keep service dependencies explicit in constructors.
- Keep business workflows in `internal/service/`.
- Keep persistence in `internal/repository/`.
- Use stable API response shapes: project-owned JSON APIs use
  `{ data, requestId }` / `{ error }`; AI Gateway model invocation success
  responses use OpenAI-compatible shapes as documented in
  `docs/services/ai-gateway/api/openapi.yaml`.
- Add or update tests for changed business logic.

---

## Testing Requirements

Use a risk-based test strategy:

| Change Type | Required Coverage |
|-------------|-------------------|
| Pure functions or validators | Unit tests |
| Service business workflows | Unit tests with mocked repositories/clients |
| Repository SQL changes | Integration tests when database test tooling exists |
| HTTP handlers | Handler tests for status code and response shape |
| Cross-service client changes | Contract-style tests or mocked HTTP server tests |
| Migration changes | Migration validation once tooling exists |

Test naming:

- Use `Test<FunctionOrBehavior>`.
- Prefer table-driven tests for validators, mappers, and policy decisions.
- Test expected errors explicitly with `errors.Is` or `errors.As`.

---

## Configuration Quality

- Read configuration from environment variables in `internal/config` using an
  `envconfig`-style structured loader.
- Validate all required variables at startup.
- Keep defaults safe for local development only.
- Do not read environment variables throughout business logic.
- Document required variables in service README or deployment docs.

---

## Code Review Checklist

Reviewers should check:

- [ ] Does the change stay within the correct service boundary?
- [ ] Are HTTP request and response contracts stable?
- [ ] Are errors classified and returned through the standard error shape?
- [ ] Is sensitive data excluded from logs and responses?
- [ ] Are database changes represented by service-owned migrations?
- [ ] Are Redis/Qdrant/MinIO responsibilities owned by the correct service?
- [ ] Are timeouts and context cancellation handled for external calls?
- [ ] Do tests cover the changed behavior?
- [ ] Can the service still run `go test ./...` and `go build ./cmd/server` independently?

---

## Common Mistakes

- Adding shared code before three services actually need the same behavior.
- Testing only handlers while business rules remain untested.
- Treating Docker Compose startup as a substitute for service-level tests.
- Allowing the gateway to accumulate all business logic.
