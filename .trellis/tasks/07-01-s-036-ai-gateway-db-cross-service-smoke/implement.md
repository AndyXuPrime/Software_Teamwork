# S-036 Implementation Plan

## Checklist

- [x] Load backend/shared specs before code edits.
- [x] Add an env-gated PostgreSQL smoke test under `services/ai-gateway/internal/repository` or an equivalent integration package.
- [x] Ensure default `go test ./...` skips the DB smoke when `AI_GATEWAY_TEST_DATABASE_URL` is absent.
- [x] Extend real provider smoke to include streaming chat if existing coverage is only non-streaming.
- [x] Update `docs/services/ai-gateway/docs/seed-runbook.md` with DB smoke, real provider run record/skip conditions, cross-service request samples, token hash, seed profile, and credential rotation guidance.
- [x] Run focused ai-gateway tests, then `cd services/ai-gateway && go test ./...`.
- [x] If PostgreSQL or real provider credentials are unavailable, stop after normal tests and report the exact missing env.

## Validation Commands

```bash
cd services/ai-gateway
go test ./...
AI_GATEWAY_TEST_DATABASE_URL="postgres://..." go test ./internal/repository -run TestPostgresRepositoryDBSmoke -count=1 -v
AI_GATEWAY_REAL_PROVIDER_SMOKE=1 \
AI_GATEWAY_REAL_PROVIDER_BASE_URL="https://api.example.com/v1" \
AI_GATEWAY_REAL_PROVIDER_API_KEY="$PROVIDER_API_KEY" \
AI_GATEWAY_REAL_CHAT_MODEL="provider-chat-model" \
AI_GATEWAY_REAL_EMBEDDING_MODEL="provider-embedding-model" \
AI_GATEWAY_REAL_EMBEDDING_DIMENSIONS="1024" \
AI_GATEWAY_REAL_RERANK_MODEL="provider-rerank-model" \
go test ./internal/http -run TestRealProviderSmoke_ExplicitEnvOnly -count=1 -v
```

## Risk Points

- PostgreSQL URL search path behavior must isolate the schema so the test never mutates a shared schema.
- Goose or migration execution must not require external network during the test.
- Provider smoke must not print raw API keys or prompts in failing assertions.
