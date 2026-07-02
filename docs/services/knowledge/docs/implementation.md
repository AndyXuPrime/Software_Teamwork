# Knowledge Service 实现说明

版本：v0.9
日期：2026-07-01
范围：`services/knowledge/` 当前实现、契约对齐、缺口和后续实现约束

## 1. 文档定位

本文档描述 `knowledge` 当前实现状态和后续实现约束。它只补充服务 README、OpenAPI、架构和技术选型文档，不覆盖这些上游契约。

权威来源：

| 类型 | 权威来源 | 本文档关系 |
| --- | --- | --- |
| 服务公开说明 | `docs/services/knowledge/README.md` | 只能补充，不能覆盖 |
| 服务 OpenAPI | `docs/services/knowledge/api/internal.openapi.yaml`；`services/knowledge/api/openapi.yaml` 是实现本地路由副本 | 只能跟随，不能另起契约 |
| Gateway 公开契约 | `docs/services/gateway/api/public.openapi.yaml` | 前端稳定契约以 gateway 为准 |
| 服务边界 | `docs/architecture/service-boundaries.md` | 必须遵守 |
| 技术基线 | `docs/architecture/technology-decisions.md` | 必须跟随 |
| 代码实现 | `services/knowledge/` | 本文档记录当前状态和差距 |

凡是本文档与上表文件冲突，以上游文件为准；发现冲突时，在“文档与实现出入”中记录并生成回写或实现任务。

## 2. 当前结论

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| 文档状态 | active | README、公开草案、数据模型、内部 OpenAPI 和实现说明存在。 |
| 代码状态 | partial | Go adapter 已实现知识库 CRUD、文档列表/上传/详情、parser-configs 运行时管理、RAGFlow runtime client、文档 chunks/content API、`knowledge-queries` 检索、MCP 工具边界和 runtime contract tests。旧 Parser Service client 路径已由 RAGFlow runtime 方案取代。 |
| 契约对齐 | partial | Gateway OpenAPI 已声明 document lifecycle、chunks、content、knowledge-queries、parser configs；这些 active routes 已由 Knowledge 和 Gateway proxy 落地。Knowledge ingestion 真实依赖 smoke、Gateway -> Knowledge owner route smoke，以及 Gateway -> Knowledge -> QA RAG env-gated smoke 已补齐；MCP、前端和 #125 完整一键 Gateway E2E 仍待后续 smoke。 |
| 数据持久化 | postgres / Redis queue / Qdrant | runtime 使用 PostgreSQL；Redis/asynq 负责任务投递；vector index 支持 Qdrant，未配置时使用 in-memory index。 |
| 测试状态 | partial | 单元、handler、contract、repository integration、platform tests、runtime route/config tests、adapter contract tests 和真实 PDF E2E 验证路径覆盖 CRUD、权限、runtime config 映射、chunks/content、`knowledge-queries`、错误 envelope 和 request id；完整 Gateway/MCP/QA #125 E2E 仍需后续联调。 |
| 依赖解耦 | documented | A-12 检索和 A-14 契约测试可依赖 `docs/api-contract.md` 2.6 与 `docs/data-models.md` 6.7 的 seeded chunk/vector fixture，不再要求 A-11 worker runtime 先完成。 |
| 建议动作 | 联调 / 人工复审 | 继续补真实 File/Parser/Redis/Qdrant/AI Gateway 端到端 smoke，以及并发/外部副作用一致性加固；人工复审任务幂等、失败状态收敛和敏感数据不泄漏。 |

## 3. 已实现

