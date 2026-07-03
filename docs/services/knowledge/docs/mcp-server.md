# Knowledge MCP Server 协议与运行说明

## 1. 当前状态与边界

本文记录当前 `develop@ce0b4774` 的 Knowledge MCP 运行事实，并指出与
[`mcp-tools.md`](mcp-tools.md) 中四个只读模型工具目标契约的差距。

当前代码已经在 `services/knowledge/internal/mcp` 提供 Streamable HTTP MCP server。
它复用 `cmd/adapter` 内的 RAGFlow runtime contract adapter，通过 in-process bridge
调用 `/internal/v1/**` Knowledge 契约，不新增知识业务事实，也不让 QA 直接读取
PostgreSQL、runtime MinIO、Elasticsearch/索引后端或 provider 内部状态。

当前实现与目标文档的关键差异：

| 项 | 当前实现 | 目标/缺口 |
| --- | --- | --- |
| 启动方式 | `cmd/adapter` 在 `KNOWLEDGE_MCP_ADDR` 非空时额外启动一个 MCP HTTP server。 | 不使用 `KNOWLEDGE_MCP_PATH`；部署/QA 配置必须写完整 endpoint。 |
| endpoint | 独立监听地址的根 endpoint，例如 `http://127.0.0.1:8093`。 | 如果后续改成 `/mcp` path，需同步本文和 QA seed。 |
| 原生工具名 | 14 个原生工具，包含检索、回答、KB CRUD、文档 CRUD、chunks/content。 | #528/#529 目标只读模型工具是 `search`、`list_documents`、`get_document`、`get_chunk`，经 QA alias 后为 `knowledge__*`。 |
| 用户上下文 | MCP HTTP 不信任调用方传入的 `X-User-*`，只接受 `X-Request-Id`；业务调用使用服务端配置的固定 `KNOWLEDGE_MCP_USER_ID`/roles/permissions。 | 若要按最终用户做知识库可见性，需 #505 后续收敛 QA discovery 与上下文传递设计。 |
| QA 默认路径 | QA 仍保留内置 `search_knowledge` retrieval tool；远程 MCP 通过 `mcp_servers` 或环境 bootstrap 显式注册。 | 四个 `knowledge__*` 默认白名单和 citation 识别仍需收敛验证。 |

## 2. 工具目录

当前 `tools/list` 的原生工具顺序由 `ToolCatalog()` 固定：

| 原生工具名 | 读写 | 用途 | 备注 |
| --- | --- | --- | --- |
| `search_knowledge` | 读 | 调 `/internal/v1/knowledge-queries`，返回 ranked chunks。 | 当前 QA citation 主路径仍识别这个名称。 |
| `answer_from_knowledge` | 读 + 模型调用 | 检索后通过 AI Gateway chat 合成回答。 | 仅在 `KNOWLEDGE_AI_GATEWAY_URL` 配置时可用。 |
| `list_knowledge_bases` | 读 | 列出可见知识库。 | 走 adapter list。 |
| `get_knowledge_base` | 读 | 读取单个知识库。 | 走 adapter get。 |
| `create_knowledge_base` | 写 | 创建知识库。 | 需要配置的 MCP caller 具备 `knowledge:write`。 |
| `update_knowledge_base` | 写 | 更新知识库。 | 需要 `knowledge:write`。 |
| `delete_knowledge_base` | 写 | 软删除知识库。 | 需要 `knowledge:write`。 |
| `list_documents` | 读 | 分页列出知识库文档。 | 原生名已与目标只读工具一致。 |
| `get_document` | 读 | 读取文档详情。 | 原生名已与目标只读工具一致。 |
| `create_document` | 写 | 上传 base64 文档内容并触发 runtime 处理。 | 需要 `knowledge:write`。 |
| `update_document` | 写 | 更新文档 tags。 | 需要 `knowledge:write`。 |
| `delete_document` | 写 | 软删除文档。 | 需要 `knowledge:write`。 |
| `list_document_chunks` | 读 | 分页列出文档 chunks。 | 目标四工具中的 `get_chunk` 尚未按 chunk ID 单点读取。 |
| `get_document_content` | 读 | 读取原始文档内容并 base64 返回。 | 输出不得直接暴露 runtime 内部对象引用。 |

