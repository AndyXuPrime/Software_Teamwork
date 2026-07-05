# 本地联调运行手册

默认本地完整联调路径是 host-run：

```text
Docker infra (postgres, redis, minio, minio-init, elasticsearch)
  -> host backend -> frontend
Real parsing -> host Knowledge runtime API + on-demand worker
```

在这条默认 host-run 路径中，不要启动业务服务容器，不要使用 `--build`，不要手工 export 一长串变量。
`config/` 是默认配置来源；用户从根 `.env.example` 复制未跟踪的 `.env.local` 后，
脚本通过 `config/ctl` 渲染 `.local/config/<profile>.env` 和 `.env.sh`。

如果本机压力过大，需要把数据库、Redis、对象存储、Knowledge runtime、OCR 和模型 provider 都放到云端，使用独立 Docker app stack：

```text
Docker app stack (auth, file, knowledge, qa, document, ai-gateway, gateway, web)
  -> cloud PostgreSQL / Redis / object storage
  -> cloud Knowledge runtime
  -> PaddleOCR cloud parser
  -> cloud model provider
```

这条路径入口是 `./scripts/docker/start.sh`，不使用根级 `deploy/docker-compose.yml`，也不启动本地 Knowledge runtime worker 或本地 OCR。

Issue #125 的 MCP 与跨服务 smoke 汇总入口见
[`Issue #125 MCP and Cross-Service Smoke`](./issue-125-smoke.md)。只有对应 slice
在当前环境实际通过后，才能在 PR 或验收记录中声明完整跨服务 smoke 通过。
Auth/Gateway/Redis 的完整本地 smoke 可直接执行：
`bash scripts/run_issue_352_smoke.sh`。

## 启动命令

先确认 Go 安装在实际运行脚本的环境中。WSL、Git Bash 和 Windows PowerShell 的
`go env` 互不等价；在哪个 shell 里运行脚本，就在那里检查：

```bash
go version
go env GOPROXY
```

默认使用官方源。中国大陆网络如果访问 GitHub、Docker Hub、PyPI、HuggingFace 或
Go modules 不稳定，运行 `./scripts/local/start.sh --china`；不要改写 `config/` 或
`.env.local`。`start.sh` 会先做 preflight，确认 `.env.local` 已存在但不会创建或覆盖它；
之后再按需准备缺失的本机工具、镜像、服务二进制、runtime `.venv` 和 artifact，长耗时
下载/构建会持续输出进度或心跳。

```bash
cp .env.example .env.local
./scripts/local/start.sh
cd apps/web && bun install && bun run dev
```

中国大陆网络：

```bash
cp .env.example .env.local
./scripts/local/start.sh --china
cd apps/web && bun install && bun run dev
```

日常再次启动：

```bash
./scripts/local/start.sh
cd apps/web && bun run dev
```

如果只想启动 infra、migration 和 seed，不启动后端服务：

```bash
./scripts/local/start.sh --infra-only
```

真实 Knowledge 解析、embedding、索引和检索链路需要 provider 配置。先在本地
`.env.local` 中填写 provider；Elasticsearch 是默认本地 infra，由
`./scripts/local/start.sh` 启动：

```text
KNOWLEDGE_AUTO_START_INGESTION=true
DOC_ENGINE=elasticsearch
KNOWLEDGE_RUNTIME_ES_URL=http://127.0.0.1:9200
```

```bash
./scripts/local/start.sh
```

中国大陆网络运行真实解析栈时，给启动命令加 `--china`：

```bash
./scripts/local/start.sh --china
```

停止：

```bash
# 在运行 bun run dev 的终端按 Ctrl-C 停前端。
./scripts/local/stop.sh
```

`stop.sh` 只停止 `.local/run/` 里记录的后端和 Knowledge runtime 进程组；Vite dev
server 是前台进程，需要在对应终端按 `Ctrl-C`。
如果 `bun run dev` 提示 `Port 5173 is in use`，用
`lsof -nP -iTCP:5173 -sTCP:LISTEN` 找到旧前端进程后停止，再重新运行前端命令。

重置本地数据：

```bash
./scripts/local/clean.sh
```

`clean.sh` 会停止 `.local/run/` 下记录的宿主机后端进程、渲染本地配置并校验 Compose，
然后执行根级 infra Compose 数据卷清理。默认需要输入 `clean` 确认；自动化或确定要
直接清空时使用 `./scripts/local/clean.sh --yes`。脚本底层执行
`docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env down -v --remove-orphans`；
不删除 Docker 镜像、`.env.local`、`.local/tools` 或 `.local/bin`。

从空数据状态复现启动问题时，按项目范围停止服务并清理 compose 数据卷；默认不删除镜像：

```bash
./scripts/local/clean.sh --yes
```

只有在确认镜像 tag、registry rewrite、daemon mirror 或本机镜像缓存本身是排障对象时，才额外删除项目镜像：

```bash
set -a
source .local/config/dev.env.sh
set +a

docker rmi \
  "${POSTGRES_IMAGE:-postgres:16-alpine}" \
  "${REDIS_IMAGE:-redis:7-alpine}" \
  "${MINIO_IMAGE:-minio/minio:RELEASE.2025-09-07T16-13-09Z}" \
  "${MINIO_MC_IMAGE:-minio/mc:RELEASE.2025-08-13T08-35-41Z}" \
  "${KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE:-docker.elastic.co/elasticsearch/elasticsearch:8.15.3}" \
  2>/dev/null || true
```

## 云端依赖 Docker app stack

这条路径适合本机 CPU/内存不足，或希望 OCR/解析/模型全部走云端时使用。它构建并运行业务服务容器和前端容器，但不启动本地 infra、Knowledge runtime 或本地 OCR。

一次性准备：

```bash
cp deploy/docker/cloud.env.example .env.docker.cloud
```

编辑 `.env.docker.cloud`，替换所有非 seed 的 `<...>` 占位值。最少需要云端 PostgreSQL、Redis、对象存储、Knowledge runtime token 和服务 token。模板默认 `DOCKER_SEED_ENABLED=false`，因此 PaddleOCR token 和模型 provider key/model 只在显式启用 seed 时必填。Document 的 Redis/asynq 连接支持 `DOCUMENT_REDIS_USERNAME`、`DOCUMENT_REDIS_PASSWORD`、`DOCUMENT_REDIS_DB` 和 `DOCUMENT_REDIS_TLS_ENABLED`，托管 Redis 要求 AUTH 或 TLS 时不要只填写 host:port。启动脚本会在发现关键占位值或本地 demo secret 未替换时直接失败，避免先构建镜像再在容器里报配置错。

