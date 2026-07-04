# Consolidate local setup into start.sh Implementation

## Checklist

- [x] Delete the old split preflight helper and remove its workflow references.
- [x] Update `scripts/local/start.sh` with preflight, idempotent prepare steps, and heartbeat output for long operations.
- [x] Change goose baseline to `v3.27.0` in scripts, docs, tests, CI, and Trellis specs.
- [x] Add/adjust tests in `scripts/tests/test_local_dev_up_script.py` for auto-prepare, skip behavior, `--china` source selection, and no `.env.local` mirror mutation.
- [x] Update local integration, Docker runbook, README, deploy docs, and validation scripts for start-only flow.
- [x] Remove the proven duplicate untracked `services/knowledge-runtime/9b5ad71b2ce5302211f9c61530b329a4922fc6a4` artifact.
- [x] Run script syntax checks.
- [x] Run focused script tests.
- [x] Run required Docker policy checks for Docker-related script/doc changes.
- [x] Re-run `./scripts/local/start.sh --china` or a safe focused mode if the full stack is already running.

## Validation Commands

```bash
bash -n scripts/local/start.sh scripts/local/stop.sh scripts/local/clean.sh scripts/local/lib/*.sh scripts/config/load-profile.sh
python3 -m unittest scripts.tests.test_local_dev_up_script scripts.tests.test_local_seed_contract scripts.tests.test_knowledge_runtime_dependency_split
python3 scripts/verify_local_seed_contract.py
python3 scripts/check_docker_policy.py
python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment
CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet
./scripts/local/start.sh --infra-only --china
```

## Risk Notes

- Do not change Docker image defaults or introduce mirror values in committed config.
- Do not write mirror values into `.env.local`; `--china` is per-run behavior plus generated `.local/config/dev.env` rewrite.
- Keep setup idempotent so repeated `start.sh` runs do not rebuild/pull/download unnecessarily.
- Do not silence child command output; progress wrappers should add context, not hide native progress bars/logs.
