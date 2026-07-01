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
| 代码状态 | partial | Go service 已实现知识库 CRUD、文档列表/上传/详情、文档 tags 更新、软删除、delete cleanup worker、File Service handoff、PostgreSQL repository、asynq enqueue、parser-configs 运行时管理、ingestion worker、Parser Service client、Knowledge-owned chunker、embedding、chunk 持久化、vector index 写入、文档 chunks/content API 和 `knowledge-queries` 检索。 |
| 契约对齐 | partial | Gateway OpenAPI 已声明 document lifecycle、chunks、content、knowledge-queries、parser configs；这些 active routes 已由 Knowledge 和 Gateway proxy 落地。Knowledge ingestion 真实依赖 smoke 和 Gateway -> Knowledge owner route smoke 已补齐；retrieval/rerank、MCP 和完整 Gateway E2E 仍待后续 smoke。 |
| 数据持久化 | postgres / Redis queue / Qdrant | runtime 使用 PostgreSQL；Redis/asynq 负责任务投递；vector index 支持 Qdrant，未配置时使用 in-memory index。 |
| 测试状态 | partial | 单元、handler、contract、repository integration、platform tests、env-gated ingestion real deps smoke 和 Gateway owner route smoke 覆盖 CRUD、权限、上传补偿、document lifecycle tags/soft delete/delete cleanup worker、queue handoff、worker 入库、File content/delete、Parser HTTP client、chunking、embedding、vector payload、Qdrant point 写入/按文档删除、parser-configs、chunks/content、`knowledge-queries`、Gateway/Auth 到 Knowledge owner route 的上下文注入、错误 envelope 和 request id；缺 retrieval/rerank 与完整 Gateway/MCP E2E 联调。 |
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
| 文档入库 worker | `services/knowledge/internal/worker`、`internal/service/ingestion.go` | `docs/services/knowledge/README.md` | worker/service tests | 消费 A10 payload，读取 File content，解析、切片、embedding、写 vector index 和 chunks，并推进 ready/failed 状态。 |
| File content reader | `services/knowledge/internal/platform/fileclient/client.go` | `docs/services/file/README.md` | fileclient tests | 调用 `/internal/v1/files/{fileId}/content`，透传 request/user/service headers，失败脱敏为 dependency error。 |
| 文档 chunks HTTP API | `services/knowledge/internal/http/server.go`、`internal/service/ingestion.go`、`internal/repository/postgres.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/services/knowledge/api/internal.openapi.yaml` | `TestDocumentChunksAndContentContract`、Gateway proxy tests | 支持 `GET /internal/v1/documents/{documentId}/chunks`，分页返回 Knowledge-owned chunk DTO，不暴露原始向量或 Qdrant payload。 |
| 原始文档 content API | `services/knowledge/internal/http/server.go`、`internal/service/service.go`、`internal/platform/fileclient/client.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/architecture/service-boundaries.md` | `TestDocumentChunksAndContentContract`、Gateway binary proxy tests | 先校验 Knowledge 文档可见性，再通过 File Service 内部读取 raw bytes；响应为二进制流，不包 JSON envelope，不暴露 `file_ref`、object key 或内部 URL。 |
| 文档 lifecycle API | `services/knowledge/internal/http/server.go`、`internal/service/service.go`、`internal/repository/postgres.go` | `docs/services/gateway/api/public.openapi.yaml`、`docs/services/knowledge/api/internal.openapi.yaml` | service/http tests、PostgreSQL repository lifecycle integration test、Gateway proxy tests | 支持 `PATCH /internal/v1/documents/{documentId}` 更新 tags，`DELETE /internal/v1/documents/{documentId}` 软删除并创建 `delete_cleanup` job；响应不暴露 `file_ref`、Qdrant point 或 embedding model。 |
| 文档 delete cleanup worker | `services/knowledge/internal/worker/handler.go`、`internal/service/delete_cleanup.go`、`internal/platform/queue/asynq.go`、`internal/platform/vector` | Issue #342、`docs/architecture/service-boundaries.md` | worker/service/repository/fileclient/vector tests | 注册并消费 `knowledge:document:delete_cleanup`，payload 只含 `requestId/jobId/documentId/knowledgeBaseId/userId`；claim job 后读取内部已软删 cleanup 视图，调用 File `DELETE /internal/v1/files/{fileId}`，按 `document_id` 删除 vector points，并把 job 标为 `succeeded`。空 `file_ref`、File 404、Qdrant point 不存在和重复投递按幂等成功处理；File/Qdrant/queue 失败只保存脱敏摘要，不恢复 deleted document；后台 reconciler 会从 PostgreSQL 重投 retryable cleanup job，补偿 Redis/asynq handoff 短暂失败。 |
| Parser Service client / chunker | `services/knowledge/internal/platform/parser`、`internal/service/chunker.go` | `docs/services/knowledge/README.md`、`docs/services/parser/README.md` | parser/client、worker/service tests | Knowledge 以流式 base64 JSON 请求调用独立 Parser Service，消费 `content/title/backend/pages`，并在 Knowledge 内完成 chunking；当前切片仍以 `content` 为主，能映射到单页时会保存 `page_start/page_end/source_pages` 和 parser 质量字段。PaddleOCR runtime 不在 Knowledge Go 进程内。 |
| embedding / vector index | `services/knowledge/internal/platform/embedding`、`internal/platform/vector` | `docs/architecture/service-boundaries.md` | platform/worker tests | local hashing 默认；可选 AI Gateway embedding；Qdrant HTTP adapter 或 in-memory index。 |
| `knowledge-queries` 检索 | `services/knowledge/internal/service/retrieval.go`、`internal/http/server.go`、Gateway proxy route | `docs/services/knowledge/docs/api-contract.md`、`docs/services/gateway/api/public.openapi.yaml` | service retrieval tests、`TestKnowledgeQueryContractWithSeededRepositoryAndFakeVector`、`TestKnowledgeQueriesRouteProxiesToKnowledge` | 基于 embedder + vector index 搜索，回 PostgreSQL hydrate chunks/documents，过滤未 ready/不可见文档，支持 tags、metadata filter、可选 AI Gateway rerank 和 local no-op rerank fallback。`topK` 和 `rerankTopN` 受最大 100 约束，rerank 结果 slice/map allocation 按有效上限分配。 |
| PostgreSQL migration/repository | `services/knowledge/migrations/0001_create_knowledge_core_tables.sql`、`0002_create_parser_configs.sql`、`internal/repository/postgres.go` | `docs/services/knowledge/docs/data-models.md` | `go test ./...`；CI 用 `KNOWLEDGE_TEST_DATABASE_URL` 跑 repository lifecycle integration test | runtime 使用 PostgreSQL，保存文档、job、parser configs 和 chunks。分页 limit/offset 转 `int32` 前在 repository 层做显式范围校验，非法页码或溢出 offset 返回 validation error，不静默截断到 `MaxInt32`。 |
| Knowledge ingestion 真实依赖 smoke | `services/knowledge/internal/integration/ingestion_real_deps_smoke_test.go` | #86 / #289、`docs/runbooks/local-integration.md` | `KNOWLEDGE_INGESTION_SMOKE=1 ... go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v` | 默认跳过；启用后使用真实 File Service、Parser Service、PostgreSQL、Qdrant 和默认 local hashing embedding，验证 fixture 上传、捕获 ingestion payload、worker handler、ready/succeeded 状态、chunk/embedding metadata 和 Qdrant point payload，并清理 File object、Qdrant collection、PostgreSQL schema。 |
| Gateway -> Knowledge owner route smoke | `services/knowledge/internal/integration/gateway_owner_route_smoke_test.go` | #86 / #289、Gateway active owner route contract | `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v` | 默认跳过；启用后先检查 File、Parser、Knowledge ready、Knowledge PostgreSQL ping 和 Redis PING，再断言伪造 `X-User-*` 且无 Bearer token 的 Gateway 请求返回 `401`，最后通过 Gateway 创建 session，调用 `GET /api/v1/knowledge-bases`，并创建/读取 run-scoped KB 校验 `createdBy` 是真实 session user 而非伪造 header。 |

