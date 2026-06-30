# RAGFlow Runtime Vendor 说明

本目录是从上游 RAGFlow 拉取的隔离源码快照，供 Knowledge/RAG 后续适配文档解析、检索、MCP 能力时参考和复用。它不是当前项目的可运行服务，也没有接入现有 Knowledge、Parser、Gateway 或 QA 微服务。

## 上游来源

- 上游仓库：https://github.com/infiniflow/ragflow
- 导入分支：`main`
- 导入提交：`45fc7feab4a0da6fec2d0fecbae67fabdc9bb3a2`
- 导入时间：2026-07-01
- 许可证：Apache License 2.0，见 `LICENSE`
- 本地来源记录：`UPSTREAM.md`

## 当前保留重点

- `deepdoc/`：文档解析、版面分析、OCR/TSR 相关源码。
- `rag/`：RAG、检索、切片、LLM 适配和 GraphRAG 相关源码。
- `api/`、`common/`、`conf/`：上游运行时、配置和服务代码参考。
- `mcp/`：MCP server/client 相关实现，后续需要为 QA 和 agent 编排团队提供能力时保留参考。
- `sdk/`、`docs/`：暂作为 API、MCP、dataset、DeepDoc 和配置参考资料保留。
- `docker/`、`Dockerfile_deepdoc_oss`：仅作为后续容器化设计参考；当前快照不能直接按上游完整产品启动。
- `test/unit_test/deepdoc/`、`test/unit_test/rag/`、`test/unit_test/mcp/`、`test/fixtures/`：保留作为解析、RAG、MCP 行为参考。

## 已清理内容

为避免误导后续开发，本地快照已清理与当前适配目标无关或已不再可运行的上游内容：

- `web/`：上游前端 UI，本项目前端由 `apps/web/` 负责。
- `helm/`：上游 Helm chart，本项目部署编排另行维护。
- `.github/`：上游 GitHub workflow、模板和元数据。
- `agent/`：上游 Python agent runtime；agent 编排不是 Knowledge/RAG 团队职责。
- `admin/`：上游 Python admin 服务和 CLI。
- `tools/`：上游独立插件、迁移、安装和开发辅助工具。
- `example/`：上游 demo 和示例调用脚本。
- 部分 `test/`：已删除针对上游 Web/Admin/Agent 表面的测试。
- 部分 `docker/`：已删除非核心 compose 变体和 OceanBase 专用启动内容。
- 其他语言 README：仅保留本文件作为本地中文说明。

## 使用边界

- 不要在现有 Go Knowledge 服务中直接 import 这里的源码。
- 不要在 vendor 清理阶段把 RAGFlow 接入 Knowledge、Parser、Gateway、QA 或 CI。
- 后续适配应优先明确 HTTP/runtime 边界，再决定哪些能力迁移到项目自己的服务目录。
- 如果要做容器化，应基于本项目服务边界重新设计 Dockerfile/Compose/部署配置；这里保留的上游 Docker 文件只作为参考。

## 注意事项

- 当前目录是 trimmed snapshot，不是完整 RAGFlow 产品源码。
- 上游完整 Docker Compose、Web UI、Agent、Admin 等路径已经被删除或部分断开。
- 若未来需要恢复完整上游产品能力，应重新从上游仓库拉取对应版本，而不是在当前 trimmed snapshot 上补齐。
- 每轮删除都应独立提交，便于审计。

## 刷新方式

如需重新同步上游，请先阅读 `UPSTREAM.md`。刷新后需要重新评估本地保留/删除边界，并更新导入提交号、许可证和清理记录。
