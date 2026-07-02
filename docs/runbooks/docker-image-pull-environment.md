# Docker 基础设施镜像拉取环境

本手册只覆盖基础设施镜像拉取。仓库默认 Docker 路径只允许：

- `postgres`
- `redis`
- `qdrant`
- `minio`
- `minio-init`

业务服务、migration、seed、Parser 和前端都不通过 Docker 启动。
Parser 的 `uv sync` 下载 Python 包，不走 Docker registry。uv 默认包索引由
`deploy/.env.example` 里的 `UV_DEFAULT_INDEX` 控制。

## 默认路径

`deploy/.env.example` 已经写入 DaoCloud registry rewrite，适合中国大陆开发网络：

```bash
cp deploy/.env.example deploy/.env
./scripts/local/dev-up.sh
```

脚本内部会执行 Compose config、pull、up、migration 和 seed。只想验证 Docker 配置时：

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env config --quiet
docker compose -f deploy/docker-compose.yml --env-file deploy/.env config --services
```

服务清单只能包含 `postgres`、`redis`、`qdrant`、`minio`、`minio-init`。

## Docker Hub Fallback

Compose 文件本身保留 Docker Hub pinned fallback。如果不想走 DaoCloud，删除
`deploy/.env` 里的 `POSTGRES_IMAGE`、`REDIS_IMAGE`、`QDRANT_IMAGE`、
`MINIO_IMAGE` 和 `MINIO_MC_IMAGE` 行即可。

优先级固定为：

```text
registry rewrite > daemon mirror > proxy
```

原因：

- registry rewrite 写在 `deploy/.env.example`，团队可审查、可复制。
- daemon mirror 是个人机器状态，适合已有稳定镜像站的人。
- proxy 依赖 shell、Docker daemon 和系统代理是否同时生效，最容易出现“终端能访问但 Docker 不走代理”。

## 环境诊断

默认 DaoCloud 路径：

```bash
python3 scripts/check_docker_environment.py --profile china --clean-env
```

Docker Hub fallback：

```bash
python3 scripts/check_docker_environment.py --profile default --clean-env
```

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
docker compose -f deploy/docker-compose.yml --env-file deploy/.env pull postgres
docker compose -f deploy/docker-compose.yml --env-file deploy/.env pull redis
docker compose -f deploy/docker-compose.yml --env-file deploy/.env pull qdrant
docker compose -f deploy/docker-compose.yml --env-file deploy/.env pull minio
docker compose -f deploy/docker-compose.yml --env-file deploy/.env pull minio-init
```

如果 DaoCloud 路径异常，先用环境诊断脚本确认 manifest 是否可用，再决定是否临时删除
`*_IMAGE` override 走 Docker Hub fallback、切换本机 daemon mirror，或配置 Docker daemon proxy。

`minio-init` 退出是正常行为。它是 `minio/mc` 客户端，完成 bucket 创建后退出。真正的
对象存储服务是 `minio`。

`ai-gateway /readyz` 返回 `503 degraded` 不是 Docker 问题。默认 seed 写入的是
placeholder provider credential；`/healthz` 成功表示服务进程可用。host-run
默认模型 profile 指向宿主机 `http://localhost:11434/v1`，不需要
`host.docker.internal`。

## 策略检查

修改 Docker 相关文件后运行：

```bash
python3 scripts/check_docker_policy.py
python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment
docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet
```

策略要求：

- 根 Compose 只包含五个基础设施服务。
- 根 Compose 不包含 `build:`。
- 默认镜像不能是 `latest`。
- `deploy/.env.example` 的基础设施镜像 rewrite 必须保持 pinned。
