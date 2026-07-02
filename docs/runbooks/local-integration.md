# 本地联调运行手册

默认联调路径只有一条：

```text
Docker infra -> host backend -> frontend
```

不要启动业务服务容器，不要使用 `--build`，不要手工 export 一长串变量。
`deploy/.env.example` 是默认配置来源；用户复制成 `deploy/.env` 后，脚本只读取它。

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
