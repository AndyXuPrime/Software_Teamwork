# Issue 550 Local Startup Diagnostics Acceptance Implementation Plan

## Checklist

- [ ] Confirm Docker Compose baseline after orphan cleanup.
- [ ] Run required static/script/unit contract checks.
- [ ] Run real normal startup:
  - `./scripts/local/dev-up.sh`
  - `LOCAL_BACKEND_STARTUP_CHECK_SECONDS=2 ./scripts/local/run-backend.sh`
  - targeted health checks or process/log evidence
- [ ] Run real stop cleanup with `./scripts/local/stop-backend.sh`.
- [ ] Run isolated old-env Go module failure simulation and capture:
  - process-local repository defaults for `GOPROXY` / `GOSUMDB`;
  - non-zero failure output;
  - unchanged temp `deploy/.env` hash.
- [ ] Run isolated early-exit simulation and capture failed service summary plus
  log tails.
- [ ] Write
  `docs/testing/reports/2026-07-03/local-startup-diagnostics-test-report.md`
  using `docs/testing/templates/test-report-template.md`.
- [ ] Rerun final checks:
  - `git diff --check`
  - any checks affected by report or small fixes
- [ ] Review final diff and prepare issue/PR completion summary.

## Validation Commands

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --services
bash -n scripts/local/dev-up.sh scripts/local/run-backend.sh scripts/local/stop-backend.sh
python3 scripts/verify_local_seed_contract.py
python3 -m unittest scripts.tests.test_local_seed_contract
./scripts/local/dev-up.sh
LOCAL_BACKEND_STARTUP_CHECK_SECONDS=2 ./scripts/local/run-backend.sh
./scripts/local/stop-backend.sh
git diff --check
```

## Risk Points

- Real startup may fail because a host dependency is missing, a local port is
  occupied, Go modules cannot download, or seed/migration state is inconsistent.
  Record the exact failure and classify it as fixed, transferred, or blocked.
- Fake-tool simulations must be isolated from the real workspace and must not
  change user-owned `deploy/.env`.
- `run-backend.sh` starts long-running host processes. Always stop them before
  ending the task.
