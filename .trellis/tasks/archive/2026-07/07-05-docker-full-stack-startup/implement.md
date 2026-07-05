# Implementation Plan

## Checklist

1. Add `trellis-before-dev` context before editing.
2. Create Docker full-stack support files:
   - `scripts/docker/start.sh`
   - `scripts/docker/stop.sh`
   - `scripts/docker/clean.sh`
   - `deploy/docker-compose.cloud.yml`
   - Dockerfiles and entrypoints under `deploy/docker/full/`
   - `deploy/docker/cloud.env.example`
3. Keep `deploy/docker-compose.yml` unchanged.
4. Add concise documentation for the new one-line command and explicitly state
   that host-run startup remains unchanged.
5. Validate shell syntax:
   - `bash -n scripts/docker/*.sh`
6. Validate existing Docker policy remains intact:
   - `python3 scripts/check_docker_policy.py`
   - `python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment`
7. Validate config and Compose parsing:
   - `CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env`
   - `docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet`
   - `docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --services`
   - `docker compose -f deploy/docker-compose.cloud.yml --env-file deploy/docker/cloud.env.example config --quiet`
8. If feasible in the current environment, run a limited build smoke:
   - `./scripts/docker/start.sh --build-only` if implemented, or
   - `docker compose -f deploy/docker-compose.full.yml --env-file .env.example build <small service>`

## Risk Points

- Container env must use service names instead of `localhost` for internal
  service-to-service URLs.
- Cloud dependency env must use reachable managed endpoints and must not point
  at host-run `localhost` defaults.
- Migration/seed jobs need access to migrations and seed SQL inside images.
- The default Docker path cannot prove real ingestion unless the external
  Knowledge runtime and PaddleOCR/model providers are reachable.
- Frontend static serving needs `/api/v1` proxying so browser requests work
  with the default API base path.
- Existing Docker policy checks may flag the new business Compose path if the
  checker assumes all Compose files are infra-only; if so, update policy tests
  deliberately without weakening the root baseline rule.

## Done Criteria

- User can start the Docker path with a short documented command.
- Existing host-run path remains documented and valid.
- Docker cloud env docs explain required cloud dependencies and cloud OCR/model
  variables.
- Checks above pass, or unrun heavy runtime checks are reported with reasons.
