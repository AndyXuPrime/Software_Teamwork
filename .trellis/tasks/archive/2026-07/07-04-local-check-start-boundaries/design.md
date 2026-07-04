# Fix local check and startup phase boundaries Design

## Boundary

The public local workflow has three operational phases:

1. `check.sh`: pre-start readiness.
2. `start.sh`: startup plus post-start observation.
3. `stop.sh`: process shutdown.

`clean.sh` stays as an explicit destructive maintenance helper and should not appear in the main human path.

## `check.sh`

`check.sh` should be side-effect light. It may read files and inspect command availability. It should avoid Docker daemon-dependent image inspection because image/container state is meaningful only after the user starts or explicitly pulls images.

Checks:

- host commands: Docker CLI, Go, `psql`, Bun, curl, Python, uv when runtime mode is not `none`
- root `.env.local`
- prepared `.local/tools/config-ctl`, `.local/tools/goose`, local seed helper
- prepared `.local/bin/*` service binaries
- runtime `.venv` and key artifact paths when requested

Suggestions:

- Official source commands for manual setup.
- Mainland China alternatives under `--china`.
- No command is executed by `check.sh`; suggestions remain text.

## `start.sh`

`start.sh` already owns the real startup sequence:

- render config through prepared `config-ctl`
- start Compose infra with `--pull never`
- run `minio-init`
- run migrations and seed
- start runtime API/worker according to runtime mode
- start prepared backend binaries

Post-start status should be part of `start.sh`, not `check.sh`.

Post-start checks:

- `docker compose ps` for the infra services after Compose startup
- process-group alive check for backend and runtime processes already started by the script
- existing HTTP readiness waits for runtime API and Elasticsearch stay in runtime startup
- tail relevant `.local/logs/*.log` when a process exits early

## Compatibility

- `start.sh` keeps default runtime mode `full` from the previous commit unless tests/docs reveal a blocking issue.
- `--runtime none`, `--runtime api`, `--infra-only`, `--backend-only`, and `--china` remain supported.
- `check.sh` should remain executable and replace the deleted `prepare.sh`.
- Existing generated Trellis archive and journal commits stay in branch history.

## Documentation Shape

Human docs:

- README first path: `cp .env.example .env.local`, `check.sh`, `start.sh`, frontend, stop.
- Download/setup details are short and point to `check.sh --china` plus runbook.

Maintenance docs/specs:

- Explain exactly which phase owns preflight checks, startup, and post-start verification.
- Keep official source defaults and mainland China mirror suggestions explicit.

