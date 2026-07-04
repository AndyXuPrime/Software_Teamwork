# Implementation Plan

## Checklist

- [x] Extend Knowledge adapter update request DTO with optional `parserConfigId`.
- [x] Resolve parser config snapshots during knowledge base update when `parserConfigId` is supplied.
- [x] Reuse creation-time conversion helpers for sanitized runtime parser config and protected credentials.
- [x] Include `parser_config_credentials` in runtime update payload only when credentials exist.
- [x] Teach runtime dataset update service to consume `parser_config_credentials` the same way dataset creation does.
- [x] Add or update Knowledge adapter tests for parser config update payloads, token redaction, and existing update compatibility.
- [x] Add targeted runtime tests for dataset update credential handling if an existing test surface is available.
- [x] Update OpenAPI contracts for the optional `parserConfigId` request field.
- [x] Run targeted Knowledge and runtime checks, then `git diff --check`.

## Validation Commands

```bash
cd services/knowledge && env -u GOROOT go test ./...
cd services/knowledge && env -u GOROOT go build ./cmd/adapter
cd services/knowledge-runtime && PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio python -m pytest <targeted-runtime-tests> -q
git diff --check
```

If a real PDF smoke is run, it must be explicitly reported because PaddleOCR cloud calls may consume quota.

## Risk Points

- Merging `parserConfigId` and `chunkStrategy` incorrectly could drop existing chunk settings.
- Runtime update may currently ignore credential payloads; creation and update must be kept consistent.
- Tests must assert tokens are not persisted or leaked.
- Existing documents must not be mutated as a side effect.
