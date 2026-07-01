# Issue 305 local integration seed baseline

## Goal

Provide a unified, idempotent local integration seed baseline for issue #305 (`S-030`) so local developers can start the root Compose stack and see consistent demo login data plus minimal Auth, AI Gateway, Knowledge, Document, and QA resources.

## Background

- Issue #305 is assigned to `Jackeyliu37` and targets branch `Special/chore/local-seed-baseline`.
- The issue depends on #286 and #289, and blocks #304.
- The root local environment is documented in `deploy/README.md` and `docs/runbooks/local-integration.md`.
- Existing seed entrypoints are `seed-local` (`deploy/seeds/001-local-demo-seed.sql`) and optional `seed-local-ai` (`deploy/seeds/002-ai-gateway-model-profiles.sql`).
- Existing local credentials are `LOCAL_ADMIN_USERNAME=admin` and `LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345` in `deploy/.env.example`.
- The issue owner added an explicit acceptance requirement: after local seed, the demo admin can log in through Gateway `POST /api/v1/sessions`, has role `admin`, has either `system:admin` or admin runtime config permissions, and can pass an admin route preflight such as parser-config/model-profile.

## Requirements

- Keep root local seed commands deterministic and documented.
- Seed a local admin user with password credentials, `admin` role, and admin runtime config permissions available through the existing role-permission model.
- Keep AI Gateway model profiles and fake/stub provider credentials as local placeholders only.
- Seed minimal Knowledge resources: the existing demo knowledge base plus one deterministic sample document and chunk suitable for local listing/query inspection without requiring private files.
- Seed minimal Document resources: report types, default template linkage, one local material, one sample report, outline, sections, and an operation/event record where useful for UI/API inspection.
- Seed minimal QA resources: one local demo conversation with completed user/assistant messages and content blocks owned by the demo admin.
- Ensure repeated seed runs update deterministic records instead of creating uncontrolled duplicates.
- Document which seed commands are CI-safe versus local/manual, default accounts, sample resource IDs, cleanup/reset, password hash source, and password rotation.
- Do not add real provider API keys, real secrets, private documents, object storage keys, or environment-specific local paths.

## Acceptance Criteria

- [ ] A local developer can follow documented commands to run seed entrypoints and obtain consistent accounts/resources.
- [ ] Running seed SQL multiple times keeps deterministic IDs and does not create duplicate demo records.
- [ ] Seed data covers Auth, AI Gateway, Knowledge, Document, and QA minimal local integration cases.
- [ ] Documentation lists environment variables, default account, sample resource IDs, local/manual versus CI-safe guidance, cleanup/reset, and password hash rotation.
- [ ] Admin login verification is documented: Gateway `POST /api/v1/sessions` with the seeded admin succeeds, response contains `admin` role and admin runtime config permissions, and a seeded admin token can reach a Gateway admin route preflight.
- [ ] Verification includes automated guard(s) for seed contract, idempotency markers, and no real secrets/private content.
- [ ] No `.local` files, local machine paths, real secrets, or private documents are committed.

## Out Of Scope

- Do not implement #304 RAG end-to-end smoke.
- Do not replace #125 cross-service smoke or claim one-command full E2E coverage.
- Do not require optional AI Gateway real-provider calls in ordinary CI.
- Do not seed real provider API keys or production credentials.