| 能力 | 代码位置 | 契约来源 | 验证方式 | 备注 |
| --- | --- | --- | --- | --- |
| 健康检查 | `services/knowledge/internal/http/server.go` | `docs/services/knowledge/api/internal.openapi.yaml` | `cd services/knowledge && go test ./...` | `GET /healthz`、`GET /readyz`。 |
| 知识库 CRUD | `services/knowledge/internal/http/server.go`、`internal/service/service.go` | `docs/services/knowledge/api/internal.openapi.yaml` | `TestKnowledgeBaseCRUDAndSoftDelete` | 支持列表、创建、详情、更新、软删除。 |
| 用户上下文和权限校验 | `services/knowledge/internal/service/service.go` | `docs/services/knowledge/README.md` | service tests | 依赖 gateway 注入的 user/permission context。 |
| 文档列表和详情 | `services/knowledge/internal/http/server.go` | `docs/services/knowledge/api/internal.openapi.yaml` | `TestDocumentListAndDetailExcludeDeletedKnowledgeBase` | 只覆盖文档元数据/状态。 |
| 文档上传 handoff | `services/knowledge/internal/platform/fileclient/client.go`、`internal/service/service.go` | `docs/services/knowledge/README.md`、`docs/services/file/README.md` | `TestUploadDocumentCreatesDocumentJobAndQueuesIngestion` | multipart 上传后调用 File `/internal/v1/files`，保存 `file_ref`，创建 processing job。 |
| Parser configs 运行时管理 | `services/knowledge/internal/http/server.go`、`internal/service/parser_config.go`、`internal/repository/postgres.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/architecture/service-boundaries.md` | `cd services/knowledge && go test ./...`；repository integration CI | 支持 list/get/create/update/delete、默认 builtin seed、上传 parser snapshot、重复名称 conflict、空配置 fallback 和 MIME 匹配选择。 |
| asynq 入队 | `services/knowledge/internal/platform/queue` | `docs/architecture/technology-decisions.md` | service tests with fake queue | 投递 `knowledge:document:ingest`，retry 次数与 `processing_jobs.max_attempts` 默认值对齐。 |
| RAGFlow runtime adapter | `services/knowledge/internal/adapter`、`internal/vendorclient`、`services/knowledge-runtime` | `docs/services/knowledge/README.md`、`docs/runbooks/local-integration.md` | adapter contract tests、runtime route tests、PDF E2E | 通过 runtime API/worker 完成 dataset/document、解析、切块、embedding、索引和检索；Go adapter 负责项目内部 HTTP 契约、权限上下文和错误归一化。 |
| 文档 chunks HTTP API | `services/knowledge/internal/http/server.go`、`internal/service/ingestion.go`、`internal/repository/postgres.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/services/knowledge/api/internal.openapi.yaml` | `TestDocumentChunksAndContentContract`、Gateway proxy tests | 支持 `GET /internal/v1/documents/{documentId}/chunks`，分页返回 Knowledge-owned chunk DTO，不暴露原始向量或 Qdrant payload。 |
| 原始文档 content API | `services/knowledge/internal/http/server.go`、`internal/service/service.go`、`internal/platform/fileclient/client.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/architecture/service-boundaries.md` | `TestDocumentChunksAndContentContract`、Gateway binary proxy tests | 先校验 Knowledge 文档可见性，再通过 File Service 内部读取 raw bytes；响应为二进制流，不包 JSON envelope，不暴露 `file_ref`、object key 或内部 URL。 |
| 文档 lifecycle API | `services/knowledge/internal/http/server.go`、`internal/service/service.go`、`internal/repository/postgres.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/services/knowledge/api/internal.openapi.yaml` | service/http tests、PostgreSQL repository lifecycle integration test、Gateway proxy tests | 支持 `PATCH /internal/v1/documents/{documentId}` 更新 tags，`DELETE /internal/v1/documents/{documentId}` 软删除并创建 `delete_cleanup` job；响应不暴露 `file_ref`、Qdrant point 或 embedding model。 |
| 文档 delete cleanup worker | `services/knowledge/internal/worker/handler.go`、`internal/service/delete_cleanup.go`、`internal/platform/queue/asynq.go`、`internal/platform/vector` | Issue #342、`docs/architecture/service-boundaries.md` | worker/service/repository/fileclient/vector tests | 注册并消费 `knowledge:document:delete_cleanup`，payload 只含 `requestId/jobId/documentId/knowledgeBaseId/userId`；claim job 后读取内部已软删 cleanup 视图，调用 File `DELETE /internal/v1/files/{fileId}`，按 `document_id` 删除 vector points，并把 job 标为 `succeeded`。空 `file_ref`、File 404、Qdrant point 不存在和重复投递按幂等成功处理；File/Qdrant/queue 失败只保存脱敏摘要，不恢复 deleted document；后台 reconciler 会从 PostgreSQL 重投 retryable cleanup job，补偿 Redis/asynq handoff 短暂失败。 |
| Runtime parser config mapping | `services/knowledge/internal/adapter/map.go`、`internal/service/parser_config.go` | `docs/services/knowledge/README.md`、`docs/architecture/service-boundaries.md` | adapter map tests、contract tests | `parser-configs` 仍是 Knowledge 管理的业务配置资源；创建 KB 时映射为 RAGFlow `parser_config`，不再调用独立 `services/parser`。 |
| embedding / vector index | `services/knowledge/internal/platform/embedding`、`internal/platform/vector` | `docs/architecture/service-boundaries.md` | platform/worker tests | local hashing 默认；可选 AI Gateway embedding；Qdrant HTTP adapter 或 in-memory index。 |
| `knowledge-queries` 检索 | `services/knowledge/internal/service/retrieval.go`、`internal/http/server.go`、Gateway proxy route | `docs/services/knowledge/docs/api-contract.md`、`docs/services/gateway/api/public.openapi.yaml` | service retrieval tests、`TestKnowledgeQueryContractWithSeededRepositoryAndFakeVector`、`TestKnowledgeQueriesRouteProxiesToKnowledge` | 基于 embedder + vector index 搜索，回 PostgreSQL hydrate chunks/documents，过滤未 ready/不可见文档，支持 tags、metadata filter、可选 AI Gateway rerank 和 local no-op rerank fallback。`topK` 和 `rerankTopN` 受最大 100 约束，rerank 结果 slice/map allocation 按有效上限分配。 |
| PostgreSQL migration/repository | `services/knowledge/migrations/0001_create_knowledge_core_tables.sql`、`0002_create_parser_configs.sql`、`internal/repository/postgres.go` | `docs/services/knowledge/docs/data-models.md` | `go test ./...`；CI 用 `KNOWLEDGE_TEST_DATABASE_URL` 跑 repository lifecycle integration test | runtime 使用 PostgreSQL，保存文档、job、parser configs 和 chunks。分页 limit/offset 转 `int32` 前在 repository 层做显式范围校验，非法页码或溢出 offset 返回 validation error，不静默截断到 `MaxInt32`。 |
| Knowledge PDF E2E | local runtime stack | #440、`docs/runbooks/local-integration.md` | 手动启动宿主机 runtime API/worker 和 host-run adapter 后上传 `DL_T_673-1999.pdf` | 覆盖真实 PDF 上传、runtime 解析、切块、embedding、索引和 `knowledge-queries` 检索；需要可用 runtime、对象存储、索引后端和 provider credential。 |
| Gateway -> Knowledge -> QA RAG smoke | `services/knowledge/internal/integration/gateway_rag_e2e_smoke_test.go` | #304、`docs/runbooks/local-integration.md` | `GATEWAY_RAG_E2E_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v` | 默认跳过；启用后通过 Gateway 创建 session/KB、上传 Markdown fixture，轮询文档 ready 和 chunkCount，调用 `knowledge-queries` 断言命中 `calibrate relay RAG-E2E-304` 和 rerank trace，再通过 QA config/session/message 验证 answer 包含 `RAG-E2E-304` 且 citations 匹配本轮 KB/doc/chunk。需要可用 AI Gateway chat profile/provider；默认 local hashing/in-memory vector 只证明等价检索数据，真实 Qdrant/AI Gateway embedding/rerank provider 需显式 env。 |

