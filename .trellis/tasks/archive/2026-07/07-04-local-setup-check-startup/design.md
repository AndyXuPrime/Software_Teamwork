# Consolidate local setup into start.sh Design

## Boundaries

`scripts/local/start.sh` becomes the standard entrypoint for both preparation and startup. It may build local Go tools/services, install `goose@v3.27.0`, run `uv sync`, download Knowledge runtime artifacts, and pull Docker infrastructure images when those artifacts/images are missing.

The committed source policy stays unchanged: official defaults in config and Compose, explicit `--china` for the current run. `start.sh --china` may mutate generated `.local/config/dev.env` after rendering, but must not write third-party defaults into `.env.local`, `config/base.yaml`, or `deploy/docker-compose.yml`.

The old split preflight helper is no longer part of the workflow and should be deleted rather than kept as a compatibility wrapper.

## Approach

- Add a prepare phase before existing startup phases:
  - preflight host commands and versions
  - ensure `.env.local`
  - build/install local tools
  - render config and apply `--china` image rewrites when requested
  - prepare Docker infrastructure images
  - prepare Knowledge runtime `.venv` and artifacts if runtime is enabled
  - build host-run service binaries unless `--infra-only`
- Use existing `run_step` / `log_*` style for every prepare step.
- Wrap long prepare commands with a heartbeat helper. The helper should run the command in the foreground-equivalent path, emit a periodic elapsed-time line while the child is still alive, return the child exit code, and avoid hiding the child command's own stdout/stderr.
- Implement artifact preparation as small idempotent functions:
  - `ensure_file` / `ensure_executable` style checks for local files
  - `docker image inspect` before `docker pull`
  - current PID file process checks before starting host services remain as-is
- Use `go -C <module-dir> build -o "$PWD/.local/..." <package>` for per-module Go builds.
- Use `GOBIN="$TOOLS_DIR"` and fixed `github.com/pressly/goose/v3/cmd/goose@v3.27.0`.
- For `--china`, pass `GOPROXY=https://goproxy.cn,direct`, `GOSUMDB=sum.golang.google.cn`, Docker image rewrite constants, runtime helper `--china`, and runtime `HF_ENDPOINT` in the current process only.

## Compatibility

Prepared environments should continue to pass, with prepare steps printing skip messages. Existing `.env.local` is preserved. Existing running Compose services and host-run process groups are reused.

The project policy remains official-by-default. The changes must not introduce active third-party defaults in `config/base.yaml`, `.env.example`, or Compose.

Changing goose from `v3.27.1` to `v3.27.0` is a project-wide baseline change and must update docs, tests, CI migration workflow, and Trellis specs together.

## Rollback

The affected behavior spans startup scripts, docs, tests, CI references to goose, and Trellis specs. If automatic preparation causes issues, keep the `goose@v3.27.0` baseline only if separately accepted; otherwise revert script/docs/spec changes together to avoid workflow drift.
