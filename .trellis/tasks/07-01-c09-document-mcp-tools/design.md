# C-09 Design

## Architecture

C-09 should add a Document-owned MCP tool adapter that sits above existing
Document service APIs. The adapter owns tool schemas, argument validation,
structured tool output, error mapping, and operation-log recording. It must not
own report-generation business rules.

Preferred shape for this PR:

```text
QA MCP client / approved MCP caller
  -> Document MCP tool adapter
      -> existing Document service layer
      -> repository / worker / File / AI Gateway through existing boundaries
```

The implementation can be packaged as an in-process Go `agent.ToolClient` style
provider, an MCP server entrypoint, or both if the existing project wiring makes
that cheap. The functional contract is tool discovery plus tool call execution
with sanitized JSON output.

## Tool Mapping

| Tool | Document capability | Result |
| --- | --- | --- |
| `generate_report_outline` | create `outline_generation` report job | `requestId`, `jobId`, `reportId`, `status`, `progress`, `error` |
| `regenerate_report_outline` | create `outline_regeneration` report job | same as above |
| `generate_report_text` | create `content_generation` report job | same as above |
| `regenerate_report_text` | create `content_regeneration` report job | same as above |
| `regenerate_report_section` | create `section_regeneration` report job with `target.sectionId` | same as above plus `sectionId` |
| `get_generation_status` | read report job | job status/progress/error summary |
| `get_template_schema` | read report template structure | template ID and safe structure fields |
| `export_report_docx` | create DOCX report file/job | `reportFileId`, `jobId`, `status`, content path only as public resource path |
| `get_report_result` | read report and optional latest file metadata | report status and safe business IDs |

## Safety Boundary

- Tool layer must not import repository internals for direct data access.
- Tool layer should accept a trusted `RequestContext` equivalent with user ID,
  roles, permissions, and request ID.
- Operation logs must use `requestSource=mcp` and `toolName=<tool>`.
- Parameter summaries should include IDs, enum values, boolean flags, and counts
  only. User prompts, requirements text, context text, raw material payloads, and
  provider options must be summarized by presence/length/count, not persisted.
- Errors must map to stable Document codes: `validation_error`,
  `unauthorized`, `forbidden`, `not_found`, `conflict`, `dependency_error`,
  `internal_error`, and `unsupported`.

## Compatibility

- Preserve existing REST routes and response DTOs.
- Preserve existing operation-log sanitization and build on it rather than
  duplicating sanitizers.
- Preserve current basic DOCX behavior. Do not present rich DOCX as available.
- Preserve current QA MCP prefixing model. If QA consumes the tool provider,
  `document__<tool>` may be the model-visible name while the Document-owned
  schema remains `<tool>`.

## Rollback

The change should be additive. Rollback can remove the new tool adapter/server
registration without changing existing report generation, report file, or QA
MCP client behavior.