## 4. 未实现

| 缺口 | 文档来源 | 影响范围 | 建议任务 |
| --- | --- | --- | --- |
| `knowledge-queries` 真实 Qdrant retrieval smoke 未闭环 | `docs/services/knowledge/docs/api-contract.md`、`docs/architecture/technology-decisions.md` | retrieval / deployment | Gateway RAG smoke 已验证通过 `knowledge-queries` 命中本轮文档；默认 Compose 仍走 in-memory vector index。需要设置 `KNOWLEDGE_QDRANT_URL=http://qdrant:6333` 后记录真实 Qdrant search、PostgreSQL hydrate 和可选 rerank 证据。 |
| 真实 AI Gateway embedding/rerank smoke 未闭环 | `docs/architecture/service-boundaries.md`、`docs/services/knowledge/docs/data-models.md` | retrieval / AI Gateway | embedding 与 rerank adapter 已实现，默认 local hashing/no-op fallback；需要带真实 provider credential 的跨服务 smoke。 |

## 5. 文档与实现出入

| 出入点 | 文档要求 | 当前实现 | 风险 | 建议处理 |
| --- | --- | --- | --- | --- |
| AI Gateway rerank smoke 状态 | AI Gateway 已实现 embeddings/rerankings endpoint，Knowledge 支持 embedding 与 rerank adapter | `knowledge-queries` 可选 rerank 已接入；本地未配置 `RERANK_MODEL` 时使用 no-op fallback | 容易把 no-op fallback 误读为真实 provider rerank 已验收 | 保留 fake/seeded 契约测试，同时补带真实 provider credential 的跨服务 smoke。 |
| Runtime host 访问前置条件 | host-run Knowledge adapter 需要能访问 RAGFlow runtime API | 如果 `VENDOR_RUNTIME_URL` 指向 Docker bridge IP 且代理环境未把该 IP 加入 `NO_PROXY`，adapter 请求会被代理截走并返回 502 | 容易把代理问题误判为 runtime 或解析失败 | 默认配置使用发布到宿主机的 `127.0.0.1:9380`；临时使用容器 IP 时必须清理代理或补 `NO_PROXY`。 |
| 公开 Knowledge 草案范围 | `docs/services/knowledge/api/public.openapi.yaml` 是服务级 public 设计草案，覆盖 deletion jobs、processing jobs、query tests、support materials、settings、statistics | runtime 已实现 KB CRUD、文档 upload/list/detail/tags/soft delete、chunks/content 和 knowledge-queries；deletion job 查询、processing job 查询、query tests、support materials、settings、statistics 仍是草案/缺口；前端稳定契约以 gateway public OpenAPI 为准 | 文件名里的 `public` 可能被误读为 active browser-facing contract | 草案文件已加说明；进入前端稳定契约前必须先更新 `docs/services/gateway/api/public.openapi.yaml`。 |
| File handoff 边界 | Knowledge 拥有文档资源，File 只保存基础 file object | 当前已按 `/internal/v1/files` 保存 raw file，通过 content API 读取，并在 delete cleanup worker 中调用 File DELETE；Knowledge 只保存/传递不透明 `file_ref`，不读取 bucket、object key、MinIO URL 或 token | File/Qdrant 失败时 cleanup job 会保持 failed 并等待 asynq 重试；Redis/asynq handoff 短暂失败时 reconciler 会从 PostgreSQL 重投 retryable job；人工排障必须从 PostgreSQL job 状态开始 | 继续补真实 Redis delivery + File/Qdrant cleanup smoke。 |
| `sqlc` 生成器版本 | 技术基线固定 `sqlc` CLI 推荐版本为 `v1.31.1` | `services/knowledge/internal/repository/sqlc/*.go` 头部仍记录 `sqlc v1.29.0`；本次版本修复不改非 Docker 生成代码 | 代码生成器版本与文档基线出入，后续 SQL 变更时容易继续沿用旧生成器 | 下次修改 Knowledge SQL 或 repository 生成代码时，使用 `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` 重新生成并提交。 |
| `go-redis` 传递依赖版本 | 技术基线固定直接 Redis client 为 `go-redis/v9@v9.21.0` | Knowledge 通过 `asynq v0.26.0` 间接带入 `go-redis/v9@v9.14.1`；本次版本修复不改非 Docker 代码依赖 | 文档基线和锁定依赖存在出入，后续队列依赖升级时可能被忽略 | 下次升级 asynq 或调整 queue 依赖时优先消除该出入；不能消除时继续在本文记录原因。 |

