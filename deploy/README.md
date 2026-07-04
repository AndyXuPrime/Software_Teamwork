# 本地启动手册

本地完整启动路径分三层：

```text
Docker: postgres + redis + minio + minio-init + elasticsearch
Host:   knowledge-runtime API, plus worker only for ingestion
Host:   knowledge-adapter
Host:   auth + file + ai-gateway + qa + document + gateway + frontend
```

`services/parser` 已由 Knowledge runtime 方案替代，不再作为本地后端服务启动。

## 直接启动

先安装 Docker、Go `1.25.x`（脚本最低接受 `go1.25.1`，当前本地基线可用 `go1.25.4`）、
uv、Bun、`psql` 客户端和 `curl`。
Go 必须安装在实际运行这些脚本的宿主机环境中；如果使用 WSL 启动脚本，Windows
里的 Go 不等于 WSL 里的 Go。

源选择采用官方默认源，国内网络用 `start.sh --china` 显式启用镜像和下载加速。默认文件不提交
active 第三方镜像值。`config/base.yaml` 里的 Docker、uv 和 Go 默认值分别指向
Docker Hub / Elastic registry pinned images、PyPI、`proxy.golang.org` 和
`sum.golang.org`。启动脚本会先做 preflight，再按需准备缺失的 Go 工具/服务二进制、
Knowledge runtime 依赖和 Docker infra images。

```bash
cp .env.example .env.local
./scripts/local/start.sh

cd apps/web
bun install
bun run dev
```

中国大陆网络：

```bash
cp .env.example .env.local
./scripts/local/start.sh --china

cd apps/web
bun install
bun run dev
```

日常再次启动时，已经准备过本机工具、镜像、二进制和前端依赖，可以直接：

```bash
./scripts/local/start.sh
cd apps/web && bun run dev
```

如果只是启动 infra、migration 和 seed，不启动后端：

```bash
./scripts/local/start.sh --infra-only
```

`start.sh` 默认启动 Knowledge runtime API + worker。真实 Knowledge 解析、embedding、
索引和检索链路还需要在本地 `.env.local` 中填写 provider。runtime `.venv`、NLTK data、
Tika jar、cl100k encoding 和 deepdoc 模型会在缺失时由 `start.sh` 准备。
Elasticsearch 是默认本地 infra，由 `./scripts/local/start.sh` 启动：

```text
DOC_ENGINE=elasticsearch
KNOWLEDGE_RUNTIME_ES_URL=http://127.0.0.1:9200
KNOWLEDGE_AUTO_START_INGESTION=true
```

只启动 runtime API：

```bash
./scripts/local/start.sh --runtime api
```

跳过 runtime：

```bash
./scripts/local/start.sh --runtime none
```

中国大陆网络下，给启动命令加 `--china`，脚本会对 Docker、Go、uv/runtime 下载和
HuggingFace runtime 访问使用本次运行的镜像设置：

```bash
./scripts/local/start.sh --china
```

停止：

```bash
# 在运行 bun run dev 的终端按 Ctrl-C 停前端。
./scripts/local/stop.sh
```

`stop.sh` 只管理 `.local/run/` 记录的后端和 Knowledge runtime 进程组；前端 Vite dev server 是前台命令，需要在对应终端按
`Ctrl-C` 单独停止。如果忘记前端在哪个终端运行，先查默认端口再杀对应进程：

```bash
lsof -nP -iTCP:5173 -sTCP:LISTEN
kill <pid>
```

清空本地 infra 数据：

```bash
./scripts/local/clean.sh
```

脚本会先停止 `.local/run/` 下记录的宿主机后端进程、渲染本地配置并校验 Compose，
再删除根级 infra Compose 的数据卷。默认需要输入 `clean` 确认；自动化场景可使用
`./scripts/local/clean.sh --yes`。脚本底层执行
`docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env down -v --remove-orphans`；
不删除 Docker 镜像、`.env.local`、`.local/tools` 或 `.local/bin`。

简单判断：

| 目标 | 命令 |
| --- | --- |
| 启动本地后端栈 | `./scripts/local/start.sh` |
| 中国大陆网络启动 | `./scripts/local/start.sh --china` |
| 只停止后端/runtime，保留 infra 数据 | `./scripts/local/stop.sh` |
| 清空本地 PostgreSQL/MinIO/Elasticsearch 数据 | `./scripts/local/clean.sh` |
| 非交互清空本地 infra 数据 | `./scripts/local/clean.sh --yes` |

