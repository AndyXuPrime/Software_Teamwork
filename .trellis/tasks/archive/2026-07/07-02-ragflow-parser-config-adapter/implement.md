# Implementation Plan

1. Check out a local branch from PR #440 head.
2. Load backend specs before editing code.
3. Add adapter mapping from `service.ParserConfig` to RAGFlow
   `parser_config`.
4. Use the mapping in knowledge-base create/update payload construction:
   explicit `chunkStrategy` wins; default parser config fills only absent values.
5. Add focused unit tests for parser backend mapping and request payloads.
6. Pass OCR provider env vars to both `knowledge-runtime-api` and
   `knowledge-runtime-worker` in compose.
7. Run focused checks:
   - `go test ./internal/adapter/... ./internal/service/...`
   - `python3 scripts/check_docker_policy.py`
   - `cd deploy && docker compose config --quiet`
8. Review diff and commit with a Conventional Commit message.

## Risk Points

- `services/knowledge/internal/adapter/map.go` request payload behavior affects
  all KB create/update calls.
- Parser config storage is optional in adapter mode; creation must not fail when
  `DATABASE_URL` is intentionally unset.
- Compose env additions must not make local startup require provider secrets.