## 6. MVP / mock / memory backend / 占位

| 项目 | 当前用途 | 退出条件 | 关联任务 |
| --- | --- | --- | --- |
| memory repository | 单元测试 | PostgreSQL integration tests 覆盖关键 CRUD 后仍可保留测试用 | 保留测试用 |
| fake file client / fake queue | 上传补偿、入队和 delete cleanup 测试 | 真实 file/Redis 集成测试补齐 | File/Redis integration smoke |
| fake runtime/vendor client | adapter、MCP 和 contract tests 的 runtime 响应输入 | 真实 runtime E2E 稳定后仍可保留为快速契约测试 | Runtime contract tests |
| seeded chunk/vector fixture | A-12 retrieval 和 A-14 contract tests | 真实 worker + Qdrant + AI Gateway smoke 稳定后仍可保留为快速契约测试 | A-12/A-14 并行开发 |
| fake vector / fake AI adapter | 检索过滤、rerank trace、错误 envelope 和 request id 测试 | 真实依赖集成测试补齐；不替代端到端 smoke | Retrieval contract tests |
| delete cleanup fake vector/file adapters | DELETE 文档后的 File/Qdrant 清理和失败重试测试 | 真实 File/Redis/Qdrant cleanup smoke 补齐后仍保留为快速契约测试 | Knowledge document cleanup smoke |