彻底冷启动时，如果还要删除本项目本地容器、卷和镜像，先确认没有其他项目依赖这些镜像：

```bash
./scripts/local/clean.sh --yes

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

## 配置来源

`config/` 是 repository-level configuration authority，也是唯一默认配置来源。
根 `.env.example` 是本地 secret 模板，用户只复制一次：

```bash
cp .env.example .env.local
```

脚本不会维护 `.env.local` 默认变量。它们通过 `config/ctl` 渲染
`.local/config/<profile>.env` 和 `.local/config/<profile>.env.sh`，让 Docker Compose
和宿主机 Go 进程拿到同一份本地配置。默认 profile 是 `dev`，也可以用
`CONFIG_PROFILE=staging` 选择其它 profile。默认使用官方源。Go modules 下载只在
`start.sh` 准备缺失 Go 工具或服务二进制时发生，并读取当前模式下的
`GOPROXY` / `GOSUMDB`。官方默认值是
`GOPROXY=https://proxy.golang.org,direct` 和 `GOSUMDB=sum.golang.org`。

中国大陆网络使用大陆镜像时，不要把仓库默认文件改成第三方代理；运行
`./scripts/local/start.sh --china`。
如果已有 `.env.local` 仍保留 TUNA、旧 Docker registry rewrite、`goproxy.cn` 或
`sum.golang.google.cn`，脚本会继续尊重这些本地值并提示你这是本地覆盖。想回到官方
默认值，重新复制 `.env.example` 后再恢复私有配置，或手动改回官方地址。

默认 demo 账号：

```text
admin / LocalDemoAdmin#12345
superadmin / LocalDemoAdmin#12345
```

Docker 镜像默认使用 Compose 里的 Docker Hub pinned tags。中国大陆网络可用
`./scripts/local/start.sh --china` 使用 `docker.1ms.run` registry rewrite；企业
镜像仓库可在本机 `.env.local` 设置 `POSTGRES_IMAGE`、`REDIS_IMAGE`、
`MINIO_IMAGE`、`MINIO_MC_IMAGE` 和 `KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE`，
但不要提交为默认值。
如果使用官方 Docker Hub pinned tags 并依赖本机代理访问，先配置宿主机/Docker 代理，
再运行 `python3 scripts/check_docker_environment.py --profile default` 验证；不要加
`--clean-env`，该参数用于直连或 daemon mirror 排查，会清掉 shell 出站代理变量。

`UV_DEFAULT_INDEX` 控制宿主机 uv 在解析或重锁依赖时使用的 Python 包索引，默认使用
官方 PyPI。`services/parser` 已退役，默认路径不再准备 standalone Parser；解析、切块、
embedding、索引和检索通过 `services/knowledge-runtime` 完成。启动脚本会在 runtime
启动模式不是 `none` 时按需执行 sync 或 artifact 下载；需要单独准备时也可手工执行：

```bash
cd services/knowledge-runtime
uv sync --python 3.13 --frozen --no-install-project --no-default-groups

# worker profile:
uv sync --python 3.13 --frozen --no-install-project --group worker
```

Artifact 和模型下载：

```bash
cd services/knowledge-runtime
uv run --no-project \
  --with "nltk>=3.9.4" \
  --with "huggingface-hub>=1.3.1" \
  ragflow_deps/download_deps.py

# 中国大陆镜像：
uv run --no-project \
  --with "nltk>=3.9.4" \
  --with "huggingface-hub>=1.3.1" \
  ragflow_deps/download_deps.py --china
```

`download_deps.py` 不再下载 uv release 包；uv 本身由本机安装或团队环境提供。提交的
`pyproject.toml` / `uv.lock` 仍保持官方 URL。需要企业源时，显式设置本机环境变量或
本机 `.env.local`。

`HF_ENDPOINT` 控制 Knowledge runtime worker 首次加载 deepdoc OCR/vision 模型时访问
HuggingFace 的地址。提交的默认配置不启用第三方 mirror；如果在中国大陆网络运行真实
解析栈，使用 `./scripts/local/start.sh --china`，脚本会在本次进程
内为未设置 `HF_ENDPOINT` 的环境使用 `https://hf-mirror.com`。企业环境也可以只在本机
`.env.local` 中设置内部 HuggingFace mirror，不要提交真实或第三方默认值。

