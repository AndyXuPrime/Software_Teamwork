# Fix cloud docker CI coverage

## Goal

Close the review gap between the cloud Docker documentation and the actual
Docker / Deploy Checks workflow. The workflow must validate the new cloud
Docker app stack and its wrapper scripts so future regressions fail in CI,
not only during manual local checks.

## Confirmed Facts

- `docs/testing/strategy.md` says `docker-deploy-checks.yml` validates both
  `deploy/docker-compose.yml` and `deploy/docker-compose.cloud.yml` with
  `docker compose ... config --quiet`.
- `.github/workflows/docker-deploy-checks.yml` currently keeps
  `knownComposeFiles` limited to `deploy/docker-compose.yml`, so changed cloud
  compose files do not enter the compose-config matrix.
- The current compose validation step uses `.local/config/dev.env` for the root
  infra Compose file and plain `docker compose -f "$compose_file" config
  --quiet` for every other file.
- `docs/testing/strategy.md` lists shell syntax checks for `scripts/docker/*.sh`
  and `deploy/docker/full/*.sh`.
- The workflow currently runs `bash -n` only for `scripts/local/*.sh`,
  `scripts/local/lib/*.sh`, and `scripts/config/load-profile.sh`.

## Requirements

- Add `deploy/docker-compose.cloud.yml` to the workflow's known Compose file
  allowlist so the existing path-derived matrix can include it safely.
- Make cloud compose changes trigger compose-config validation, including
  changes to `deploy/docker/cloud.env.example`.
- Validate `deploy/docker-compose.cloud.yml` with
  `--env-file deploy/docker/cloud.env.example`, matching the testing docs and
  the cloud stack's required interpolation inputs.
- Expand Docker/deploy workflow detection and shell syntax coverage to include
  `scripts/docker/*.sh` and `deploy/docker/full/*.sh`.
- Preserve the existing root Compose validation behavior:
  `CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env`
  followed by root compose validation with `.local/config/dev.env`.
- Keep the path-derived matrix allowlisted; do not switch to untrusted arbitrary
  compose paths.

## Acceptance Criteria

- [x] A PR that changes `.github/workflows/docker-deploy-checks.yml` or
      `deploy/docker-compose.cloud.yml` validates both root infra Compose and
      cloud Compose in the compose-config matrix.
- [x] A PR that changes only `deploy/docker-compose.cloud.yml` or
      `deploy/docker/cloud.env.example` validates the cloud Compose config with
      `--env-file deploy/docker/cloud.env.example`.
- [x] The policy-check job runs shell syntax checks for `scripts/docker/*.sh`
      and `deploy/docker/full/*.sh` in addition to existing local/config scripts.
- [x] Local verification covers workflow script syntax, Docker policy tests,
      root Compose config, cloud Compose config, and whitespace checks.

## Out Of Scope

- Changing Docker stack behavior, image tags, service definitions, or startup
  script runtime logic.
- Adding production deployment semantics to the cloud Docker app stack.
