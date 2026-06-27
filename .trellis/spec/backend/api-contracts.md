# API Contracts

> Contract-first rules for gateway-facing and cross-service HTTP APIs.

---

## Scenario: Gateway Contract-First API

### 1. Scope / Trigger

- Trigger: any new or changed frontend-facing gateway endpoint, gateway
  response envelope, frontend API client DTO, or cross-service route ownership.
- Applies to `services/gateway/`, browser API clients under `apps/frontend/`,
  and the domain service that owns the endpoint's business state.

### 2. Signatures

Gateway public endpoints are documented in:

```text
docs/api/gateway.openapi.yaml
```

Public routes use these prefixes:

```text
GET /healthz
GET /readyz
/api/v1/**
```

Every OpenAPI operation must include:

- `operationId`
- `tags`
- `summary`
- at least one success response
- at least one `4XX` response for user-callable operations
- `x-owner-service` for routes backed by a service boundary

### 3. Contracts

Gateway success envelope:

```json
{
  "data": {},
  "requestId": "req_123"
}
```

Gateway paginated envelope:

```json
{
  "data": [],
  "page": {
    "page": 1,
    "pageSize": 20,
    "total": 100
  },
  "requestId": "req_123"
}
```

Gateway error envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "request validation failed",
    "requestId": "req_123",
    "fields": {
      "name": "is required"
    }
  }
}
```

Public IDs are strings. Public timestamps use OpenAPI `date-time`.

Gateway must pass request context to downstream services with these headers
when values are available:

| Header | Purpose |
| --- | --- |
| `X-Request-Id` | Correlate frontend request, gateway logs, and downstream logs. |
| `X-User-Id` | Authenticated user identity. |
| `X-User-Roles` | Comma-separated authenticated roles. |
| `X-User-Permissions` | Comma-separated authenticated permissions. |
| `X-Forwarded-For` | Original client address chain. |
| `X-Forwarded-Proto` | Original request protocol. |

### 4. Validation & Error Matrix

| Condition | Public response |
| --- | --- |
| Invalid request shape or field value | `400 validation_error` |
| Missing or invalid authentication | `401 unauthorized` |
| Authenticated caller lacks permission | `403 forbidden` |
| Resource does not exist or is hidden | `404 not_found` |
| State conflict | `409 conflict` |
| Rate or quota exceeded | `429 rate_limited` |
| Downstream service or infrastructure failed | `502 dependency_error` |
| Unexpected gateway failure | `500 internal_error` |

Do not forward raw downstream error bodies, SQL details, object keys, tokens,
prompts, vector payloads, or internal URLs to the frontend.

### 5. Good/Base/Bad Cases

- Good: add a gateway route to `docs/api/gateway.openapi.yaml`, mark
  `x-owner-service`, use the standard envelope, and update
  `docs/service-boundaries.md` if ownership is new.
- Base: proxy a domain-service route through gateway without changing the
  domain response shape, but still normalize errors to the gateway envelope.
- Bad: add a frontend call directly to `services/knowledge` or embed Qdrant,
  MinIO, SQL, prompt, or report-generation logic in gateway.

### 6. Tests Required

When implementation exists:

- Gateway handler tests assert status code, response envelope, and request id.
- Error tests cover validation, auth failure, forbidden, not found, and
  dependency failure where applicable.
- Cross-service client tests use mocked HTTP servers and assert propagated
  context headers.
- Frontend API client tests assert request path, response normalization, and
  error-code mapping.

For documentation-only contract changes:

- Run an OpenAPI linter against `docs/api/gateway.openapi.yaml`.
- Parse the YAML and verify `$ref` targets resolve.
- Check route prefix consistency: health routes stay unversioned, public API
  routes use `/api/v1/**`.

### 7. Wrong vs Correct

#### Wrong

```text
frontend -> services/knowledge/search
gateway handler -> Qdrant query -> raw vector payload response
```

#### Correct

```text
frontend -> gateway /api/v1/search
gateway -> knowledge service
knowledge service -> retrieval infrastructure
gateway -> normalized SearchResponse or ErrorResponse
```

## Related Documents

- `docs/gateway.md`
- `docs/api/gateway.openapi.yaml`
- `docs/service-boundaries.md`
- `docs/frontend-backend-contract.md`