启动：

```bash
./scripts/docker/start.sh
```

浏览器入口：

```text
http://localhost:18080
```

查看日志：

```bash
docker compose -f deploy/docker-compose.cloud.yml --env-file .env.docker.cloud logs -f
```

停止和清理：

```bash
./scripts/docker/stop.sh
./scripts/docker/clean.sh --yes
```

云端 Docker path 的 migration 和 seed 在容器内执行。默认 `DOCKER_SEED_ENABLED=false`，seed 容器直接成功退出，并允许省略 `POSTGRES_ADMIN_URL`、`PADDLEOCR_ACCESS_TOKEN` 和 `AI_GATEWAY_LOCAL_PROVIDER_*`。如果临时云库确实要写入本地 demo seed，显式设置 `DOCKER_SEED_ENABLED=true`；此时 `POSTGRES_ADMIN_URL` 用于静态 seed 中的 `\connect`，`AI_GATEWAY_LOCAL_SEED_ENABLED=true` 会写入云端模型 provider profile，`PADDLEOCR_ACCESS_TOKEN` 会把默认 parser config 切到 `paddleocr_cloud`。启动脚本和 seed 容器都会拒绝 `local-dev-*`、`local-demo-*`、`change-me` 和 `<...>` 占位 secret。Seed 关闭时，QA/Document 不会从 `AI_GATEWAY_LOCAL_CHAT_MODEL` 回填运行时模型；需要覆盖 profile model 时显式设置 `MODEL_ID` 或 `DOCUMENT_AI_GATEWAY_MODEL`。托管 Redis 需要 ACL/TLS 时，同时配置 Gateway 的 `GATEWAY_REDIS_USERNAME` / `GATEWAY_REDIS_TLS_ENABLED` 和 Document 的 `DOCUMENT_REDIS_USERNAME` / `DOCUMENT_REDIS_TLS_ENABLED`。

不要把 `.env.docker.cloud` 提交到仓库。它不经过 `config/ctl` 渲染，也不会被 `./scripts/local/start.sh` 使用。

## 启动后应该看到什么

- 前端：`http://localhost:5173`
- Gateway：`http://localhost:8080`
- 云端 Docker app stack 前端：`http://localhost:18080`
- 本机 OpenAI-compatible 模型服务默认地址：`http://localhost:11434/v1`
- 默认 demo 管理员：`admin` / `LocalDemoAdmin#12345`
- 默认 demo 超管：`superadmin` / `LocalDemoAdmin#12345`
- 后端日志：`.local/logs/*.log`
- 后端进程组 PID：`.local/run/*.pid`

快速确认：

```bash
curl --noproxy '*' -fsS http://localhost:8080/healthz
curl --noproxy '*' -fsS http://localhost:8080/readyz
```

`ai-gateway /readyz` 在 placeholder profile 下返回 `503 degraded` 是预期行为。
这表示真实 provider credential 尚未配置，不表示进程失败。

## Document MCP 注册与工具发现

`start.sh` 在 QA 数据库插入 `mcp_servers.alias=document` 的非敏感元数据；`.env.local`
提供 `MCP_SERVER_TOKEN`。`start.sh` 在宿主机启动 Document `http://localhost:8085/mcp`
和 QA，根级 Compose 不启动这两个业务服务容器。

确认 seed：

```bash
psql "$QA_DATABASE_URL" -c \
  "select alias, transport, endpoint_url, token_encrypted is null as token_from_env, enabled from mcp_servers where alias = 'document';"
```

fresh local volume 的预期值为 `document`、`streamable_http`、
`http://localhost:8085/mcp`、`token_from_env=true`、`enabled=true`。已有管理员配置不会被
seed 覆盖；同名记录被显式禁用时，环境 bootstrap 也不会把它重新启用。

运行现有 9 个工具的发现与调用 smoke：

```bash
cd services/qa
QA_DOCUMENT_MCP_SMOKE=1 \
go test ./internal/platform/mcpclient -run '^TestDocumentMCPReportToolsSmoke$' -count=1 -v
```

该 smoke 验证 `tools/list`、`document__` 前缀、默认白名单、生成 job accepted/status、
DOCX export/result 安全摘要及无权限结果。真实模型 provider 仍不可用时，可以验证 MCP
client 与 Document 工具，不代表完整 QA Agent + LLM 链路通过。Issue #510 的
`document__generate_report_from_content` 合并前不在当前 9 工具验收范围内。

常见失败：

| 现象 | 排查 |
| --- | --- |
| QA 日志 `connection failed` | 检查 Document 已启动、`MCP_SERVER_URL=http://localhost:8085/mcp`。 |
| Document 返回 `401` | 检查 `MCP_SERVER_TOKEN` 与 `DOCUMENT_MCP_SERVICE_TOKEN`/`INTERNAL_SERVICE_TOKEN` 一致，header 均为 `Authorization`。 |
| `tools/list` 成功但模型看不到工具 | 检查 active QA config 的 `enabledToolNames`；默认只放行 5 个常用 `document__...` 工具。 |
| job 一直 pending/running | 查 `.local/logs/document.log`、Redis/asynq worker、AI Gateway profile；不要在 MCP HTTP 调用中做无界等待。 |

精确工具参数、返回结构和 Agent 多轮流程见
[Document MCP 工具契约](../services/document/docs/mcp-tools.md)。

## 谁负责什么

- `start.sh`：标准本地入口。它先检查 Docker、Go、Python、uv、psql、curl 等宿主机环境
  和 Go 版本，并确认 `.env.local` 已由用户创建但不改写；再按需准备 `.local/tools`、
  `.local/bin`、Docker infra images、Knowledge runtime `.venv` 和 runtime artifact。Knowledge
  runtime `.venv` 会按 `--runtime` 模式校验 dependency profile，默认 full 模式会补齐 worker
  group，并校验 `pyproject.toml`、`uv.lock` 和 `download_deps.py` fingerprint。Go 本地产物会用
  `.local/stamps/` 记录源码 fingerprint；存在且匹配才跳过，源码变化会自动重建，`--skip-prepare`
  遇到过期产物会失败。Go 构建/安装、Docker pull、uv sync 和模型/artifact 下载都会输出
  原生命令进度或周期性心跳。