## 7. 运行与配置

| 项目 | 当前状态 | 缺口 |
| --- | --- | --- |
| 启动命令 | `cd services/knowledge && go run ./cmd/adapter` | 需要可访问的 host-run `services/knowledge-runtime` API。 |
| 环境变量 | adapter 模式需要 `VENDOR_RUNTIME_URL`、`KNOWLEDGE_SERVICE_TOKEN` 或 `INTERNAL_SERVICE_TOKEN`、`KNOWLEDGE_AUTO_START_INGESTION`；`DATABASE_URL` 或 `KNOWLEDGE_DATABASE_URL` 用于 parser-config admin。runtime 依赖 PostgreSQL、Redis、MinIO、Elasticsearch 和 provider 配置 | 仍需按部署环境补真实依赖连通性检查；所有 `/internal/v1/**` 调用必须带匹配的 `X-Service-Token`。 |
| PostgreSQL / migration | `migrations/0001_create_knowledge_core_tables.sql`、`0002_create_parser_configs.sql`，runtime `pgx/v5` | goose apply CI 已覆盖 migration；repository lifecycle 由 `KNOWLEDGE_TEST_DATABASE_URL` 集成测试覆盖。 |
| Redis / queue | 使用 `asynq` client 投递 ingestion 和 delete cleanup；worker 在同进程消费 `knowledge:document:ingest` 与 `knowledge:document:delete_cleanup` | 后续可按部署形态拆分独立 worker 进程。 |
| Object storage / vector store / AI provider | RAGFlow runtime 通过 MinIO、Elasticsearch/向量索引和 provider 配置完成文档保存、embedding、索引和检索 | 本地 PDF E2E 覆盖上传、解析、切块、索引和查询；仍需完整 Gateway/MCP/QA 联调。 |
| Knowledge runtime | Knowledge 通过 `VENDOR_RUNTIME_URL` 调 `services/knowledge-runtime` 的 RAGFlow API；worker 负责后台 parse/index 任务 | runtime image rebuild 在网络异常时可能卡在资源仓库 clone；本地可先复用已构建健康容器验证链路。 |

