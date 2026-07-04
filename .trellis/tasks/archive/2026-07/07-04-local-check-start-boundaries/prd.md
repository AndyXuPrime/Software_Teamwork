# Fix local check and startup phase boundaries

## Goal

Make the local scripts match a clear user workflow:

```text
preflight check -> start required services -> post-start status/log check -> stop
```

`check.sh` should answer "can this machine start the project, and what manual setup is missing?"
`start.sh` should answer "start the stack and show what actually came up."

## Requirements

- `check.sh` is a pre-start readiness check only.
  - It must not start Docker containers.
  - It must not run Docker image inspection as a post-start substitute.
  - It must not run downloads, `docker pull`, `go build`, `go install`, `uv sync`, runtime artifact downloads, migrations, seed, or service startup.
  - It should check prerequisites needed before startup: `.env.local`, required host commands, prepared local tools/binaries, and runtime files when runtime startup is requested.
  - It should print concise manual setup suggestions for missing items, including official-source commands and mainland China alternatives.
- `start.sh` owns startup and post-start checks.
  - It should start the required infra Docker services, migrations, seed data, backend binaries, and Knowledge runtime according to selected options.
  - It should keep `--pull never` and never download Go/uv/runtime artifacts.
  - After startup, it should print a clear status summary for Docker infra, host-run backend processes, runtime mode, and relevant health/readiness checks.
  - If a service exits early or readiness fails, it should show the current stage, failed component, and relevant log tail.
- Human-facing docs should keep the main path simple: `check.sh`, `start.sh`, frontend command, `stop.sh`.
- AI/maintenance docs and specs should document the boundary: preflight check before startup, post-start checks after startup, manual setup suggestions only.
- Keep root `.env.example -> .env.local` as the local secret/env template. Do not move initialization into scripts beyond reading/rendering the existing env files.
- Keep mainland China support explicit through `check.sh --china` suggestions and `start.sh --china` runtime mirror behavior. Do not commit active third-party defaults.

## Acceptance Criteria

- [x] `./scripts/local/check.sh` does not call Docker Compose `up`, `docker pull`, `docker image inspect`, `go build`, `go install`, `uv sync`, `download_deps.py`, migrations, seed, or service startup.
- [x] `./scripts/local/check.sh --china` prints mainland China manual setup suggestions without editing `.env.local` or committed config.
- [x] `./scripts/local/start.sh` still uses `docker compose up --pull never`, host-run migrations/seed, prepared `.local/tools`, prepared `.local/bin`, and prepared runtime `.venv`.
- [x] `./scripts/local/start.sh` performs and prints post-start status for infra and host-run services, including useful log tails on failure.
- [x] Startup failure hints recommend `./scripts/local/check.sh` for preflight diagnostics.
- [x] README first path is easy for humans to scan and does not expose internal setup categories.
- [x] Runbooks/specs/tests describe the same split between preflight check, startup, and post-start verification.
- [x] Existing local seed, Docker policy, Compose config, and script unit checks pass.

## Notes

- This fixes the previous attempt where `check.sh` drifted into Docker image checks and setup suggestions while `start.sh` lacked enough post-start reporting.