- `start.sh` 会渲染 `.local/config/<profile>.env`，等待 `postgres` / `redis` / `minio` /
  `elasticsearch` health checks，单独运行一次性 `minio-init`，执行 migration 和 demo seed，
  再启动 host-run 后端进程组。Compose 启动仍使用 `--pull never`，镜像拉取只发生在
  prepare 阶段发现所选镜像缺失时。
- `start.sh` 默认启动 Knowledge runtime API + worker，适合验证 PDF 上传、解析、切块、
  embedding、索引和检索；`--runtime api` 只启动 API，`--runtime none` 跳过 runtime。
  脚本只会自动把 loopback runtime URL 加入 `NO_PROXY`，并对本机 health check 使用
  `curl --noproxy '*'`。外部 runtime URL 继续尊重宿主机代理环境。
- 官方默认 Go module 设置是 `https://proxy.golang.org,direct` / `sum.golang.org`。
  `start.sh --china` 会在本次 Go 工具/服务构建和 goose 安装中使用
  `GOPROXY=https://goproxy.cn,direct` / `GOSUMDB=sum.golang.google.cn`。这不是 Docker
  镜像源，也不是 Knowledge runtime 的 `UV_DEFAULT_INDEX`。长期企业 Go 源可以写在
  `.env.local`；`start.sh` 会在构建 config renderer、安装 goose 和构建 seed helper 前先读取
  `GOPROXY`、`GOSUMDB`、`GOPRIVATE`、`GONOPROXY`、`GONOSUMDB` 和 `GOINSECURE`。
- `stop.sh`：按 `.local/run/` 中记录的进程组停止后端和 runtime，避免留下真实服务占用端口。
- `clean.sh`：停止 host-run 进程并删除本地 infra Compose 数据卷；不删除 Docker images、
  `.env.local`、`.local/tools` 或 `.local/bin`。
- `scripts/docker/start.sh`：校验 `.env.docker.cloud` 后对 `deploy/docker-compose.cloud.yml`
  执行 `up -d --build`，用于云端依赖 Docker app stack；seed 默认关闭，显式启用
  seed 时拒绝本地 demo secret。
- `scripts/docker/stop.sh` / `scripts/docker/clean.sh`：只管理 `deploy/docker-compose.cloud.yml`
  创建的 app stack 容器和本地卷，不停止 host-run 后端，也不清理根级 infra 数据卷。
- `config/`：默认配置和环境 profile；根 `.env.example`：本地 secret 模板；
  `.env.local`：未跟踪本地配置。脚本只渲染 `.local/config/` 运行时产物。
- 这些本地入口脚本都必须在命令行输出彩色的开始、成功、警告和失败摘要。失败摘要应
  说明当前阶段、退出码和下一步排查入口，不能只靠用户自己翻日志猜状态。`NO_COLOR=1`
  可关闭颜色，`FORCE_COLOR=1` 可在非 TTY 中强制开色。

## 故障判断

Infra 拉取慢：

- 默认是 Compose 里的 Docker Hub pinned tags。
- 中国大陆网络运行 `./scripts/local/start.sh --china`，脚本会在本次运行的生成态
  `.local/config/dev.env` 中使用 `docker.1ms.run` registry rewrite，并在镜像缺失时拉取
  这组镜像；`.env.local` 不会被脚本改写。
- 已配置 Docker daemon mirror 时，运行 `python3 scripts/check_docker_environment.py --profile all --clean-env`。
- 代理只作为最后选择；shell proxy、daemon proxy 和 registry rewrite 是三条不同路径。
  验证官方 Docker Hub 路径经 shell/Docker 代理可达时，运行
  `python3 scripts/check_docker_environment.py --profile default`，不要加 `--clean-env`。

Knowledge runtime 启动慢：

- 默认 `UV_DEFAULT_INDEX=https://pypi.org/simple`，`start.sh` 准备 runtime 依赖时使用官方
  PyPI。中国大陆网络运行 `./scripts/local/start.sh --china`，脚本会把本次 runtime
  准备切到中国大陆镜像路径；提交的 `pyproject.toml` / `uv.lock` 仍保持官方 URL。
- `HF_ENDPOINT` 用于 runtime worker 首次导入 deepdoc OCR/vision 模块时下载
  `InfiniFlow/deepdoc`。提交的默认配置不启用第三方 mirror；中国大陆网络运行真实解析
  栈时使用 `./scripts/local/start.sh --china`，脚本会在本次进程内为
  未设置 `HF_ENDPOINT` 的环境使用 `https://hf-mirror.com`。企业环境可只在本机
  `.env.local` 中设置内部 HuggingFace 镜像。
- runtime API 走宿主机启动；worker 仅在真实 ingestion smoke 或生产调度时启动，
  不通过根级 Docker Compose 构建或运行。
- 只验证查询链路时优先使用：

  ```bash
  ./scripts/local/start.sh --runtime api
  ```

- 需要真实 PDF 解析链路时优先使用：

  ```bash
  # First set provider vars in .env.local; Elasticsearch starts with default infra.
  ./scripts/local/start.sh
  python3 scripts/local/knowledge-pdf-e2e.py /path/to/DL_T_673-1999.pdf
  ```

  如果 runtime API 已在容器网络中运行但没有映射 `9380` 到宿主机，显式传入容器 bridge
  地址后启动后端即可。该地址不是 loopback，脚本不会自动加入 `NO_PROXY`；如果本机代理
  会截获私网地址，请在本机显式补 `NO_PROXY`：

  ```bash
  export NO_PROXY="${NO_PROXY:-localhost,127.0.0.1,::1},172.22.0.6"
  VENDOR_RUNTIME_URL=http://172.22.0.6:9380 \
    ./scripts/local/start.sh --backend-only --no-runtime
  ```

- 默认保留 `.env.example` 里的 `ENABLE_TIMEOUT_ASSERTION=1`，让 runtime
  worker 的解析、embedding 和存储调用超时保护生效。
- 不要恢复 `services/parser`；PDF 解析、切块、embedding、索引和检索由 Knowledge
  runtime worker 完成。

Go modules 下载慢或超时：

- `start.sh` 会在准备缺失 Go tools、`goose@v3.27.0` 和 host-run 服务二进制时下载 Go
  modules。默认保留 `.env.example` 里的官方
  `GOPROXY=https://proxy.golang.org,direct` 和 `GOSUMDB=sum.golang.org`；中国大陆网络使用
  `./scripts/local/start.sh --china`，脚本会在本次 Go 准备阶段设置
  `GOPROXY=https://goproxy.cn,direct` 和 `GOSUMDB=sum.golang.google.cn`。
  企业网络可以把长期 Go 源覆盖写入 `.env.local`，这些值会在首次构建 config renderer/goose
  之前生效。