`GOPROXY` 和 `GOSUMDB` 控制宿主机 Go 工具链下载模块和校验数据库的路径，默认使用
`https://proxy.golang.org,direct` 和 `sum.golang.org`，用于手工构建
本机 goose/config 工具和服务二进制。如果在中国大陆网络看到
`Get "https://proxy.golang.org/...": i/o timeout`，先运行 `./scripts/local/start.sh --china`
使用 `GOPROXY=https://goproxy.cn,direct` 和 `GOSUMDB=sum.golang.google.cn`。
无法访问公开镜像的企业环境可以在本机 `.env.local` 中改为企业 Go proxy / checksum DB。
`start.sh` 会在构建 config renderer、安装 goose 和构建 seed helper 前先读取 `.env.local`
里的 Go 源相关变量，因此首次准备本地 Go tools 也会使用这些长期本地覆盖。

## 脚本职责

`./scripts/local/start.sh`：

- 先做 preflight，确认 `.env.local` 已由用户创建但不创建或覆盖它；再按需准备 config
  renderer、`goose@v3.27.0`、AI Gateway seed helper、host-run 服务二进制、Knowledge
  runtime `.venv`/artifact 和 Docker infra images。Go 产物只有在 `.local/stamps/` 中的源码
  fingerprint 匹配时才复用；源码变化会自动重建，`--skip-prepare` 遇到过期产物会失败。长耗时
  下载/构建会持续输出进度或心跳。Knowledge runtime `.venv` 会校验 profile 以及
  `pyproject.toml` / `uv.lock` / `download_deps.py` fingerprint。
- 使用 config renderer 渲染配置，使用 `.local/tools/goose` 执行 migration。
- 使用已准备好的 infra images 启动 Compose，默认 `--pull never`。
- 应用本地 demo seed、AI Gateway profile seed 和 QA Document MCP seed。
- 启动 `.local/bin/` 下的 host-run 后端二进制；默认启动已准备好的 Knowledge runtime
  API + worker，可用 `--runtime api` 或 `--runtime none` 调整。
- 启动后输出 Docker infra `ps`、host-run 进程组状态和 `.local/logs/*.log` 位置。

`./scripts/local/stop.sh`：

- 按 `.local/run/*.pid` 停止 host-run 后端和 runtime 进程组并清理 pid 文件。

`./scripts/local/clean.sh`：

- 停止 host-run 进程并删除本地 Compose 数据卷；不删除 Docker images、`.env.local`、
  `.local/tools` 或 `.local/bin`。

## Knowledge Runtime

Knowledge 文档上传、解析、切块、embedding、索引和检索通过 Knowledge runtime 完成。
默认本地栈只启动 Knowledge adapter；真实 ingestion/retrieval 需要显式启动 runtime
及其外部依赖。本地 Knowledge adapter 读取：

```text
VENDOR_RUNTIME_URL=http://127.0.0.1:9380
VENDOR_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me
KNOWLEDGE_RUNTIME_SERVICE_TOKEN=local-dev-runtime-service-token-change-me
KNOWLEDGE_RUNTIME_READINESS_MODE=query
KNOWLEDGE_AUTO_START_INGESTION=true
```

runtime API 和 worker 在宿主机启动；本地默认 adapter 使用
`http://127.0.0.1:9380`。runtime API 只需要 `X-Service-Token`，由
Knowledge adapter 使用 `VENDOR_RUNTIME_SERVICE_TOKEN` 转发；全局 runtime
scope 在 runtime 进程内通过 `KNOWLEDGE_RUNTIME_SCOPE_ID` 配置。
不要再启动 `services/parser`，也不要把 runtime 放回根级 Compose。

启用真实 ingestion 前，需要先准备：

- 从默认配置复制本地环境文件：

  ```bash
  cp .env.example .env.local
  ```

