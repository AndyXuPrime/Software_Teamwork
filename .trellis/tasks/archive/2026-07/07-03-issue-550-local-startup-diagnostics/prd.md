# Issue 550 local startup diagnostics acceptance

## Goal

Complete GitHub issue #550 / T-016 by validating the local startup diagnostic
work delivered for #542, cleaning the local Docker Compose state back to the
current infrastructure-only baseline, and publishing a reproducible test report
under `docs/testing/reports/2026-07-03/`.

The test result must tell reviewers whether the startup scripts now surface Go
module mirror failures, early backend exits, start/stop summaries, and
Go/Docker/uv documentation boundaries without rewriting a user's local
`deploy/.env`.

## Confirmed Facts

- #550 is open and depends on #542. #542 is closed, so the acceptance task can
  proceed.
- `deploy/docker-compose.yml` currently defines only `postgres`, `redis`,
  `qdrant`, `minio`, and `minio-init`.
- Local Docker had orphan business-service containers from an older Compose
  baseline. They were removed with `docker compose -f deploy/docker-compose.yml
  --env-file deploy/.env up -d --remove-orphans`; the remaining running
  containers are the four long-running infrastructure services and all are
  healthy.
- The relevant runtime contract is documented in
  `.trellis/spec/backend/quality-guidelines.md`,
  `.trellis/spec/cicd.md`, `docs/testing/strategy.md`,
  `README.md`, `deploy/README.md`, and
  `docs/runbooks/local-integration.md`.
- `scripts/local/run-backend.sh` starts host-run services for Auth, File,
  Knowledge, AI Gateway, QA, Document, and Gateway. It must not start the
  retired standalone Parser.

## Requirements

- Validate the standard Docker/Compose state after orphan cleanup and record
  the result.
- Run the script/static contract checks named by #550:
  - `bash -n scripts/local/dev-up.sh scripts/local/run-backend.sh scripts/local/stop-backend.sh`
  - `python3 scripts/verify_local_seed_contract.py`
  - `python3 -m unittest scripts.tests.test_local_seed_contract`
  - `git diff --check`
- Validate four local-startup paths with reproducible evidence:
  - normal startup path: `dev-up.sh`, `run-backend.sh`, and health/log evidence
    sufficient to prove the host-run backend came up;
  - Go module failure path: an isolated old `deploy/.env` missing `GOPROXY` and
    `GOSUMDB`, with a simulated `go mod download` failure, proving the script
    uses process-local repository defaults and does not rewrite the env file;
  - service early-exit path: an isolated run where `go mod download` succeeds
    but `go run` exits immediately, proving `run-backend.sh` reports failed
    services and `.local/logs/<service>.log` tails instead of silently claiming
    success;
  - stop cleanup path: `stop-backend.sh` stops/removes host-run pid files and
    reports a success summary.
- Generate the required report at
  `docs/testing/reports/2026-07-03/local-startup-diagnostics-test-report.md`
  using the repository testing report template.
- Record any unavailable environment dependency as an explicit skipped/blocked
  item with residual risk.
- If a small issue is found in the tested surface, fix it in this task and
  rerun relevant checks. If a large issue is found, create or link a follow-up
  owner issue instead of broadening this task.

## Acceptance Criteria

- [ ] Docker state is back to the documented infra-only baseline; orphan
  business service containers are removed and evidence is captured.
- [ ] The test report contains evidence for normal startup, Go module failure,
  service early exit, and stop cleanup paths.
- [ ] The old `deploy/.env` simulation proves scripts do not rewrite local env
  files while still using process-local `GOPROXY` / `GOSUMDB` defaults.
- [ ] Failure output points to the next diagnostic entry, including current
  `GOPROXY` / `GOSUMDB` for Go module failures and log tails for early exits.
- [ ] Static/script/unit checks pass or any failure is documented with
  owner/follow-up.
- [ ] The final issue/PR summary can cite the report path, executed commands,
  environment, results, and remaining risk.

## Out of Scope

- Full business E2E and real model-provider validation.
- Restoring or validating the retired standalone Parser.
- Changing Docker image policy, Compose services, or local startup behavior
  unless a small acceptance-blocking issue is discovered.
