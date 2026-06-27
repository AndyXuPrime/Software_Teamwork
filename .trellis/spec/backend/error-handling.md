# Error Handling

> How Go backend services propagate, classify, log, and return errors.

---

## Overview

Backend services use explicit Go errors internally and stable JSON error
responses at HTTP boundaries. Errors should preserve enough context for logs
without leaking secrets or implementation details to clients.

Handlers are responsible for translating domain/service errors into HTTP
status codes. Service and repository layers should return typed or classifiable
errors, not HTTP responses.

---

## Error Categories

Every service should classify errors into these categories:

| Category | HTTP Status | Meaning |
|----------|-------------|---------|
| `validation_error` | `400` | Request shape or field value is invalid |
| `unauthorized` | `401` | Authentication is missing or invalid |
| `forbidden` | `403` | Authenticated caller lacks permission |
| `not_found` | `404` | Requested resource does not exist or is hidden |
| `conflict` | `409` | Request conflicts with current state |
| `rate_limited` | `429` | Caller exceeded a rate or quota limit |
| `dependency_error` | `502` | Downstream service or infrastructure failed |
| `internal_error` | `500` | Unexpected server-side failure |

Prefer service-local error constructors or sentinel errors over matching error
strings.

---

## Error Type Pattern

Use a small application error type per service when errors need structured
classification:

```go
type Code string

const (
    CodeValidation Code = "validation_error"
    CodeNotFound   Code = "not_found"
)

type AppError struct {
    Code    Code
    Message string
    Err     error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }
```

Rules:

- Wrap lower-level errors with `%w`.
- Do not expose database driver messages directly to HTTP clients.
- Keep user-facing messages stable and concise.
- Use `errors.Is` and `errors.As` for classification.

---

## Error Propagation

- Repository layer wraps infrastructure failures with operation context.
- Service layer converts expected repository failures into domain errors.
- Handler layer converts domain errors into HTTP responses.
- Middleware should recover panics, log them, and return `internal_error`.
- Do not log the same error at every layer. Log once at the boundary where the request context is available.

Example:

```go
user, err := svc.GetUser(ctx, id)
if err != nil {
    writeError(w, r, err)
    return
}
```

---

## API Error Response

All services should return a stable JSON error shape:

```json
{
  "error": {
    "code": "validation_error",
    "message": "email is required",
    "requestId": "req_123"
  }
}
```

Optional field-level validation details may be included:

```json
{
  "error": {
    "code": "validation_error",
    "message": "request validation failed",
    "requestId": "req_123",
    "fields": {
      "email": "must be a valid email address"
    }
  }
}
```

Rules:

- Include `requestId` whenever available.
- Do not include stack traces in HTTP responses.
- Do not include SQL, object keys, tokens, access keys, or internal URLs.
- Use the same response shape in `gateway` and internal services.

---

## Cross-Service Errors

When calling another service over HTTP:

- Treat non-2xx responses as dependency failures unless the downstream error is part of the caller's contract.
- Preserve downstream `requestId` in logs when available.
- Convert downstream errors to caller-owned error codes before returning to frontend.
- Use timeouts for every HTTP client.
- Do not blindly forward internal service error messages to users.

---

## Common Mistakes

- Returning raw `err.Error()` to clients.
- Matching errors by string content.
- Logging sensitive request bodies on validation or dependency failures.
- Converting all downstream failures to `500` without distinguishing dependency errors.
- Letting `gateway` and services return different error shapes.
