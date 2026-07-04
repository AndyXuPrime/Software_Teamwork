# 电力行业知识管理系统

本仓库用于开发一个面向电力行业的知识管理系统。系统目标是沉淀行业文档、文件资料和结构化知识，并提供知识检索、智能问答和文档生成能力。

当前仓库已从架构与工程规范建设进入核心能力落地、联调与 CI 验证收口阶段。前端应用、主要后端服务、本地基础设施 Compose 和路径过滤 CI 已落地；后续重点是补齐跨服务闭环、真实运行时 smoke、端到端验收和发布流水线。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React + TypeScript |
| 后端 | Go 微服务 |
| 服务通信 | RESTful HTTP API |
| 关系数据库 | PostgreSQL |
| 缓存 / 队列 | Redis |
| 检索 / 索引后端 | Knowledge runtime doc engine（Elasticsearch） |
| 对象存储 | MinIO |
| 本地基础设施 | Docker Compose 拉取 PostgreSQL、Redis、MinIO、Elasticsearch |
| CI/CD | GitHub Actions |
| 仓库结构 | Monorepo |

前端当前落地在 `apps/web/`，使用 Bun + Vite。后续如果切换构建工具，再同步更新启动命令和 CI 细节。

## 系统架构

系统采用以网关为入口的微服务架构：

```text
frontend
   |
   v
gateway service
   |
   +--> auth service
   +--> file service
   +--> 智能问答
   +--> 知识库
   +--> Knowledge runtime
   +--> 文档生成
   +--> AI Gateway

基础设施:
postgres + redis + minio + minio-init + elasticsearch
```

服务职责：

| 服务 | 职责 |
| --- | --- |
| `frontend` | 面向用户的 React + TypeScript 应用，通过网关访问后端能力。 |
| `gateway` | 后端统一入口，负责路由、鉴权上下文传递、聚合接口和跨服务请求协调。 |
| `auth` | 用户身份、登录认证、权限控制、令牌或会话管理。 |
| `file` | 文件上传、文件元数据、对象存储协调，以及文件处理流程入口。 |
| `qa` | 智能问答服务，作为 Agent Host 管理会话、ReAct 循环、MCP 工具调用、引用和回答持久化。 |
| `knowledge` | 知识导入状态、Knowledge runtime 适配、解析/切块/索引状态、元数据管理和检索协调。 |
| `document` | 报告、材料、知识摘要等文档生成流程。 |
| `ai-gateway` | 模型 profile、provider credential、chat/embedding/rerank 调用的统一内部入口。 |

基础设施职责：

| 组件 | 用途 |
| --- | --- |
| PostgreSQL | 业务数据、用户数据、文件元数据、知识元数据。 |
| Redis | 缓存、会话、短期任务状态或轻量队列。 |
| Elasticsearch | Knowledge runtime 的 active doc engine，负责检索索引与相似度检索。 |
| MinIO | 原始文件、生成文档和其他对象数据。 |

项目文档入口：

- 文档索引：[docs/README.md](docs/README.md)

Gateway 基础契约文档：

当前已确定的前后端公开契约覆盖 gateway 健康检查、auth、knowledge-owned 知识库/文档上传/文档处理/原文件内容/切片/检索接口、qa-owned 智能问答接口，以及 document-owned 报告生成接口。File 服务只作为后端内部基础文件能力，不直接拥有前端公开 API。管理后台概览/指标聚合接口仍在设计中，暂在 OpenAPI 中标记为缺失占位。

所有前端到 gateway、gateway 到下游服务、服务到服务的 HTTP API 都必须使用 RESTful 资源路径，由 HTTP method 表达动作。除 `/healthz`、`/readyz` 外，不在稳定 path 中使用 `login`、`logout`、`register`、`download`、`search`、`generate`、`export`、`retry`、`revoke` 等动作词。

- Gateway 服务规划：[docs/services/gateway/README.md](docs/services/gateway/README.md)
- Auth 服务接口文档：[docs/services/auth/README.md](docs/services/auth/README.md)
- File 服务接口文档：[docs/services/file/README.md](docs/services/file/README.md)
- Knowledge 服务接口文档：[docs/services/knowledge/README.md](docs/services/knowledge/README.md)
- Knowledge 服务接口文档：[docs/services/knowledge/README.md](docs/services/knowledge/README.md)
- Gateway OpenAPI 契约：[docs/services/gateway/api/public.openapi.yaml](docs/services/gateway/api/public.openapi.yaml)
- 服务边界矩阵：[docs/architecture/service-boundaries.md](docs/architecture/service-boundaries.md)
- 前后端集成契约：[docs/architecture/frontend-backend-contract.md](docs/architecture/frontend-backend-contract.md)

