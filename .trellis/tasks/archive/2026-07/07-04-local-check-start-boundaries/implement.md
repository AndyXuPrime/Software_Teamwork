# Fix local check and startup phase boundaries Implementation Plan

## Steps

1. Refactor `scripts/local/check.sh`. Done.
   - Remove Docker image inspection.
   - Keep command/file/runtime readiness checks.
   - Keep official and mainland China manual setup suggestions.
   - Ensure output is readable in captured logs and TTY.
2. Refactor `scripts/local/start.sh`. Done.
   - Keep startup behavior and `--pull never`.
   - Add/adjust post-start infra status output after Compose startup.
   - Ensure failure hints point to `check.sh`.
   - Keep log-tail behavior for early host process exits.
3. Update tests. Done.
   - `check.sh` must not call Docker pull/image inspect or Go/uv download/build commands.
   - `start.sh` must own Compose startup/status and runtime/backend process checks.
   - Contract checker tokens should reflect the preflight/start/post-check split.
4. Update docs/specs. Done.
   - README short human path.
   - deploy/runbook/service docs with the new boundary.
   - Trellis specs and testing strategy.
5. Validate. Done.
   - Passed: `bash -n scripts/local/check.sh scripts/local/start.sh scripts/local/stop.sh scripts/local/clean.sh scripts/local/lib/*.sh scripts/config/load-profile.sh`
   - Passed: `python3 -m py_compile scripts/verify_local_seed_contract.py scripts/check_docker_policy.py scripts/check_docker_environment.py scripts/tests/test_local_dev_up_script.py scripts/tests/test_knowledge_runtime_dependency_split.py`
   - Passed: `UV_CACHE_DIR=/tmp/codex-uv-cache PATH=/usr/local/go/bin:$PATH uv run --no-project --with pyyaml python -m unittest discover scripts/tests` (63 tests)
   - Passed: `python3 scripts/verify_local_seed_contract.py`
   - Passed: `python3 scripts/check_docker_policy.py`
   - Passed: `python3 scripts/check_docker_environment.py --skip-network --clean-env`
   - Passed: `PATH=/usr/local/go/bin:$PATH CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env`
   - Passed: `docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet`
   - Passed: `docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --services`
   - Passed: `git diff --check`
   - Observed: `./scripts/local/check.sh --runtime none` now reports missing prepared host tools/binaries and suggestions without Docker image inspection.

## Risk Points

- `check.sh` must not accidentally run Docker/startup logic while checking.
- `start.sh --infra-only` should not require runtime `.venv`.
- `start.sh --backend-only` should skip infra/migration/seed but still check selected runtime mode.
- Docs must not reintroduce deleted `prepare.sh`.

## Rollback

If the refactor fails, revert only the latest boundary-fix commit and keep the earlier branch commits intact.
