# Auth Service

`services/auth` owns users, credentials, roles, permissions, sessions, token
hash metadata, revocations, and security events. Public frontend access still
goes through gateway; this service exposes only operational endpoints in the
current baseline.

## Current Scope

Implemented now:

- Independent Go module.
- `GET /healthz`
- `GET /readyz`
- `POST /internal/v1/users`
- `POST /internal/v1/sessions`
- `GET /internal/v1/users/{userId}`
- `GET /internal/v1/users/{userId}/permissions`
- `GET /internal/v1/sessions/{sessionId}`
- `DELETE /internal/v1/sessions/{sessionId}`
- Validated runtime configuration.
- PostgreSQL migration for auth users, credentials, roles, permissions, user
  roles, role permissions, sessions, revocations, and security events.
- Seed migration for `standard`, `admin`, and `super_admin` roles plus baseline
  permission strings.
- Service-local `sqlc.yaml` plus explicit-column query files.
- `pgx` repository adapter for user, credential, session, role, permission,
  revocation, and security-event persistence.
- Argon2id password hashes and opaque bearer access-token issuance with
  versioned HMAC token hashes.
- Repository, service, config, and HTTP tests.

Out of scope for this baseline:

- Gateway public `/api/v1/**` route implementation and Redis session caching.
- User, role, and permission management endpoints beyond default role assignment
  and source reads needed by gateway.
- Public `/api/v1/**` routes.

## Local Run

```bash
cd services/auth
go test ./...
go build ./cmd/server
AUTH_HTTP_ADDR=:8001 go run ./cmd/server
```

Check the service:

```bash
curl http://localhost:8001/healthz
curl http://localhost:8001/readyz
```

Without `AUTH_DATABASE_URL`, `/readyz` returns `503` with `postgres` marked
`not_configured`. This is intentional so the process can start locally while
readiness still reflects durable store availability.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `AUTH_HTTP_ADDR` | `:8001` | HTTP listen address. |
| `AUTH_SERVICE_VERSION` | `0.1.0` | Service version returned by health checks. |
| `AUTH_ENV` | `local` | Runtime environment label. |
| `AUTH_DATABASE_URL` | unset | PostgreSQL connection string. Required for readiness. |
| `AUTH_INTERNAL_SERVICE_TOKEN` | unset | Shared service-to-service token expected in `X-Service-Token`. Required when `AUTH_DATABASE_URL` is set. |
| `AUTH_TOKEN_HASH_SECRET` | unset | HMAC secret for access-token hashes. Required when `AUTH_DATABASE_URL` is set. |
| `AUTH_TOKEN_HASH_KEY_VERSION` | `v1` | Version label embedded in token hash values. |
| `AUTH_SESSION_TTL` | `24h` | Access-token session lifetime. Accepts Go duration strings or seconds. |
| `AUTH_DEFAULT_ROLE_CODE` | `standard` | Role assigned to newly created users. |
| `AUTH_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |
| `AUTH_READINESS_TIMEOUT` | `2s` | PostgreSQL readiness check timeout. |

Do not log `AUTH_DATABASE_URL`, `AUTH_INTERNAL_SERVICE_TOKEN`, or
`AUTH_TOKEN_HASH_SECRET` because they may contain credentials.

## Migration

Run the project-pinned goose version, then apply the service-owned migrations:

```bash
cd services/auth
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$AUTH_DATABASE_URL" up
```

The first migration is forward and down capable:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$AUTH_DATABASE_URL" down
```

## sqlc

The service keeps SQL query files under `internal/repository/queries/` and
`sqlc.yaml` at the service root. Generate code with the pinned command:

```bash
cd services/auth
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

Generated code is committed under `internal/repository/sqlc/`; the repository
adapter maps generated rows to auth service-domain structs before returning to
service callers.

## Repository Notes

- PostgreSQL access uses `pgx/v5`.
- Query files must not use `SELECT *`.
- Repository methods return service-domain structs, not generated SQL row
  structs.
- Token hashes and password hashes must never be returned by HTTP handlers or
  written to logs.
