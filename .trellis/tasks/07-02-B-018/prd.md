# QA 全局系统提示词版本化、热重载与审计

## Goal

将全局 `systemPrompt` 纳入 `qa_config_versions`，与检索配置、Agent 终止策略、工具白名单一起原子保存。清理 legacy `qa_runtime_settings` 双写路径。审计只保存版本元数据，不泄露完整提示词。

## Requirements

- 新增可回滚 migration，添加 `system_prompt TEXT NOT NULL DEFAULT ''` 列到 `qa_config_versions`，CHECK ≤ 20000 bytes
- 迁移现有 `qa_runtime_settings.system_prompt` 到当前 active QA config version
- 运行时只从当前 active QA config version 读取提示词；无 DB 配置时回退 `AGENT_SYSTEM_PROMPT` bootstrap
- 扩展 QA config repository、service DTO 和 handler，使管理员能读取当前完整提示词并发布包含新提示词的配置版本
- 非空、去除首尾空白、最大 20000 bytes 校验，返回稳定 validation error 字段
- 发布 active 版本触发原子 runtime reload；新提问获取新 snapshot，运行中回答持有旧 snapshot
- 每个 response run 保存 `qa_config_version_id`，追溯生成时版本
- 审计只保存版本 ID、版本号、操作者、prompt 长度等元数据，不在 `admin_audit_logs`、应用日志、错误或指标中保存完整提示词
- 边界：不实现按用户/角色/会话覆盖提示词；不实现管理端 React 页面（F-* 任务负责）

## Acceptance Criteria

- [x] 升级已有数据库后，当前全局提示词迁入 active QA config version 且内容不丢失
- [x] 新安装从 `AGENT_SYSTEM_PROMPT` 建立或回退安全默认值
- [x] 发布新版本后下一次提问使用新提示词；运行中回答继续使用旧提示词
- [x] 不同用户使用同一个 active 全局提示词
- [x] `response_run.qaConfigVersionId` 能追溯提示词版本
- [x] 空字符串和超过 20000 bytes 提示词被稳定拒绝
- [x] `qa:settings:read`/`qa:settings:write` 权限控制
- [x] 完整提示词不出现在普通 QA 响应、SSE、错误、日志、metrics 或 admin audit data 中
- [x] `go build ./cmd/server ./cmd/agent` 和 `go test ./...` 通过

## Notes

- PR: [#476](https://github.com/Sakayori-Iroha-168/Software_Teamwork/pull/476)
- Branch: `JerryTeam/feat/qa-versioned-system-prompt`
- Depends on: [S-048] QA 全局 Agent 系统提示词版本契约与权限边界 #464