当 `EMBEDDING_PROVIDER=ai_gateway` 时，`EMBEDDING_MODEL` 必须匹配解析出的 AI Gateway embedding profile `model`。`AI_GATEWAY_EMBEDDING_PROFILE_ID` 可留空以使用 AI Gateway 默认启用的 embedding profile，但 provider 调用前仍会强制校验 model 匹配。

### 7.1 Delete cleanup worker 运维约束

- 任务类型：`knowledge:document:delete_cleanup`。
- 启动方式：当前和 HTTP server、ingestion worker 共用 `cd services/knowledge && go run ./cmd/server` 进程，asynq mux 同时注册 `knowledge:document:ingest` 和 `knowledge:document:delete_cleanup`。
- 依赖 env：`DATABASE_URL` 或 `KNOWLEDGE_DATABASE_URL`、`FILE_SERVICE_BASE_URL`、`KNOWLEDGE_REDIS_ADDR`、`KNOWLEDGE_SERVICE_TOKEN` 必填；`QDRANT_URL` 为空时使用 memory vector index，非空时使用 `QDRANT_COLLECTION` 和可选 `QDRANT_API_KEY`。
- payload：只允许 `requestId`、`jobId`、`documentId`、`knowledgeBaseId`、`userId`。不得加入 `file_ref`、bucket、object key、MinIO URL、签名 URL、service token 或 Qdrant payload。
- 重试语义：worker 先 claim PostgreSQL `processing_jobs`，再读已软删 document cleanup 视图；File 404、空 `file_ref`、Qdrant point 不存在和重复投递视为幂等成功。File/Qdrant 失败会把 job 标为 `failed`，保存 `file cleanup failed` 或 `vector cleanup failed` 这类脱敏摘要，并返回错误给 asynq 重试；不会恢复 `knowledge_documents.deleted_at`。同进程 reconciler 每分钟从 PostgreSQL 扫描 retryable `queued`、dependency-failed 和 stale-running `delete_cleanup` job 并重新入队，Redis/asynq 仍不是长期事实源。
- 状态事实：PostgreSQL 的 `knowledge_documents.deleted_at` 和 `processing_jobs` 是长期事实来源；Redis/asynq 只负责投递和短期重试。

排查 SQL：

```sql
SELECT id, document_id, status, current_stage, attempts, max_attempts,
       error_code, error_message, updated_at
FROM processing_jobs
WHERE job_type = 'delete_cleanup'
ORDER BY updated_at DESC
LIMIT 20;

SELECT d.id AS document_id, d.knowledge_base_id, d.current_job_id,
       j.status, j.attempts, j.max_attempts, j.error_code, j.error_message
FROM knowledge_documents d
JOIN processing_jobs j ON j.id = d.current_job_id
WHERE d.deleted_at IS NOT NULL
  AND j.job_type = 'delete_cleanup'
  AND j.status IN ('queued', 'running', 'failed')
ORDER BY j.updated_at DESC;
```

## 8. 测试与验证