- 在 `.env.local` 中配置 AI Gateway embedding/rerank profile。外部 provider
  API key 只放入 AI Gateway local seed 或管理 API，不放入 Knowledge runtime。
  推荐配置。这里的 runtime model 值只是 RAGFlow 兼容 label；provider model、
  base URL、credential 和调用审计以 AI Gateway profile 为准：

  ```text
  KNOWLEDGE_RUNTIME_AI_GATEWAY_SERVICE_TOKEN=local-dev-internal-service-token-change-me
  KNOWLEDGE_RUNTIME_EMBEDDING_FACTORY=AI_GATEWAY
  KNOWLEDGE_RUNTIME_EMBEDDING_MODEL=BAAI/bge-m3
  KNOWLEDGE_RUNTIME_EMBEDDING_BASE_URL=http://127.0.0.1:8086/internal/v1
  KNOWLEDGE_RUNTIME_RERANK_FACTORY=AI_GATEWAY
  KNOWLEDGE_RUNTIME_RERANK_MODEL=BAAI/bge-reranker-v2-m3
  KNOWLEDGE_RUNTIME_RERANK_BASE_URL=http://127.0.0.1:8086/internal/v1
  KNOWLEDGE_VENDOR_EMBEDDING_ID=BAAI/bge-m3@default@AI_GATEWAY
  KNOWLEDGE_VENDOR_RERANK_ID=BAAI/bge-reranker-v2-m3@default@AI_GATEWAY
  KNOWLEDGE_AUTO_START_INGESTION=true
  DOC_ENGINE=elasticsearch
  KNOWLEDGE_RUNTIME_ES_URL=http://127.0.0.1:9200
  ```

  direct provider（例如 `SILICONFLOW`）仍可通过显式配置
  `KNOWLEDGE_RUNTIME_MODEL_API_KEY` 使用，但这会绕过 AI Gateway invocation
  审计和 usage 聚合，只应作为本地/应急路径。

- 如果希望 AI Gateway 的本地默认 profile 也自动指向同一组 provider，在
  `.env.local` 中额外填写一组本地 seed 变量，并显式设置
  `AI_GATEWAY_LOCAL_SEED_ENABLED=true`。只有启用该开关时，
  `./scripts/local/start.sh` 才会在 SQL demo seed 后运行
  `scripts/local/render_ai_gateway_local_seed.go` 并把生成的 SQL 写入 PostgreSQL。
  overlay 会使用 `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` 加密 API key，并更新
  `default-chat`、`default-embedding`、`default-rerank`。chat、embedding、rerank
  三类本地模型变量都需要填写；脚本还会新增或激活 QA `llm_config_versions` 中匹配
  `default-chat` 的版本，避免 QA persisted runtime config 继续请求 placeholder model。
  这些 `AI_GATEWAY_LOCAL_*` 变量只用于本地 seed overlay；运行时模型配置权威仍是
  AI Gateway profile，不是 `.env.local`。

  ```text
  AI_GATEWAY_LOCAL_SEED_ENABLED=true
  AI_GATEWAY_LOCAL_PROVIDER=siliconflow
  AI_GATEWAY_LOCAL_PROVIDER_BASE_URL=https://api.siliconflow.cn/v1
  AI_GATEWAY_LOCAL_PROVIDER_API_KEY=<your provider key>
  AI_GATEWAY_LOCAL_CHAT_MODEL=deepseek-ai/DeepSeek-V3
  AI_GATEWAY_LOCAL_EMBEDDING_MODEL=BAAI/bge-m3
  AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS=1024
  AI_GATEWAY_LOCAL_RERANK_MODEL=BAAI/bge-reranker-v2-m3
  AI_GATEWAY_LOCAL_RERANK_TOP_N=5
  ```

  QA 的 persisted LLM 配置由同一个本地 env seed overlay 同步。QA/Document 的
  host-run env 可将 `MODEL_ID` / `DOCUMENT_AI_GATEWAY_MODEL` 留空或保留默认
  `local-placeholder-chat`；`start.sh` 会在本次 host-run 进程里把它们对齐到
  `AI_GATEWAY_LOCAL_CHAT_MODEL`，避免请求模型与 `default-chat` profile 不一致。

- `./scripts/local/start.sh --runtime api` 只启动已经准备好的 host-run runtime API，
  不启动 worker，适合 `KNOWLEDGE_RUNTIME_READINESS_MODE=query` 的查询链路验证。
