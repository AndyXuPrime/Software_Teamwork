# Discussion 65 research notes

Source: `https://github.com/Sakayori-Iroha-168/Software_Teamwork/discussions/65`

## Access notes

- `gh` CLI is not installed in this environment.
- Public GitHub page fetching through local `curl` stalled, and no GitHub token was available in `GITHUB_TOKEN` or `GH_TOKEN`.
- The runtime web reader successfully loaded the discussion page and exposed the discussion body plus one comment.

## Extracted discussion points

- Current contract is a fixed backend orchestration flow:
  - user question,
  - QA intent recognition,
  - flow selection,
  - optional Knowledge retrieval,
  - AI Gateway answer generation,
  - citation formatting,
  - streaming return and persistence.
- The desired target is: QA runs the ReAct loop, AI Gateway handles Function Calling, and MCP Client discovers and executes tools.
- Target architecture:
  - Frontend -> Gateway -> QA Service as Agent Host.
  - QA contains ReAct Loop, MCP Client Manager, Tool Policy / permission checks, SSE output, and session/run state.
  - QA calls AI Gateway for LLM function-calling transport.
  - QA calls MCP Client, which connects to Knowledge, Document, and future MCP servers.
- ReAct execution:
  - QA creates `response_run`.
  - MCP Client fetches available tools.
  - QA converts MCP tools into Function Calling `tools`.
  - QA calls AI Gateway.
  - If the LLM returns `tool_calls`, QA validates tool, arguments, and permissions, executes `tools/call` through MCP Client, stores the tool call, adds a `role=tool` result back to messages, and loops.
  - If the LLM returns final text, QA stores the answer and citations.
- Safety:
  - ReAct Action is model-returned `tool_calls`.
  - Observation is MCP tool execution result.
  - Do not store raw model Thought.
  - Frontend only sees safe processing summaries.
- First tool set:
  - `search_knowledge`
  - `get_citation_source`
- Later report-generation tools:
  - `generate_report_outline`
  - `generate_report_text`
  - `get_generation_status`
  - `get_report_result`
  - `export_report_docx`
- Tools still call Gateway `/api/v1`; they must not bypass Gateway and directly call business services.
- AI Gateway gap:
  - Existing contract mentions `finish_reason = tool_calls` and `role = tool`.
  - Missing fields: `tools`, `tool_choice`, `parallel_tool_calls`, `assistant.tool_calls`, `tool_call_id`, streaming tool-call delta.
  - AI Gateway only passes through function-calling fields; it does not execute tools.
- SSE should retain existing event direction and add agent-state events:
  - `message.created`
  - `agent.iteration.started`
  - `tool.started`
  - `tool.completed`
  - `tool.failed`
  - `reasoning.step`
  - `answer.delta`
  - `citation.delta`
  - `answer.completed`
  - `error`
- SSE and public API must not return complete tool arguments, internal URLs, or raw document content.
- Existing tables remain:
  - `conversations`
  - `messages`
  - `response_runs`
  - `message_content_blocks`
  - `response_process_steps`
  - `response_stream_events`
  - `citations`
- New tables proposed:
  - `agent_model_invocations`
  - `agent_tool_calls`
- `response_runs` should add:
  - `current_iteration`
  - `max_iterations`
  - `termination_reason`
  - `effective_tool_names`
  - `effective_retrieval_config`
- Suggested default controls:
  - `maxIterations = 5`
  - single tool timeout 10 seconds
  - single model timeout 60 seconds
  - overall timeout 120 seconds
  - whitelist tools and MCP servers
  - validate JSON Schema on every call
  - trim available tools by user permission
  - limit tool result length and item count
  - retry read-only tool failures once
  - require idempotency keys for writes and avoid blind retries
  - prevent prompt injection in tool results from changing tool permissions
  - require HTTPS, restricted egress, and independent credentials for remote MCP to reduce SSRF and token-passthrough risks
- Comment: "harness" implementation can reference `https://github.com/shareAI-lab/learn-claude-code`.

## Contract implications

- QA owns:
  - public QA session/message contracts through gateway,
  - response runs and agent iteration state,
  - user-visible step events,
  - tool-call records and sanitized tool observations,
  - citation snapshots,
  - cancellation/recovery semantics.
- MCP Client owns:
  - MCP server registration,
  - tool discovery and schema normalization,
  - permission checks for concrete tool execution,
  - tool execution transport and timeout handling.
- AI Gateway owns:
  - model profile selection,
  - OpenAI-compatible chat completion request/response normalization,
  - streaming chunk forwarding,
  - function/tool-calling payload pass-through,
  - provider error normalization.
- AI Gateway must not:
  - execute MCP tools,
  - persist QA sessions/messages,
  - decide business permissions for tools,
  - expose provider credentials or raw provider errors.

## API design notes

- Keep frontend-facing endpoints resource-oriented:
  - sessions,
  - messages,
  - response runs,
  - events,
  - citations,
  - config versions.
- Treat tool calls as resources/events, not action paths such as `/call-tool`.
- Public SSE events should be a QA-owned protocol, not raw AI Gateway OpenAI chunks.
- Public events can expose sanitized agent progress such as `step`, `tool_call`, `tool_result`, `citation`, `message_delta`, `done`, and `error`.
- Do not expose private chain of thought. Store and stream only user-visible reasoning summaries.
- Tool arguments/results must be sanitized before persistence, logs, SSE, or public API responses.
