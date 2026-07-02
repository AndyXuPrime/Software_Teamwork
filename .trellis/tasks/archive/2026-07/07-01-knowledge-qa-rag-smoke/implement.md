# Implementation Plan

## Steps

1. Inspect current smoke and contract code.
   - Confirm existing Knowledge ingestion smoke helpers and QA citation extraction path.
   - Confirm AI Gateway smoke expectations and Compose profile variables.
2. Add the minimum RAG smoke implementation.
   - Reuse or extend service-local integration tests where practical.
   - Include sample document text, sample question, expected hit assertion, answer assertion, and citation assertion.
   - Add stage-labelled prechecks and failure messages for File, Parser, Knowledge, AI Gateway, and QA.
3. Update runbooks and implementation docs.
   - Add startup order and command sequence to `docs/runbooks/local-integration.md`.
   - Keep real-provider setup explicit and separate from default fake/stub/local runs.
   - Update Knowledge/QA implementation status tables if needed.
4. Run targeted checks.
   - `cd services/knowledge && go test ./internal/integration`
   - `cd services/qa && go test ./internal/platform/modelclient ./internal/service ./internal/service/tools`
   - Any new service-local package tests added by the implementation.
   - `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
   - `python3 scripts/check_docker_policy.py` only if Docker/Compose/image/env policy files change.
   - `git diff --check`
5. Finish Trellis workflow.
   - Run `trellis-check` equivalent after code edits.
   - Decide whether a spec update is warranted.
   - Commit with a Conventional Commit message when verification is complete.

## Rollback Points

- If full QA runtime smoke proves too dependent on unavailable provider setup, keep a Knowledge retrieval smoke plus QA citation/AI Gateway runbook and document the real-provider blocker clearly.
- If Compose changes are required, validate Compose config before running builds.
- If a service boundary bug is discovered, stop and update this task's PRD/design before implementing new product behavior.
