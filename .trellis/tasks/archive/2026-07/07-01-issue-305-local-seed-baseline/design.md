# Issue 305 design

## Scope And Boundaries

This task extends the existing root local/demo seed baseline. It does not add a production deployment seed path and does not convert optional env-gated smoke tests into required CI checks.

Seed data stays in `deploy/seeds/`:

- `001-local-demo-seed.sql` owns default Auth, Knowledge, Document, and QA records.
- `002-ai-gateway-model-profiles.sql` continues to own optional AI Gateway placeholder profiles and fake encrypted credentials.
- A cleanup/reset SQL entrypoint may be added under `deploy/seeds/` if the docs need a targeted reset path beyond `docker compose down -v`.

Docs stay in existing authority locations:

- `deploy/README.md` is the primary operator entrypoint for seed commands, resources, and reset instructions.
- `docs/runbooks/local-integration.md` records how this seed fits the broader local integration state and avoids over-claiming full E2E smoke coverage.

## Data Design

Use deterministic IDs so local docs, smoke tests, and future #304 can reference the same resources:

- Auth:
  - user `usr_local_admin`, username `admin`.
  - credential `cred_local_admin_password`.
  - role assignment to existing `admin` role only.
  - The `admin` role already has `admin:model-profile:write` and `admin:parser-config:write`; no `super_admin` escalation is required for #305.
- Knowledge:
  - knowledge base `kb_local_demo`.
  - sample document `doc_local_demo_seed` under that KB.
  - sample chunk `chunk_local_demo_seed_001` with `local_hashing` metadata and no private source content.
- Document:
  - report types remain `summer_peak_inspection` and `coal_inventory_audit`.
  - default template references point to the existing placeholder template IDs from `services/document/migrations/0003_seed_initial_report_defaults.sql`.
  - sample material and report use deterministic UUIDs.
  - sample outline/section rows use manual/completed states, so no worker or AI provider is triggered.
- QA:
  - conversation, user message, assistant message, and content blocks use deterministic UUIDs.
  - The sample is completed-state read data only, so it does not start an Agent Run.
- AI Gateway:
  - keep profiles `default-chat`, `default-embedding`, and `default-rerank`.
  - keep fake encrypted credentials as placeholders; docs must state they are not usable provider secrets.

All inserts must use `ON CONFLICT` or equivalent guarded updates. Updates should avoid modifying unrelated user-created local records.

## Verification Design

Add a Python stdlib seed contract checker under `scripts/` with unit tests in `scripts/tests/` before changing seed/docs. The checker should validate:

- expected deterministic resource IDs appear in seed SQL and docs;
- `001-local-demo-seed.sql` includes Auth, Knowledge, Document, and QA baseline sections;
- `002-ai-gateway-model-profiles.sql` includes the three AI Gateway placeholder profiles and fake credential markers;
- seed SQL uses idempotent conflict handling for deterministic inserts;
- documentation includes default credentials, password hash source/rotation, cleanup/reset, and admin login/admin-route verification commands;
- forbidden real-secret/private-content patterns are absent.

Run targeted tests plus compose config parsing. If Docker is unavailable or Compose config is slow because of local environment constraints, record the exact limitation instead of claiming it passed.

## Operational Notes

- Prefer least privilege: keep the local admin on role `admin`; rely on existing runtime config permissions rather than assigning `super_admin`.
- Do not document local machine paths in PR text or committed docs.
- Avoid editing Chinese docs unless needed; when editing UTF-8 docs, preserve encoding and verify no mojibake is introduced.
- Cleanup should offer both full volume reset and targeted local-demo deletion where safe.
