# AI Gateway 实现说明

版本：v0.1
日期：2026-06-29
范围：`services/ai-gateway/` 当前实现、契约对齐、缺口和后续实现约束

## 1. 文档定位

本文档描述 `ai-gateway` 当前实现状态和后续实现约束。它只补充服务 README、OpenAPI、架构和技术选型文档，不覆盖这些上游契约。

权威来源：

| 类型 | 权威来源 | 本文档关系 |
| --- | --- | --- |
| 服务公开说明 | `docs/services/ai-gateway/README.md` | 只能补充，不能覆盖 |
| 服务 OpenAPI | `docs/services/ai-gateway/api/openapi.yaml` | 只能跟随，不能另起契约 |
| Gateway 公开契约 | `docs/services/gateway/api/openapi.yaml` | 前端稳定契约以 gateway 为准 |
| 服务边界 | `docs/architecture/service-boundaries.md` | 必须遵守 |
| 技术基线 | `docs/architecture/technology-decisions.md` | 必须跟随 |
| 代码实现 | `services/ai-gateway/` | 本文档记录当前状态和差距 |

凡是本文档与上表文件冲突，以上游文件为准；发现冲突时，在“文档与实现出入”中记录并生成回写或实现任务。

## 2. 当前结论

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| 文档状态 | active | README、数据模型和内部 OpenAPI 存在。 |
| 代码状态 | partial | Go service、PostgreSQL repository、model profile CRUD、credential encryption、service-token auth、chat completions 和 provider invocation 记录已实现；embeddings/rerankings routes 返回 501。 |
| 契约对齐 | partial | `/internal/v1/model-profiles/**` 和 `/internal/v1/chat/completions` 基本实现；embeddings/rerankings 在 OpenAPI 中声明但未实现。 |
| 数据持久化 | postgres | runtime 使用 PostgreSQL 和 AES-GCM 加密列保存 provider credentials。 |
| 测试状态 | partial | config/service/http/middleware/provider tests 覆盖 profile、安全、非流式 chat、流式 chat、脱敏和失败记录；缺真实 provider/DB smoke。 |
| 建议动作 | 补实现 / 回写文档 | 实现 embedding/rerank provider adapters，补真实 provider 和 migration smoke。 |

## 3. 已实现

| 能力 | 代码位置 | 契约来源 | 验证方式 | 备注 |
| --- | --- | --- | --- | --- |
| 健康/就绪检查 | `services/ai-gateway/internal/http/server.go` | AI Gateway OpenAPI | `cd services/ai-gateway && go test ./...` | ready 检查 repo/profile 状态。 |
| model profile CRUD | `internal/http/server.go`、`internal/service/service.go` | AI Gateway OpenAPI / Gateway admin model profiles | HTTP/service tests | 支持 list/create/get/update/delete。 |
| provider credential 加密 | `internal/service/crypto.go` | AI Gateway README | `TestCreateModelProfileDoesNotReturnAPIKey` | 响应只返回 `apiKeyConfigured`。 |
| sensitive default parameter validation | `internal/service/service.go` | AI Gateway README | service tests | 拒绝敏感参数进入 profile defaults。 |
| service-token/caller-service auth | `internal/middleware/auth.go`、`internal/http/server.go` | AI Gateway README | auth/http tests | 要求 token hash 和 `X-Caller-Service`。 |
| OpenAI-compatible chat completions | `internal/http/server.go`、`internal/service/chat.go`、`internal/provider/chat_http.go` | AI Gateway OpenAPI | fake provider HTTP tests | 支持非流式和流式请求，转发 function-calling 字段。 |
| provider invocation 记录 | `migrations/0004`-`0005`、`internal/service/chat.go`、`internal/repository/postgres.go` | 数据模型 / observability | service/http tests | 记录 provider、model、status、token usage、duration 和错误归一摘要，不保存 prompt/API key。 |
| PostgreSQL schema/repository | `migrations/0001`-`0005`、`internal/repository/postgres.go` | 数据模型 | repository path via service tests | runtime 使用 `pgx/v5`。 |

## 4. 未实现

| 缺口 | 文档来源 | 影响范围 | 建议任务 |
| --- | --- | --- | --- |
| Embeddings 未实现 | AI Gateway OpenAPI / Knowledge requirements | Knowledge / vector indexing | 待确认：实现 embedding provider adapter。 |
| Rerankings 未实现 | AI Gateway OpenAPI / Knowledge retrieval | Knowledge / QA | 待确认：实现 rerank adapter。 |
| Provider error normalization 只覆盖 chat | AI Gateway README | API / observability | 待确认：补真实 provider smoke 和 embedding/rerank fake tests。 |
| 配额、指标未落地 | 数据模型 / 技术基线 | observability / admin | 待确认：后续增强。 |

