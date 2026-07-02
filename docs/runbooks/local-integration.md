# 本地联调运行手册

默认联调路径只有一条：

```text
Docker infra -> host backend -> frontend
```

不要启动业务服务容器，不要使用 `--build`，不要手工 export 一长串变量。
`deploy/.env.example` 是默认配置来源；用户复制成 `deploy/.env` 后，脚本只读取它。

Issue #125 的 MCP 与跨服务 smoke 汇总入口见
[`Issue #125 MCP and Cross-Service Smoke`](./issue-125-smoke.md)。只有对应 slice
在当前环境实际通过后，才能在 PR 或验收记录中声明完整跨服务 smoke 通过。

## 启动命令

```bash
cp deploy/.env.example deploy/.env
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh
cd apps/web && bun install && bun run dev
```

日常再次启动：

```bash
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh
cd apps/web && bun run dev
```

停止：

```bash
./scripts/local/stop-backend.sh
```

重置本地数据：

```bash
./scripts/local/stop-backend.sh
docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v
```

## 启动后应该看到什么

- 前端：`http://localhost:5173`
- Gateway：`http://localhost:8080`
- 本机 OpenAI-compatible 模型服务默认地址：`http://localhost:11434/v1`
- 默认 demo 管理员：`admin` / `LocalDemoAdmin#12345`
- 后端日志：`.local/logs/*.log`
- 后端进程组 PID：`.local/run/*.pid`

快速确认：

```bash
curl --noproxy '*' -fsS http://localhost:8080/healthz
curl --noproxy '*' -fsS http://localhost:8080/readyz
```

`ai-gateway /readyz` 在 placeholder profile 下返回 `503 degraded` 是预期行为。
这表示真实 provider credential 尚未配置，不表示进程失败。

## 谁负责什么

- `dev-up.sh`：infra pull/up、等待 Compose health checks、Qdrant collection
  初始化、migration、demo seed。
- `run-backend.sh`：Parser uv 依赖准备、后端进程启动、日志和进程组 PID。uv 的
  Python 包索引来自 `deploy/.env` 里的 `UV_DEFAULT_INDEX`，不走 Docker 镜像源。
- `stop-backend.sh`：按 `.local/run/` 中记录的进程组停止后端，避免只杀掉
  `go run` / `uv run` wrapper 后留下真实服务占用端口。
- `deploy/.env`：本地配置。脚本不生成、不改写、不维护第二套默认值。

## 故障判断

Infra 拉取慢：

- 默认保留 `deploy/.env.example` 里的显式 registry rewrite。
- 已配置 Docker daemon mirror 时，运行 `python3 scripts/check_docker_environment.py --profile all --clean-env`。
- 代理只作为最后选择；shell proxy、daemon proxy 和 registry rewrite 是三条不同路径。

Parser uv 依赖慢：

- 默认保留 `deploy/.env.example` 里的 `UV_DEFAULT_INDEX`。
- 如果公司网络只能访问 PyPI 或自建源，改 `deploy/.env` 里的 `UV_DEFAULT_INDEX`。
- uv 下载的是 Python 包；Docker registry rewrite 不影响它。
- 第一次准备 PaddleOCR extra 会下载几十个包；确认 `services/parser/uv.lock`
  里的 URL 也是清华源，而不是 `pypi.org` 或 `files.pythonhosted.org`。

后端没起来：

- 先看 `.local/logs/<service>.log`。
- Knowledge ingestion 到 embedding/index 阶段失败时，先确认
  `QDRANT_URL`、`QDRANT_COLLECTION` 和 `EMBEDDING_DIMENSION` 与 dev-up 初始化一致。
- Auth、File、Knowledge、QA、Document、AI Gateway 优先查数据库和 migration。
- Gateway 优先查 Redis、Auth URL 和下游服务端口。
- File/Knowledge/Parser 内部调用 `401` 时，检查 `INTERNAL_SERVICE_TOKEN` 是否一致。

WSL 内存高：

- 先看 `docker stats`。
- 当前 Docker 只跑 infra；内存压力主要来自 PostgreSQL、Qdrant、MinIO、Parser OCR 或本机后端进程。
- 不需要保留环境时先停后端，再执行 `docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v`。

