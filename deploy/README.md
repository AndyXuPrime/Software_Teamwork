# 本地启动手册

本次更新后，Docker 只负责基础设施：

```text
Docker: postgres + redis + qdrant + minio + minio-init
Host:   auth + file + parser + knowledge + ai-gateway + qa + document + gateway + frontend
```

## 直接启动

先安装 Docker、Go `1.25.x`、uv、Bun、`psql` 客户端和 `curl`。
PostgreSQL server 由 Docker 启动；Parser 的 Python 运行时由 uv 按项目配置处理。

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
Go/uv 进程拿到同一份本地配置。

默认 demo 账号：

```text
admin / LocalDemoAdmin#12345
superadmin / LocalDemoAdmin#12345
```

`deploy/.env.example` 已经内置中国大陆开发网络默认镜像源。需要直连 Docker Hub 时，
从 `deploy/.env` 删除 `POSTGRES_IMAGE`、`REDIS_IMAGE`、`QDRANT_IMAGE`、
`MINIO_IMAGE` 和 `MINIO_MC_IMAGE` 这几行即可回到 Compose 里的 Docker Hub pinned tags。

`UV_DEFAULT_INDEX` 控制宿主机 uv 在解析或重锁依赖时使用的 Python 包索引，默认使用
清华 PyPI 镜像以加速 Parser 首次准备 PaddleOCR 依赖。当前 `services/parser/uv.lock`
也锁定到清华源；`run-backend.sh` 使用 `uv sync --frozen`，所以删除
`deploy/.env` 里的 `UV_DEFAULT_INDEX` 不会让已锁定的包下载 URL 切回官方 PyPI。
默认路径需要保留清华源基线。无法访问清华源的环境应先按网络/代理路径解决；如必须
使用 PyPI 或自建源，需要用同一索引重新生成 `services/parser/uv.lock`，并在合并前
重锁回清华源，否则 `scripts/verify_local_seed_contract.py` 会报错。首次准备 OCR
runtime 仍会下载几十个 Python 包。

## 脚本职责

`./scripts/local/dev-up.sh`：

- 校验 `deploy/docker-compose.yml`。
- 拉取并启动 `postgres`、`redis`、`qdrant`、`minio`、`minio-init`，并等待
  Compose health checks 通过。
- 创建或校验 `QDRANT_COLLECTION`，向量维度来自 `EMBEDDING_DIMENSION`。
- 在宿主机执行各服务 goose migration。
- 用 `psql` 依次应用本地 demo 数据、AI Gateway profile 和 QA Document MCP
  注册 seed。Document MCP seed 只保存 endpoint/alias 等非敏感元数据；token 来自
  `deploy/.env` 的 `MCP_SERVER_TOKEN`。

`./scripts/local/run-backend.sh`：

- 用 uv 准备 Parser 运行依赖，包括 PaddleOCR extra；Python 包下载走
  `deploy/.env` 里的 `UV_DEFAULT_INDEX`，不是 Docker registry。
- 按顺序启动 `auth`、`file`、`parser`、`knowledge`、`ai-gateway`、`qa`、`document`、`gateway`。
- 日志写入 `.local/logs/`，进程组 leader PID 写入 `.local/run/`，供
  `stop-backend.sh` 停掉 `go run` / `uv run` 及其子进程。

## 快速确认

```bash
curl --noproxy '*' -fsS http://localhost:8080/healthz
curl --noproxy '*' -fsS http://localhost:8080/readyz
```

`http://localhost:8086/readyz` 在 placeholder profile 下返回 `503 degraded` 是预期行为，
表示还没配置真实模型 provider credential，不代表 AI Gateway 进程失败。
默认本地模型 profile 的 OpenAI-compatible 地址是 `http://localhost:11434/v1`。
Document MCP 的默认 endpoint 是 `http://localhost:8085/mcp`，QA 将其工具暴露为
`document__<tool>`；完整工具参数和 Agent 工作流见
[Document MCP 工具契约](../docs/services/document/docs/mcp-tools.md)。

## 排障入口

- Docker 拉取慢、registry rewrite、daemon mirror、proxy 和 WSL 内存：
  [docs/runbooks/docker-image-pull-environment.md](../docs/runbooks/docker-image-pull-environment.md)
- 本地联调顺序、端口和故障判断：
  [docs/runbooks/local-integration.md](../docs/runbooks/local-integration.md)