## 4. 未实现

| 缺口 | 文档来源 | 影响范围 | 建议任务 |
| --- | --- | --- | --- |
| `knowledge-queries` 真实 Qdrant retrieval smoke 未闭环 | `docs/services/knowledge/docs/api-contract.md`、`docs/architecture/technology-decisions.md` | retrieval / deployment | ingestion smoke 已验证真实 Qdrant collection 创建、point 写入和 payload 读取；仍需补通过 `knowledge-queries` 走 Qdrant search、PostgreSQL hydrate 和可选 rerank 的真实依赖 smoke。 |
| 真实 AI Gateway embedding/rerank smoke 未闭环 | `docs/architecture/service-boundaries.md`、`docs/services/knowledge/docs/data-models.md` | retrieval / AI Gateway | embedding 与 rerank adapter 已实现，默认 local hashing/no-op fallback；需要带真实 provider credential 的跨服务 smoke。 |

## 5. 文档与实现出入

| 出入点 | 文档要求 | 当前实现 | 风险 | 建议处理 |
| --- | --- | --- | --- | --- |
| AI Gateway rerank smoke 状态 | AI Gateway 已实现 embeddings/rerankings endpoint，Knowledge 支持 embedding 与 rerank adapter | `knowledge-queries` 可选 rerank 已接入；本地未配置 `RERANK_MODEL` 时使用 no-op fallback | 容易把 no-op fallback 误读为真实 provider rerank 已验收 | 保留 fake/seeded 契约测试，同时补带真实 provider credential 的跨服务 smoke。 |
| Parser image build/cache 前置条件 | 本地 Compose 可通过 `--build` 或已有镜像启动 Parser | 若本机缺少 `software-teamwork-local-parser:latest`，`docker compose up --no-build file parser knowledge` 会失败；若临时 build，可能在拉取 `python:3.12-slim` metadata 时受 Docker Hub 超时阻塞 | 容易把 Knowledge/Gateway smoke 失败误判为代码问题 | runbook 已记录先使用 `deploy/.env.china.example`、预构建/缓存 Parser image，再运行 `--no-build` 或 owner route smoke。 |
| 公开 Knowledge 草案范围 | `docs/services/knowledge/api/public.openapi.yaml` 是服务级 public 设计草案，覆盖 deletion jobs、processing jobs、query tests、support materials、settings、statistics | runtime 已实现 KB CRUD、文档 upload/list/detail/tags/soft delete、chunks/content 和 knowledge-queries；deletion job 查询、processing job 查询、query tests、support materials、settings、statistics 仍是草案/缺口；前端稳定契约以 gateway public OpenAPI 为准 | 文件名里的 `public` 可能被误读为 active browser-facing contract | 草案文件已加说明；进入前端稳定契约前必须先更新 `docs/services/gateway/api/public.openapi.yaml`。 |
| File handoff 边界 | Knowledge 拥有文档资源，File 只保存基础 file object | 当前已按 `/internal/v1/files` 保存 raw file，通过 content API 读取，并在 delete cleanup worker 中调用 File DELETE；Knowledge 只保存/传递不透明 `file_ref`，不读取 bucket、object key、MinIO URL 或 token | File/Qdrant 失败时 cleanup job 会保持 failed 并等待 asynq 重试；Redis/asynq handoff 短暂失败时 reconciler 会从 PostgreSQL 重投 retryable job；人工排障必须从 PostgreSQL job 状态开始 | 继续补真实 Redis delivery + File/Qdrant cleanup smoke。 |
| `sqlc` 生成器版本 | 技术基线固定 `sqlc` CLI 推荐版本为 `v1.31.1` | `services/knowledge/internal/repository/sqlc/*.go` 头部仍记录 `sqlc v1.29.0`；本次版本修复不改非 Docker 生成代码 | 代码生成器版本与文档基线出入，后续 SQL 变更时容易继续沿用旧生成器 | 下次修改 Knowledge SQL 或 repository 生成代码时，使用 `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` 重新生成并提交。 |
| `go-redis` 传递依赖版本 | 技术基线固定直接 Redis client 为 `go-redis/v9@v9.21.0` | Knowledge 通过 `asynq v0.26.0` 间接带入 `go-redis/v9@v9.14.1`；本次版本修复不改非 Docker 代码依赖 | 文档基线和锁定依赖存在出入，后续队列依赖升级时可能被忽略 | 下次升级 asynq 或调整 queue 依赖时优先消除该出入；不能消除时继续在本文记录原因。 |