```bash
cd ../services/knowledge
GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 \
GATEWAY_BASE_URL='http://127.0.0.1:8080' \
KNOWLEDGE_SERVICE_BASE_URL='http://127.0.0.1:8083' \
FILE_SERVICE_BASE_URL='http://127.0.0.1:8082' \
PARSER_SERVICE_BASE_URL='http://127.0.0.1:8087' \
KNOWLEDGE_TEST_DATABASE_URL='postgres://knowledge_app:knowledge_app_dev@127.0.0.1:5432/knowledge_system?sslmode=disable' \
KNOWLEDGE_REDIS_ADDR='127.0.0.1:6379' \
GATEWAY_SMOKE_USERNAME='admin' \
GATEWAY_SMOKE_PASSWORD='LocalDemoAdmin#12345' \
go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v
```

该测试会：

- `GET /readyz` 检查 File、Parser 和 Knowledge。
- 使用 `KNOWLEDGE_TEST_DATABASE_URL` ping Knowledge PostgreSQL。
- 使用 `KNOWLEDGE_REDIS_ADDR` 发送 Redis `PING`。
- 调用带伪造 `X-User-*` 但无 Bearer token 的 `GET /api/v1/knowledge-bases`，
  断言 Gateway 返回 `401 unauthorized`。
- `POST /api/v1/sessions` 创建真实 Gateway session。
- 带 Bearer token 调用 `GET /api/v1/knowledge-bases`，同时发送一个伪造的
  `X-User-Id`；Gateway 应忽略该伪造 header 并注入 Auth/session cache 中的上下文。
- 通过 Gateway 创建并读取一个 run-scoped knowledge base，断言 `createdBy` 等于真实
  session user，而不是伪造的 `X-User-Id`。

常见失败：

- `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 requires ...`：缺必填 env；失败只列 key，不列值。
- `parser readyz request failed` 或 Docker 报缺 `software-teamwork-local-parser:latest`：
  先按上面的 Parser 镜像构建/缓存前置条件处理。
- Gateway session 返回 `401`：确认 `seed-local` 已完成，并使用
  `.env.example` 中的 `admin` / `LocalDemoAdmin#12345` 或显式
  `GATEWAY_SMOKE_USERNAME` / `GATEWAY_SMOKE_PASSWORD`。
- Gateway Knowledge route 返回 `401`：Gateway session cache/Redis 可能不可用；查
  `docker compose logs gateway redis auth`。
- Gateway Knowledge route 返回 `502`：Knowledge owner route 或 service token 配置异常；
  查 `docker compose logs gateway knowledge` 并用相同 `X-Request-Id` 搜索。

### Gateway -> Knowledge -> QA RAG 端到端 smoke

该 smoke 是 Issue #304 的最小 RAG 验收样例。它通过 Gateway public
`/api/v1/**` 创建 session、创建知识库、上传文档、轮询 Knowledge ingestion、
调用 `knowledge-queries`，再配置 QA 的临时 LLM/retrieval config，创建 QA session/message，
并断言 answer 和 citation 摘要。它不替代 #125 的完整跨服务/MCP smoke。

样例固定在测试代码中：

| 项目 | 值 |
| --- | --- |
| 文档文件名 | `gateway-rag-e2e-smoke.md` |
| 样例问题 | `What calibration marker must be checked for the RAG E2E smoke?` |
| 预期命中 | `calibrate relay RAG-E2E-304` |
| 预期 citation 字段 | `knowledgeBaseId`、`documentId`、`chunkId` 非空且匹配本轮资源；`contentPreview` 或 `text` 包含预期命中。 |

前置要求：

- 根级 Compose 需要带 `--profile ai` 启动，使 AI Gateway、migration 和 placeholder
  profile seed 可用。
- `QA_SETTINGS_OPEN=true` 只建议在本地 smoke 环境启用，用于允许测试通过 Gateway
  创建本轮 QA/LLM config versions。也可改用具备 `qa:settings:write` 权限或
  `QA_ADMIN_USER_IDS` 的账号。
- `QA_SMOKE_CHAT_PROFILE_ID` 和 `QA_SMOKE_CHAT_MODEL` 必须指向 AI Gateway 中可实际
  调用的 chat profile/model。`.env.example` 的 `default-chat` /
  `local-placeholder-chat` 只在 `host.docker.internal:11434/v1` 后面有可用
  OpenAI-compatible provider 时可用；真实 provider 或受控 stub provider 仍需显式配置。
