# Docker 基础设施镜像拉取环境

本手册覆盖两类 Docker 路径：

- 默认本地联调的基础设施镜像拉取。
- 独立 cloud Docker app stack 的基础镜像拉取和构建源策略。

默认本地联调的根级 Docker 路径只允许：

- `postgres`
- `redis`
- `minio`
- `minio-init`
- `elasticsearch`

在默认本地联调路径中，业务服务、migration、seed、Knowledge runtime 和前端都不通过根级 Docker Compose 启动。
Knowledge runtime 的 `uv sync` 下载 Python 包，不走 Docker registry。uv 默认包索引由
`config/base.yaml` 里的 `UV_DEFAULT_INDEX` 控制；默认是官方 PyPI。Knowledge runtime
Python 依赖和 artifact 由 `./scripts/local/start.sh` 按当前源模式准备；中国大陆网络使用
`--china`。不要把 `pyproject.toml` 或 `uv.lock` 的默认 URL 改成第三方代理。
Go 模块下载是宿主机 Go 工具链行为，不属于 Docker registry 问题。`start.sh` 会在构建
`config-ctl`、安装 goose 和构建 seed helper 前先读取 shell 环境与 `.env.local` 中的
`GOPROXY` / `GOSUMDB` 等 Go 源变量；渲染配置后，profile 中的 Go 源默认值继续用于后续
服务二进制准备。默认是官方 `proxy.golang.org` / `sum.golang.org`。中国大陆网络用
`./scripts/local/start.sh --china` 在本次准备阶段启用 Go mirror。
`./scripts/local/start.sh` 会在缺少所选 infra image 时执行 `docker pull`，随后用
`docker compose up --pull never` 启动。

`deploy/docker-compose.cloud.yml` 是另一条显式 cloud Docker app stack。它允许 `build:`，
用于构建业务服务和前端镜像，但外接云端 PostgreSQL、Redis、对象存储、Knowledge
runtime、PaddleOCR 和模型 provider。它不改变根级 `deploy/docker-compose.yml`
的 infra-only 基线；模板默认 `DOCKER_SEED_ENABLED=false`，不会默认写入本地 demo seed。

## 源策略契约

当前契约是默认官方源、国内网络显式 `--china`。
PR review 和 agent 检查应把 active 第三方镜像默认值视为回归；缺少这些 active 默认值
不是回归。中国大陆用户的一等路径是运行 `./scripts/local/start.sh --china`，或使用本机
未提交的 `.env.local` / 企业镜像覆盖。

## 默认路径

默认使用 Compose 里的 Docker Hub pinned tags：

```bash
cp .env.example .env.local
./scripts/local/start.sh
```

中国大陆网络拉取 Docker 基础设施镜像慢时，显式使用：

```bash
./scripts/local/start.sh --china
```

该模式在本次运行的生成态 compose env 中使用 `POSTGRES_IMAGE`、`REDIS_IMAGE`、
`MINIO_IMAGE`、`MINIO_MC_IMAGE` 和 `KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE`
的 `docker.1ms.run` registry rewrite，不改写 `.env.local`。如果镜像缺失，`start.sh`
会先拉取所选镜像，再用 `docker compose up --pull never` 启动并执行 migration 和 seed。
只想验证 Docker 配置时：

```bash
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --services
```

服务清单只能包含 `postgres`、`redis`、`minio`、`minio-init`、
`elasticsearch`。

## Docker 镜像源选择

Compose 文件本身保留 Docker Hub pinned defaults。需要企业 registry 或长期本地
override 时，可只在本机 `.env.local` 设置 `POSTGRES_IMAGE`、`REDIS_IMAGE`、
`MINIO_IMAGE`、`MINIO_MC_IMAGE` 和 `KNOWLEDGE_RUNTIME_ELASTICSEARCH_IMAGE`，
不要提交成默认值。已有旧
`.env.local` 如果仍保留第三方 registry 值，脚本会尊重本地配置；不传 `--china` 时会提示
这是本地覆盖。

优先级固定为：

```text
registry rewrite > daemon mirror > proxy
```

原因：

- registry rewrite 通过 `--china` 或本地 `.env.local` 显式选择，团队可审查、可复制。
- daemon mirror 是个人机器状态，适合已有稳定镜像站的人。
- proxy 依赖 shell、Docker daemon 和系统代理是否同时生效，最容易出现“终端能访问但 Docker 不走代理”。

## Cloud Docker app stack

云端依赖 Docker 路径使用：

```bash
cp deploy/docker/cloud.env.example .env.docker.cloud
./scripts/docker/start.sh
```

它构建这些镜像：