- `./scripts/local/start.sh --runtime full` 启动 host-run runtime API 和 worker，并生成
  `.local/knowledge-runtime/service_conf.yaml` 供 runtime API 和 worker 使用。它不直接
  执行 `docker build`、`docker run` 或 runtime 依赖下载；本地 Elasticsearch 由同一个
  `start.sh` 通过默认根 Compose 基础设施管理。如果你要使用已有宿主机或外部
  Elasticsearch，把 `KNOWLEDGE_RUNTIME_ES_URL` 改成实际地址。
- runtime worker 第一次启动会按需下载 `InfiniFlow/deepdoc` 等 deepdoc 模型。中国大陆
  网络用 `./scripts/local/start.sh --runtime full --china` 显式启用
  `HF_ENDPOINT=https://hf-mirror.com` 的本次进程覆盖。worker 生命周期可由本地
  `start.sh --runtime full` 或外部 systemd、K8s/KEDA、supervisor 等工具管理。
  runtime helpers 只会自动把 loopback runtime URL 加入 `NO_PROXY`，并对本机 health
  check 使用 `curl --noproxy '*'`。外部 runtime URL 继续尊重宿主机代理环境；如果使用
  Docker bridge IP 且本机代理会截获私网地址，请在本机显式补 `NO_PROXY`。

只有可信本地 provider 才可显式设置 `KNOWLEDGE_RUNTIME_ALLOW_EMPTY_MODEL_API_KEY=1`。

## 快速确认

```bash
curl --noproxy '*' -fsS http://localhost:8080/healthz
curl --noproxy '*' -fsS http://localhost:8080/readyz
curl --noproxy '*' -fsS http://localhost:8083/healthz
```

`http://localhost:8083/readyz` 在 runtime 未启动或 runtime 外部依赖未配置时返回
`503 degraded` 是预期行为；真实 ingestion/retrieval 配好后再用它确认 Knowledge
adapter 到 runtime 的链路。

`http://localhost:8086/readyz` 在 placeholder profile 下返回 `503 degraded` 是预期行为，
表示还没配置真实模型 provider credential，不代表 AI Gateway 进程失败。
默认本地模型 profile 的 OpenAI-compatible 地址是 `http://localhost:11434/v1`。如果
`.env.local` 设置 `AI_GATEWAY_LOCAL_SEED_ENABLED=true`，`start.sh` 会用
`AI_GATEWAY_LOCAL_*` 覆盖 `default-chat`、`default-embedding`、`default-rerank`
三条默认 profile，并同步激活 QA 的 LLM 配置到同一个 `default-chat` profile/model。
Document MCP 的默认 endpoint 是 `http://localhost:8085/mcp`，QA 将其工具暴露为
`document__<tool>`；完整工具参数和 Agent 工作流见
[Document MCP 工具契约](../docs/services/document/docs/mcp-tools.md)。

## Seed Data

`start.sh` 按顺序应用 `deploy/seeds/001-local-demo-seed.sql`、
`deploy/seeds/002-ai-gateway-model-profiles.sql`、
`deploy/seeds/003-qa-document-mcp.sql` 和
`deploy/seeds/004-qa-default-knowledge-base.sql`。最后一个 seed 会清理
legacy `kb_local_demo` 默认绑定，让 QA 的 `defaultKnowledgeBaseIds` 保持空列表；
空列表表示通过 Knowledge MCP / Knowledge query 搜索所有已索引知识库。如果
`.env.local` 填了
`AI_GATEWAY_LOCAL_SEED_ENABLED=true`、`AI_GATEWAY_LOCAL_PROVIDER`、
`AI_GATEWAY_LOCAL_PROVIDER_BASE_URL`、`AI_GATEWAY_LOCAL_PROVIDER_API_KEY`、
`AI_GATEWAY_LOCAL_CHAT_MODEL`、`AI_GATEWAY_LOCAL_EMBEDDING_MODEL`、
`AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS`、`AI_GATEWAY_LOCAL_RERANK_MODEL` 和
`AI_GATEWAY_LOCAL_RERANK_TOP_N`，随后才会通过
`scripts/local/render_ai_gateway_local_seed.go` 生成 SQL 并加密写入本地 AI Gateway
provider credential；未启用该开关时不会应用这个 overlay。它也会激活匹配的 QA
`llm_config_versions` 版本。