- `.env.local` 如果设置镜像值，脚本会尊重本地覆盖并提示；
  若 proxy 或 checksum DB 不可达或下载超时，脚本会在终端直接失败并打印当前有效
  `GOPROXY` / `GOSUMDB`，而不是只把错误藏在 `.local/logs/*.log`。
- Go modules 下载不走 Docker registry rewrite，也不受 `UV_DEFAULT_INDEX` 影响。
- 如果 Go 构建出现 `Get "https://proxy.golang.org/...": i/o timeout`，
  中国大陆网络先运行 `./scripts/local/start.sh --china`；其他网络检查企业代理
  或本机 Go 配置。
- `.env.local` 不会被脚本自动改写；想恢复官方默认值，重新复制
  `.env.example` 后再恢复本机私有配置。
- 如果需要把镜像配置持久写入当前 shell 使用的 Go 全局配置，在运行脚本的同一个环境中执行：

  ```bash
  go env -w GOPROXY=https://goproxy.cn,direct
  go env -w GOSUMDB=sum.golang.google.cn
  ```

- 之后重新运行 `./scripts/local/stop.sh` 和 `./scripts/local/start.sh`，
  再用 `curl --noproxy '*' -fsS http://localhost:8080/healthz` 验证 Gateway。

后端没起来：

- 先看 `start.sh` 命令行失败摘要；它会说明 infra、migration、seed、runtime、服务启动
  或短窗口进程检查中哪一步失败。
- 先看 `.local/logs/<service>.log`。
- Knowledge ingestion 到 embedding/index 阶段失败时，先确认 `VENDOR_RUNTIME_URL`
  指向可访问的 runtime API，并检查宿主机 runtime worker 是否在处理任务。
- Knowledge adapter 已启动但 runtime 或 worker 状态不清楚时，可通过内部诊断口查看
  runtime URL、API ping 和 worker heartbeat。该诊断口只要求内部服务 token，不代表
  终端用户登录态：

  ```bash
  curl --noproxy '*' -fsS \
    -H "X-Service-Token: ${INTERNAL_SERVICE_TOKEN:-local-dev-internal-service-token-change-me}" \
    http://127.0.0.1:8083/internal/v1/runtime/status
  ```

- Auth、File、Knowledge、QA、Document、AI Gateway 优先查数据库和 migration。
- Gateway 优先查 Redis、Auth URL 和下游服务端口。
- File/Document/QA 内部 file 调用 `401` 时，检查 `INTERNAL_SERVICE_TOKEN`
  是否一致，以及启用 File caller allowlist 后是否传递了
  `X-Caller-Service`。出现 `403` 时，检查
  `FILE_ALLOWED_CREATE_CALLERS`、`FILE_ALLOWED_READ_CALLERS`、
  `FILE_ALLOWED_DELETE_CALLERS` 是否包含实际调用方。Knowledge runtime
  调用 `401` 时，检查 `VENDOR_RUNTIME_SERVICE_TOKEN` 与
  `KNOWLEDGE_RUNTIME_SERVICE_TOKEN` 是否一致。经 Gateway/Knowledge 返回给
  浏览器的这类 runtime 认证失败应是 `502 dependency_error`，不是用户 session
  失效；前端不应因此掉登录。
- 如果某个服务日志显示 `bind: address already in use`，用
  `lsof -nP -iTCP:<port> -sTCP:LISTEN` 查占用者。旧的 host-run 或 `tmux`
  会话可能仍在监听默认端口；先停止旧进程，或在 `.env.local` 修改对应端口后
  重新启动。

WSL 内存高：

- 先看 `docker stats`。
- 当前默认 Docker 只跑 infra；内存压力主要来自 PostgreSQL、Redis、MinIO、
  Elasticsearch、宿主机 Knowledge runtime 或本机后端进程。
- 不需要保留环境时执行 `./scripts/local/clean.sh`；确认后会停止宿主机后端并删除本地 infra 数据卷。

```bash
cd ../services/knowledge
GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 \
GATEWAY_BASE_URL='http://127.0.0.1:8080' \
KNOWLEDGE_SERVICE_BASE_URL='http://127.0.0.1:8083' \
VENDOR_RUNTIME_URL='http://127.0.0.1:9380' \
KNOWLEDGE_TEST_DATABASE_URL='postgres://knowledge_app:knowledge_app_dev@127.0.0.1:5432/knowledge_system?sslmode=disable' \
KNOWLEDGE_REDIS_ADDR='127.0.0.1:6379' \
GATEWAY_SMOKE_USERNAME='admin' \
GATEWAY_SMOKE_PASSWORD='LocalDemoAdmin#12345' \
go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v
```

该测试会：

- `GET /readyz` 检查 Gateway、Knowledge 和 Knowledge runtime 可达性。
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
- `vendor ping failed` 或 Knowledge readiness 返回 `vendor_runtime_ok=false`：
  先确认 `VENDOR_RUNTIME_URL` 指向 `http://127.0.0.1:9380`，runtime API 已在宿主机启动。
- Gateway session 返回 `401`：确认 `./scripts/local/start.sh` 已完成 seed SQL，并使用
  `.env.example` 中的 `admin` / `LocalDemoAdmin#12345` 或显式
  `GATEWAY_SMOKE_USERNAME` / `GATEWAY_SMOKE_PASSWORD`。
- Gateway Knowledge route 返回 `401`：Gateway session cache/Redis 可能不可用；查
  `.local/logs/gateway.log`、`.local/logs/auth.log`，并确认 Redis infra 已启动。
- Gateway Knowledge route 返回 `502`：Knowledge owner route 或 service token 配置异常；
  查 `.local/logs/gateway.log` 和 `.local/logs/knowledge.log` 并用相同 `X-Request-Id` 搜索。

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

- 已按本手册“启动命令”使用 `./scripts/local/start.sh` 启动 infra、migration、seed 和
  宿主机后端服务。
- `QA_SETTINGS_OPEN=true` 只建议在本地 smoke 环境启用，用于允许测试通过 Gateway
  创建本轮 QA/LLM config versions；需要在启动后端前写入 `.env.local`。
  也可改用具备 `qa:settings:write` 权限或 `QA_ADMIN_USER_IDS` 的账号。