- 默认 Knowledge 使用 local hashing embedding 和 in-memory vector index。需要证明
  Qdrant runtime 查询时，在启动前设置 `KNOWLEDGE_QDRANT_URL=http://qdrant:6333`。
  需要真实 AI Gateway embedding/rerank 时，再设置 `EMBEDDING_PROVIDER=ai_gateway`、
  `KNOWLEDGE_AI_GATEWAY_BASE_URL=http://ai-gateway:8086`、embedding profile/model
  和 `RERANK_MODEL` / `RERANK_PROFILE_ID`。

启动本地栈：

```bash
cd deploy
cp .env.example .env
# 可选：中国大陆 Docker 构建 overlay
# cat .env.china.example >> .env
QA_SETTINGS_OPEN=true DOCKER_BUILDKIT=1 docker compose --env-file .env --profile ai up -d --build gateway ai-gateway
```

运行 smoke：

```bash
cd ../services/knowledge
GATEWAY_RAG_E2E_SMOKE=1 \
GATEWAY_BASE_URL='http://127.0.0.1:8080' \
FILE_SERVICE_BASE_URL='http://127.0.0.1:8082' \
PARSER_SERVICE_BASE_URL='http://127.0.0.1:8087' \
KNOWLEDGE_SERVICE_BASE_URL='http://127.0.0.1:8083' \
QA_SERVICE_BASE_URL='http://127.0.0.1:8084' \
AI_GATEWAY_BASE_URL='http://127.0.0.1:8086' \
KNOWLEDGE_TEST_DATABASE_URL='postgres://knowledge_app:knowledge_app_dev@127.0.0.1:5432/knowledge_system?sslmode=disable' \
KNOWLEDGE_REDIS_ADDR='127.0.0.1:6379' \
GATEWAY_SMOKE_USERNAME='admin' \
GATEWAY_SMOKE_PASSWORD='LocalDemoAdmin#12345' \
QA_SMOKE_CHAT_PROFILE_ID='default-chat' \
QA_SMOKE_CHAT_MODEL='local-placeholder-chat' \
go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v
```

预期结果：

```text
=== RUN   TestGatewayRAGE2ESmoke
--- PASS: TestGatewayRAGE2ESmoke (...s)
PASS
```

测试会创建 run-scoped knowledge base 和文档，并在清理阶段先调用 Gateway
`DELETE /api/v1/documents/{documentId}` 触发 File/vector cleanup，再按
chunks、jobs、documents、knowledge base 的顺序删除本轮 Knowledge PostgreSQL 行。
测试开始前会读取当前 active QA config 和 LLM config，结束时通过 Gateway settings API
重新创建并激活一份等价恢复版本。本轮 smoke 创建的 QA config versions 仍会作为本地运行
记录留存，因为它们是 QA settings 的真实业务资源；只在临时本地栈运行该 smoke。

常见失败和定位：

| 阶段 | 典型失败 | 排查 |
| --- | --- | --- |
| File | `File stage: gateway document upload returned HTTP ...` | 查 `docker compose logs gateway file`，确认 File ready、`INTERNAL_SERVICE_TOKEN` 一致、上传大小未超限。不要打印 multipart body 或 object key。 |
| Parser | `Parser stage: ready document did not record parserBackend` 或文档状态 `failed` | 查 `docker compose logs knowledge parser`，确认 Parser ready、`PARSER_SERVICE_TOKEN` 一致；Markdown fixture 不需要真实 OCR 模型下载。 |
| Knowledge ingestion | `document ... did not become ready` 或 `chunkCount = 0` | 查 `docker compose logs knowledge redis postgres qdrant`；检查 `processing_jobs` 状态、Redis/asynq 投递和 Qdrant/local vector 配置。 |
| Knowledge retrieval | `Knowledge retrieval stage: ...`、无 expected hit 或 rerank trace 异常 | 查 `docker compose logs gateway knowledge ai-gateway qdrant`；确认 `knowledge-queries` 返回 ready 文档 chunk，`rerank=true` 在无 `RERANK_MODEL` 时只证明 no-op fallback trace。 |
| AI Gateway | QA message 返回 `502`、model error 或 provider unavailable | 查 `docker compose logs qa ai-gateway`；确认 chat profile enabled、model exact-match、credential/provider 可用。不要粘贴 provider 原始错误 body 或 API key。 |
| QA | QA config POST `403/400`、answer 未完成、无 citation | 确认 `QA_SETTINGS_OPEN=true` 或账号具备 `qa:settings:write`；确认模型支持 OpenAI-compatible tool/function calling，并且 QA config 只启用 `search_knowledge`。 |