Seeded local resources:

| Area | Deterministic resource |
| --- | --- |
| Auth | user `usr_local_admin`, username `admin`, password `LocalDemoAdmin#12345`, role `admin` |
| Auth permissions | `admin:model-profile:write`, `admin:parser-config:write`, `qa:settings:read`, and `qa:settings:write` |
| Knowledge | knowledge base `kb_local_demo`, document `doc_local_demo_seed`, chunk `chunk_local_demo_seed_001` |
| Document | material `22222222-2222-4222-8222-222222222201`, report `22222222-2222-4222-8222-222222222301`, outline `22222222-2222-4222-8222-222222222401` |
| AI Gateway | optional placeholder profiles `default-chat`, `default-embedding`, and `default-rerank`; `.env` local seed can replace their provider `base_url` and encrypted `api_key` |
| QA | conversation `33333333-3333-4333-8333-333333333301`, user message `33333333-3333-4333-8333-333333333401`, assistant message `33333333-3333-4333-8333-333333333402`; default KB list intentionally empty for global Knowledge MCP search |

AI Gateway 的静态 seed 不保存真实 provider key，只保存可识别的本地 placeholder。
需要本机真实 provider 时，在 `.env.local` 中设置：

```bash
AI_GATEWAY_LOCAL_SEED_ENABLED=true
AI_GATEWAY_LOCAL_PROVIDER=siliconflow
AI_GATEWAY_LOCAL_PROVIDER_BASE_URL=https://api.siliconflow.cn/v1
AI_GATEWAY_LOCAL_PROVIDER_API_KEY=<local-provider-api-key>
AI_GATEWAY_LOCAL_CHAT_MODEL=deepseek-ai/DeepSeek-V3
AI_GATEWAY_LOCAL_EMBEDDING_MODEL=BAAI/bge-m3
AI_GATEWAY_LOCAL_EMBEDDING_DIMENSIONS=1024
AI_GATEWAY_LOCAL_RERANK_MODEL=BAAI/bge-reranker-v2-m3
AI_GATEWAY_LOCAL_RERANK_TOP_N=5
```

再次运行 `./scripts/local/start.sh` 后，脚本会用
`AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` 加密 provider key，更新 AI Gateway 默认
profiles，并新增/激活一条 QA `llm_config_versions` 记录指向 `default-chat`。
生成的 SQL 不应提交，真实 key 只留在本机 `.env`。

`minio-init` 会创建两个本地 bucket：`software-teamwork-local` 供 File service
使用，`software-teamwork-knowledge` 供 Knowledge runtime 使用。

`001-local-demo-seed.sql` 里的本地管理员密码 hash 是
`LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345` 对应的 `argon2id` PHC 字符串，
参数为 `m=65536`、`t=3`、`p=2`、16-byte salt、32-byte key。轮换本地密码时，
需要一起更新 `.env.example`、`001-local-demo-seed.sql` 和本文档，然后重新
运行 `./scripts/local/start.sh` 让 host-run seed SQL 生效。不要把 demo 密码或 hash 用在共享环境或长期环境。

## 排障入口

- Docker 拉取慢、registry rewrite、daemon mirror、proxy 和 WSL 内存：
  [docs/runbooks/docker-image-pull-environment.md](../docs/runbooks/docker-image-pull-environment.md)
- 本地联调顺序、端口和故障判断：
  [docs/runbooks/local-integration.md](../docs/runbooks/local-integration.md)

后端启动后，可以通过 Gateway 确认 seed 管理员和权限：

```bash
curl --noproxy '*' -fsS http://localhost:8080/api/v1/sessions \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"LocalDemoAdmin#12345"}'
```

响应应包含 `admin` 角色，以及 `admin:model-profile:write`、
`admin:parser-config:write`、`qa:settings:read` 等本地演示权限。拿到 token 后，
可以请求 `/api/v1/admin/parser-configs` 或 `/api/v1/admin/model-profiles` 确认
Gateway 管理路由鉴权正常。

只清理本地 demo seed 数据：

```bash
set -a
source .local/config/dev.env.sh
set +a
psql "$POSTGRES_ADMIN_URL" -v ON_ERROR_STOP=1 -f deploy/seeds/099-local-demo-cleanup.sql
```