- `QA_SMOKE_CHAT_PROFILE_ID` 必须指向 AI Gateway 中可实际调用的 chat profile。
  `QA_SMOKE_CHAT_MODEL` 仅在 smoke 需要显式校验 model/profile exact-match 时设置。
  `.env.example` 的 `default-chat` 只在 `localhost:11434/v1` 后面有可用
  OpenAI-compatible provider 时可用；真实 provider 可通过 `.env.local` 的
  `AI_GATEWAY_LOCAL_PROVIDER_BASE_URL`、`AI_GATEWAY_LOCAL_PROVIDER_API_KEY` 和
  `AI_GATEWAY_LOCAL_CHAT_MODEL` 由 `./scripts/local/start.sh` 自动写入本地 seed。
- 默认 Knowledge adapter 使用 query-first readiness，不要求 runtime worker 已在启动时
  运行。需要证明真实上传、解析、embedding、索引和检索时，先让
  `./scripts/local/start.sh` 通过根级 Compose 默认 infra 启动本地 Elasticsearch，
  再用 `./scripts/local/start.sh --runtime full` 显式启动 host-run runtime API、worker
  和 Knowledge adapter。第一次启用前先执行
  `cp .env.example .env.local`，并在本地 `.env.local` 中显式设置
  `KNOWLEDGE_AUTO_START_INGESTION=true`、`DOC_ENGINE=elasticsearch`、
  `KNOWLEDGE_RUNTIME_ES_URL=http://127.0.0.1:9200` 及对应
  `KNOWLEDGE_RUNTIME_EMBEDDING_*` / `KNOWLEDGE_RUNTIME_RERANK_*` / vendor model id
  变量。上传 ingestion 会调用 `/documents/parse` 入队，不预检或等待 worker heartbeat。
  worker 生命周期由 `./scripts/local/start.sh --runtime full`、systemd、K8s/KEDA、
  supervisor 或同类外部工具管理。`start.sh` 不直接执行 Docker build/run；
  如使用外部 Elasticsearch，把 `KNOWLEDGE_RUNTIME_ES_URL` 指向实际地址。

启动本地栈：

```bash
cp .env.example .env.local
# 如需运行本 smoke，可先在 .env.local 中设置 QA_SETTINGS_OPEN=true。
./scripts/local/start.sh
```

运行 smoke：

```bash
cd services/knowledge
GATEWAY_RAG_E2E_SMOKE=1 \
GATEWAY_BASE_URL='http://127.0.0.1:8080' \
VENDOR_RUNTIME_URL='http://127.0.0.1:9380' \
KNOWLEDGE_SERVICE_BASE_URL='http://127.0.0.1:8083' \
QA_SERVICE_BASE_URL='http://127.0.0.1:8084' \
AI_GATEWAY_BASE_URL='http://127.0.0.1:8086' \
KNOWLEDGE_TEST_DATABASE_URL='postgres://knowledge_app:knowledge_app_dev@127.0.0.1:5432/knowledge_system?sslmode=disable' \
KNOWLEDGE_REDIS_ADDR='127.0.0.1:6379' \
GATEWAY_SMOKE_USERNAME='admin' \
GATEWAY_SMOKE_PASSWORD='LocalDemoAdmin#12345' \
QA_SMOKE_CHAT_PROFILE_ID='default-chat' \
go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v
```

预期结果：

```text
=== RUN   TestGatewayRAGE2ESmoke
--- PASS: TestGatewayRAGE2ESmoke (...s)
PASS
```

测试会创建 run-scoped knowledge base 和文档，并在清理阶段先调用 Gateway
`DELETE /api/v1/documents/{documentId}` 触发 Knowledge runtime 文档删除，再按
chunks、jobs、documents、knowledge base 的顺序删除本轮 Knowledge PostgreSQL 行。
测试开始前会读取当前 active QA config 和 LLM config，结束时通过 Gateway settings API
重新创建并激活一份等价恢复版本。本轮 smoke 创建的 QA config versions 仍会作为本地运行
记录留存，因为它们是 QA settings 的真实业务资源；只在临时本地栈运行该 smoke。

常见失败和定位：

| 阶段 | 典型失败 | 排查 |
| --- | --- | --- |
| File | `File stage: gateway document upload returned HTTP ...` | 查 `.local/logs/gateway.log` 和 `.local/logs/file.log`，确认 File ready、`INTERNAL_SERVICE_TOKEN` 一致、上传大小未超限。不要打印 multipart body 或 object key。 |
| Knowledge runtime | `Parser stage: ready document did not record parserBackend` 或文档状态 `failed` | 查 `.local/logs/knowledge.log` 和 runtime API/worker 终端日志，确认 `VENDOR_RUNTIME_URL`、`VENDOR_RUNTIME_SERVICE_TOKEN`、`KNOWLEDGE_RUNTIME_SERVICE_TOKEN` 一致；Markdown fixture 不需要真实 OCR 模型下载。 |
| Knowledge ingestion | `document ... did not become ready` 或 `chunkCount = 0` | 查 `.local/logs/knowledge.log` 和 runtime worker 日志，并确认 PostgreSQL、Redis、MinIO、Elasticsearch 处于 healthy；检查 runtime task 状态和对象存储 bucket。 |
| Knowledge retrieval | `Knowledge retrieval stage: ...`、无 expected hit 或 rerank trace 异常 | 查 `.local/logs/gateway.log`、`.local/logs/knowledge.log` 和 `.local/logs/ai-gateway.log`；确认 `knowledge-queries` 返回 ready 文档 chunk，runtime 显式配置 `KNOWLEDGE_RUNTIME_RERANK_FACTORY=AI_GATEWAY` 和 rerank profile；只有显式配置兼容 model label 时才要求它与 profile model 匹配。并在 AI Gateway `provider_invocations` 里看到 `caller_service=knowledge`、`operation=reranking`。 |
| AI Gateway | QA message 返回 `502`、model error 或 provider unavailable | 查 `.local/logs/qa.log` 和 `.local/logs/ai-gateway.log`；确认 chat profile enabled、credential/provider 可用；如果显式设置了 `MODEL_ID`，再确认它与 profile model exact-match。不要粘贴 provider 原始错误 body 或 API key。 |
| QA | QA config POST `403/400`、answer 未完成、无 citation | 确认 `QA_SETTINGS_OPEN=true` 或账号具备 `qa:settings:write`；确认模型支持 OpenAI-compatible tool/function calling；确认 QA config 启用了 `knowledge__search` 或 fallback `search_knowledge`，且默认 KB 列表为空以允许全量知识库检索。 |