## 5. 文档与实现出入

| 出入点 | 文档要求 | 当前实现 | 风险 | 建议处理 |
| --- | --- | --- | --- | --- |
| README 状态记录 | README 曾称当前文档不落地 `services/ai-gateway/` 代码 | 实际已有 Go module、migrations、repository、profile API；本次已回写 README | 后续若重复写实现状态，容易再次漂移 | README 只链接 implementation，当前状态在本文维护。 |
| Model invocation API | OpenAPI 声明 chat/embeddings/rerankings | chat completions 已实现；embeddings/rerankings 仍由 `handleModelInvocationNotImplemented` 返回 OpenAI-style 501 | QA/Document 的 chat 路径可接 AI Gateway；Knowledge embedding/rerank 闭环仍阻塞 | 继续实现 embedding/rerank adapters，并补跨服务 smoke。 |
| Service token 格式 | README 要求 `X-Service-Token`，配置使用 token hashes | 代码读取 `AI_GATEWAY_SERVICE_TOKEN_HASHES`，不接受明文配置 | 安全部署需要先生成 hash | README 保留 hash 格式说明，补生成示例。 |
| pgx version | 技术基线早期记录 pgx/v4 | AI Gateway 使用 `pgx/v5` | 版本统一策略不清 | 更新技术基线为混用现状。 |

## 6. MVP / mock / memory backend / 占位

| 项目 | 当前用途 | 退出条件 | 关联任务 |
| --- | --- | --- | --- |
| embedding/rerank invocation 501 | 占住契约并返回 OpenAI-style error | embedding/rerank provider adapters 落地 | 待确认 |
| memory repository in tests | profile service/http 单元测试 | 保留测试用 | 无 |

## 7. 运行与配置

| 项目 | 当前状态 | 缺口 |
| --- | --- | --- |
| 启动命令 | `cd services/ai-gateway && go run ./cmd/server` | 需要 PostgreSQL、token hashes、credential encryption key。 |
| 环境变量 | `AI_GATEWAY_DATABASE_URL`、`AI_GATEWAY_SERVICE_TOKEN_HASHES`、`AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF`、`AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY`、timeouts/body size | 需要 provider-specific runtime smoke。 |
| PostgreSQL / migration | `migrations/0001`-`0005`，`sqlc.yaml`，runtime repository | 需要 migration CI/smoke。 |
| Redis / queue | 当前不使用 | 后续配额/熔断按需。 |
| Object storage / vector store / AI provider | provider profiles 已存储；chat provider HTTP adapter 已实现 | embedding/rerank adapters 和真实 provider smoke。 |

## 8. 测试与验证

| 验证项 | 命令或步骤 | 当前结果 | 缺口 |
| --- | --- | --- | --- |
| 单元测试 | `cd services/ai-gateway && go test ./...` | pass（本次执行） | 不覆盖真实 provider。 |
| 集成测试 | goose apply + profile CRUD against DB | missing | 需要 PostgreSQL。 |
| 契约测试 | HTTP tests for profile、chat completion、streaming、embedding/rerank 501 routes | partial | 未从 OpenAPI 自动校验完整 schema。 |
| 手工 smoke | Gateway admin model profile CRUD + chat invocation | not run | 需要 gateway/auth/Redis/ai-gateway/provider。 |

## 9. 建议任务

| 任务 | 类型 | 优先级 | 依据 | 说明 |
| --- | --- | --- | --- | --- |
| 实现 embeddings/rerankings | 新任务 | P0 | Knowledge 依赖 | 支持 Knowledge indexing/retrieval。 |
| 增加 provider smoke / contract tests | 新任务 | P1 | 错误归一和脱敏要求 | 在 fake tests 基础上覆盖真实 provider 配置、timeout、401/429/5xx、stream cancel。 |
| 补 token hash 生成文档 | 回写文档 | P1 | 部署可操作性 | 避免明文 token 配置误用。 |

## 10. 最近检查记录

| 日期 | 检查人/工具 | 代码基准 | 结论 |
| --- | --- | --- | --- |
| 2026-06-29 | Codex after rebase | `808c589` + working tree | AI Gateway 已实现模型配置、凭据安全、chat completions 和 provider invocation 记录；embedding/rerank 仍未实现，是 Knowledge/RAG 闭环阻塞点。 |
| 2026-06-29 | Codex goal | `eddf917` + working tree | AI Gateway 已实现模型配置和凭据安全基础；当时模型调用三大 endpoint 仍未实现，后续 `develop` 已补 chat completions。 |
