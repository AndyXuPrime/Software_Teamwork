# Issue 550 Local Startup Diagnostics Acceptance Design

## Boundaries

This is a validation and reporting task. The intended code surface is limited to
the generated test report unless an acceptance-blocking defect is found.

The local runtime boundary follows the current project contract:

- Docker Compose runs shared infrastructure only: `postgres`, `redis`,
  `qdrant`, `minio`, and one-shot `minio-init`.
- Backend business services run on the host through
  `scripts/local/run-backend.sh`.
- Go module downloads are controlled by `GOPROXY` / `GOSUMDB`.
- Python package index behavior is controlled by `UV_DEFAULT_INDEX`.
- Docker image pulls are controlled by Compose image variables / registry
  rewrite, daemon mirror, or proxy.

## Evidence Strategy

Normal startup uses the real workspace and current `deploy/.env` so it validates
the integration path contributors will use locally. Before running it, any
legacy Compose business-service containers are removed with `--remove-orphans`
so host ports are free and the Docker baseline matches policy.

Failure scenarios use an isolated temporary Git worktree or copy so the test can
modify `deploy/.env` and put fake tools earlier on `PATH` without touching the
developer's real `deploy/.env` or relying on slow/flaky network failures. These
scenarios still run the real repository scripts:

- Go module failure: fake `go env` reports upstream defaults and fake
  `go mod download` returns a proxy timeout. The expected output includes
  repository defaults used for this run, the failed service, effective
  `GOPROXY` / `GOSUMDB`, and remediation guidance. The temp `deploy/.env` hash
  must be unchanged before/after.
- Early service exit: fake `go mod download` succeeds and fake `go run` prints a
  service failure then exits quickly. The expected output includes
  `backend startup failed for:` and service log tails.

Stop cleanup uses the real workspace after normal startup. It validates that
process groups are stopped and `.local/run/*.pid` files are removed.

## Compatibility

The task must avoid changing user-owned local configuration. Any temporary
`deploy/.env` manipulation happens outside the main workspace and is cleaned up
after evidence is captured.

The test report should be committed as a normal documentation artifact. Large
runtime failures discovered during smoke testing are reported rather than hidden
behind local cleanup.

## Rollback

If the normal startup leaves host-run backend processes alive, run
`./scripts/local/stop-backend.sh`. If Docker is left in an unexpected state, run
`docker compose -f deploy/docker-compose.yml --env-file deploy/.env ps` to
inspect and `docker compose -f deploy/docker-compose.yml --env-file deploy/.env
up -d --remove-orphans` to restore the infra-only baseline.
