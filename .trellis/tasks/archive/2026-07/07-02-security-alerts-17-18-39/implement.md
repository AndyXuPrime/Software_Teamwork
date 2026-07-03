# Implementation Plan

## Checklist

- [x] Upgrade `golang.org/x/net` in `services/file` to `v0.55.0` or newer using
      Go tooling.
- [x] Replace QA citation metadata map capacity arithmetic with an allocation
      pattern that does not depend on `len(item.Metadata)+1`.
- [x] Add focused unit tests for citation metadata serialization behavior.
- [x] Replace frontend QA chat `Math.random()` ID generation with Web Crypto.
- [x] Add Go module proxy defaults to `deploy/.env.example`.
- [x] Update local startup docs/runbooks to distinguish Go module proxy,
      Parser uv index, and Docker image registry paths.
- [x] Extend local seed/startup contract verification for the Go proxy defaults.
- [x] Run focused checks:
      - `go test ./...` from `services/file`
      - `go test ./...` from `services/qa`
      - `go build ./cmd/server` from changed Go services
      - `go build ./cmd/agent` from `services/qa`
      - `bun run --cwd apps/web check`
      - `bun run --cwd apps/web build`
      - `python3 scripts/verify_local_seed_contract.py`
      - `python3 scripts/tests/test_local_seed_contract.py`
      - `git diff --check`
- [x] Attempt `bun run --cwd apps/web test:e2e`; browser binary download was
      resolved through local mirror cache, but the current machine lacks
      Chromium system libraries such as `libatk-1.0.so.0` and `sudo` is not
      available to install `playwright install-deps chromium`.
- [x] Re-query or cite GitHub alert state after commit if possible. GitHub still
      reports alerts 17, 18, and 39 as `open` on the current baseline; Dependabot
      and CodeQL are expected to close them after a pushed branch or PR scan.

## Risk Points

- File service dependency updates may pull related Go x/* modules forward; keep
  Go tooling output scoped to `services/file`.
- QA metadata helper must preserve attachment metadata normalization exactly.
- Frontend secure ID fallback must work in Vitest/jsdom and browser runtimes
  without relying on Node-only APIs.
- Existing users with an old copied `deploy/.env` will need to copy or add the
  new `GOPROXY` / `GOSUMDB` lines manually; docs should make this visible.
