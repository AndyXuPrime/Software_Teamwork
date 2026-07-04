# Local Startup Script Overhaul Implementation Plan

## Checklist

- [ ] Add shared shell helper files under `scripts/local/lib/`.
- [ ] Refactor `dev-up.sh` to use a single source-selection step and one common
      stage flow for default and `--china`.
- [ ] Fix China Elasticsearch image selection or add a bounded preflight with a
      precise diagnostic.
- [ ] Refactor Knowledge runtime entrypoints to use shared helper logic.
- [ ] Remove or demote `scripts/check_powershell_encoding.ps1`.
- [ ] Make `knowledge-pdf-e2e.py` require an existing PDF path before smoke
      execution.
- [ ] Update tests for source selection, helper behavior, and removed script
      surface.
- [ ] Update README/runbooks if command names or diagnostic behavior changes.
- [ ] Run validation.

## Validation Commands

```bash
bash -n scripts/local/*.sh scripts/config/load-profile.sh
python3 -m unittest \
  scripts.tests.test_check_docker_policy \
  scripts.tests.test_check_docker_environment \
  scripts.tests.test_local_dev_up_script \
  scripts.tests.test_local_seed_contract \
  scripts.tests.test_verify_gateway_active_api
python3 scripts/check_docker_policy.py
python3 scripts/check_docker_environment.py --skip-network --clean-env
PATH="/usr/local/go/bin:$PATH" CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --services
git diff --check
```

Network validation when available:

```bash
python3 scripts/check_docker_environment.py --profile china --clean-env --timeout 30
```

## Risk Points

- `dev-up.sh` is the highest-risk file because it owns migrations and seed.
- Knowledge runtime scripts have repeated behavior; refactor in small pieces and
  keep shell syntax checks green after each step.
- CI path filters include `scripts/local/**`, so helper file additions must be
  covered by existing Docker deploy checks.
