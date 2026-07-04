# Local startup script overhaul

## Goal

Simplify and harden the local startup scripts so contributors can start the
repository with official sources by default, opt into mainland China mirrors
with `--china`, and get the same startup flow after source selection. The
scripts should be easier to maintain, faster to run, and free of misleading or
unused helper paths.

## Background

- The current local startup path is `./scripts/local/dev-up.sh` followed by
  `./scripts/local/run-backend.sh`; shutdown uses
  `./scripts/local/stop-backend.sh`.
- The default profile renders official Docker/PyPI/Go sources.
- `dev-up.sh --china` currently rewrites Docker image variables and also sets
  Go/uv/runtime download mirrors for the current process.
- DaoCloud currently returns `403 Forbidden` for
  `docker.m.daocloud.io/docker.elastic.co/elasticsearch/elasticsearch:8.15.3`,
  while the official Elastic image is reachable and
  `docker.m.daocloud.io/elasticsearch:8.15.3` has a manifest.
- `scripts/local/run-knowledge-parse-stack.sh`,
  `scripts/local/run-knowledge-runtime-api.sh`,
  `scripts/local/start-knowledge-runtime-worker.sh`, and
  `scripts/local/watch-knowledge-runtime-worker-idle.sh` duplicate local
  runtime helpers such as URL normalization, `NO_PROXY`, process-group checks,
  HF mirror handling, and generated runtime config handling.
- `scripts/check_powershell_encoding.ps1` has no repository references and is
  not part of the local startup or CI path.
- `scripts/local/knowledge-pdf-e2e.py` is useful as a manual smoke but its
  default fixture `DL_T_673-1999.pdf` is not present in the current worktree.

## Requirements

- R1. Preserve the documented public startup commands:
  `./scripts/local/dev-up.sh`, `./scripts/local/run-backend.sh`, and
  `./scripts/local/stop-backend.sh`.
- R2. Keep official sources as the default for Docker images, Go modules, uv,
  and runtime artifact downloads.
- R3. Keep `--china` as an explicit current-process-only source selection. It
  must not edit `.env.local`, `.local/config/*.env`, `config/`, or committed
  defaults.
- R4. After source selection, default and `--china` modes must run the same
  startup stages in the same order.
- R5. Fix the current `--china` Docker image failure mode for Elasticsearch or
  fail with a precise service/image diagnostic before the rest of startup work
  begins.
- R6. Reduce duplicated local script logic by introducing shared shell helpers
  for common startup behavior instead of adding more copy-pasted functions.
- R7. Keep scripts fast by avoiding unnecessary repeated profile renders,
  repeated dependency checks, and expensive runtime setup when the target state
  is already satisfied.
- R8. Remove, rename, or demote scripts that are not part of a documented or CI
  path, while keeping compatibility wrappers where CI or docs still reference
  them.
- R9. Keep Docker policy boundaries intact: root Compose remains infra-only,
  pull-only, and pinned; business services stay host-run.
- R10. Update docs and tests to reflect the simplified script set and source
  selection behavior.

## Acceptance Criteria

- [ ] `./scripts/local/dev-up.sh` defaults to official sources and does not
      activate third-party mirrors without `--china` or local untracked
      overrides.
- [ ] `./scripts/local/dev-up.sh --china` uses explicit current-process image
      overrides and then follows the same validation, pull, infra health,
      MinIO init, migration, and seed flow as default mode.
- [ ] China Docker image selection no longer points Elasticsearch at the
      DaoCloud path that returns `403 Forbidden`, or the script surfaces a
      targeted preflight failure that names the image and fallback options.
- [ ] Shared local helper code replaces repeated shell functions across the
      Knowledge runtime scripts without changing their documented behavior.
- [ ] `scripts/check_powershell_encoding.ps1` is removed or moved out of the
      active scripts surface, with no broken references.
- [ ] `knowledge-pdf-e2e.py` no longer silently assumes a missing fixture; it
      either requires an explicit PDF path or prints a clear missing-fixture
      error.
- [ ] Issue-number smoke script entrypoints are either retained as thin
      compatibility wrappers or renamed with docs updated; no CI reference is
      broken.
- [ ] Required checks pass: shell syntax for local scripts, Docker policy,
      Docker environment unit tests, local startup script tests, local seed
      contract tests, gateway verifier tests as affected, Compose config, and
      `git diff --check`.

## Out Of Scope

- Changing business service behavior or API contracts.
- Replacing Docker Compose with another process manager.
- Making third-party registries committed defaults.
- Solving every external mirror outage; scripts should make source choice and
  failure causes clear.
