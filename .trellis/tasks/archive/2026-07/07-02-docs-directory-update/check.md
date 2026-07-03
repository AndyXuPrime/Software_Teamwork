# Trellis 任务全量检查记录

日期：2026-07-03

## 范围

- PR：#502 `docs(architecture): refresh current capability gaps`
- 分支：`SakayoriTeam/docs/update-docs`
- 最新基线：`origin/develop@ce0b4774`
- 本轮新增范围：
  - rebase 到最新 `develop`，吸收已合入的 #529 Knowledge MCP 工具 schema 文档。
  - 将 #531 Knowledge MCP server 协议与运行说明纳入本 PR，并按当前代码事实修正。
  - 再次检查架构、服务实现、QA、测试矩阵和 Trellis 任务记录中的当前状态/缺口。
  - 检查 PR 描述是否符合仓库模板。

## 事实核对

- #529 已合入 `develop`，`docs/services/knowledge/docs/mcp-tools.md` 来自最新基线。
- #531 仍是 open PR；本 PR 只吸收其 MCP server 文档内容，不带入对方 Trellis 任务元数据。
- 当前 `services/knowledge/internal/mcp` 实际发布 14 个原生工具，并通过
  `KNOWLEDGE_MCP_ADDR` 启动独立 Streamable HTTP endpoint。
- #529 文档中的四个 `knowledge__*` 工具是目标收敛契约；默认 QA RAG 当前仍走内置
  `search_knowledge`。
- `services/parser` 已退役；当前 Knowledge 运行路径是 adapter ->
  `services/knowledge-runtime` RAGFlow runtime。

## 检查项

| 检查 | 结果 | 说明 |
| --- | --- | --- |
| 最新基线 | pass | 已 fetch 并 rebase 到 `origin/develop@ce0b4774`。 |
| PR #529 文档纳入 | pass | #529 已在基线，本文档和 PR 描述明确记录其状态。 |
| PR #531 文档纳入 | pass | 新增 `docs/services/knowledge/docs/mcp-server.md`，并更新 Knowledge README 入口。 |
| Knowledge MCP 代码/文档一致性 | pass | 对照 `cmd/adapter`、`internal/mcp`、`adapterconfig` 和 `deploy/.env.example`，将文档改为 `KNOWLEDGE_MCP_ADDR` 独立 endpoint 与 14 个原生工具。 |
| 旧 Parser 当前路径扫描 | pass | 新增/触碰文档不把旧 `services/parser` 当作当前依赖；历史记录保留为历史路径。 |
| Knowledge runtime 边界扫描 | pass | `api-contract.md`、`data-models.md`、`permission-matrix.md` 和 `technology-decisions.md` 已从旧 File/Qdrant/asynq 路径调整为当前 RAGFlow runtime adapter 口径。 |
| PR 描述格式 | pending | 提交并推送后使用中文 PR 模板重写 #502 body。 |

## 已运行验证

- `bun run --cwd apps/web api:generate`：pass。
- `python3 -m unittest scripts.tests.test_verify_gateway_active_api`：pass。
- `python3 scripts/verify_gateway_active_api.py`：pass。
- Knowledge MCP / Knowledge docs JSON fence 解析：pass。
- changed Markdown relative-link check：pass。
- stale current-state scan：pass，剩余 `Qdrant` / `File Service` / `asynq` 命中均为本地 infra、Document/File 边界或明确历史/兼容说明。
- `cd services/knowledge && go test ./...`：pass。
- `git diff --check`：pass。

完整前端 build/check、全仓 Go 测试和真实 env-gated smoke 不属于本轮文档变更的必跑项；
本轮没有服务代码语义变更，Knowledge 侧已用 `go test ./...` 覆盖。

## 剩余风险

- #505 仍为 open issue；Knowledge MCP server 代码已随 #440 进入 `develop`，但默认
  QA 接入、四个 `knowledge__*` 目标工具、citation 识别和 #125 smoke 仍未收敛。
- #531 原 PR 尚未合并；本 PR 已吸收并修正其文档内容，后续可能需要关闭或同步对方 PR。
- 真实 RAGFlow runtime/provider、Knowledge MCP + QA + frontend 的完整端到端验收仍需
  env-gated 或共享环境 smoke。
