# Implementation Plan

## Steps

1. Strengthen cloud seed safety defaults.
   - Change `deploy/docker/cloud.env.example` to default `DOCKER_SEED_ENABLED=false`.
   - Update comments around seed and token fields so demo seed is explicit opt-in.
   - Add placeholder-token rejection to `scripts/docker/start.sh`.
   - Add the same guard to `deploy/docker/full/seed.sh`.

2. Strengthen Docker boundary policy.
   - Make `scripts/check_docker_policy.py` wording distinguish local infra-only compose from the explicit cloud app stack.
   - Add/adjust unit tests that prove cloud `build:` is allowed only for `deploy/docker-compose.cloud.yml`, and local heavy dependencies remain forbidden in cloud compose.

3. Align docs/spec/agent guidance.
   - Update `AGENTS.md`, README, deploy README, local integration runbook, Docker image pull runbook, architecture docs, test strategy, and Trellis Docker/backend specs as needed.
   - Ensure docs no longer imply the cloud path defaults to writing demo seed into cloud databases.

4. Validate.
   - `bash -n scripts/docker/start.sh scripts/docker/stop.sh scripts/docker/clean.sh scripts/local/start.sh scripts/local/stop.sh scripts/local/clean.sh scripts/local/lib/*.sh scripts/config/load-profile.sh`
   - `sh -n deploy/docker/full/*.sh`
   - `python3 scripts/check_docker_policy.py`
   - `python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_cloud_docker_start_script`
   - Broader Docker unittest suite if policy/startup scripts changed.
   - `CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env`
   - `docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet`
   - `docker compose -f deploy/docker-compose.cloud.yml --env-file deploy/docker/cloud.env.example config --quiet`
   - A temp seed-enabled cloud env with placeholder tokens must fail in `scripts/docker/start.sh` before compose.
   - `git diff --check`

## Rollback Points

- Seed safety changes are limited to cloud Docker env/script paths; reverting them should not touch local seed SQL.
- Policy wording/test changes should remain separate from compose topology changes.
