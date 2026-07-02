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
| 向量数据库 | Qdrant |
| 对象存储 | MinIO |
| 本地基础设施 | Docker Compose 拉取 PostgreSQL、Redis、Qdrant、MinIO |
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
   +--> parser runtime
   +--> 文档生成
   +--> AI Gateway

基础设施:
postgres + redis + qdrant + minio
```

服务职责：

| 服务 | 职责 |
| --- | --- |
| `frontend` | 面向用户的 React + TypeScript 应用，通过网关访问后端能力。 |
| `gateway` | 后端统一入口，负责路由、鉴权上下文传递、聚合接口和跨服务请求协调。 |
| `auth` | 用户身份、登录认证、权限控制、令牌或会话管理。 |
| `file` | 文件上传、文件元数据、对象存储协调，以及文件处理流程入口。 |
| `qa` | 智能问答服务，作为 Agent Host 管理会话、ReAct 循环、MCP 工具调用、引用和回答持久化。 |
| `knowledge` | 知识导入状态、切分、索引、元数据管理和检索协调。 |
| `parser` | 内部文档解析运行时，将 raw bytes 转成规范化 parsed content；首期目标为 Python/PaddleOCR，不通过 gateway 暴露。 |
| `document` | 报告、材料、知识摘要等文档生成流程。 |
| `ai-gateway` | 模型 profile、provider credential、chat/embedding/rerank 调用的统一内部入口。 |

基础设施职责：

| 组件 | 用途 |
| --- | --- |
| PostgreSQL | 业务数据、用户数据、文件元数据、知识元数据。 |
| Redis | 缓存、会话、短期任务状态或轻量队列。 |
| Qdrant | 向量索引和相似度检索。 |
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
- Parser Runtime 服务文档：[docs/services/parser/README.md](docs/services/parser/README.md)
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
│   ├── parser/
│   │   ├── api/
│   │   └── src/
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

默认启动路径是：Docker 只拉取并启动基础设施，所有业务服务都在宿主机运行。
不要使用任何业务服务容器、服务级 Compose、构建参数或生产 Compose。

先准备本机工具：

- Docker Engine 或 Docker Desktop，带 Compose v2。
- Go `1.25.x`，例如 `1.25.1` 或 `1.25.4` 都可。
- uv。Parser 的 Python `3.12` 运行时由 uv 按项目配置选择或安装。
- Bun。
- `psql` 客户端。PostgreSQL server 由 Docker 启动，本机不用装 PostgreSQL server。
- `curl`。

第一次启动：

```bash
git clone https://github.com/Sakayori-Iroha-168/Software_Teamwork.git
cd Software_Teamwork

cp deploy/.env.example deploy/.env
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh

cd apps/web
bun install
bun run dev
```

日常启动：

```bash
./scripts/local/dev-up.sh
./scripts/local/run-backend.sh
cd apps/web && bun run dev
```

停止后端：

```bash
./scripts/local/stop-backend.sh
```

`deploy/.env.example` 是唯一默认配置来源。用户只复制成 `deploy/.env`；
脚本不会创建、改写或维护另一套默认变量，只会读取 `deploy/.env` 让宿主机进程拿到配置。
默认 demo 账号来自 `deploy/.env.example`：`admin` / `LocalDemoAdmin#12345`，
`superadmin` / `LocalDemoAdmin#12345`。
`UV_DEFAULT_INDEX` 也在这份文件里，默认使用清华 PyPI 镜像加速 Parser 首次准备
PaddleOCR 依赖；它影响 uv，不影响 Docker。Parser 的 `uv.lock` 同样锁到清华源，
`uv sync --frozen` 会按锁文件里的 URL 下载包，不能靠删除 `UV_DEFAULT_INDEX` 切回
官方 PyPI。第一次启动仍会下载几十个 OCR 依赖包，之后会走 uv 缓存。

`./scripts/local/dev-up.sh` 会拉取并启动 `postgres`、`redis`、`qdrant`、`minio`、
`minio-init`，等待基础设施健康后创建/校验 Knowledge 的 Qdrant collection，
再执行本机 migration 和 demo seed。
`./scripts/local/run-backend.sh` 会启动 `auth`、`file`、`parser`、`knowledge`、
`ai-gateway`、`qa`、`document` 和 `gateway`，日志在 `.local/logs/`。

`ai-gateway /readyz` 在 placeholder credential 下返回 `503 degraded` 是预期行为，
不代表服务没起。默认本地模型 profile 指向宿主机 `http://localhost:11434/v1`。
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
| Migration CI | `services/**/migrations/**` | 校验 migration 文件名并用 `goose@v3.27.1` apply 到 PostgreSQL 16。 |
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