| 验证项 | 命令或步骤 | 当前结果 | 缺口 |
| --- | --- | --- | --- |
| 单元测试 | `cd services/knowledge && go test ./...` | pass（2026-07-01，本地 Go 1.26.4；需允许 `httptest` 本地端口监听） | 主要使用 memory/fake 依赖，并覆盖 document lifecycle tags/soft delete/delete cleanup worker、parser-configs 管理、fallback、conflict、上传 snapshot、chunks/content、`knowledge-queries`、错误 envelope 和 request id；env-gated ingestion/owner smoke 默认跳过，不破坏普通 CI。 |
| Repository 集成测试 | `KNOWLEDGE_TEST_DATABASE_URL=... go test ./internal/repository -count=1` | CI 覆盖 repository lifecycle；无 env 时本地跳过 | 覆盖 PostgreSQL repository 的 CRUD、tags 更新、软删除可见性、内部 cleanup target 视图和 `delete_cleanup` job 创建；不覆盖 File/Redis/Qdrant。 |
| Knowledge runtime targeted tests | `cd services/knowledge-runtime && PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio python -m pytest <targeted tests> -q` | pass（2026-07-02，本地 targeted route/config tests） | 不替代真实 PDF E2E。 |
| Knowledge PDF E2E | 启动 runtime API/worker 和 host-run adapter 后上传 `DL_T_673-1999.pdf` | available / manual | 覆盖真实 PDF 上传、解析、切块、embedding、索引和检索；需要真实 provider credential 和本地 runtime。 |
| Gateway -> Knowledge owner route smoke | `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v` | available（2026-07-01 新增；默认 skip） | 覆盖伪造 `X-User-*` 未认证请求拒绝、Gateway session 创建、KB `createdBy` 真实 session user 断言，以及 Parser/File/Redis/PostgreSQL ready 前置检查；不覆盖完整 Gateway route matrix。 |
| Gateway -> Knowledge -> QA RAG smoke | `GATEWAY_RAG_E2E_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v` | available（2026-07-01 新增；默认 skip；本轮只跑默认 skip 编译检查） | 覆盖最小 RAG 样例的 Gateway 上传、ingestion ready/chunks、`knowledge-queries` 命中、QA answer 和 citations；需要可用 AI Gateway chat profile/provider，真实 provider 不进入普通 CI。 |
| 端到端上传/删除联调 | Gateway + Redis delivery + PostgreSQL + File + Parser + Qdrant end-to-end upload/delete | partial | A-021 直接调用 worker handler 消费捕获的 ingestion payload，刻意不把 Redis/asynq delivery 可靠性扩进本 smoke；delete cleanup 当前由 unit/integration repository 测试覆盖，仍缺真实 Redis delivery + File/Qdrant cleanup smoke。完整 Gateway/Redis/MCP E2E 仍由 #125 等任务覆盖。 |
| 契约测试 | gateway route matrix + Knowledge handler tests | pass（2026-06-30） | document lifecycle、chunks、content、knowledge-queries、parser-configs 等 active path 已补 contract/request-id/error envelope 覆盖。 |
| 手工 smoke | 启动 PostgreSQL、File、Redis 后上传文档 | not run | 需要可复现脚本或 Compose。 |

## 9. 建议任务

| 任务 | 类型 | 优先级 | 依据 | 说明 |
| --- | --- | --- | --- | --- |
| 补真实 delete cleanup smoke | 新任务 | P0 | 当前已实现 worker，但仍缺真实 Redis delivery + File/Qdrant cleanup smoke | 通过真实 File Service、Redis/asynq 和 Qdrant 验证删除文档后 File DELETE、Qdrant filter delete、job succeeded/failed 重试语义。 |
| 补真实 Qdrant/AI Gateway retrieval-rerank 证据 | 新任务 | P0 | #304 已提供最小 Gateway RAG smoke，但默认路径允许 local hashing/in-memory vector 和 no-op rerank fallback | 在真实 Qdrant、AI Gateway embedding/rerank provider、Redis/asynq delivery 环境下记录 `knowledge-queries` search/rerank 证据，并继续由 #125 覆盖 MCP/前端完整 E2E。 |

## 10. 最近检查记录

