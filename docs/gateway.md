# Gateway 服务规划

本文档定义 `gateway` 服务在项目初期的职责边界和基础契约。目标是让前端只依赖一个稳定入口，同时让 `auth`、`file`、`knowledge`、`qa`、`document` 等服务可以按清晰边界并行开发。

## 设计原则

- `gateway` 是面向前端的后端统一入口，不是业务大单体。
- 前端只调用 `gateway` 暴露的 `/api/v1/**` 接口，不直接调用内部服务。
- `gateway` 通过 HTTP/REST 调用内部服务，不 import 其他服务的 Go `internal/` 包。
- 领域业务规则尽量留在拥有该领域数据和流程的服务中。
- 跨服务聚合接口必须有明确前端场景，不能把所有服务编排都放进 `gateway`。
- OpenAPI 契约先行，代码实现必须跟随契约变更。

## Gateway 应负责

| 能力 | 说明 |
| --- | --- |
| Public API surface | 暴露前端使用的 `/api/v1/**` HTTP API。 |
| Routing | 将请求转发到 `auth`、`file`、`knowledge`、`qa`、`document` 等内部服务。 |
| Auth context | 校验或委托校验用户身份，并向下游传递用户、角色、权限和 request id。 |
| Response contract | 对前端保持统一成功响应、分页响应和错误响应结构。 |
| Request correlation | 生成或透传 `X-Request-Id`，并要求下游服务保留该 request id。 |
| Cross-service aggregation | 仅为明确页面场景提供少量聚合读接口，例如管理后台概览。 |
| Streaming entrypoint | 为问答和报告生成暴露 SSE/流式入口，业务生成逻辑仍归领域服务。 |
| Edge policy | 集中处理 CORS、基础请求头、请求大小原则、健康检查和公开 API 命名。 |

## Gateway 不应负责

| 领域 | 归属服务 | Gateway 不做什么 |
| --- | --- | --- |
| 用户、密码、会话、角色权限数据 | `auth` | 不保存密码，不维护用户表，不实现 RBAC 持久化。 |
| 文件对象、文件元数据、对象存储协调 | `file` | 不直接操作 MinIO，不生成业务 object key。 |
| 知识库、文档切片、向量索引、检索策略 | `knowledge` | 不执行切片、嵌入、Qdrant 查询或重排序。 |
| 问答、意图识别、RAG、LLM 调用 | `qa` | 不拼 prompt，不执行 RAG pipeline，不保存对话业务状态。 |
| 报告大纲、章节生成、DOCX 导出 | `document` | 不生成报告内容，不操作报告模板业务规则。 |
| 服务数据库迁移 | 各领域服务 | 不拥有其他服务的 migrations。 |

## Public API 命名

第一版公开 API 使用统一版本前缀：

```text
/api/v1
```

健康检查接口不带版本前缀：

```text
/healthz
/readyz
```

推荐路径分组：

| Gateway path | 初始 owner | 说明 |
| --- | --- | --- |
| `/api/v1/auth/**` | `auth` | 注册、登录、登出、当前用户。 |
| `/api/v1/users/me` | `auth` | 当前用户资料，可作为 `/auth/me` 的别名或后续稳定入口。 |
| `/api/v1/knowledge-bases/**` | `knowledge` | 知识库 CRUD、策略配置、知识库内文档列表。 |
| `/api/v1/documents/**` | `file` / `knowledge` | 文件上传和原文件归 `file`，切片与索引状态归 `knowledge`。 |
| `/api/v1/search` | `knowledge` | 面向用户的知识检索。 |
| `/api/v1/chat/**` | `qa` | 会话、消息、流式问答。 |
| `/api/v1/reports/**` | `document` | 报告记录、大纲、章节生成、导出和下载。 |
| `/api/v1/admin/overview` | Aggregated | 管理后台概览，可由 gateway 聚合多个服务的只读指标。 |

当某个 endpoint 涉及两个服务时，文档必须显式标注 workflow owner。默认规则是：拥有核心业务状态的服务拥有流程，gateway 只做入口和上下文传递。

## 认证与上下文传递

认证机制后续可选择 token 或 cookie，但公开契约先约定上下文传递规则。

前端请求：

- 登录类接口不要求认证。
- 业务接口必须携带认证凭据。
- 前端不直接设置用户身份 header，用户身份由 gateway 认证后注入。

Gateway 调用下游服务时应传递：

| Header | 说明 |
| --- | --- |
| `X-Request-Id` | 贯穿一次前端请求的 request id。 |
| `X-User-Id` | 已认证用户 ID。 |
| `X-User-Roles` | 逗号分隔的角色列表。 |
| `X-User-Permissions` | 逗号分隔的权限列表，字段细节由 auth 契约细化。 |
| `X-Forwarded-For` | 原始客户端地址链。 |
| `X-Forwarded-Proto` | 原始协议。 |

下游服务仍需在自己的边界做权限校验，不能只依赖前端传参。

## 响应约定

成功响应使用稳定 JSON envelope：

```json
{
  "data": {},
  "requestId": "req_123"
}
```

分页响应：

```json
{
  "data": [],
  "page": {
    "page": 1,
    "pageSize": 20,
    "total": 100
  },
  "requestId": "req_123"
}
```

错误响应遵循后端规范：

```json
{
  "error": {
    "code": "validation_error",
    "message": "request validation failed",
    "requestId": "req_123",
    "fields": {
      "name": "is required"
    }
  }
}
```

错误码初始集合：

| Code | HTTP status |
| --- | --- |
| `validation_error` | `400` |
| `unauthorized` | `401` |
| `forbidden` | `403` |
| `not_found` | `404` |
| `conflict` | `409` |
| `rate_limited` | `429` |
| `dependency_error` | `502` |
| `internal_error` | `500` |

## SSE 与流式接口

问答和报告生成需要流式输出。初期约定：

- Gateway 暴露 `text/event-stream` endpoint。
- 业务生成逻辑由 `qa` 或 `document` 服务负责。
- Gateway 负责传递认证上下文和 request id。
- 流式事件结构先在 OpenAPI 中标记，完整断线重连策略后续补充。

推荐事件类型：

| Event | 用途 |
| --- | --- |
| `progress` | 进度或阶段变化。 |
| `message` | 文本增量。 |
| `citation` | 问答引用来源增量。 |
| `done` | 流式任务完成。 |
| `error` | 流式任务失败。 |

## 健康检查

| Endpoint | 说明 |
| --- | --- |
| `GET /healthz` | 进程存活检查，只表示 gateway 进程可响应。 |
| `GET /readyz` | 就绪检查，后续可包含关键下游依赖状态。 |

## 后续扩展

本轮只定义基础契约包。以下内容后续单独细化：

- 下游服务完整内部 API 索引。
- 超时、重试、熔断和断线重连策略。
- API 版本兼容策略。
- 限流、审计和安全事件记录。
- 多端 BFF 拆分条件。