QA 注册 alias 后，模型侧工具名为 `<alias>__<原生工具名>`。例如 alias 为
`knowledge` 时，当前检索工具会变成 `knowledge__search_knowledge`，不是
`mcp-tools.md` 目标契约中的 `knowledge__search`。

## 3. 传输、鉴权与上下文

- 传输协议为 MCP Streamable HTTP。
- 当前 MCP server 绑定在独立 `http.Server` 上，不挂载到主 Knowledge HTTP handler。
- MCP HTTP 请求必须携带 `X-Service-Token`，值与 `KNOWLEDGE_SERVICE_TOKEN` 或
  `INTERNAL_SERVICE_TOKEN` 一致；缺失或错误时返回 `401`。
- 当前不支持自定义 token header；`X-Service-Token` 是代码内固定校验 header。
- 服务端忽略调用方伪造的 `X-User-Id`、`X-User-Roles` 和 `X-User-Permissions`。
  这些业务上下文来自启动配置：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `KNOWLEDGE_MCP_ADDR` | 空 | 非空时启动 MCP server，例如 `127.0.0.1:8093`。 |
| `KNOWLEDGE_MCP_USER_ID` | `knowledge_mcp_service` | MCP bridge 调用 adapter 时使用的固定用户 ID。 |
| `KNOWLEDGE_MCP_PERMISSIONS` | `knowledge:read` | 固定权限集合；写工具需要包含 `knowledge:write`。 |
| `KNOWLEDGE_MCP_ROLES` | 空 | 固定角色集合。 |
| `KNOWLEDGE_SERVICE_TOKEN` / `INTERNAL_SERVICE_TOKEN` | 无 | MCP HTTP 入口校验和 bridge 调 adapter 的服务令牌。 |
| `KNOWLEDGE_AI_GATEWAY_URL` | 空 | 配置后启用 `answer_from_knowledge` 的 AI Gateway chat client。 |
| `KNOWLEDGE_AI_GATEWAY_SERVICE_TOKEN` | `INTERNAL_SERVICE_TOKEN` fallback | `answer_from_knowledge` 调 AI Gateway 的服务令牌。 |

`X-Request-Id` 会从 MCP HTTP 请求透传到 adapter；缺失时 server 生成 `mcp_*`
request id。不得把 service token、provider credential、runtime URL、SQL、object key
或内部索引 payload 写入工具输出、SSE、前端响应或日志摘要。

## 4. QA 接入路径

QA 侧远程 MCP 的正式配置路径是数据库 `mcp_servers`，环境变量
`MCP_TRANSPORT`/`MCP_SERVER_*` 只作为 bootstrap 或本地调试入口。QA Manager 会：

1. 读取 runtime configuration 中启用的 MCP server。
2. 用 `streamable_http` 连接 endpoint。
3. 调 `tools/list`。
4. 通过 alias 把原生工具名前缀化为 `<alias>__<tool>`。
5. 与内置工具合并；重复工具名会导致 discovery 失败。
6. 用 agent tool policy 按 `enabledToolNames` 过滤。

本地调试 Knowledge MCP 可使用：

```powershell
cd D:\PROJECTS\Software_Teamwork\services\knowledge
$env:KNOWLEDGE_MCP_ADDR = "127.0.0.1:8093"
$env:KNOWLEDGE_MCP_USER_ID = "knowledge_mcp_service"
$env:KNOWLEDGE_MCP_PERMISSIONS = "knowledge:read"
$env:KNOWLEDGE_SERVICE_TOKEN = "local-dev-internal-service-token-change-me"
$env:VENDOR_RUNTIME_URL = "http://127.0.0.1:9380"
$env:VENDOR_RUNTIME_SERVICE_TOKEN = "local-dev-internal-service-token-change-me"
go run ./cmd/adapter
```

