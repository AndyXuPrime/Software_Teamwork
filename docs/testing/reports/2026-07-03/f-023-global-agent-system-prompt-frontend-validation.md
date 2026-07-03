# F-023 Global Agent System Prompt Frontend Validation

Date: 2026-07-03
Branch: `Frontend/feat/global-agent-system-prompt`
Scope: admin Agent prompt page, generated Gateway types, QA config version publish flow.

## Commands

| Command                            | Result | Notes                                                     |
| ---------------------------------- | ------ | --------------------------------------------------------- |
| `bun run --cwd apps/web check`     | pass   | typecheck, test typecheck, lint, and format check passed. |
| `bun run --cwd apps/web build`     | pass   | Vite build passed; bundle size warning only.              |
| `bun run --cwd apps/web test:unit` | pass   | 27 files, 87 tests passed.                                |

## Browser / API Validation

| Acceptance Item                                                    | Result       | Evidence                                                                                                                                                                                             |
| ------------------------------------------------------------------ | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Admin with `qa:settings:read` can view current prompt and metadata | pass         | `admin-prompts-page-viewport.png` shows menu, `/admin/prompts`, version, created time, created by, active status, full prompt.                                                                       |
| Admin with `qa:settings:write` can publish new versions            | pass         | Published version 2 with temporary smoke rule, then version 3 restored the original prompt.                                                                                                          |
| Standard user cannot enter prompt page or read prompt API          | pass         | Browser route shows forbidden page; Gateway `GET /api/v1/qa-config-versions/current` returned 403 for standard user.                                                                                 |
| Global impact copy is visible                                      | pass         | Page and confirm dialog show: all users, next question effective, in-flight answers unaffected.                                                                                                      |
| Empty and oversized prompts are blocked before submit              | pass         | Browser validation showed empty prompt disables publish with `系统提示词不能为空`; 20001 bytes disables publish with `系统提示词不能超过 20000 UTF-8 bytes`.                                         |
| Prompt-only publish preserves other QA fields                      | pass         | DB comparison showed versions 2 and 3 preserved `top_k`, `use_rerank`, `max_iterations`, and `enabled_tool_names` from version 1.                                                                    |
| Save failure keeps draft                                           | pass         | Covered by `qa-system-prompt.test.tsx`; failed mutation keeps draft and displays requestId.                                                                                                          |
| Success refreshes current version/cache                            | pass         | UI updated to version 2 after publish and version 3 after restore.                                                                                                                                   |
| Chat/SSE/logs do not expose full system prompt                     | pass         | Chat UI did not show full prompt or smoke rule; browser console had no prompt leak; QA logs had no match for full prompt/smoke rule.                                                                 |
| Next question uses new config version                              | partial pass | Response run for `F-023 prompt smoke check` referenced QA config version 2. The run failed with `dependency_error`, so model text could not prove smoke-answer compliance in this local environment. |

## Screenshot Evidence

Stored under `.local/evidence/F-023/`:

- `admin-prompts-page-viewport.png`
- `admin-prompts-published.png`
- `admin-prompts-restored.png`
- `standard-user-forbidden.png`

## Database Evidence

Latest QA config versions after restore:

```text
version 3 active=true  prompt_bytes=125 has_smoke_rule=false created_by=usr_local_admin
version 2 active=false prompt_bytes=237 has_smoke_rule=true  created_by=usr_local_admin
version 1 active=false prompt_bytes=125 has_smoke_rule=false created_by=system
```

Field preservation:

```text
version 2: top_k=true, rerank=true, max_iterations=true, tools=true
version 3: top_k=true, rerank=true, max_iterations=true, tools=true
```

Run snapshot:

```text
question: F-023 prompt smoke check
qa_config_version: 2
status: failed
stop_reason: dependency_error
request_id: req_eb34a636ff0e6d22
```

## Residual Risk

The real QA run loaded the newly published QA config version, but AI/dependency execution returned `dependency_error`; therefore the browser could not observe the temporary prompt rule in the assistant answer. Current active prompt was restored to version 3 without the smoke rule.