## 6. MVP / mock / memory backend / 占位

| 项目 | 当前用途 | 退出条件 | 关联任务 |
| --- | --- | --- | --- |
| memory repository | 单元测试 | PostgreSQL integration tests 覆盖关键 CRUD 后仍可保留测试用 | 保留测试用 |
| fake file client / fake queue | 上传补偿、入队和 delete cleanup 测试 | 真实 file/Redis 集成测试补齐 | File/Redis integration smoke |
| fake parser client | A-11 worker、A-12/A-14 契约测试的 parsed content 输入 | 真实 Parser service smoke 稳定后仍可保留为快速契约测试 | Parser contract tests |
| seeded chunk/vector fixture | A-12 retrieval 和 A-14 contract tests | 真实 worker + Qdrant + AI Gateway smoke 稳定后仍可保留为快速契约测试 | A-12/A-14 并行开发 |
| fake vector / fake AI adapter | 检索过滤、rerank trace、错误 envelope 和 request id 测试 | 真实依赖集成测试补齐；不替代端到端 smoke | Retrieval contract tests |
| delete cleanup fake vector/file adapters | DELETE 文档后的 File/Qdrant 清理和失败重试测试 | 真实 File/Redis/Qdrant cleanup smoke 补齐后仍保留为快速契约测试 | Knowledge document cleanup smoke |

