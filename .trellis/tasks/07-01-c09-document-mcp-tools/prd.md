# C-09 Document MCP tools

## Goal

Implement the first stable Document MCP tool surface for report generation so QA
or another approved MCP caller can trigger and inspect report generation through
safe, structured tool calls without bypassing the Document service boundary.

This task maps the C-009 issue to the current `develop` implementation. It
should expose report generation, status, template-schema, export, and result
lookup capabilities as MCP tools, while returning only safe summaries and
business IDs.

## Confirmed Facts

- GitHub issue: #105 `[C-009] Document MCP 工具能力与安全摘要`.
- Issue state: open, assigned to `heavenllrt`, status `Ready`.
- Suggested branch: `PrimeTeam/feat/document-mcp-tools`.
- Dependencies #91, #100, #101, #103, #151, and #259 are closed.
- Blocker #125 is still open and covers MCP/cross-service end-to-end smoke
  scripts; C-09 can implement and test service-local behavior, but final
  cross-service smoke remains a documented risk until #125 lands.
- `docs/services/document/docs/implementation.md` records Document MCP tools as
  not implemented.
- `services/document` already implements report jobs, attempts, events, basic AI
  outline/content generation, report files/content, basic DOCX export, settings,
  statistics, and operation logs.
- `services/qa` already has an MCP client, prefixed tool namespace support, a
  composite `ToolClient`, and sanitized `agent_tool_calls` summaries.
- Existing Document operation logs include `request_source`, `tool_name`,
  `parameter_summary_json`, `operation_result`, `error_message`, and metadata.

## Requirements

- Provide these Document MCP tool capabilities:
  - `generate_report_outline`
  - `regenerate_report_outline`
  - `generate_report_text`
  - `regenerate_report_text`
  - `regenerate_report_section`
  - `get_generation_status`
  - `get_template_schema`
  - `export_report_docx`
  - `get_report_result`
- Each tool must define a stable JSON input schema with required and optional
  fields.
- Tool calls must use existing Document business services or the documented
  gateway `/api/v1/**` contract shape; they must not directly access the
  database, File object storage, MinIO object keys, Qdrant, or model providers.
- Tool results must return only safe summaries, business resource IDs, status,
  progress, and stable error codes.
- Tool results must not expose secrets, provider raw errors, prompt text, full
  private parameters, File Service internal IDs, storage paths, object keys,
  signed URLs, SQL errors, or internal service URLs.
- Generation tools must create/return `ReportJob` resources using existing
  Document job semantics:
  - outline generation -> `outline_generation`
  - outline regeneration -> `outline_regeneration`
  - text generation -> `content_generation`
  - text regeneration -> `content_regeneration`
  - section regeneration -> `section_regeneration`
- `export_report_docx` must create a report file/job through the existing basic
  DOCX export path and must not claim Pandoc/LibreOffice rich DOCX support.
- Capabilities that depend on future rich DOCX behavior must return stable
  unsupported/not-ready semantics rather than fake success.
- Each MCP tool invocation must record a safe Document operation-log summary:
  tool name, request ID, source `mcp`, parameter summary, status/result, and
  stable error code where applicable.
- High-risk delete, overwrite, bulk rebuild, direct file download, and provider
  configuration operations are out of scope and must not be exposed as tools.

## Acceptance Criteria

- [ ] Tool names and input schemas are stable and covered by tests.
- [ ] Tool calls reuse Document service boundaries and do not introduce direct
  database/File/MinIO/Qdrant/provider access in the tool layer.
- [ ] `get_generation_status`, `get_report_result`, and `export_report_docx`
  cover success, not-ready/conflict, forbidden/no permission, and dependency
  failure in tests.
- [ ] Tools depending on incomplete rich DOCX behavior return stable
  unsupported/not-ready errors instead of success.
- [ ] Tool results and operation logs are sanitized and do not include secrets,
  raw prompts, internal storage paths, File internal IDs, provider raw errors, or
  full private parameters.
- [ ] MCP invocation summaries can be queried through Document operation logs
  using `requestSource=mcp` and `toolName=<tool>`.
- [ ] Documentation records deployment/call shape, permission boundary, current
  supported capabilities, and #125 end-to-end smoke risk.
- [ ] Service-local validation passes: `cd services/document && go test ./...`
  and `cd services/document && go build ./cmd/server`.

## Out Of Scope

- Implementing QA Agent behavior changes beyond consuming normal MCP tools.
- Replacing #101 generation orchestration.
- Replacing #160 rich DOCX/Pandoc/LibreOffice tooling.
- Adding delete, overwrite, bulk rebuild, direct download, template upload, or
  material upload MCP tools.
- Exposing raw MCP schemas through the frontend Gateway public contract.

## Open Questions

- None blocking planning. The implementation should use the minimum stable
  in-process Document tool provider/server shape that fits the existing QA MCP
  `ToolClient` contract, and document any deployment limitation for follow-up
  review.
