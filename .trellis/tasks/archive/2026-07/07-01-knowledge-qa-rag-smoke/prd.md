# Knowledge 到 QA RAG 端到端验收样例

## Goal

Issue #304 / S-029 requires a minimum, reproducible RAG acceptance sample that proves a document can move from Knowledge ingestion to retrieval, then through QA answer generation with citation evidence. The deliverable must give developers and operators one executable command or a clear runbook that identifies failures by File, Parser, Knowledge, AI Gateway, or QA stage.

## Background

- Authoritative issue: <https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/304>.
- Priority: P0, module `deploy`, labels `backend`, `deployment`, `service:knowledge`, `service:qa`, `service:ai-gateway`.
- Dependencies named by the issue: #236 document lifecycle, #84 / PR #277 `knowledge-queries`, #288 QA to AI Gateway smoke, #289 Knowledge ingestion real dependency smoke, and #305 local seed data.
- Current Knowledge facts in `docs/services/knowledge/docs/implementation.md`: ingestion worker, File handoff, Parser client, chunk persistence, Qdrant adapter, `knowledge-queries`, and existing env-gated ingestion / Gateway owner-route smokes are implemented.
- Current QA facts in `docs/services/qa/docs/implementation.md`: QA session/message/Agent Loop, AI Gateway chat client, Knowledge retrieval client, citation extraction, and existing QA -> AI Gateway smoke are implemented, but full QA + Knowledge + AI Gateway RAG smoke is still unproved.
- Current runbook facts in `docs/runbooks/local-integration.md`: root Compose is a partial local integration baseline; existing sections cover File, Knowledge ingestion, and Gateway -> Knowledge owner-route smokes, but not the minimum RAG chain required by #304.

## Requirements

1. Provide a minimum sample document, question, expected retrieval hit, and expected citation shape for a local or CI-controlled environment.
2. Exercise public or service-to-service contracts only. The smoke may create isolated test schemas, but it must not bypass File/Parser/Knowledge business APIs by directly writing production tables or vector stores as the acceptance path.
3. Verify Knowledge ingestion reaches completed state and produces chunks plus vector-search data or equivalent retrieval data.
4. Verify `knowledge-queries` returns the expected chunk, with rerank covered when explicitly configured and with a local/no-op path still accepted for default CI-like runs.
5. Verify QA can generate an answer through AI Gateway and return or persist citation summaries derived from Knowledge results.
6. Provide failure diagnostics that distinguish File, Parser, Knowledge, AI Gateway, and QA stages.
7. Keep test and runbook output sanitized: no API keys, object keys, prompt raw text, uploaded full document body, provider raw error body, service token, database URL credentials, or internal storage URL in ordinary output.
8. Synchronize the local integration documentation in `docs/runbooks/local-integration.md` or service docs.

## Acceptance Criteria

- [ ] A developer has one env-gated test command or a clear runbook command sequence for the minimum RAG acceptance sample.
- [ ] The sample includes a document fixture, question, expected hit text, and expected citation fields.
- [ ] Verification covers Knowledge ingestion completion, retrieval hit, QA answer, and citations.
- [ ] Failure output or runbook triage maps failures to File, Parser, Knowledge, AI Gateway, or QA stages.
- [ ] Default unit tests remain runnable without external providers; true provider execution remains explicit and opt-in.
- [ ] No test or docs path encourages logging secrets, object keys, prompt raw text, full document body, vector payload, provider raw error body, or internal URLs.
- [ ] Relevant docs are updated and do not claim #125 full cross-service/MCP smoke is complete.

## Out Of Scope

- Replacing #125 full cross-service/MCP smoke.
- Implementing new product capabilities beyond the smoke/runbook wiring.
- Making real provider smoke mandatory in default CI.
- Building frontend E2E coverage.
- Changing Gateway public API semantics unless current contracts are discovered to be broken.

## Evidence To Recheck Before Completion

- `services/knowledge/internal/integration/*smoke_test.go`
- `services/qa/internal/platform/modelclient/ai_gateway_smoke_test.go`
- `services/qa/internal/service/citations.go`
- `docs/runbooks/local-integration.md`
- `docs/services/knowledge/docs/implementation.md`
- `docs/services/qa/docs/implementation.md`
- `docs/services/ai-gateway/docs/provider-adapters.md`
