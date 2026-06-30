# 修复服务文档一致性问题

## Goal

修复当前 `develop` 文档中已经确认的状态漂移，避免 Gateway active paths、服务级 OpenAPI 版本、Knowledge route 落地状态和 Document internal contract 入口互相冲突。

## Requirements

- 更新 `docs/architecture/service-boundaries.md` 的缺失契约登记，明确 Gateway public OpenAPI 不是空白文件，只有管理概览和指标聚合占位项停留在 `x-missing-contracts`，不进入 active paths。
- 更新 `docs/architecture/technology-decisions.md` 的 OpenAPI 基线和契约版本表，使其匹配实际 `docs/services/*/api/*.yaml` 与 `services/*/api/openapi.yaml` 的 `openapi:` 头。
- 更新 `docs/architecture/current-capability-matrix.md`，移除 Knowledge active routes 仍为阶段性 501 的过期说法，保留真实跨依赖 smoke 等仍未完成的缺口。
- 更新 Document implementation 权威来源，明确 `docs/services/document/api/internal.openapi.yaml` 是内部运行和 report job contract 来源。
- 澄清 `docs/README.md` 中服务级 public OpenAPI 的路径前缀语义，避免误读为所有服务级 public path 必须直接写 `/api/v1`。

## Acceptance Criteria

- [x] `service-boundaries.md` 不再声称 `docs/services/gateway/api/public.openapi.yaml` 为空白。
- [x] `technology-decisions.md` 不再声称 Knowledge/QA/Document 所有契约均为 OpenAPI `3.0.3`。
- [x] `current-capability-matrix.md` 不再把已落地的 Knowledge document lifecycle、chunks/content 和 `knowledge-queries` active routes 写成阶段性 501 或仍需补齐的路由本身。
- [x] Document implementation 的权威来源包含 public 和 internal OpenAPI。
- [x] 文档补丁不修改服务代码或机器可读 OpenAPI 契约语义。

## Definition of Done

- 运行文本搜索确认旧问题表述已消失或只保留在正确上下文。
- 运行 `git diff --check`。
- 说明未运行代码测试的原因：本次只修改 Markdown。

## Technical Approach

采用文档事实对齐，不改 OpenAPI 或服务代码。以 Gateway OpenAPI、服务 implementation 文档和实际 OpenAPI 文件头为事实来源，更新架构和入口文档中的过期描述。

## Out of Scope

- 不新增或删除 OpenAPI path/schema。
- 不修改 Gateway/Knowledge/Document/QA 服务代码。
- 不修复现有 Docker 文档版本审计任务。

## Technical Notes

- 已核对 `docs/services/gateway/api/public.openapi.yaml` 存在大量 active paths，`x-missing-contracts` 只列 `GET /api/v1/admin-overview` 和 `GET /api/v1/admin-metrics`。
- 已核对 `docs/services/document/api/internal.openapi.yaml`、`docs/services/knowledge/api/internal.openapi.yaml`、`docs/services/qa/api/internal.openapi.yaml` 以及对应 `services/*/api/openapi.yaml` 使用 OpenAPI `3.1.0`。
- 已核对 Gateway 和 Knowledge implementation 文档记录 document lifecycle、chunks/content 和 `knowledge-queries` proxy/handler 已落地。