QA 环境 bootstrap 示例：

```powershell
$env:MCP_TRANSPORT = "streamable_http"
$env:MCP_SERVER_ALIAS = "knowledge"
$env:MCP_SERVER_URL = "http://127.0.0.1:8093"
$env:MCP_SERVER_TOKEN = "local-dev-internal-service-token-change-me"
$env:MCP_SERVER_TOKEN_HEADER = "X-Service-Token"
$env:MCP_TOOL_TIMEOUT = "30s"
```

当前仓库尚未提供 Knowledge MCP 的默认 QA seed；默认 RAG smoke 仍要求模型使用内置
`search_knowledge` 工具。把 Knowledge MCP 接入默认 QA 配置时，必须同步更新
`enabledToolNames`、citation 工具名识别和 #125 smoke。

## 5. Agent 调用建议

当前默认 QA RAG 主路径仍是内置 `search_knowledge`。显式注册 Knowledge MCP 后：

- 只读问答优先使用 alias 后的检索工具，例如 `knowledge__search_knowledge`。
- 浏览库/文档时使用 `knowledge__list_knowledge_bases`、`knowledge__list_documents`
  和 `knowledge__get_document`。
- 需要上下文展开时，当前实现可用 `knowledge__list_document_chunks` 或
  `knowledge__get_document_content`；目标 `knowledge__get_chunk` 仍待实现/收敛。
- 写工具默认不应进入普通 Agent 白名单；只有配置了 `knowledge:write` 的受控管理场景
  才能启用 create/update/delete 工具。
- `answer_from_knowledge` 会调用 AI Gateway chat，属于模型调用增强工具；缺少
  `KNOWLEDGE_AI_GATEWAY_URL` 时会返回安全错误，不应作为默认可用能力宣传。

## 6. 联调与排障

联调至少覆盖：

- 无 `X-Service-Token` 调 MCP endpoint 返回 `401`。
- `tools/list` 返回 `ToolCatalog()` 中的 14 个原生工具。
- `search_knowledge` 能通过 adapter bridge 调用 `knowledge-queries`。
- 调用方伪造 `X-User-Permissions: knowledge:write` 时，写工具仍按服务端固定权限拒绝。
- `answer_from_knowledge` 未配置 `KNOWLEDGE_AI_GATEWAY_URL` 时返回安全错误。
- QA alias 前缀后不会与内置 `search_knowledge` 或其他 MCP server 工具重名。

常见排障顺序：

1. 检查 `KNOWLEDGE_MCP_ADDR` 是否非空，以及 adapter 日志是否出现
   `knowledge MCP server listening`。
2. 确认 QA 的 `MCP_SERVER_URL` 指向独立 MCP endpoint 根地址，而不是
   `KNOWLEDGE_HTTP_ADDR` 或 `/mcp` path。
3. 对齐 QA `MCP_SERVER_TOKEN` 与 Knowledge `KNOWLEDGE_SERVICE_TOKEN` /
   `INTERNAL_SERVICE_TOKEN`。
4. 检查 `MCP_SERVER_TOKEN_HEADER=X-Service-Token`。
5. 检查 `MCP_SERVER_ALIAS` 是否符合 `^[a-z0-9_]{2,32}$`，以及 Agent config
   `enabledToolNames` 是否包含 alias 后工具名。
6. 用 `X-Request-Id` 关联 QA 与 Knowledge adapter 日志。

## 7. 版本与兼容性

当前实现是 `knowledge-mcp` `0.1.0`。新增只读工具或新增可选字段属于向后兼容；
删除/改名工具、改变原生工具名、改变鉴权 header、收紧已有字段范围或改变错误语义
属于破坏性变更，必须同步：

- 本文；
- [`mcp-tools.md`](mcp-tools.md)；
- QA tool policy/default config；
- citation 工具名识别；
- MCP client/platform tests；
- #125 或对应 env-gated smoke runbook。
