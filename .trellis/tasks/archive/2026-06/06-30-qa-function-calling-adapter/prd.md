# QA AI Gateway Function Calling Adapter

## Background

Issue #90 (`B-04`) requires the QA service to call AI Gateway through the
OpenAI-compatible `/internal/v1/chat/completions` contract and support function
calling transport without direct provider access.

Authoritative references:

- `docs/services/qa/README.md`
- `docs/services/ai-gateway/README.md`
- `docs/services/ai-gateway/api/openapi.yaml`
- `.trellis/spec/backend/api-contracts.md`
- `.trellis/spec/backend/error-handling.md`
- `.trellis/spec/backend/logging-guidelines.md`
- `.trellis/spec/backend/quality-guidelines.md`

## Scope

Enhance the QA model client and request context propagation so QA can use AI
Gateway function calling transport.

The implementation must:

- Send OpenAI-compatible `tools`, `tool_choice`, `parallel_tool_calls`, messages,
  model, profile, and max token fields to AI Gateway.
- Preserve QA ownership of MCP tool policy and execution; AI Gateway only
  transports function calling fields.
- Forward internal `X-Service-Token`, `X-Caller-Service`, `X-Request-Id`, and
  user-triggered `X-User-Id` headers.
- Parse non-streaming `assistant.tool_calls` into the agent loop representation.
- Parse streaming `delta.tool_calls` chunks into indexed internal tool-call
  state, preserving incremental function name and argument semantics.
- Normalize AI Gateway failures to stable project errors without exposing raw
  provider bodies, prompts, API keys, or provider tokens.
- Keep model invocation persistence on the existing #89 response-run path with
  provider/profile/model, finish reason, token summary, status, and latency.

## Acceptance Criteria

- QA calls AI Gateway only; it does not direct-connect to OpenAI, SiliconFlow, or
  local providers.
- Function-calling request payloads include the documented transport fields.
- Tool-call responses can be converted into `agent.ToolCall` values.
- Streaming tool-call deltas can be merged by `index` into complete internal
  tool calls.
- User and request context headers are propagated to AI Gateway.
- Gateway validation and dependency failures are returned through stable
  `validation_error` / `dependency_error` classification.
- Tests prove secret-safe error handling and avoid full prompt/provider body
  leakage.

## Out of Scope

- AI Gateway provider implementation.
- MCP tool permission decisions or execution changes beyond the existing QA
  agent loop.
- Frontend changes.