如果只需要验证 Knowledge ingestion 和 retrieval，不要运行本 RAG smoke；先使用上面的
`KNOWLEDGE_INGESTION_SMOKE` 或 `GATEWAY_KNOWLEDGE_OWNER_SMOKE` 缩小范围。

### QA + Auth + Gateway 宿主机联调检查

```bash
./scripts/local/start.sh
```

该路径适合验证 Auth、QA、Gateway 的基础 ready 状态和 QA 非 provider 依赖路径：

```bash
curl -fsS http://localhost:8081/readyz
curl -fsS http://localhost:8084/readyz
curl -fsS http://localhost:8080/readyz
```

日志查看 `.local/logs/auth.log`、`.local/logs/qa.log`、`.local/logs/gateway.log`。
触发真实 LLM 调用、LLM connection test 或 Agent Run 时，确保宿主机 `ai-gateway`
已由 `start.sh` 启动，并且 `.env.local` 中的 profile/provider 配置可用。

### Document 宿主机联调检查

```bash
./scripts/local/start.sh
```

该路径适合验证 Document PostgreSQL、Redis、migration、job enqueue 和 worker 状态机：

```bash
curl -fsS http://localhost:8085/readyz
```

注意：模板、材料和报告文件 bytes 需要 File Service；真实大纲/正文生成需要 AI Gateway。
当前基础 DOCX 导出使用 Document 内置 `SimpleDOCXGenerator`，不需要 Pandoc/LibreOffice；
Pandoc/LibreOffice 仅是后续富 DOCX worker 工具链。日志查看
`.local/logs/document.log`、`.local/logs/file.log` 和 `.local/logs/ai-gateway.log`。
Document worker 会执行 `report_file_creation` 的基础 DOCX 导出；其他大纲/正文生成类
job 依赖可用的 AI Gateway chat profile/provider。

### AI Gateway host-run

PR #487 之后 AI Gateway 作为本机进程运行（不再有 AI Gateway Docker 业务服务）。
`start.sh` 会自动执行 ai-gateway migration 并写入本地 placeholder profile，也会随其他
服务一起启动 ai-gateway。
如果 `.env.local` 设置 `AI_GATEWAY_LOCAL_SEED_ENABLED=true`，`start.sh` 会在静态
seed 后读取 `AI_GATEWAY_LOCAL_PROVIDER`、`AI_GATEWAY_LOCAL_PROVIDER_BASE_URL`、
`AI_GATEWAY_LOCAL_PROVIDER_API_KEY`、`AI_GATEWAY_LOCAL_CHAT_MODEL`、
`AI_GATEWAY_LOCAL_EMBEDDING_MODEL`、`AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS`、
`AI_GATEWAY_LOCAL_RERANK_MODEL` 和 `AI_GATEWAY_LOCAL_RERANK_TOP_N`，生成加密
credential，并覆盖 `default-chat`、`default-embedding`、`default-rerank`。同一
overlay 还会新增/激活一条 QA LLM 配置版本，让 QA 的 active model 与
`default-chat` profile model 一致。

AI Gateway 服务 token 运行时只接受 hash，如需手动验证：

```bash
TOKEN=dev-internal-service-token-change-me
printf '%s' "$TOKEN" | shasum -a 256 | awk '{print "sha256:" $1}'
```

环境变量通过 `.env.local` 统一加载（`AI_GATEWAY_DATABASE_URL`、
`AI_GATEWAY_SERVICE_TOKEN_HASHES`、`AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` 等均在
`.env` 中定义）。若要单独调试，可手动 source 后运行：

```bash
set -a && source .local/config/dev.env.sh && set +a

cd services/ai-gateway
go run github.com/pressly/goose/v3/cmd/goose@v3.27.0 \
  -dir migrations postgres "$AI_GATEWAY_DATABASE_URL" up
go run ./cmd/server
```

创建可调用 profile 后，`/internal/v1/chat/completions`、`/internal/v1/embeddings` 和 `/internal/v1/rerankings` 都会按 profile 解析 provider、model、base URL 和 API key。当前 fake-provider 单元测试覆盖了协议形态；真实 provider smoke 仍需单独执行并记录。

### 报告大纲/正文 AI 生成配置

报告大纲生成（`outline_generation`）和正文生成（`content_generation`）调用 AI Gateway 的
`default-chat` profile。**本地 seed 写入的是 placeholder URL，不带真实模型，默认不可用。**
拉取最新 develop 后如果点击"生成大纲"一直失败，根本原因就是 AI Gateway 没有指向可用的
LLM provider。

可以选择用 `.env.local` overlay 自动配置，或按以下管理端步骤手动配置后重试。
overlay 配置示例：

```bash
AI_GATEWAY_LOCAL_SEED_ENABLED=true
AI_GATEWAY_LOCAL_PROVIDER=siliconflow
AI_GATEWAY_LOCAL_PROVIDER_BASE_URL=https://api.siliconflow.cn/v1
AI_GATEWAY_LOCAL_PROVIDER_API_KEY=<local-provider-api-key>
AI_GATEWAY_LOCAL_CHAT_MODEL=deepseek-ai/DeepSeek-V4-Flash
AI_GATEWAY_LOCAL_EMBEDDING_MODEL=BAAI/bge-m3
AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS=1024
AI_GATEWAY_LOCAL_RERANK_MODEL=BAAI/bge-reranker-v2-m3
AI_GATEWAY_LOCAL_RERANK_TOP_N=5
```

修改后重新运行 `./scripts/local/start.sh`，再启动或重启后端。脚本会用
`AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` 加密 API key，只把密文写入 PostgreSQL。
`default-chat`、`default-embedding`、`default-rerank` 会被更新；如果配置了 chat model，
脚本还会激活一条匹配的 QA LLM config version。host-run 的 `MODEL_ID` 和
`DOCUMENT_AI_GATEWAY_MODEL` 可保持为空或默认 `local-placeholder-chat`；`start.sh`
会在本次 host-run 进程里把它们对齐到 `AI_GATEWAY_LOCAL_CHAT_MODEL`，避免请求模型与
`default-chat` profile 不一致。

也可以继续通过管理端 UI 覆盖 profile 和 credential（Admin → 模型配置），但真实
API key 仍只应保存在本地 `.env.local`、受控 secret store 或浏览器提交请求中，不要提交。

