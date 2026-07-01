# 本地启动手册

本地默认路径分两层：

```text
Docker: postgres + redis + qdrant + minio + minio-init
Host:   auth + file + knowledge + ai-gateway + qa + document + gateway + frontend
```

`services/parser` 已由 Knowledge 的 RAGFlow runtime 方案替代，不再作为本地后端服务启动。

## 直接启动

先安装 Docker、Go `1.25.x`、Bun、`psql` 客户端和 `curl`。

```bash
cp deploy/.env.example deploy/.env
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh

cd apps/web
bun install
bun run dev
```

日常再次启动时，已经执行过 `bun install` 可以直接：

```bash
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh
cd apps/web && bun run dev
```

停止后端：

```bash
./scripts/local/stop-backend.sh
```

清空本地 infra 数据：

```bash
./scripts/local/stop-backend.sh
docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v
```

## 配置来源

`deploy/.env.example` 是唯一默认配置来源。用户只复制一次：

```bash
cp deploy/.env.example deploy/.env
```

脚本不会生成、改写或维护另一套默认变量。它们只读取 `deploy/.env`，让宿主机
Go 进程拿到同一份本地配置。

默认 demo 账号：

```text
admin / LocalDemoAdmin#12345
superadmin / LocalDemoAdmin#12345
```

`deploy/.env.example` 已经内置中国大陆开发网络默认镜像源。需要直连 Docker Hub 时，
从 `deploy/.env` 删除 `POSTGRES_IMAGE`、`REDIS_IMAGE`、`QDRANT_IMAGE`、
`MINIO_IMAGE` 和 `MINIO_MC_IMAGE` 这几行即可回到 Compose 里的 Docker Hub pinned tags。

## 脚本职责

`./scripts/local/dev-up.sh`：

- 校验 `deploy/docker-compose.yml`。
- 拉取并启动 `postgres`、`redis`、`qdrant`、`minio`、`minio-init`，并等待
  Compose health checks 通过。
- 如果 `QDRANT_URL` 已设置，则创建或校验 `QDRANT_COLLECTION`。
- 在宿主机执行各服务 goose migration。
- 用 `psql` 依次应用本地 demo 数据、AI Gateway profile 和 QA Document MCP
  注册 seed。Document MCP seed 只保存 endpoint/alias 等非敏感元数据；token 来自
  `deploy/.env` 的 `MCP_SERVER_TOKEN`。

`./scripts/local/run-backend.sh`：

- 按顺序启动 `auth`、`file`、`knowledge`、`ai-gateway`、`qa`、`document`、`gateway`。
- Knowledge 运行 `cmd/adapter`，并通过 `VENDOR_RUNTIME_URL` 调用 RAGFlow runtime。
- 日志写入 `.local/logs/`，进程组 leader PID 写入 `.local/run/`，供
  `stop-backend.sh` 停掉 `go run` 及其子进程。

## Knowledge / RAGFlow

Knowledge 文档上传、解析、切块、embedding、索引和检索通过 RAGFlow runtime 完成。
本地 Knowledge adapter 读取：

```text
VENDOR_RUNTIME_URL=http://host.docker.internal:9380
KNOWLEDGE_AUTO_START_INGESTION=true
DOC_ENGINE=elasticsearch
```

如果 runtime 以 Compose profile 运行，Compose 网络内默认地址应指向
`http://knowledge-runtime-api:9380`；如果 runtime 在宿主机运行，则使用
`http://127.0.0.1:9380` 或 Docker host gateway 可达地址。不要再启动
`services/parser`。

## 快速确认

```bash
curl --noproxy '*' -fsS http://localhost:8080/healthz
curl --noproxy '*' -fsS http://localhost:8080/readyz
curl --noproxy '*' -fsS http://localhost:8083/readyz
```

`http://localhost:8086/readyz` 在 placeholder profile 下返回 `503 degraded` 是预期行为，
表示还没配置真实模型 provider credential，不代表 AI Gateway 进程失败。
默认本地模型 profile 的 OpenAI-compatible 地址是 `http://localhost:11434/v1`。
Document MCP 的默认 endpoint 是 `http://localhost:8085/mcp`，QA 将其工具暴露为
`document__<tool>`；完整工具参数和 Agent 工作流见
[Document MCP 工具契约](../docs/services/document/docs/mcp-tools.md)。

## Seed Data

`dev-up.sh` 应用 `deploy/seeds/001-local-demo-seed.sql` 和
`deploy/seeds/002-ai-gateway-model-profiles.sql`。

Seeded local resources:

| Area | Deterministic resource |
| --- | --- |
| Auth | user `usr_local_admin`, username `admin`, password `LocalDemoAdmin#12345`, role `admin` |
| Auth permissions | `admin:model-profile:write`, `admin:parser-config:write`, `qa:settings:read`, and `qa:settings:write` |
| Knowledge | knowledge base `kb_local_demo`, document `doc_local_demo_seed`, chunk `chunk_local_demo_seed_001` |
| Document | material `22222222-2222-4222-8222-222222222201`, report `22222222-2222-4222-8222-222222222301`, outline `22222222-2222-4222-8222-222222222401` |
| QA | conversation `33333333-3333-4333-8333-333333333301`, user message `33333333-3333-4333-8333-333333333401`, assistant message `33333333-3333-4333-8333-333333333402` |
| AI Gateway | optional placeholder profiles `default-chat`, `default-embedding`, and `default-rerank` |

## 排障入口

- Docker 拉取慢、registry rewrite、daemon mirror、proxy 和 WSL 内存：
  [docs/runbooks/docker-image-pull-environment.md](../docs/runbooks/docker-image-pull-environment.md)
- 本地联调顺序、端口和故障判断：
  [docs/runbooks/local-integration.md](../docs/runbooks/local-integration.md)

## Common Dependency Failures

| Symptom | Likely cause | Check |
| --- | --- | --- |
| `gateway /readyz` returns `503 dependency_error` | Redis, auth, or required owner service base URL configuration is not ready | `docker compose ps`, service logs under `.local/logs/` |
| `auth /readyz` returns `postgres unavailable` | Auth migration or PostgreSQL failed | `docker compose logs postgres`; check `.local/logs/auth.log` |
| Knowledge upload/query returns `502 dependency_error` | RAGFlow runtime unreachable or ES/MinIO not ready | Check `VENDOR_RUNTIME_URL`, runtime API, worker, Elasticsearch, and MinIO |
| QA message call fails on model invocation | AI Gateway profile is not running, fake local credential is still in use, or host provider is not listening on `host.docker.internal:11434` | Check `.local/logs/ai-gateway.log` and `.local/logs/qa.log` |
| MinIO bucket missing | `minio-init` did not complete | `docker compose logs minio minio-init` |
| Host port conflict | Another local process uses a default port | Change the matching `*_PORT` in `deploy/.env` |