## 7. 运行与配置

| 项目 | 当前状态 | 缺口 |
| --- | --- | --- |
| 启动命令 | `cd services/knowledge && go run ./cmd/server` | 需要 PostgreSQL、File Service 和 Redis。 |
| 环境变量 | `DATABASE_URL`、`FILE_SERVICE_BASE_URL`、`PARSER_SERVICE_BASE_URL`、`KNOWLEDGE_REDIS_ADDR`、`KNOWLEDGE_SERVICE_TOKEN` 必填；另有 embedding、AI Gateway、Qdrant、HTTP/version/env/max upload/shutdown 配置 | 仍需按部署环境补真实依赖连通性检查。 |
| PostgreSQL / migration | `migrations/0001_create_knowledge_core_tables.sql`、`0002_create_parser_configs.sql`，runtime `pgx/v5` | goose apply CI 已覆盖 migration；repository lifecycle 由 `KNOWLEDGE_TEST_DATABASE_URL` 集成测试覆盖。 |
| Redis / queue | 使用 `asynq` client 投递 ingestion 和 delete cleanup；worker 在同进程消费 `knowledge:document:ingest` 与 `knowledge:document:delete_cleanup` | 后续可按部署形态拆分独立 worker 进程。 |
| Object storage / vector store / AI provider | 通过 File Service 保存、读取和删除 raw file；默认 local hashing + memory vector index；可选 Qdrant adapter、AI Gateway embedding adapter 和 rerank adapter 已接入 | ingestion smoke 已覆盖真实 File/Parser/PostgreSQL/Qdrant 写入；仍需真实 AI Gateway provider、retrieval/rerank、delete cleanup 真实依赖和完整 Gateway/MCP 联调。 |
| Parser runtime | Knowledge 通过 `PARSER_SERVICE_BASE_URL` 调 `services/parser` 的 `/internal/v1/parsed-documents`；Parser Service 以 Python/FastAPI/PaddleOCR PP-StructureV3 独立部署，并返回页级 `ParsedDocument.pages` | 仍需真实 PP-StructureV3 模型 smoke 和部署环境资源配置。 |