## 目标目录结构

```text
.
├── apps/
│   └── web/
├── services/
│   ├── gateway/
│   │   └── go.mod
│   ├── auth/
│   │   └── go.mod
│   ├── file/
│   │   └── go.mod
│   ├── qa/
│   │   └── go.mod
│   ├── knowledge/
│   │   └── go.mod
│   └── document/
│       └── go.mod
├── deploy/
│   └── docker-compose.yml
├── docs/
├── .github/
│   └── workflows/
└── .trellis/
    ├── spec/
    └── tasks/
```

每个 Go 微服务维护独立的 `go.mod`，作为独立的构建和测试单元。除非后续明确引入共享库，否则不要默认跨服务共享 Go 包。

## 本次更新后的本地启动方法

默认启动路径是：Docker 只启动基础设施，业务服务和 Knowledge runtime 都在宿主机运行。
启动脚本会先做本机 preflight，再按需准备缺失的本地工具、服务二进制、Knowledge runtime
依赖和基础设施镜像，之后启动 infra、migration、seed、runtime 和后端服务。

本机需要：Docker/Compose v2、Go `1.25.x`（脚本最低接受 `go1.25.1`，当前本地基线可用
`go1.25.4`）、uv、Bun、`psql`、`curl`、Python 3。
默认使用官方源：Docker Hub/Elastic registry、PyPI、`proxy.golang.org` 和
`sum.golang.org`；仓库不提交 active 第三方镜像值。中国大陆网络直接给启动脚本加
`--china`。

第一次启动前：

```bash
git clone https://github.com/Sakayori-Iroha-168/Software_Teamwork.git
cd Software_Teamwork

cp .env.example .env.local
./scripts/local/start.sh

cd apps/web
bun install
bun run dev
```

中国大陆网络第一次启动：

```bash
cp .env.example .env.local
./scripts/local/start.sh --china

cd apps/web
bun install
bun run dev
```

日常启动：

```bash
./scripts/local/start.sh
cd apps/web && bun run dev
```

命令边界：

| 命令 | 会做什么 | 不会做什么 |
| --- | --- | --- |
| `./scripts/local/start.sh` | 检查宿主机环境，补齐缺失工具、二进制、runtime 依赖和 infra images，启动 infra、migration、seed、Knowledge runtime 和后端服务。 | 不创建或覆盖 `.env.local`；不启动前端 Vite。 |
| `./scripts/local/stop.sh` | 停止 `.local/run/` 中记录的 host-run 后端和 Knowledge runtime 进程组。 | 不停止 Docker infra；不停止前端 Vite。 |
| `./scripts/local/clean.sh` | 先 stop，再删除本地 infra Compose 容器和数据卷。 | 不删除 Docker images、`.env.local`、`.local/tools` 或 `.local/bin`。 |

停止后端和 Knowledge runtime：

```bash
./scripts/local/stop.sh
```

清空本地 infra 数据：

```bash
./scripts/local/clean.sh
```

`stop.sh` 只停止 `start.sh` 记录在 `.local/run/` 里的宿主机后端和 Knowledge runtime
进程组；不会停止 Docker infra，也不会停止前端 Vite，前端仍在对应终端按 `Ctrl+C` 结束。
`clean.sh` 会先执行 stop，再删除本地 Compose 容器和 PostgreSQL/MinIO/Elasticsearch
数据卷；不会删除 Docker images、`.env.local`、`.local/tools` 或 `.local/bin`。非交互环境
要直接清理时使用 `./scripts/local/clean.sh --yes`。

如果启动失败，直接看 `start.sh` 的阶段摘要和 `.local/logs/*.log`。脚本会说明失败发生在
preflight、prepare、infra、migration、seed、runtime 还是后端启动阶段。

`config/` 是唯一默认配置来源，根 `.env.example` 是本地 secret 模板。用户只复制成
未跟踪的 `.env.local`；脚本通过 `config/ctl` 渲染 `.local/config/dev.env` 和
`.local/config/dev.env.sh`，让 Docker Compose 和宿主机进程拿到同一份 profile 配置。
`start.sh` 只检查 `.env.local` 是否存在；缺失时退出并提示复制命令，不会创建或覆盖它。
默认 demo 账号来自 `.env.example`：`admin` / `LocalDemoAdmin#12345`，
`superadmin` / `LocalDemoAdmin#12345`。
Go modules 下载只在 `start.sh` 准备缺失或过期 Go 工具/服务二进制时发生；脚本会用
`.local/stamps/` 记录源码 fingerprint，用户 `git pull`、切分支或修改 Go 源码后会自动重建
相关本地产物。`--skip-prepare` 只允许复用与当前源码匹配的产物，不匹配时会失败并提示
重新运行不带 `--skip-prepare` 的启动命令。
`start.sh` 会在构建 config renderer、安装 goose、构建 seed helper 之前读取 `.env.local`
里的 `GOPROXY`、`GOSUMDB`、`GOPRIVATE`、`GONOPROXY`、`GONOSUMDB` 和 `GOINSECURE`，
所以长期企业 Go 源可以只写在 `.env.local`。`--china` 会在本次运行中覆盖这些 Go 源设置。
Knowledge runtime 的 uv sync 会校验 runtime profile 以及 `pyproject.toml` / `uv.lock` /
`download_deps.py` fingerprint，依赖输入变化时会自动重新 sync；artifact 下载仍只在缺失时
由 `start.sh` 自动执行。