| 日期 | 检查人/工具 | 代码基准 | 结论 |
| --- | --- | --- | --- |
| 2026-07-01 | Codex | Issue #342 branch | 实现 Knowledge 文档 delete cleanup worker：删除文档后软删并投递 `knowledge:document:delete_cleanup`，worker 幂等调用 File DELETE 和按 `document_id` 清理 vector points，失败摘要脱敏写入 `processing_jobs`；仍需真实 Redis delivery + File/Qdrant cleanup smoke 和人工复审。 |
| 2026-07-01 | Codex | Issue #304 branch | 新增 env-gated `TestGatewayRAGE2ESmoke`，默认 skip；启用后通过 Gateway 上传最小 Markdown fixture，验证 Knowledge ingestion ready/chunkCount、`knowledge-queries` 命中、QA answer 包含 `RAG-E2E-304`，并校验 citation 摘要匹配本轮 KB/doc/chunk。 |
| 2026-07-01 | Codex | A-021 working tree | 历史路径：新增 env-gated `TestKnowledgeIngestionRealDepsSmoke`，当时验证 File Service、Parser Service、Knowledge worker、local hashing embedding 到 Qdrant 的路径；当前 RAGFlow runtime 方案已取代独立 Parser Service。 |
| 2026-07-01 | Codex CodeQL follow-up | working tree | 继续收敛合并后仍 open 的 rerank allocation 告警：rerank result ordering 的 slice/map capacity 改为 `maxRetrievalTopK` 常量，`limit` 仅作为业务截断条件，避免用户控制值继续流入 allocation size。 |
| 2026-07-01 | Codex | A-021 scope update | 历史路径：新增 env-gated `TestGatewayKnowledgeOwnerRouteSmoke` 和 Parser image 构建/缓存前置说明；当前 RAGFlow runtime 方案不再恢复 Parser image。 |
| 2026-06-30 | Codex full-day audit | `develop@92d3afc` | 历史路径：Knowledge 当时包含 ingestion worker、Parser Service client、parser-configs runtime management、chunks/content、`knowledge-queries`、AI Gateway embedding/rerank adapter、document PATCH/DELETE lifecycle 和 Gateway proxy；当前 RAGFlow runtime 方案已取代独立 Parser Service。 |
| 2026-06-30 | Codex | A-014 working tree | 补齐 chunks/content internal route、Gateway proxy、seeded/fake-backed `knowledge-queries` contract、错误 envelope 和 request id 测试；当时 document PATCH/DELETE 与真实 Qdrant/AI Gateway smoke 仍待后续任务。 |
| 2026-06-30 | Codex | PR #273 | 文档 PATCH/DELETE lifecycle 已落地：tags 更新、软删除、cleanup job 创建、Gateway proxy 和 PostgreSQL repository lifecycle 集成测试；真实 File/Qdrant cleanup worker 和跨依赖 smoke 仍待后续任务。 |
| 2026-06-30 | Codex | working tree | 补充 A-11/A-12/A-14 解耦契约：A-12/A-14 可用 seeded chunks、fake vector/AI adapter 做契约和 handler 测试；完整 ingestion runtime 仍由 A-11 交付。 |
| 2026-06-30 | Codex | A-13 PR #249 | parser-configs 运行时管理已落地并合入：Knowledge 内部 API、Gateway proxy、默认 builtin seed、上传 snapshot、conflict 映射和前端管理入口均已覆盖。 |
| 2026-06-30 | Codex | PR #226 docs extraction | 历史路径：当时从 PR #226 单独抽出 Parser Runtime 文档和 OpenAPI 契约；当前分支已删除旧 Parser 文档。 |
| 2026-06-29 | Codex goal | `eddf917` + working tree | Knowledge 已完成 KB CRUD 和文档上传 handoff；当时 parser config 与入库 worker、chunks、content、retrieval 均为关键缺口，其中 parser config 已由 A-13 补齐。 |
| 2026-06-29 | Codex | A11 branch | 历史路径：Knowledge 当时完成文档上传 handoff、入库 worker、Parser Service 解析调用、切片、embedding、vector 写入和 chunk 持久化；当前 RAGFlow runtime 方案已替代独立 Parser Service。 |