当 `EMBEDDING_PROVIDER=ai_gateway` 时，`EMBEDDING_MODEL` 必须匹配解析出的 AI Gateway embedding profile `model`。`AI_GATEWAY_EMBEDDING_PROFILE_ID` 可留空以使用 AI Gateway 默认启用的 embedding profile，但 provider 调用前仍会强制校验 model 匹配。

### 7.1 Delete cleanup worker 运维约束

- 任务类型：`knowledge:document:delete_cleanup`。
- 启动方式：当前和 HTTP server、ingestion worker 共用 `cd services/knowledge && go run ./cmd/server` 进程，asynq mux 同时注册 `knowledge:document:ingest` 和 `knowledge:document:delete_cleanup`。
- 依赖 env：`DATABASE_URL`、`FILE_SERVICE_BASE_URL`、`KNOWLEDGE_REDIS_ADDR`、`KNOWLEDGE_SERVICE_TOKEN` 必填；`QDRANT_URL` 为空时使用 memory vector index，非空时使用 `QDRANT_COLLECTION` 和可选 `QDRANT_API_KEY`。
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
| Parser 服务测试 | `cd services/parser && uv run ruff check . && uv run pytest && uv run python -m compileall src tests` | available / not run in this documentation pass | Parser 拥有独立 service-local 测试；默认使用 fake OCR backend，不下载 PaddleOCR 模型，真实 PaddleOCR smoke 需显式环境变量。 |
| Knowledge ingestion 真实依赖 smoke | `KNOWLEDGE_INGESTION_SMOKE=1 ... go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v` | available（2026-07-01 新增；默认 skip） | 覆盖 PostgreSQL/File Service/Parser Service/local hashing embedding/Qdrant 写入和清理；不覆盖 Redis delivery、`knowledge-queries`、rerank、MCP 或 Gateway E2E。 |
| Gateway -> Knowledge owner route smoke | `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v` | available（2026-07-01 新增；默认 skip） | 覆盖伪造 `X-User-*` 未认证请求拒绝、Gateway session 创建、KB `createdBy` 真实 session user 断言，以及 Parser/File/Redis/PostgreSQL ready 前置检查；不覆盖完整 Gateway route matrix。 |
| 端到端上传/删除联调 | Gateway + Redis delivery + PostgreSQL + File + Parser + Qdrant end-to-end upload/delete | partial | A-021 直接调用 worker handler 消费捕获的 ingestion payload，刻意不把 Redis/asynq delivery 可靠性扩进本 smoke；delete cleanup 当前由 unit/integration repository 测试覆盖，仍缺真实 Redis delivery + File/Qdrant cleanup smoke。完整 Gateway/Redis/MCP E2E 仍由 #125 等任务覆盖。 |
| 契约测试 | gateway route matrix + Knowledge handler tests | pass（2026-06-30） | document lifecycle、chunks、content、knowledge-queries、parser-configs 等 active path 已补 contract/request-id/error envelope 覆盖。 |
| 手工 smoke | 启动 PostgreSQL、File、Redis 后上传文档 | not run | 需要可复现脚本或 Compose。 |

## 9. 建议任务

| 任务 | 类型 | 优先级 | 依据 | 说明 |
| --- | --- | --- | --- | --- |
| 补真实 delete cleanup smoke | 新任务 | P0 | 当前已实现 worker，但仍缺真实 Redis delivery + File/Qdrant cleanup smoke | 通过真实 File Service、Redis/asynq 和 Qdrant 验证删除文档后 File DELETE、Qdrant filter delete、job succeeded/failed 重试语义。 |
| 补真实 retrieval/rerank/Gateway 端到端 smoke | 新任务 | P0 | A-021 已覆盖 ingestion 写入，但不覆盖查询与 Gateway 总入口 | 覆盖 gateway 上传、Redis/asynq delivery、`knowledge-queries` 可命中、可选 AI Gateway embedding/rerank、MCP/QA 调用入口。 |

## 10. 最近检查记录