`./scripts/local/start.sh` 会准备缺失的 infra images，启动并等待 `postgres`、`redis`、
`minio`、`elasticsearch` 健康；随后运行 `minio-init`、migration、demo seed、后端服务、
Knowledge runtime API 和 worker。启动后会输出 Docker infra、host-run 进程组和日志位置；
服务日志在 `.local/logs/`。
`start.sh`、`stop.sh` 和 `clean.sh` 都会输出开始、成功、警告和失败摘要。

`ai-gateway /readyz` 在 placeholder credential 下返回 `503 degraded` 是预期行为，
不代表服务没起。默认本地模型 profile 指向宿主机 `http://localhost:11434/v1`。
本机需要真实 provider 时，在 `.env.local` 设置 `AI_GATEWAY_LOCAL_SEED_ENABLED=true`
和 `AI_GATEWAY_LOCAL_*` 后重新运行 `./scripts/local/start.sh`；脚本会加密写入默认
AI Gateway profiles，并同步 QA active LLM model。
完整排障见 [deploy/README.md](deploy/README.md) 和
[Docker 镜像拉取环境与镜像源](docs/runbooks/docker-image-pull-environment.md)。

单个 Go 服务可以直接调试：

```bash
cd services/gateway
go mod download
go run ./cmd/server
```

运行质量检查：

```bash
# 前端
bun run --cwd apps/web check
bun run --cwd apps/web build

# Go 服务示例
cd services/gateway
go test ./...
go build ./cmd/server
```

## CI/CD 约定

GitHub Actions 应按服务路径拆分检查，避免一个服务的小改动触发所有服务的完整构建。

推荐流水线：

| 阶段 | 触发范围 | 内容 |
| --- | --- | --- |
| PR Guard | 所有 PR | 检查 PR 目标分支、fork 协作规则和分支同步状态。 |
| Commitlint | 所有 PR | 检查 Conventional Commits。 |
| Frontend CI | `apps/web/**`、根前端依赖文件 | 安装依赖，执行 `bun run --cwd apps/web check`、`build`、`test:unit` 和 Playwright smoke。 |
| Go Service CI | `services/<service>/**` | 只对变更服务执行 `go test ./...` 和 `go build ./cmd/server`；QA 额外构建 `cmd/agent`。 |
| Migration CI | `services/**/migrations/**` | 校验 migration 文件名并用 `goose@v3.27.0` apply 到 PostgreSQL 16。 |
| Docker / Deploy Checks | `deploy/docker-compose.yml`、Docker 镜像源文档、Docker policy 脚本 | 验证 infra-only Compose config、镜像 tag/overlay policy 和环境诊断脚本；不处理业务服务、不部署。 |

当前 `deploy/docker-compose.yml` 是本地/演示基础设施基线，不是生产部署基线，也不包含业务服务容器。

## 协作规范

本仓库采用 fork + PR 的协作方式：

- 从主仓库最新 `develop` 创建个人分支。
- 所有日常开发 PR 指向 `develop`。
- 禁止直接向 `develop` 或 `main` push 功能、修复或文档修改。
- `main` 只用于发布合并。

完整流程见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## Commit 规范

所有 commit 必须遵循 Conventional Commits：

```text
<type>(<scope>): <subject>
```

示例：

```text
feat(gateway): add health check endpoint
fix(auth): handle expired token
docs(readme): describe service architecture
chore(ci): add go service workflow
```

完整规则见 [.trellis/spec/guides/commit-convention.md](.trellis/spec/guides/commit-convention.md)。

## 工程规范

后续实现应遵循 Trellis spec：

- 后端规范：[.trellis/spec/backend/](.trellis/spec/backend/)
- 前端规范：[.trellis/spec/frontend/](.trellis/spec/frontend/)
- CI/CD 规范：[.trellis/spec/cicd.md](.trellis/spec/cicd.md)
- 共享思考指南：[.trellis/spec/guides/](.trellis/spec/guides/)
