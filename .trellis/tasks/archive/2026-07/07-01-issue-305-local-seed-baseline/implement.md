# Issue 305 implementation plan

## Plan

- [ ] Write a failing seed contract test under `scripts/tests/`.
- [ ] Add the minimal checker implementation under `scripts/`.
- [ ] Extend `deploy/seeds/001-local-demo-seed.sql` with Knowledge sample document/chunk, Document sample material/report/outline/sections, and QA sample conversation/messages.
- [ ] Add a targeted cleanup/reset seed SQL if it reduces manual cleanup ambiguity.
- [ ] Update `deploy/README.md` with commands, default credentials, sample resource IDs, idempotency, admin login verification, password hash source/rotation, local/manual versus CI-safe seed guidance, and cleanup.
- [ ] Update `docs/runbooks/local-integration.md` only where it needs to reflect the broader seed coverage without claiming full E2E.
- [ ] Run `python -m unittest scripts.tests.test_local_seed_contract`.
- [ ] Run relevant existing script tests.
- [ ] Run Docker Compose config parsing for default and optional AI profile if Docker is available.
- [ ] Run `git diff --check`.
- [ ] Perform Trellis quality check, then commit with a Conventional Commits message.

## Validation Commands

```powershell
python -m unittest scripts.tests.test_local_seed_contract
python -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_verify_gateway_active_api
python scripts/verify_local_seed_contract.py
docker compose --env-file deploy/.env.example -f deploy/docker-compose.yml config --quiet
docker compose --env-file deploy/.env.example -f deploy/docker-compose.yml --profile ai config --quiet
git diff --check
```

Optional manual local smoke after containers are running:

```powershell
cd deploy
docker compose --env-file .env.example up -d --build gateway
$body = @{ username = "admin"; password = "LocalDemoAdmin#12345" } | ConvertTo-Json
$session = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/sessions -ContentType "application/json" -Body $body
$headers = @{ Authorization = "Bearer $($session.data.session.accessToken)" }
Invoke-RestMethod -Uri http://localhost:8080/api/v1/admin/parser-configs -Headers $headers
```

## Review Risks

- SQL table shape drift can break seed inserts; mitigate with schema-aware inserts based on current migrations.
- Review may reject over-claiming; docs must distinguish static seed availability from full cross-service E2E smoke.
- Real secrets must not appear; fake AI credential material must be explicitly marked as local placeholder data.
- `deploy/README.md` already contains seed text; keep updates concise and avoid duplicating facts across sections.
