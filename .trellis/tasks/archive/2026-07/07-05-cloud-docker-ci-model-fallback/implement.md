# Implementation Plan

1. Update `.github/workflows/docker-deploy-checks.yml`.
   - Add a `deploy/docker/full/**` policy-trigger pattern.
   - Keep compose file selection behavior unchanged unless compose/dependency
     files change.

2. Add regression coverage for workflow detect.
   - Prefer a Python unit test that extracts/evaluates the embedded
     `actions/github-script` or a focused fixture mirroring the pattern logic.
   - Cover `deploy/docker/full/go-service.Dockerfile` and `deploy/docker/full/nginx.conf`.

3. Fix cloud model fallback.
   - Remove `AI_GATEWAY_LOCAL_CHAT_MODEL` fallback from Document and QA runtime env.
   - Make `AI_GATEWAY_LOCAL_CHAT_MODEL` empty/commented in `deploy/docker/cloud.env.example`.
   - Update docs/spec to clarify seed-only versus runtime model override values.

4. Validate.
   - Shell syntax for Docker/local scripts.
   - `python3 scripts/check_docker_policy.py`.
   - Related unittest suite including the new detect test.
   - `CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env`.
   - Root and cloud compose config; cloud render should show empty runtime model envs.
   - `actionlint` for workflow changes.
   - `git diff --check`.
