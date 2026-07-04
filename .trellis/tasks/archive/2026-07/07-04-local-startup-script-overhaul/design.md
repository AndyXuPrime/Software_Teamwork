# Local Startup Script Overhaul Design

## Boundaries

The implementation changes repository tooling under `scripts/`, local startup
documentation, and related tests. It does not change service APIs, migrations,
seed data semantics, or the root Compose service boundary.

## Source Selection Model

Default mode keeps the rendered profile values and official fallbacks:

- Docker images: Compose defaults or rendered official image variables.
- Go modules: `https://proxy.golang.org,direct` and `sum.golang.org`.
- uv: `https://pypi.org/simple`.
- Runtime model/artifact mirrors: no forced mirror.

`--china` is a current-process overlay:

- It exports explicit Docker image variables before Compose config/pull.
- It exports Go/uv/runtime mirror variables only for the command process.
- It does not write local env files.
- After applying overlays, it uses the same stage runner as default mode.

## Script Structure

Add a small shared shell library under `scripts/local/lib/`:

- `common.sh`: status logging, truthy parsing, URL normalization, `NO_PROXY`.
- `process.sh`: process-group pid helpers for host-run startup/stop scripts.
- `knowledge-runtime.sh`: runtime config generation and HF mirror handling.

Keep public entrypoints in place:

- `dev-up.sh`
- `run-backend.sh`
- `stop-backend.sh`
- `run-knowledge-runtime-api.sh`
- `start-knowledge-runtime-worker.sh`
- `run-knowledge-parse-stack.sh`

Avoid sourcing helpers from generated `.local` paths; helpers are committed
scripts and must work from a clean checkout.

## Compatibility

CI and docs currently reference `scripts/run_issue_125_smoke.sh` and
`scripts/run_issue_352_smoke.sh`. Rename only if compatibility wrappers remain.
The first implementation pass may keep the file names and document their smoke
role instead of renaming them.

Remove `scripts/check_powershell_encoding.ps1` from the active scripts surface
because no current docs, CI, or tests reference it.

## Performance

Avoid expensive work when not needed:

- Render the config profile once per entrypoint.
- Run Knowledge runtime dependency sync only when the script starts a runtime
  component that needs it.
- Use `uv sync --check` before running full sync where that pattern already
  exists.
- Do Docker image preflight only for selected `--china` images, and keep it
  bounded by a short timeout.

## Failure Handling

Docker pull failures should include:

- selected mode (`official` or `china`);
- service/image list;
- current source-selection hint;
- a specific Elasticsearch hint when the selected China image is known to be
  problematic.

Knowledge PDF E2E should fail before network calls if the PDF path does not
exist.

## Rollback

The change is tooling-only. Rollback is restoring the previous scripts and docs.
Keep public entrypoints compatible so a partial rollback does not strand local
users.
