# S-036 Design

## Boundaries

- Code changes stay in `services/ai-gateway` unless repository evidence shows a cross-service test must change. Cross-service deliverables are request samples and troubleshooting notes, not broad Knowledge/QA/Document implementation changes.
- Public service contracts are not changed unless code already diverges from `docs/`; any non-required public doc mismatch is noted instead of widening task scope.
- The DB smoke is opt-in through `AI_GATEWAY_TEST_DATABASE_URL` so default CI remains independent of PostgreSQL.

## DB Smoke Shape

- Use a unique PostgreSQL schema per test run and drop it during cleanup.
- Apply `services/ai-gateway/migrations` before repository operations, using the pinned goose migration path or the repository-established equivalent where a direct library dependency would be excessive.
- Exercise the real `repository.PostgresRepository` through the service layer for profile lifecycle and directly through repository methods for invocation summary persistence.
- Verify encrypted credential storage by checking ciphertext does not contain the raw key and decrypting only through `service.CredentialEncryptor`.
- Verify credential rotation by updating a profile with a new key, then checking the previous credential is no longer active and the new credential is active.
- Verify usage aggregate by recording a successful invocation and checking one hourly aggregate row for caller/profile/operation.

## Runbook Shape

- Extend `docs/services/ai-gateway/docs/seed-runbook.md` because it already owns token hash, seed profile, fake provider, real provider, and downstream service examples.
- Add DB smoke command and environment variable requirements.
- Add credential rotation with `PATCH /internal/v1/model-profiles/{profileId}` and a safe verification query that never prints raw credentials.
- Make real provider coverage explicit: existing real smoke covers non-streaming chat, embedding, rerank; streaming needs the real smoke entry to set `stream=true` and `Accept: text/event-stream`.
- Add cross-service samples for:
  - Knowledge embedding and rerank.
  - QA chat.
  - Document chat and profile validation.

## Security

- Test constants may use fake secrets but must assert they are absent from responses, invocation rows, and failure output.
- Docs examples must use placeholders for service tokens and provider keys.
- Error troubleshooting should reference normalized AI Gateway errors rather than provider raw bodies.

## Rollback

- Revert the new DB smoke test file and runbook additions if the test introduces unreliable local dependencies.
- No schema changes are planned, so rollback does not require migrations.
