# QA SSE Real-Environment Behavior Verification

## Goal

Complete GitHub issue #498 / T-013 by running, or explicitly blocking with evidence, a real-environment verification of QA SSE behavior and public payload redaction.

The user value is to prove that the QA streaming path works outside fake-backed unit tests: Gateway/Auth/QA/AI Gateway and, where available, Knowledge/tool dependencies must produce stable SSE events, replayable records, safe public payloads, and clear failure evidence.

## Confirmed Facts

- Source issue: https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/498
- Suggested branch: `Test/test/qa-sse-behavior-verification`
- Module scope: `qa / gateway`
- Required report path: `docs/testing/reports/YYYY-MM-DD/qa-sse-behavior-test-report.md`
- Dependencies #495, #496, #497, and #524 are closed as of planning.
- This is a test task. It must produce execution evidence, not only a checklist.
- QA SSE fake-backed tests already cover contract names, heartbeat/replay boundaries, tool/citation payload safety, and redaction behavior.
- Real provider / full cross-service behavior is env-gated and depends on a usable AI Gateway chat profile.
- Static code evidence to verify at runtime:
  - `services/qa/internal/http/server.go:316` sets the SSE response path and heartbeat.
  - `services/qa/internal/service/agent/loop.go:112` runs the agent loop and emits model/tool/reasoning events.
  - `services/qa/internal/service/qa.go:385` creates runs and persists stream events.
  - `services/qa/internal/service/qa.go:702` currently emits `answer.delta` after the final assistant content is assembled; the real SSE test must verify whether public answer streaming is incremental or effectively one-shot.

## Requirements

1. Verify basic SSE connectivity for `POST /internal/v1/qa-sessions/{sessionId}/messages` with `Accept: text/event-stream`.
2. Record whether the real stream contains the documented event types:
   `message.created`, `agent.iteration.started`, `reasoning.step`, `tool.started`, `tool.completed`, `answer.delta`, `citation.delta`, `answer.completed`, and `heartbeat`.
3. Verify event sequencing:
   event `id:` or payload `seq` / service sequence must be strictly increasing for persisted non-heartbeat events.
4. Verify answer streaming behavior:
   `answer.delta` should not be a single final full-answer push if the real provider path is expected to stream.
5. Verify tool behavior under `mode: knowledge_qa` when the environment can trigger a knowledge/tool call:
   `tool.started` and `tool.completed` must pair by `toolCallId`, and public arguments/results must be summaries only.
6. Run safety redaction scanning against the raw SSE capture and relevant logs for API keys, bearer tokens, secrets, prompt text, internal URLs, database URLs, object storage keys, provider raw error bodies, raw vector payloads, and full document text.
7. Verify error handling:
   invalid session id returns 404; empty message returns 400 `validation_error`; client disconnect records the response run as `cancelled` when a run is actually started.
8. Verify replay:
   capture a successful `responseRunId`, call `GET /internal/v1/qa-sessions/{sessionId}/events?responseRunId={runId}`, and compare replayed non-heartbeat events with the original stream.
9. Check `.local/logs/qa.log` and related service logs for unexpected `panic` or non-redacted `ERROR` details.
10. If the local environment cannot satisfy the real-provider or full-service prerequisites, record the exact missing dependency, commands attempted, observed output, residual risk, and classify the final result as environment-blocked rather than passing.

## Acceptance Criteria

- [ ] A report exists at `docs/testing/reports/YYYY-MM-DD/qa-sse-behavior-test-report.md`.
- [ ] The report uses the repository testing template structure and records issue #498 / T-013.
- [ ] The report lists commands, environment snapshot, request IDs, evidence files, and pass/fail/blocked/skipped status for every test item.
- [ ] Raw SSE capture and redaction scan output are committed or referenced from the report without leaking secrets.
- [ ] Real SSE event types, answer streaming continuity, and tool-call behavior are either verified in a real environment or explicitly marked blocked with the missing dependency.
- [ ] Redaction scan is run for every captured raw SSE/log artifact, with each hit classified.
- [ ] Any major product/backend defect discovered is not fixed in this test task unless it is small and local; instead it is recorded and, if actionable, transferred to a new owner issue.
- [ ] The PR body links the report and summarizes commands, environment result, remaining risk, and `Closes #498`.

## Out Of Scope

- Frontend rendering and UI behavior; those were covered by #495, #496, and #497.
- Model answer quality or retrieval ranking quality.
- Refactoring QA, Gateway, Knowledge, or AI Gateway production behavior unless a small test harness or documentation correction is required.
- Adding this real-provider verification to default CI.
- Treating fake-backed unit tests, env-gated smoke tests, and real-provider manual evidence as the same result.

## Open Questions

None blocking. If no usable AI Gateway chat profile/provider is available locally, proceed by recording the environment blocker in the report as required by #498.
