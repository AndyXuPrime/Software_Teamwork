# Make local startup downloads opt-in

## Goal

Make the local scripts clean and explicit: startup uses already-present local
tools, images, binaries, and runtime environments; network/download work is not
hidden in the scripts. The public check helper only reports the current
environment and prints official/mainland China setup suggestions.

## Requirements

- Public local shell entrypoints should be reduced to a small set:
  check, start, stop, and clean.
- Initialization/start scripts must not run `go mod download`, `go run
  module@version`, `uv sync`, Knowledge runtime artifact downloads, or forced
  Docker pulls by default.
- Hard-to-download dependencies should have documented manual commands and
  check-script suggestions, including official and mainland China variants.
- Knowledge runtime dependency and artifact downloads must remain manual, even
  when `--china` is passed.
- `--china` must keep printing mainland mirror/proxy suggestions without
  rewriting committed config or `.env.local`.
- The root `.env.example -> .env.local` path remains the local secret/env
  template; `deploy/` remains Docker Compose infrastructure only.
- Old public startup/download shell entrypoints should be removed or replaced by
  the smaller public contract.
- README, deploy docs, runbooks, and Trellis specs must describe the new
  default and the manual setup suggestion path.
- The investigation of the `.env` entrypoint change must identify the exact
  commits and author.

## Acceptance Criteria

- [ ] `check.sh` only reports current environment readiness and setup
  suggestions, and does not download, build, pull images, run `uv sync`, or edit
  `.env.local`.
- [ ] `start.sh` does not run `uv sync`, `go mod download`,
  `go run module@version`, runtime artifact downloads, or Docker pulls.
- [ ] Check suggestions cover Go tools/service binaries, infrastructure images,
  Knowledge runtime API/worker environments, and Knowledge runtime artifacts.
- [ ] `--china` prints mirror suggestions and documents runtime mirror startup
  behavior without changing committed defaults or `.env.local`.
- [ ] Tests and contract checkers cover the default no-download behavior, the
  smaller public script set, and the manual setup suggestion path.
- [ ] Startup docs and specs are consistent about root `.env.example` /
  `.env.local`, local preconfigured environments, and opt-in preparation.

## Notes

- User specifically asked not to spend time on the official proxy chain; focus
  on clean mirror/no-mirror behavior and avoiding unnecessary downloads.
