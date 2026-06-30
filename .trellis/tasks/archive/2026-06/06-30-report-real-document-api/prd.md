# F-014 Report real Document API

## Goal

完成 GitHub issue #280：Report 前端去除 silent fallback，并按 Gateway Document active paths 接入真实 Document API。目标是把 Report workspace 从“页面骨架 + 假数据兜底”推进到 L2 真实 API 接入，同时对尚未 ready 的 AI 生成、Document MCP tools 和富 DOCX 能力做显式未就绪/禁用状态。

## Requirements

- 从最新 `upstream/develop` 派生个人 fork 功能分支 `Frontend/feat/report-real-document-api`。
- 只修改前端相关代码和必要测试，前端源码保持在 `apps/web/src/`。
- 移除 Report 生产路径中的 silent fallback 数据，包括但不限于 `fallbackTypes`、`fallbackTemplates`、`fallbackMaterials`、`fallbackReports`、`fallbackOutline`、`fallbackSections`。
- Report type、template、material、report、outline、section、job、event、report file/content 等能力按 Gateway `/api/v1/**` Document active paths 接入真实 API。
- API 失败时展示 gateway 错误 envelope 的 `message` 和 `requestId`；不能把错误吞掉后显示假成功或假空列表。
- 501、`not_implemented`、`dependency_error` 或后端未 ready 能力必须展示未就绪、依赖失败或禁用状态。
- 真实 AI 大纲/正文生成、Document MCP tools、Pandoc/LibreOffice 富 DOCX 不作为本任务完成能力；页面上只能显式说明未就绪或禁用。
- 复用已有 typed client、TanStack Query、加载/空/错误/权限状态组件和前端约定。
- 按 TDD 方式先补失败测试，再实现生产代码。

## Acceptance Criteria

- [ ] 生产路径中不存在报告模块 silent fallback 数据。
- [ ] API 错误会展示可排查的 `requestId`。
- [ ] 501、`dependency_error` 或未就绪能力不会显示假成功。
- [ ] 基础 CRUD、outline/section 保存、job/event 查询和 report file content 读取按当前 Gateway 契约调用。
- [ ] 不直连 Document/File/AI Gateway 内部地址。
- [ ] 新增或更新的测试先经历 RED，再由实现变绿。
- [ ] `bun run --cwd apps/web check` 通过。
- [ ] `bun run --cwd apps/web build` 通过。
- [ ] `git diff --check` 通过。

## Definition of Done

- 代码、测试和必要文档已提交到本地分支。
- PR 指向主仓库 `develop`，来源为个人 fork 分支。
- PR 描述包含 `Closes #280`。
- 未完成风险明确写入 PR 描述。

## Technical Approach

- 先阅读当前 Report 前端实现，定位 fallback 数据和 API 调用边界。
- 先写测试覆盖 fallback 移除、错误 requestId 展示、未就绪/501 gating 等行为。
- 再改 API 调用与页面状态，优先使用现有 API client 和组件，不新增平行请求层。
- 若 Gateway OpenAPI/generated types 已覆盖 Document 路径，直接使用现有 typed client；若生成文件落后，优先查已有项目生成流程，不手工改 generated 文件。

## Out of Scope

- 不实现后端真实 AI 生成。
- 不实现 Document MCP tools。
- 不实现 Pandoc/LibreOffice 富 DOCX。
- 不修改 Gateway OpenAPI，除非后续明确确认契约错误。
- 不把 API-boundary mock test 视为真实 L3 smoke。
- 不处理 Knowledge 或 QA Chat follow-up。

## Technical Notes

- GitHub issue: #280
- 权威依据：
  - `docs/collaboration/frontend-readiness-task-plan.md`
  - `docs/architecture/frontend-backend-contract.md`
  - `docs/services/gateway/api/openapi.yaml`
  - `docs/services/gateway/docs/active-api-owner-map.md`
  - `docs/services/document/docs/implementation.md`
  - `.trellis/spec/frontend/quality-guidelines.md`
- 相关依赖：#108 #114 #115 #158 #159 #160 #161 #162
- 并行任务：#125 #264