| 日期 | 检查人/工具 | 代码基准 | 结论 |
| --- | --- | --- | --- |
| 2026-07-01 | Codex | Issue #342 branch | 实现 Knowledge 文档 delete cleanup worker：删除文档后软删并投递 `knowledge:document:delete_cleanup`，worker 幂等调用 File DELETE 和按 `document_id` 清理 vector points，失败摘要脱敏写入 `processing_jobs`；仍需真实 Redis delivery + File/Qdrant cleanup smoke 和人工复审。 |
| 2026-07-01 | Codex | A-021 working tree | 新增 env-gated `TestKnowledgeIngestionRealDepsSmoke`，默认 skip；启用后验证 fixture 从 File Service、Parser Service、Knowledge worker、local hashing embedding 到 Qdrant point 写入和 PostgreSQL 状态更新，并记录清理策略。 |
| 2026-07-01 | Codex CodeQL follow-up | working tree | 继续收敛合并后仍 open 的 rerank allocation 告警：rerank result ordering 的 slice/map capacity 改为 `maxRetrievalTopK` 常量，`limit` 仅作为业务截断条件，避免用户控制值继续流入 allocation size。 |
| 2026-07-01 | Codex | A-021 scope update | 根据 Gateway/Auth 前置发现，新增 env-gated `TestGatewayKnowledgeOwnerRouteSmoke` 和 Parser image 构建/缓存前置说明；owner smoke 启用后先查 File/Parser/Knowledge ready、PostgreSQL 和 Redis，再通过 Gateway session 调用 `GET /api/v1/knowledge-bases`。 |
| 2026-06-30 | Codex full-day audit | `develop@92d3afc` | 复核今日 PR/issue：Knowledge 已包含 ingestion worker、Parser Service client、parser-configs runtime management、chunks/content、`knowledge-queries`、AI Gateway embedding/rerank adapter、document PATCH/DELETE lifecycle 和 Gateway proxy；Parser PP-StructureV3/runtime readiness 已由 Parser 服务承接。剩余为 delete cleanup worker、#289 真实依赖 ingest/query smoke、真实 Qdrant collection smoke 和真实 AI Gateway embedding/rerank smoke。 |
| 2026-06-30 | Codex | A-014 working tree | 补齐 chunks/content internal route、Gateway proxy、seeded/fake-backed `knowledge-queries` contract、错误 envelope 和 request id 测试；当时 document PATCH/DELETE 与真实 Qdrant/AI Gateway smoke 仍待后续任务。 |
| 2026-06-30 | Codex | PR #273 | 文档 PATCH/DELETE lifecycle 已落地：tags 更新、软删除、cleanup job 创建、Gateway proxy 和 PostgreSQL repository lifecycle 集成测试；真实 File/Qdrant cleanup worker 和跨依赖 smoke 仍待后续任务。 |
| 2026-06-30 | Codex | working tree | 补充 A-11/A-12/A-14 解耦契约：A-12/A-14 可用 seeded chunks、fake vector/AI adapter 做契约和 handler 测试；完整 ingestion runtime 仍由 A-11 交付。 |
| 2026-06-30 | Codex | A-13 PR #249 | parser-configs 运行时管理已落地并合入：Knowledge 内部 API、Gateway proxy、默认 builtin seed、上传 snapshot、conflict 映射和前端管理入口均已覆盖。 |
| 2026-06-30 | Codex | PR #226 docs extraction | 从 PR #226 单独抽出 Parser Runtime 文档和 OpenAPI 契约；当前分支只记录契约，未引入 Knowledge worker 实现代码。 |
| 2026-06-29 | Codex goal | `eddf917` + working tree | Knowledge 已完成 KB CRUD 和文档上传 handoff；当时 parser config 与入库 worker、chunks、content、retrieval 均为关键缺口，其中 parser config 已由 A-13 补齐。 |
| 2026-06-29 | Codex | A11 branch | Knowledge 已完成文档上传 handoff、入库 worker、Parser Service 解析调用、切片、embedding、vector 写入和 chunk 持久化；当时 content 与 knowledge-queries 仍待补齐，A-14 working tree 已补 active route/contract 覆盖。 |
