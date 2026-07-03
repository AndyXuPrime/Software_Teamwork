# Design

## Verification Boundary

This task is a real-environment verification and reporting task. It should avoid broad product changes. The implementation surface is limited to:

- test report and evidence files under `docs/testing/reports/2026-07-03/`;
- optional small helper scripts only if manual curl evidence would otherwise be error-prone;
- Trellis task artifacts.

The preferred validation path is Gateway/Auth/QA/AI Gateway through real local services. Internal QA endpoints may be used for the issue-required `/internal/v1/**` checks, but public Gateway paths should be used when creating authenticated sessions if practical.

## Environment Assumptions

- Infrastructure is started by `./scripts/local/dev-up.sh`.
- Backend services are started on the host by `./scripts/local/run-backend.sh`.
- The default seeded AI Gateway profiles are placeholders. A usable chat profile/provider is required before real model calls can pass.
- The local demo account is `admin / LocalDemoAdmin#12345`.
- Logs live under `.local/logs/`.

## Evidence Layout

Use the execution date `2026-07-03` for this run:

```text
docs/testing/reports/2026-07-03/
  qa-sse-behavior-test-report.md
  qa-sse-raw-success.sse
  qa-sse-raw-validation-error.sse
  qa-sse-redaction-scan.txt
  qa-sse-replay.json
  qa-sse-environment.txt
```

Only commit evidence that is safe after redaction scanning. If a file contains a token, provider key, full prompt, internal URL with sensitive query values, or provider raw body, do not commit the raw file; record a sanitized excerpt and the reason in the report.

## Test Matrix

- Connectivity: response headers and terminal stream behavior.
- Event contract: documented event set, unknown event detection, ordering.
- Streaming: count and timing of `answer.delta` events.
- Tool behavior: knowledge QA tool start/complete pairing and safe summaries.
- Redaction: raw SSE and logs scanned for forbidden patterns.
- Error paths: missing session, empty message, client disconnect/cancelled run.
- Replay: successful run events are replayable and ordered.
- Logs: QA/Gateway/AI Gateway logs do not leak sensitive material.

## Defect Handling

- Small local issue in scripts/reporting: fix in this task and document the fix.
- Major behavior defect such as one-shot answer streaming, missing tool event, unsafe payload, or broken replay: record evidence, create/link a follow-up owner issue if needed, and mark the relevant test as fail or transferred.
- Missing provider or unavailable service: mark as blocked/skipped with exact commands and residual risk.

## Rollback

This task should not change production code. Rollback is deleting the generated report/evidence files and Trellis task artifacts before commit.