> **代理注意**：ai-gateway 以本机进程运行（PR #487 后不再使用 Docker），直接继承
> 宿主机代理环境变量。如果所在网络需要代理才能访问外部 provider，在 `.env.local`
> 中添加：
>
> ```bash
> HTTP_PROXY=http://127.0.0.1:7897
> HTTPS_PROXY=http://127.0.0.1:7897
> ```
>
> 然后重启 ai-gateway 进程（本地标准路径用 `./scripts/local/stop.sh` 后再运行
> `./scripts/local/start.sh`）。端口号改成本机实际代理端口。

#### 验证

配置完成后确认 profile 可用：

```bash
# AI Gateway 健康检查
curl -fs http://localhost:8086/readyz && echo "ok"

# 快速冒烟：直接调 chat completions（profile 决定 provider/model）
INTERNAL_TOKEN="local-dev-internal-service-token-change-me"
curl -X POST http://localhost:8086/internal/v1/chat/completions \
  -H "X-Service-Token: $INTERNAL_TOKEN" \
  -H "X-Caller-Service: document" \
  -H "Content-Type: application/json" \
  -d '{"profile_id":"default-chat","messages":[{"role":"user","content":"hi"}]}'
```

返回包含 `choices` 字段则 provider 正常，之后在前端选择「迎峰度夏检查报告」
（`summer_peak_inspection`）或「煤库存审计报告」（`coal_inventory_audit`）重新提交，
即可触发大纲生成。正文生成可继续通过报告页面的「正文生成」任务或单章重新生成验证。

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
| Auth/Gateway/QA 宿主机联调检查 | 各服务 `GET /readyz` + 目标 Gateway API smoke | Gateway `/readyz` 只证明 Redis/Auth 和 owner URL 配置；QA `GET /readyz` 证明 QA 进程与自身依赖 ready；真实 AI 调用仍可能因 AI Gateway profile/provider 未配置失败。 |
| Document 宿主机联调检查 | 创建 report job 后查询 job/attempt/events | 非文件生成类任务会入队并由 worker 推进为 succeeded；不会生成真实 AI 大纲/正文。若额外提供 File Service，`report_file_creation` 可生成基础 DOCX 并通过 content endpoint 读取成功文件。 |
| AI Gateway profile | 创建 chat/embedding/rerank profile，调用对应内部 endpoint | fake provider 和兼容 provider 应返回 OpenAI-style body；真实 provider 需手工验证。 |
| Gateway contract | `python3 scripts/verify_gateway_active_api.py` | active path、owner、security 和 owner map 不漂移。 |
| File PostgreSQL + MinIO | `FILE_MINIO_POSTGRES_SMOKE=1 ... go test ./internal/integration -run TestFileMinIOPostgresSmoke -count=1 -v` | 只在真实 PostgreSQL/MinIO 可用时运行；验证 upload、metadata、content read、delete 和清理状态。File service 默认本地配置已使用 PostgreSQL + MinIO；生产化验证还应显式设置非本地 `FILE_ENV`、caller allowlist 和 `FILE_ALLOWED_CONTENT_TYPES`，确认 memory backend guard、`401`/`403` 和 MIME 拒绝路径符合预期。 |
| Knowledge runtime route/config | `cd services/knowledge-runtime && PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio --with filelock --with ruamel-yaml python -m pytest test/routes/test_config_utils.py test/routes/test_route_registry.py test/routes/test_gateway_auth.py test/routes/test_runtime_dependency_check.py -q` | 验证 runtime 配置脱敏、路由 allowlist、service token、clean DB provisioning 纯逻辑和 host-run dependency guard。 |
| Knowledge ingestion real deps | `KNOWLEDGE_INGESTION_SMOKE=1 ... go test ./internal/integration -run '^TestKnowledgeIngestionRealDepsSmoke$' -count=1 -v` | 只在 Knowledge adapter 以 `KNOWLEDGE_AUTO_START_INGESTION=true` 重启后，且 Knowledge runtime API、runtime worker、Redis/MinIO/Elasticsearch/PostgreSQL 和 embedding provider 均可用时运行；验证 Markdown fixture 上传、解析、切片、embedding、索引写入、ready 状态、chunk 回看和检索命中。当前 Knowledge 主路径不依赖 File Service。 |
| Gateway -> Knowledge owner route | `GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayKnowledgeOwnerRouteSmoke$' -count=1 -v` | 只在 Gateway/Auth/Redis/Knowledge/PostgreSQL 和 Knowledge runtime 可用时运行；验证伪造 `X-User-*` 未认证请求被拒绝，并用 KB `createdBy` 断言 Gateway 注入真实 session user。 |
| Gateway -> Knowledge -> QA RAG | `GATEWAY_RAG_E2E_SMOKE=1 ... go test ./internal/integration -run '^TestGatewayRAGE2ESmoke$' -count=1 -v` | 只在 Gateway/Auth/Redis/Knowledge/Knowledge runtime/QA/AI Gateway 和可用 chat profile/provider 可用时运行；验证上传、ingestion ready、`knowledge-queries` 命中、QA answer 和 citation 摘要。 |
| 前端 Gateway 类型 | `bun run --cwd apps/web api:generate` 后检查 diff | 生成类型应与 Gateway OpenAPI 保持同步。 |

## 已知缺口

| 缺口 | 影响 | 跟踪 |
| --- | --- | --- |
| 根级跨服务 smoke 缺失 | 即使使用 `deploy/docker-compose.yml` 启动本地/演示基线，也不能自动证明 Auth/Gateway/File/Knowledge/QA/Document/AI Gateway 链路可用。 | #125 |
| 跨服务契约测试和 E2E smoke 缺失 | 不能自动证明前端 -> Gateway -> 多服务链路可用。 | #125 |
| Gateway `/readyz` 非完整依赖诊断 | `/readyz` 不请求所有 owner service `/readyz`，也不执行业务级 smoke；它通过不等于 Knowledge/QA/Document/AI Gateway 全链路可用。 | #353、#125、#352 |
| Knowledge runtime 真实解析/检索 smoke 不在普通 CI 中运行 | CI 覆盖 route/config/auth/provisioning 和语法门禁；真实 PDF 解析、OCR 质量、embedding、Elasticsearch/MinIO/Redis 组合仍需在具备依赖的本地或部署环境手动记录。 | #125 |
| Knowledge/QA RAG smoke 仍为显式 opt-in | File 自身 PostgreSQL + MinIO smoke 已有；Knowledge ingestion 真实依赖 smoke 覆盖 Knowledge runtime/PostgreSQL/Redis/MinIO/Elasticsearch 写入和状态更新，当前 Knowledge 主路径不依赖 File Service；Gateway -> Knowledge -> QA RAG smoke 已提供最小验收样例，但依赖可用 AI Gateway chat profile/provider，且不覆盖 MCP、前端或 #125 完整一键 E2E。 | #125、#152、#154、#304 |
| 生产部署基线缺失 | 当前 `deploy/docker-compose.yml` 是本地/演示基线，不能直接当生产部署。 | #150 |
| Document 真实 AI 生成和富 DOCX 工具链未落地 | 报告 job 状态机和基础 DOCX 导出可用；真实大纲/正文生成、Pandoc/LibreOffice 富 DOCX 转换和跨服务内容读取 smoke 仍需补齐。 | #160、#223 |
| Document 跨服务 smoke 仍缺失 | settings/statistics/logs 已在服务端落地，但管理端、Gateway、File Service、Document worker 串联 smoke 仍未一键化。 | #159、#221 |
| QA Agent Run MVP 和权限一致性仍在推进 | QA 会话/消息基础可用，完整 Agent 编排和 403 一致性仍需收口。 | #157、#217 |
| 前端业务 E2E 覆盖不足 | 已有 Playwright 基础 smoke；Knowledge、QA、Document 等完整业务流程仍需随页面能力扩展。 | #117、#163 |

