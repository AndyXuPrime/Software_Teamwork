# Refactor API contracts and design goals from Discussion 65

## Goal

Update the existing API contract and design documentation to reflect GitHub Discussion #65: the QA module should be treated as an Agent Host that runs an iterative ReAct-style loop, uses MCP tools through an MCP Client layer, and calls AI Gateway for OpenAI-compatible model/function-calling transport. The output of this task is documentation and OpenAPI contract alignment, not backend implementation.

## What I already know

- User asked to read `https://github.com/Sakayori-Iroha-168/Software_Teamwork/discussions/65` and refactor the current API contracts and design goals.
- Discussion #65 reframes QA away from a fixed RAG orchestration pipeline.
- QA owns session/message/run persistence, Agent Loop decisions, tool-call bookkeeping, public SSE events, and user-visible reasoning steps.
- MCP Client is the execution boundary for tool discovery and tool calls. QA should not embed concrete tool implementations.
- AI Gateway should remain an internal model gateway. It should pass through OpenAI-compatible function/tool calling fields, but it should not execute MCP tools or own QA business state.
- Public frontend calls still go through gateway `/api/v1/**`; frontend must not call `qa`, `ai-gateway`, or MCP internals directly.

## Requirements

- Refactor `docs/services/qa.md` around Agent Host responsibilities, ReAct iterations, MCP tool planning/execution, tool-call events, citations, and persistence.
- Refactor `docs/services/ai-gateway.md` to state the function-calling transport contract and explicitly exclude tool execution.
- Update `docs/architecture/frontend-backend-contract.md` so frontend streaming guidance matches the Agent Host public contract direction.
- Update `docs/api/gateway.openapi.yaml` QA missing-contract metadata so the placeholder operations mention agent runs, tool calls, and MCP configuration gaps.
- Keep existing RESTful resource path rules and gateway response envelopes.
- Do not expose internal prompts, private chain of thought, provider errors, MCP tool arguments with secrets, storage keys, vector payloads, or internal URLs.

## Acceptance Criteria

- [ ] QA service documentation describes Agent Host as the primary design target.
- [ ] QA service documentation separates public gateway contract, internal QA behavior, MCP Client boundary, AI Gateway boundary, and persistence rules.
- [ ] AI Gateway documentation includes OpenAI-compatible tool/function-calling fields and makes tool execution out of scope.
- [ ] Frontend/backend contract explains that public QA streaming is a gateway SSE contract, not raw AI Gateway streaming.
- [ ] Gateway OpenAPI still parses as YAML after updates.
- [ ] OpenAPI `x-missing-contracts` no longer frames QA only as generic chat/RAG; it lists the agent/MCP gaps that must be finalized before implementation.

## Definition of Done

- Documentation changes are scoped to API contract/design artifacts.
- YAML parse check passes for `docs/api/gateway.openapi.yaml`.
- Relevant Markdown links are kept intact.
- Git diff is reviewed for accidental unrelated changes.

## Out of Scope

- Implementing QA, MCP Client, AI Gateway, or gateway code.
- Adding stable QA active paths to OpenAPI before request/response schemas are fully finalized.
- Adding frontend API clients.
- Changing package manager or build configuration.

## Technical Notes

- Research notes: [`research/discussion-65.md`](research/discussion-65.md).
- Relevant specs read:
  - `.trellis/spec/backend/api-contracts.md`
  - `.trellis/spec/backend/error-handling.md`
  - `.trellis/spec/backend/logging-guidelines.md`
  - `.trellis/spec/guides/cross-layer-thinking-guide.md`
- Existing primary files:
  - `docs/services/qa.md`
  - `docs/services/ai-gateway.md`
  - `docs/architecture/frontend-backend-contract.md`
  - `docs/api/gateway.openapi.yaml`

## Decision (ADR-lite)

**Context**: The previous QA contract described a mostly fixed sequence: intent classification, route selection, retrieval, rerank, generation, and citation post-processing. Discussion #65 requires a more general Agent Host that can iteratively decide tool calls and use MCP tools.

**Decision**: Keep public API resource-oriented through gateway, but revise QA design around agent runs, response steps, tool calls, MCP Client interaction, and sanitized SSE events. AI Gateway remains an OpenAI-compatible model transport layer and only forwards/normalizes tool/function-calling payloads.

**Consequences**: QA API details remain missing-contract placeholders until exact schemas are finalized. The docs now reserve the right extension points without committing the frontend to unstable agent/tool-call schemas.