- Go 服务镜像：基于 `golang:1.25-alpine` 和 `alpine:3.22`。
- migration/seed 工具镜像：基于 `golang:1.25-alpine` 和 `alpine:3.22`，内含 `goose@v3.27.0`、`psql` 和 AI Gateway seed helper。
- 前端镜像：基于 `oven/bun:1.3.12-alpine` 构建，再用 `nginx:1.27-alpine` 服务静态资源并反代 Gateway。

可在 `.env.docker.cloud` 覆盖：

```text
DOCKER_IMAGE_REGISTRY_PREFIX=
GO_DOCKER_GOPROXY=https://proxy.golang.org,direct
GO_DOCKER_GOSUMDB=sum.golang.org
GO_DOCKER_IMAGE=golang:1.25-alpine
ALPINE_DOCKER_IMAGE=alpine:3.22
ALPINE_MIRROR=
BUN_DOCKER_IMAGE=oven/bun:1.3.12-alpine
NGINX_DOCKER_IMAGE=nginx:1.27-alpine
```

不要把这些默认 tag 改成 `latest`。中国大陆网络或企业 registry 可以用
`DOCKER_IMAGE_REGISTRY_PREFIX` 或本机 Docker daemon mirror/proxy；Go module 下载由
`GO_DOCKER_GOPROXY` / `GO_DOCKER_GOSUMDB` 控制；Alpine `apk` 包索引由
`ALPINE_MIRROR` 控制。这三类源和 Docker registry 不是同一类问题。

只验证 cloud Compose 配置：

```bash
docker compose -f deploy/docker-compose.cloud.yml --env-file deploy/docker/cloud.env.example config --quiet
```

## 环境诊断

中国大陆 registry rewrite 路径：

```bash
python3 scripts/check_docker_environment.py --profile china --clean-env
```

Docker Hub 默认路径：

```bash
python3 scripts/check_docker_environment.py --profile default --clean-env
```

Docker Hub 通过当前代理环境访问：

```bash
export HTTP_PROXY=http://127.0.0.1:<proxy-port>
export HTTPS_PROXY=http://127.0.0.1:<proxy-port>
export NO_PROXY=localhost,127.0.0.1,::1
python3 scripts/check_docker_environment.py --profile default
```

`--clean-env` 会故意清掉出站代理变量，适合验证直连/daemon mirror；验证官方源经代理可达时不要加它。

完整诊断：

```bash
python3 scripts/check_docker_environment.py --profile all --clean-env
```

CI 只跑无网络诊断：

```bash
python3 scripts/check_docker_environment.py --skip-network --clean-env
```

## 常见现象

拉取进度卡住时，先区分是哪个镜像卡住：

```bash
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull postgres
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull redis
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull minio
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull minio-init
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env pull elasticsearch
```

如果 registry rewrite 路径异常，先用环境诊断脚本确认 manifest 是否可用，再决定是否临时去掉
`--china` / 本地 `*_IMAGE` override 走 Docker Hub、切换本机 daemon mirror，或配置
Docker daemon proxy。

`minio-init` 退出是正常行为。它是 `minio/mc` 客户端，完成 bucket 创建后退出。真正的
对象存储服务是 `minio`。

`ai-gateway /readyz` 返回 `503 degraded` 不是 Docker 问题。本地 host-run 默认 seed 写入的是
placeholder provider credential；cloud Docker app stack 默认不执行 demo seed。`/healthz` 成功表示服务进程可用。host-run
默认模型 profile 指向宿主机 `http://localhost:11434/v1`，不需要
`host.docker.internal`。

## 策略检查

修改 Docker 相关文件后运行：

```bash
python3 scripts/check_docker_policy.py
python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment
CONFIG_SECRET_FILE=.env.example ./scripts/config/load-profile.sh --print-compose-env
docker compose -f deploy/docker-compose.yml --env-file .local/config/dev.env config --quiet
docker compose -f deploy/docker-compose.cloud.yml --env-file deploy/docker/cloud.env.example config --quiet
```

策略要求：

- 根 Compose 只包含五个基础设施服务：`postgres`、`redis`、`minio`、
  `minio-init`、`elasticsearch`。
- 根 Compose 不包含 `build:`；`deploy/docker-compose.cloud.yml` 是明确批准且唯一允许 `build:` 的业务 Docker app stack。
- 默认镜像不能是 `latest`。
- `config/base.yaml` 和 `.env.example` 不能默认启用第三方基础设施镜像 rewrite；大陆网络使用
  `./scripts/local/start.sh --china` 或本地 `.env.local` override。
- 所有基础设施镜像 tag 必须 pinned，不能使用 `latest`。
