# Implementation Plan

1. Add `deploy/seeds/003-qa-document-mcp.sql`, wire it into `dev-up.sh`, and
   update cleanup and seed-contract verification.
2. Add Document MCP bootstrap variables to `deploy/.env.example`.
3. Update QA runtime MCP merging and add focused unit tests.
4. Add `docs/services/document/docs/mcp-tools.md`; link/update Document and QA
   service docs plus the local integration runbook.
5. Run formatting and focused tests, then the required Docker policy, seed,
   Compose, QA, and Document checks.
6. Inspect the final diff, archive the Trellis task, commit only scoped files,
   push the fork branch, and open a draft PR targeting upstream `develop`.

## Validation

- `gofmt` on changed Go files
- `cd services/qa && go test ./internal/service/...`
- `cd services/document && go test ./... && go build ./cmd/server`
- `python3 scripts/verify_local_seed_contract.py`
- `python3 -m unittest scripts.tests.test_local_seed_contract`
- `python3 scripts/check_docker_policy.py`
- `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
- `git diff --check`

## Risk and rollback points

- Runtime merge semantics are security-sensitive; tests must cover stored-token
  precedence and disabled-record precedence.
- Seed changes must remain idempotent and must not include plaintext or fixed
  encrypted service tokens.
- Do not modify root Compose service membership.
