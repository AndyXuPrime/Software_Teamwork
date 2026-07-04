# Consolidate local setup into start.sh

## Goal

Make local setup and startup a single user-facing flow. Developers should be able to run `./scripts/local/start.sh` or `./scripts/local/start.sh --china` from the repository root and have the script preflight the host environment, prepare missing local artifacts in order, start already-running components idempotently, and report a clear final status.

The old split preflight helper is removed from the local startup path. The real preparation logic belongs to `start.sh`.

## Background

- The previous flow required one script to print manual commands, then `./scripts/local/start.sh` to consume prepared artifacts. Users had to copy several commands, some of which used awkward `cd ... && ... ../../...` paths.
- The split flow also let users prepare official Docker image names, then run `start.sh --china` and need the rewritten `docker.1ms.run` images.
- The new flow should keep source selection coherent: no `--china` means official sources; `--china` means Docker registry rewrite, Go proxy mirror, and Knowledge runtime mirror behavior for the current run.
- Existing `start.sh` already has useful status handling for Compose health, migrations, seed, runtime, backend process groups, and log tails; the prepare phase should use the same visible step style.
- Current environment observation: after adding `go` to PATH, Docker has only the five `docker.1ms.run` infra images in use, so there are no duplicate official Docker images to delete.
- A duplicate untracked file `services/knowledge-runtime/9b5ad71b2ce5302211f9c61530b329a4922fc6a4` has the same SHA256 as `services/knowledge-runtime/ragflow_deps/cl100k_base.tiktoken`; it is a local artifact cleanup candidate, not a Docker image.
- `github.com/pressly/goose/v3@v3.27.1` requires Go `1.25.7`, which conflicts with the local Go `1.25.1` baseline. `v3.27.0` has `go 1.25.0` in upstream `go.mod` and is the desired fixed goose version for this task.

## Requirements

- `start.sh` must preflight required host commands before preparation: Docker, Go, Python, uv when runtime is enabled, psql, curl, and any other command it will invoke.
- `start.sh` must validate Go compatibility before attempting Go builds or goose install. Go `1.25.1` and later `1.25.x` patch versions should pass; lower versions or unsupported majors should fail with an actionable message.
- `start.sh` must automatically prepare missing local artifacts in order:
  - `.env.local` from `.env.example` when missing.
  - `.local/tools/config-ctl`.
  - `.local/tools/goose` using `github.com/pressly/goose/v3/cmd/goose@v3.27.0`.
  - `.local/tools/render-ai-gateway-local-seed`.
  - `.local/bin/*` host-run service binaries.
  - Knowledge runtime `.venv` and required runtime artifacts when runtime startup is enabled.
  - Docker infrastructure images for the selected source mode.
- Preparation steps must be idempotent: if an artifact, image, running Compose service, or host process is already present, the script should skip or reuse it and continue.
- `--china` must apply consistently across preparation and startup: Docker image names, Go `GOPROXY` / `GOSUMDB`, Knowledge runtime download helper `--china`, and runtime `HF_ENDPOINT`.
- The script must stream progress by step and provide a final success/failure summary with next actions.
- Long-running prepare operations must provide ongoing feedback so the terminal does not look frozen. This applies to Docker pulls, Go module/tool downloads, Go builds, `uv sync`, and runtime artifact/model downloads. Native command progress is acceptable when present; otherwise the wrapper should print periodic elapsed-time status.
- Existing manually started host-run services must be detected via current PID files and skipped; already-running Docker services must be reused by Compose.
- The old split preflight helper must be removed from the documented flow and repository entrypoints.
- The scripts must not change the committed default source strategy: official sources by default, explicit `--china` for mirror/rewrite suggestions.
- Documentation and tests must reflect the start-only setup path and `goose@v3.27.0`.
- Local duplicate dependency artifacts discovered during this session may be removed only when proven redundant.

## Acceptance Criteria

- [ ] `./scripts/local/start.sh --china` can prepare missing local tools, service binaries, Knowledge runtime dependencies, and Docker images with coherent mainland China source settings.
- [ ] `./scripts/local/start.sh` can prepare the same categories with official defaults.
- [ ] Re-running `./scripts/local/start.sh --china` on an already-prepared/running local stack skips existing work and reports running status instead of failing.
- [ ] `goose@v3.27.0` is used consistently in scripts, docs, tests, CI migration commands, and Trellis specs.
- [ ] Startup preflight fails early with concrete messages when Go/Python/uv/Docker requirements are not met.
- [ ] Long-running downloads/builds print ongoing progress or heartbeat output while they run.
- [ ] Tests cover the prepare phase, skip behavior, source selection, and updated goose version.
- [ ] Required Docker policy checks for Docker-related script/doc changes pass.
- [ ] Local environment remains ready after changes.
- [ ] No unnecessary Docker images remain from the mistaken official-source pull path; if none exist, record that no Docker cleanup was needed.

## Notes

Out of scope:

- Changing committed default Docker image tags, Go proxy defaults, PyPI defaults, or HuggingFace mirror defaults.
- Adding business service containers back to Docker Compose.
- Reintroducing the retired standalone parser service or business-service Docker containers.
