# Gateway Service

`gateway` is the public backend entrypoint for frontend, admin, backend module,
and tool callers. This first implementation slice only provides the service
skeleton, operational health checks, request edge middleware, and standard JSON
response helpers. It does not implement downstream business handlers or own a
business database.

Authoritative contracts and service boundaries:

- `docs/services/gateway/README.md`
- `docs/services/gateway/api/openapi.yaml`
- `docs/services/gateway/docs/data-models.md`
- `docs/architecture/service-boundaries.md`
- `docs/architecture/technology-decisions.md`

## Local Run

```bash
cd services/gateway
go run ./cmd/server
```

Default address:

```text
:8080
```

Health checks:

```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
```

Both endpoints return the project success envelope and include `X-Request-Id`:

```json
{
  "data": {
    "status": "ok",
    "service": "gateway",
    "version": "0.1.0",
    "environment": "local"
  },
  "requestId": "req_123"
}
```

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `GATEWAY_HTTP_ADDR` | `:8080` | HTTP listen address. |
| `GATEWAY_SERVICE_VERSION` | `0.1.0` | Version reported by health checks and startup logs. |
| `GATEWAY_ENV` | `local` | Runtime environment label. |
| `GATEWAY_MAX_BODY_BYTES` | `10485760` | Maximum request body size enforced at the gateway edge. |
| `GATEWAY_REQUEST_TIMEOUT` | `30s` | Per-request context timeout. |
| `GATEWAY_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |
| `GATEWAY_CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed CORS origins. |
| `GATEWAY_CORS_ALLOWED_METHODS` | `GET,POST,PATCH,DELETE,OPTIONS` | Comma-separated allowed CORS methods. |
| `GATEWAY_CORS_ALLOWED_HEADERS` | `Authorization,Content-Type,X-Request-Id` | Comma-separated allowed CORS headers. |
| `GATEWAY_CORS_ALLOW_CREDENTIALS` | `false` | Whether CORS credentialed requests are allowed. |

## Tests

Run service-local checks from this directory:

```bash
go test ./...
go build ./cmd/server
```

## Scope Guardrails

- No PostgreSQL, Redis, MinIO, Qdrant, or provider client is initialized in this
  skeleton.
- Gateway does not persist auth, file, knowledge, QA, document, or AI Gateway
  business state.
- Public JSON responses use `{ "data": ..., "requestId": ... }` for success and
  `{ "error": { "code": ..., "message": ..., "requestId": ... } }` for errors.
- Future business routes must follow `docs/services/gateway/api/openapi.yaml`
  and delegate domain logic to the owning service.
