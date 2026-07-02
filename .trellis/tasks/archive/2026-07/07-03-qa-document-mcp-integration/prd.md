# QA Agent Document MCP integration

## Goal

Complete Issue #511 by making the host-run QA service discover the Document
Streamable HTTP MCP server automatically in the standard local integration
workflow, and document the Document MCP tool contracts and Agent call flow.

## Background

- The current root Compose policy permits infrastructure containers only;
  QA and Document run on the host through `scripts/local/run-backend.sh`.
- Document already serves a token-protected, stateless MCP endpoint at `/mcp`.
- QA loads enabled database `mcp_servers` records and otherwise falls back to
  one environment bootstrap MCP server. A seeded record currently suppresses
  the environment bootstrap entirely, including its token.
- Issue #510 / PR #521 owns `generate_report_from_content`; it is not present in
  the current `develop` baseline and must not be copied into this task.

## Requirements

1. Add the canonical host-run Document MCP bootstrap variables to
   `deploy/.env.example`, using the existing `MCP_SERVER_*` names consumed by QA.
2. Add an idempotent local seed for `mcp_servers.alias=document` and run it from
   `scripts/local/dev-up.sh` after QA migrations.
3. Preserve an explicitly disabled database record, but merge a matching
   environment bootstrap token into an enabled seed record that has no stored
   token. Append the bootstrap server when the database contains only unrelated
   aliases.
4. Add repeatable contract checks for the new seed and runtime merge behavior.
5. Update local integration documentation and add a Document MCP contract page
   covering transport/authentication, all current tools, input parameters,
   common output, async workflow, QA aliasing, safety, and the #510 dependency.
6. Do not add QA or Document business services to root Docker Compose.

## Acceptance Criteria

- [ ] `dev-up.sh` applies an idempotent seed that creates an enabled
      `mcp_servers` row with alias `document`, transport `streamable_http`, and
      endpoint `http://localhost:8085/mcp`.
- [ ] QA combines the seeded record with the matching environment bootstrap
      credential and can initialize/list Document MCP tools after restart.
- [ ] A disabled database record is not re-enabled by environment bootstrap.
- [ ] Local seed contract tests and QA service tests cover the behavior.
- [ ] The runbook includes discovery, call, failure/degraded-mode, and
      verification steps without implying business services run in Compose.
- [ ] Document MCP documentation lists each tool and its exact parameters, and
      distinguishes the current nine tools from #510's pending content tool.
- [ ] Docker policy checks, Compose config, QA tests, and documentation checks
      required by the changed surface pass.

## Out of Scope

- Implementing or cherry-picking `generate_report_from_content` from #510.
- Modifying the ReAct loop or MCP composite tool architecture.
- Frontend work, production deployment, or adding business-service containers.
- New retry, timeout, or fallback policies for Document tool calls.