如果只需要验证 Knowledge ingestion 和 retrieval，不要运行本 RAG smoke；先使用上面的
`KNOWLEDGE_INGESTION_SMOKE` 或 `GATEWAY_KNOWLEDGE_OWNER_SMOKE` 缩小范围。

### QA + Auth + Gateway 局部环境

```bash
cd services/qa
docker compose up --build
```

该 Compose 适合验证 Auth、QA、Gateway 的基础 ready 状态和 QA 非 provider 依赖路径：

```bash
curl -fsS http://localhost:8081/readyz
curl -fsS http://localhost:8084/readyz
curl -fsS http://localhost:8080/readyz
```

注意：默认 `AI_GATEWAY_URL` 指向 Compose 网络内的 `http://ai-gateway:8086/internal/v1/chat/completions`，但该 Compose 没有 `ai-gateway` 服务。触发真实 LLM 调用、LLM connection test 或 Agent Run 时，需要额外启动 AI Gateway 并改写 `QA_AI_GATEWAY_URL`。

### Document 局部环境

```bash
cd services/document
docker compose up --build
```

该 Compose 适合验证 Document PostgreSQL、Redis、migration、job enqueue 和 worker 状态机：

```bash
curl -fsS http://localhost:8085/readyz
```

注意：模板、材料和报告文件 bytes 需要 File Service；真实大纲/正文生成需要 AI Gateway。当前基础 DOCX 导出使用 Document 内置 `SimpleDOCXGenerator`，不需要 Pandoc/LibreOffice；Pandoc/LibreOffice 仅是后续富 DOCX worker 工具链。当前 Compose 只给 File/AI Gateway 下游设置 URL，不启动这些下游服务，所以 Document-only 环境不能完整读取生成文件内容。Document worker 会执行 `report_file_creation` 的基础 DOCX 导出；其他大纲/正文生成类 job 仍只完成 job/attempt 状态流转。

### AI Gateway root profile / host-run

根级 `deploy/docker-compose.yml` 的 `--profile ai` 会启动 AI Gateway、执行 migration，并通过
`seed-local-ai` 写入本地 placeholder profile。下面的 host-run 示例用于单独调试服务进程。
AI Gateway 服务 token 运行时只接受 hash：

```bash
TOKEN=dev-internal-service-token-change-me
printf '%s' "$TOKEN" | shasum -a 256 | awk '{print "sha256:" $1}'
```

最小 host-run 环境示例：

```bash
export AI_GATEWAY_HTTP_ADDR=:8086
export AI_GATEWAY_DATABASE_URL='postgres://ai_gateway:ai_gateway@localhost:5436/ai_gateway?sslmode=disable'
export AI_GATEWAY_SERVICE_TOKEN_HASHES='sha256:<token-sha256-hex>'
export AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF=local-dev-key-v1
export AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY='<long-local-secret>'

cd services/ai-gateway
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$AI_GATEWAY_DATABASE_URL" up
go run ./cmd/server
```

创建可调用 profile 后，`/internal/v1/chat/completions`、`/internal/v1/embeddings` 和 `/internal/v1/rerankings` 都会按 profile 解析 provider、model、base URL 和 API key。当前 fake-provider 单元测试覆盖了协议形态；真实 provider smoke 仍需单独执行并记录。

### 报告大纲/正文 AI 生成配置

报告大纲生成（`outline_generation`）和正文生成（`content_generation`）调用 AI Gateway 的
`default-chat` profile。**本地 seed 写入的是 Ollama 占位 URL，不带真实模型，默认不可用。**
拉取最新 develop 后如果点击"生成大纲"一直失败，根本原因就是 AI Gateway 没有指向可用的
LLM provider。

按以下任一方案配置后重试即可。

#### 方案 A：本机 Ollama

