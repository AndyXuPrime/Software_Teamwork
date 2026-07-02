# Design

## Approach

Add an opt-in RAG smoke asset that reuses the repository's existing service boundaries and local integration conventions. The default path should be deterministic with local hashing embedding and a controlled/fake AI Gateway provider when available; a real-provider path remains manual and explicitly configured.

The smoke must be useful even when a developer cannot run the whole Compose stack. Therefore the implementation should pair executable checks with a runbook section that lists exact environment variables, startup order, expected output, and stage-specific failure triage.

## Boundary Map

1. File stage: Knowledge uploads the sample document through File Service handoff. Failure indicates File Service readiness, service token mismatch, or object persistence issues.
2. Parser stage: Knowledge ingestion invokes Parser Service for the sample document. Failure indicates Parser readiness, token mismatch, or parser runtime errors.
3. Knowledge ingestion stage: Knowledge worker records ready document, succeeded processing job, chunks, embedding metadata, and Qdrant/local vector state.
4. Knowledge retrieval stage: `knowledge-queries` searches the ready document and hydrates the expected chunk. Optional rerank goes through AI Gateway when rerank env is configured.
5. AI Gateway stage: QA sends chat completion requests only through AI Gateway. Fake/stub or real provider setup is explicit.
6. QA stage: QA creates a session/message run, invokes the Knowledge search tool, returns an answer, and extracts citation summaries without exposing raw tool results publicly.

## Implementation Shape

- Prefer adding or extending env-gated Go integration tests over a shell script when the test can reuse service-local helpers and assertions.
- Keep helper output intentionally terse and sanitized. Fatal messages should list missing env keys or failing stage names, not secret values.
- Put reusable sample data in code or a small fixture file with a unique phrase, expected question, and expected citation fields.
- Update `docs/runbooks/local-integration.md` with the exact default and real-provider commands.
- Update implementation-status docs for Knowledge and QA if the smoke materially changes the current known gap list.

## Compatibility

- Ordinary `go test ./...` must skip the new smoke unless an explicit env gate is set.
- The smoke may share helpers with existing Knowledge integration tests when package boundaries allow it.
- Compose docs must preserve the existing statement that root local Compose is partial and does not replace #125.
- No new shared Go package should be introduced for one smoke.

## Risk Controls

- Use run-scoped IDs, schemas, and Qdrant collection names when the test writes data.
- Add cleanup for File objects, Qdrant collections, and PostgreSQL schemas where the test creates them.
- Keep provider errors normalized and discard response bodies in diagnostics.
- Prefer service-level clients and HTTP APIs; do not import another service's `internal` packages across service boundaries.