## PR 前判断

- 只改文档：至少执行 `git diff --check`，并检查新增链接、相对路径和实现事实。
- 改后端服务：执行对应服务 `go test ./...` 和 `go build ./cmd/server`；QA 还要 `go build ./cmd/agent`。
- 改 migration：执行 goose apply；如果服务有 env-gated repository integration tests，尽量使用本地 PostgreSQL 跑一遍。
- 改 Knowledge runtime 契约或运行时：检查 `services/knowledge-runtime/README.md`、Knowledge adapter 文档和本 runbook 是否一致；运行 runtime targeted pytest、`python -m compileall api common rag deploy`、Knowledge adapter `go test ./...` / `go build ./cmd/adapter`。如触碰真实解析、embedding、Elasticsearch、Redis 或 MinIO 组合，尽量追加真实 PDF E2E，并在 PR 记录中区分 unit/contract 检查与真实依赖 smoke。
- 改 Gateway OpenAPI：执行 `python3 scripts/verify_gateway_active_api.py`，前端类型相关改动还要执行 `bun run --cwd apps/web api:generate` 并检查生成 diff。
- 改前端：执行 `bun install --frozen-lockfile`、`bun run --cwd apps/web check`、`bun run --cwd apps/web build`、`bun run --cwd apps/web test:unit`；关键页面改动再跑 `bun run --cwd apps/web test:e2e`。

## Appendix: AI Gateway real-provider cross-service acceptance

Use this checklist only in a protected local or shared environment that can hold
real provider credentials. Do not commit `.env` values, provider keys, service
tokens, full prompts, document text, embedding payloads, or provider raw error
bodies.

1. Start the local stack (infra + all backend services):
   `./scripts/local/start.sh`.
2. Check AI Gateway readiness:
   `curl -s http://127.0.0.1:8086/readyz`.
   `missing` means the profile or active credential is absent; `placeholder`
   means the seeded local fake credential is still present; `ok` means a
   non-placeholder credential is configured, not that the external provider has
   accepted it.
3. Create or patch chat, embedding, and rerank profiles through
   `/internal/v1/model-profiles` with `X-Service-Token`,
   `X-Caller-Service`, and `X-Request-Id`. The `model` in smoke requests must
   exactly match the selected profile. For local-only validation you may instead
   set `AI_GATEWAY_LOCAL_SEED_ENABLED=true` and the `AI_GATEWAY_LOCAL_*` values
   in `.env.local`, then rerun `./scripts/local/start.sh`.
4. Run direct AI Gateway smoke with
   `AI_GATEWAY_REAL_PROVIDER_SMOKE=1`. Chat is the minimum; embedding and rerank
   run only when the provider and env vars support them.
5. Run QA validation by creating a Gateway session, creating a QA session, and
   sending a message through Gateway public `/api/v1/**` or the documented
   `QA_AI_GATEWAY_SMOKE=1` service-client test. Use the same request id to
   search `.local/logs/gateway.log`, `.local/logs/qa.log`, and
   `.local/logs/ai-gateway.log`.
6. Run Document validation for `summer_peak_inspection` and
   `coal_inventory_audit`: create a report, create an `outline_generation`,
   `section_regeneration`, or `content_generation` job, poll jobs, events, and
   sections until terminal state, then search
   `.local/logs/gateway.log`, `.local/logs/document.log`, and
   `.local/logs/ai-gateway.log` by request id. Rich DOCX
   Pandoc/LibreOffice worker validation is not part of this checklist.
7. Run Knowledge validation with `KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY=AI_GATEWAY`
   and `KNOWLEDGE_RUNTIME_RERANK_FACTORY=AI_GATEWAY`. Set
   `KNOWLEDGE_RUNTIME_EMBEDDING_BASE_URL` and `KNOWLEDGE_RUNTIME_RERANK_BASE_URL`
   to `http://127.0.0.1:8086/internal/v1`, use `default-embedding` /
   `default-rerank`, and keep `X-Service-Token`, `X-Caller-Service=knowledge`,
   and `X-Request-Id` visible in logs. A passing Knowledge query alone is not
   enough; verify AI Gateway `provider_invocations` contains
   `caller_service=knowledge` rows for both `operation=embedding` and
   `operation=reranking` with the request id or generated `stw-kgw-*` id.

Acceptance record template:

| Area | Command or request | Expected proof | Logs |
| --- | --- | --- | --- |
| AI Gateway | `/readyz` + real provider smoke | profile status is not `placeholder`; smoke succeeds for configured operations | `.local/logs/ai-gateway.log` |
| QA | session/message or `QA_AI_GATEWAY_SMOKE=1` | answer path reaches AI Gateway and returns normalized response/error | `.local/logs/gateway.log`, `.local/logs/qa.log`, `.local/logs/ai-gateway.log` |
| Knowledge | AI Gateway embedding/rerank path | selected runtime factories are `AI_GATEWAY`; `provider_invocations` records `caller_service=knowledge` for embedding and reranking | `.local/logs/gateway.log`, `.local/logs/knowledge.log`, `.local/logs/ai-gateway.log` |
| Document | `summer_peak_inspection` and `coal_inventory_audit` report job flow | job/events/sections reach expected terminal state | `.local/logs/gateway.log`, `.local/logs/document.log`, `.local/logs/ai-gateway.log` |