先安装 [Ollama](https://ollama.com) 并拉取一个支持对话的模型，例如：

```bash
ollama pull qwen2.5:7b
```

Ollama 默认监听 `http://localhost:11434`，Docker 内通过 `host.docker.internal:11434`
访问。seed 的 `default-chat` profile 已指向该地址，只需把 model 改成你实际拉取的名字：

```bash
# 获取管理员 token（本地默认值）
ADMIN_TOKEN="atk_v1_Z4EHKs54YdxqTTXuYTjBaRnYNL7XO6sIGw4WBd7DMRo"

curl -X PATCH http://localhost:8080/api/v1/admin/model-profiles/default-chat \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"qwen2.5:7b"}'
```

#### 方案 B：OpenAI 兼容 provider（以 DeepSeek 为例）

将 `default-chat` profile 指向 DeepSeek，然后在管理端「模型配置」页面的 Credentials
表单里填入 API key（**不要把 key 写进任何文件或命令行历史**）：

```bash
ADMIN_TOKEN="atk_v1_Z4EHKs54YdxqTTXuYTjBaRnYNL7XO6sIGw4WBd7DMRo"

curl -X PATCH http://localhost:8080/api/v1/admin/model-profiles/default-chat \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"https://api.deepseek.com/v1","model":"deepseek-chat"}'
```

API key 通过管理端 UI 配置（Admin → 模型配置 → default-chat → 更新凭证），
不经过命令行，避免泄漏到 shell 历史。

#### 验证

配置完成后确认 profile 可用：

```bash
# AI Gateway 健康检查
curl -fs http://localhost:8086/readyz && echo "ok"

# 快速冒烟：直接调 chat completions（model 填你实际配置的模型名）
INTERNAL_TOKEN="local-dev-internal-service-token-change-me"
curl -X POST http://localhost:8086/internal/v1/chat/completions \
  -H "X-Service-Token: $INTERNAL_TOKEN" \
  -H "X-Caller-Service: document" \
  -H "Content-Type: application/json" \
  -d '{"profile_id":"default-chat","model":"deepseek-chat","messages":[{"role":"user","content":"hi"}]}'
```

返回包含 `choices` 字段则 provider 正常，之后在前端选择「迎峰度夏检查报告」类型
（`summer_peak_inspection`）重新提交即可触发大纲生成。

> **注意**：当前仅 `summer_peak_inspection` 报告类型支持 AI 大纲/正文生成，其他类型
> 会在 job 执行阶段返回 `unsupported report type` 校验错误。

## 手动联调顺序

完整链路尚未一键化时，建议按下面顺序缩小问题范围：

1. 单服务测试和 build 先通过：`go test ./...`、`go build ./cmd/server`。
2. 对有 migration 的服务执行 goose apply smoke。
3. 启动 Auth、Gateway、目标领域服务和该领域服务的数据库。
4. 需要模型调用时再启动 AI Gateway，并创建对应 `purpose=chat|embedding|rerank` 的 enabled/default profile。
5. 需要文件 bytes 时再启动 File Service；不要让领域服务直接暴露 object key、bucket 或内部 URL。
6. 通过 Gateway public `/api/v1/**` 验证前端可见能力；只在服务间 smoke 中直连 `/internal/v1/**`。

## 冒烟检查清单

| 场景 | 检查 | 当前预期 |
| --- | --- | --- |
| Auth/Gateway/QA 局部环境 | 各服务 `GET /readyz` + 目标 Gateway API smoke | Gateway `/readyz` 只证明 Redis/Auth 和 owner URL 配置；QA `GET /readyz` 证明 QA 进程与自身依赖 ready；真实 AI 调用仍可能因 AI Gateway profile/provider 未配置失败。 |
| Document 局部环境 | 创建 report job 后查询 job/attempt/events | 非文件生成类任务会入队并由 worker 推进为 succeeded；不会生成真实 AI 大纲/正文。若额外提供 File Service，`report_file_creation` 可生成基础 DOCX 并通过 content endpoint 读取成功文件。 |
| AI Gateway profile | 创建 chat/embedding/rerank profile，调用对应内部 endpoint | fake provider 和兼容 provider 应返回 OpenAI-style body；真实 provider 需手工验证。 |
| Gateway contract | `python3 scripts/verify_gateway_active_api.py` | active path、owner、security 和 owner map 不漂移。 |
| Parser PaddleOCR model | `PARSER_PADDLEOCR_SMOKE=1 PARSER_PADDLEOCR_ALLOW_DOWNLOAD=1 uv run pytest -m paddleocr_smoke -s` | 只在本机具备 PaddleOCR extra 和可用模型下载/缓存时运行；验证真实模型加载和最小 fixture OCR 非空。 |
| File PostgreSQL + MinIO | `FILE_MINIO_POSTGRES_SMOKE=1 ... go test ./internal/integration -run TestFileMinIOPostgresSmoke -count=1 -v` | 只在真实 PostgreSQL/MinIO 可用时运行；验证 upload、metadata、content read、delete 和清理状态。 |
| Knowledge ingestion real deps | `KNOWLEDGE_INGESTION_SMOKE=1 ... go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v` | 只在 PostgreSQL/File/Parser/Qdrant 可用时运行；验证 fixture 上传、解析、切片、embedding、Qdrant point 写入和状态更新。 |
| Gateway -> Knowledge owner route | `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v` | 只在 Gateway/Auth/Redis/Knowledge/File/Parser/PostgreSQL 可用时运行；验证伪造 `X-User-*` 未认证请求被拒绝，并用 KB `createdBy` 断言 Gateway 注入真实 session user。 |
| Gateway -> Knowledge -> QA RAG | `GATEWAY_RAG_E2E_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v` | 只在 Gateway/Auth/Redis/File/Parser/Knowledge/QA/AI Gateway 和可用 chat profile/provider 可用时运行；验证上传、ingestion ready、`knowledge-queries` 命中、QA answer 和 citation 摘要。 |
| 前端 Gateway 类型 | `bun run --cwd apps/web api:generate` 后检查 diff | 生成类型应与 Gateway OpenAPI 保持同步。 |

## 已知缺口

| 缺口 | 影响 | 跟踪 |
| --- | --- | --- |
| 根级跨服务 smoke 缺失 | 即使使用 `deploy/docker-compose.yml` 启动本地/演示基线，也不能自动证明 Auth/Gateway/File/Knowledge/QA/Document/AI Gateway 链路可用。 | #125 |
| 跨服务契约测试和 E2E smoke 缺失 | 不能自动证明前端 -> Gateway -> 多服务链路可用。 | #125 |
| Gateway `/readyz` 非完整依赖诊断 | `/readyz` 不请求所有 owner service `/readyz`，也不执行业务级 smoke；它通过不等于 Knowledge/QA/Document/AI Gateway 全链路可用。 | #353、#125、#352 |
| Parser 真实 OCR smoke 不在普通 CI 中运行 | Parser 已有 env-gated 真实 PaddleOCR 模型 smoke，但 CI 仍使用 fake OCR backend；真实模型、OCR 质量和部署资源需要在具备模型的本地或部署环境手动记录。 | #125 |
| Knowledge/QA RAG smoke 仍为显式 opt-in | File 自身 PostgreSQL + MinIO smoke 已有；Knowledge ingestion 真实依赖 smoke 已覆盖 File/Parser/PostgreSQL/Qdrant 写入和状态更新；Gateway -> Knowledge -> QA RAG smoke 已提供最小验收样例，但依赖可用 AI Gateway chat profile/provider，且不覆盖 MCP、前端或 #125 完整一键 E2E。 | #125、#152、#154、#304 |
| 生产部署基线缺失 | 当前 `deploy/docker-compose.yml` 是本地/演示基线，不能直接当生产部署。 | #150 |
| Document 真实 AI 生成和富 DOCX 工具链未落地 | 报告 job 状态机和基础 DOCX 导出可用；真实大纲/正文生成、Pandoc/LibreOffice 富 DOCX 转换和跨服务内容读取 smoke 仍需补齐。 | #160、#223 |
| Document 跨服务 smoke 仍缺失 | settings/statistics/logs 已在服务端落地，但管理端、Gateway、File Service、Document worker 串联 smoke 仍未一键化。 | #159、#221 |
| QA Agent Run MVP 和权限一致性仍在推进 | QA 会话/消息基础可用，完整 Agent 编排和 403 一致性仍需收口。 | #157、#217 |
| 前端业务 E2E 覆盖不足 | 已有 Playwright 基础 smoke；Knowledge、QA、Document 等完整业务流程仍需随页面能力扩展。 | #117、#163 |

## PR 前判断

- 只改文档：至少执行 `git diff --check`，并检查新增链接、相对路径和实现事实。
- 改后端服务：执行对应服务 `go test ./...` 和 `go build ./cmd/server`；QA 还要 `go build ./cmd/agent`。
- 改 migration：执行 goose apply；如果服务有 env-gated repository integration tests，尽量使用本地 PostgreSQL 跑一遍。
- 改 Parser 契约或运行时：检查 `docs/services/parser/api/internal.openapi.yaml`、`services/parser/api/openapi.yaml`（实现本地副本）、Parser README、Knowledge ingestion 文档和 `parser-service.yml` 是否一致；运行 `cd services/parser && uv run ruff check . && uv run pytest && uv run python -m compileall src tests`。如触碰 PaddleOCR runtime 或部署资源，尽量追加 `PARSER_PADDLEOCR_SMOKE=1` 的真实模型 smoke，并在 PR 记录中区分 fake OCR 与真实模型结果。
- 改 Gateway OpenAPI：执行 `python3 scripts/verify_gateway_active_api.py`，前端类型相关改动还要执行 `bun run --cwd apps/web api:generate` 并检查生成 diff。
- 改前端：执行 `bun install --frozen-lockfile`、`bun run --cwd apps/web check`、`bun run --cwd apps/web build`、`bun run --cwd apps/web test:unit`；关键页面改动再跑 `bun run --cwd apps/web test:e2e`。

## Appendix: AI Gateway real-provider cross-service acceptance

Use this checklist only in a protected local or shared environment that can hold
real provider credentials. Do not commit `.env` values, provider keys, service
tokens, full prompts, document text, embedding payloads, or provider raw error
bodies.

1. Start the local AI profile stack:
   `cd deploy && docker compose --profile ai up -d --build`.
2. Check AI Gateway readiness:
   `curl -s http://127.0.0.1:8086/readyz`.
   `missing` means the profile or active credential is absent; `placeholder`
   means the seeded local fake credential is still present; `ok` means a
   non-placeholder credential is configured, not that the external provider has
   accepted it.
3. Create or patch chat, embedding, and rerank profiles through
   `/internal/v1/model-profiles` with `X-Service-Token`,
   `X-Caller-Service`, and `X-Request-Id`. The `model` in smoke requests must
   exactly match the selected profile.
4. Run direct AI Gateway smoke with
   `AI_GATEWAY_REAL_PROVIDER_SMOKE=1`. Chat is the minimum; embedding and rerank
   run only when the provider and env vars support them.
5. Run QA validation by creating a Gateway session, creating a QA session, and
   sending a message through Gateway public `/api/v1/**` or the documented
   `QA_AI_GATEWAY_SMOKE=1` service-client test. Use the same request id to
   search `docker compose logs gateway qa ai-gateway`.
6. Run Document validation for `summer_peak_inspection`: create a report,
   create an `outline_generation` or `content_generation` job, poll jobs,
   events, and sections until terminal state, then search
   `docker compose logs gateway document ai-gateway` by request id. Rich DOCX
   Pandoc/LibreOffice worker validation is not part of this checklist.
7. Run Knowledge validation in the intended mode. The default local path uses
   local hashing embeddings and no-op/empty rerank configuration; do not count
   it as real AI Gateway embedding/rerank. For real provider validation set
   `EMBEDDING_PROVIDER=ai_gateway`,
   `KNOWLEDGE_AI_GATEWAY_BASE_URL=http://ai-gateway:8086`, and the embedding or
   rerank profile/model env vars, then inspect Knowledge and AI Gateway logs by
   request id.

Acceptance record template:

| Area | Command or request | Expected proof | Logs |
| --- | --- | --- | --- |
| AI Gateway | `/readyz` + real provider smoke | profile status is not `placeholder`; smoke succeeds for configured operations | `docker compose logs ai-gateway` |
| QA | session/message or `QA_AI_GATEWAY_SMOKE=1` | answer path reaches AI Gateway and returns normalized response/error | `docker compose logs gateway qa ai-gateway` |
| Knowledge | local hashing or AI Gateway embedding/rerank path | selected path is explicitly named; real provider path has profile/model env | `docker compose logs gateway knowledge ai-gateway qdrant` |
| Document | `summer_peak_inspection` report job flow | job/events/sections reach expected terminal state | `docker compose logs gateway document ai-gateway redis` |