完整重置本地 infra 数据：

```bash
./scripts/local/clean.sh --yes
./scripts/local/start.sh
```

AI Gateway 的本地 placeholder profile 只用于 readiness 检查，里面不是可用的真实
provider key。真正调用模型前，需要通过管理端配置真实 provider credential，或在
`.env.local` 中启用 `AI_GATEWAY_LOCAL_SEED_ENABLED=true` 的本地 overlay。默认
OpenAI-compatible 地址是 `http://localhost:11434/v1`。

## Request ID 排障

所有服务都会返回或透传 `X-Request-Id`。

```bash
rid=req_local_debug_001
curl --noproxy '*' -fsS http://localhost:8080/readyz -H "X-Request-Id: ${rid}"
rg "${rid}" .local/logs
```

前端问题先记录响应里的 `requestId` 或 `X-Request-Id`，再查 Gateway 日志。如果
Gateway 报依赖错误，用同一个 id 继续查 owner service 日志。

## Knowledge 集成确认

Knowledge 主路径通过 Gateway 暴露：

```bash
curl --noproxy '*' -fsS http://localhost:8080/api/v1/knowledge-bases \
  -H "Authorization: Bearer ${token}" \
  -H "X-Request-Id: req_knowledge_local_001"
curl --noproxy '*' -fsS http://localhost:8080/api/v1/knowledge-bases/kb_local_demo/documents \
  -H "Authorization: Bearer ${token}"
curl --noproxy '*' -fsS http://localhost:8080/api/v1/documents/<documentId>/chunks \
  -H "Authorization: Bearer ${token}"
curl --noproxy '*' -fsS http://localhost:8080/api/v1/knowledge-queries \
  -H "Authorization: Bearer ${token}" \
  -H 'Content-Type: application/json' \
  -d '{"query":"local demo","topK":3}'
```

Knowledge 路由依赖 `VENDOR_RUNTIME_URL` 指向 Knowledge runtime。runtime 负责 PDF
解析、切块、embedding、索引和检索；不要再启动 `services/parser`。

宿主机启动 runtime API；查询-only 用 API helper，需要真实 ingestion 时启动完整
parse stack：

```bash
cp .env.example .env.local
# Edit .env.local with AI_GATEWAY runtime embedding/rerank variables,
# optional explicit direct-provider fallback variables,
# optional AI_GATEWAY_LOCAL_PROVIDER_* seed variables,
# KNOWLEDGE_AUTO_START_INGESTION=true, and DOC_ENGINE=elasticsearch.
./scripts/local/start.sh --runtime api
./scripts/local/start.sh
```

更多 runtime 说明见 [services/knowledge-runtime/README.md](../services/knowledge-runtime/README.md)。

## Common Dependency Failures

| Symptom | Likely cause | Check |
| --- | --- | --- |
| `gateway /readyz` returns `503 dependency_error` | Redis, auth, or required owner service base URL configuration is not ready | `docker compose ps`, service logs under `.local/logs/` |
| `auth /readyz` returns `postgres unavailable` | Auth migration or PostgreSQL failed | `docker compose logs postgres`; check `.local/logs/auth.log` |
| Knowledge upload/query returns `502 dependency_error` | Knowledge runtime unreachable or ES/MinIO not ready | Check `VENDOR_RUNTIME_URL`, runtime API, worker, Elasticsearch, and MinIO |
| Knowledge adapter `/readyz` says `task executor heartbeat is missing` | Runtime worker exited or is still downloading deepdoc models | Check `.local/logs/knowledge-runtime-worker.log`; for mainland China rerun `./scripts/local/start.sh --runtime full --china` or set a local/internal `HF_ENDPOINT` |
| QA message call fails on model invocation | AI Gateway profile is not running, fake local credential is still in use, or host provider is not listening on `host.docker.internal:11434` | Check `.local/logs/ai-gateway.log` and `.local/logs/qa.log` |
| MinIO bucket missing | `minio-init` did not complete or one of `software-teamwork-local` / `software-teamwork-knowledge` is absent | `docker compose logs minio minio-init` |
| Host port conflict | Another local process or old host-run/tmux session uses a default port | `lsof -nP -iTCP:<port> -sTCP:LISTEN`; stop the old process or change the matching `*_PORT` in `.env.local` |
