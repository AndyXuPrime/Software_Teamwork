# Knowledge Runtime（裁剪版）

本目录是上游 [RAGFlow](https://github.com/infiniflow/ragflow) 的隔离快照，作为 Knowledge 的 **vendor 运行时** 部署。Go 契约适配器在 `services/knowledge/cmd/adapter`，通过 `VENDOR_RUNTIME_URL` 调用本目录 Python API（`:9380`）。

完整上游信息与 refresh 步骤见 [`UPSTREAM.md`](UPSTREAM.md)。

## 进程

| 服务 | 端口 | 入口 | 职责 |
| --- | --- | --- | --- |
| `knowledge-runtime-api` | `:9380` | `api/ragflow_server.py` | 数据集/文档/检索 HTTP API |
| `knowledge-runtime-worker` | n/a | `rag/svr/task_executor.py` | deepdoc 解析、分块、嵌入（Redis 队列） |

共用 PostgreSQL（`knowledge_system`）、MinIO（`software-teamwork-knowledge`）、Elasticsearch、Redis。

## 已裁剪的产品面

上游 Web UI、Agent、Admin、Chat、用户注册/登录、Go HTTP 运行时、容器内 nginx、vendor 自带 docker-compose 等已移除。运行时信任 Gateway 注入的 `X-Tenant-Id` / `X-User-Id`。

## 主要目录

| 路径 | 说明 |
|------|------|
| `api/` | Python REST API 与 DB 服务（adapter 调用面） |
| `deepdoc/` | 文档解析器与视觉模型 |
| `rag/` | 分块、嵌入、检索、GraphRAG、任务执行 |
| `docker/` | 容器 entrypoint |
| `conf/` | 运行时配置（compose 覆盖见 `service_conf.compose.yaml`） |
| `common/data_source/` | 多源连接器参考代码（默认不启用） |
| `docs/` | parser/RAG 参考文档 |

## 本地验证

```bash
bash -n docker/entrypoint.sh
python3 -m py_compile api/apps/__init__.py rag/prompts/generator.py
```

## 许可证

Apache License 2.0，详见 [`LICENSE`](LICENSE)。
