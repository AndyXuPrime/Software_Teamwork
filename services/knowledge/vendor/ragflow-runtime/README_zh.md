# RAGFlow Runtime（裁剪版）

本目录是上游 [RAGFlow](https://github.com/infiniflow/ragflow) 的隔离快照，挂载在 Knowledge 服务域下，供后续逐步适配文档解析、RAG 检索与 MCP 工具化能力。

完整上游信息与 refresh 步骤见 [`UPSTREAM.md`](UPSTREAM.md)。

## 保留范围

- **文档解析**：`deepdoc/`、`rag/app/`
- **RAG / 检索**：`rag/`（含 GraphRAG、RAPTOR、mindmap 索引）
- **检索反馈加权**：`api/db/services/chunk_feedback_service.py`（默认关闭，设 `CHUNK_FEEDBACK_ENABLED=true` 启用；根据引用 chunk 的点赞/点踩调整 `pagerank_fea`）
- **MCP / 工具化**：`mcp/`
- **容器化参考**：`docker/`、`Dockerfile*`、`build.sh`
- **对应测试**：`test/unit_test/deepdoc/`、`test/unit_test/rag/`、`test/unit_test/mcp/` 及相关 REST 集成测试

## 已裁剪的产品面

上游完整产品中的 Web UI、Agent、Admin、Chat、Dify 集成、用户注册/登录/租户协作、OceanBase 运维端点等已移除或不再暴露。运行时仍保留 API token / JWT 鉴权中间层，用于保护 dataset / document / retrieval / MCP 等核心 API。

## 主要目录

| 路径 | 说明 |
|------|------|
| `deepdoc/` | 文档解析器与视觉模型 |
| `rag/` | 分块、嵌入、检索、GraphRAG、任务执行 |
| `mcp/` | MCP server / client |
| `api/` | Python REST API 与 DB 服务 |
| `internal/` | Go API 与 ingestion 运行时 |
| `docker/` | Compose 与启动脚本参考 |
| `common/data_source/` | 多源连接器参考代码（默认不启用运行时） |
| `docs/` | 保留的 parser/RAG/MCP 参考文档 |

## 本地验证

```bash
# Python 语法抽查
python3 -m py_compile api/apps/__init__.py rag/prompts/generator.py

# Shell 脚本语法
bash -n docker/entrypoint.sh && bash -n docker/launch_backend_service.sh

# Go 路由注册测试
go test ./internal/router/...
```

## 许可证

Apache License 2.0，详见 [`LICENSE`](LICENSE)。
