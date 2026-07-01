# 测试证据记录模板

> 使用方式：复制本模板到测试 issue 评论或 PR body 的验证部分。复杂测试如需完整报告，使用 `docs/testing/templates/test-report-template.md` 并归档到 `docs/testing/reports/YYYY-MM-DD/`。

## 测试范围

- Issue：
- 被测分支：
- 被测 commit：
- Base branch：
- 测试负责人：
- 测试环境：本地 / CI / Docker Compose / env-gated / 真实 provider / 人工验收
- 测试层级：本地自动化 / CI 自动化 / env-gated smoke / 真实 provider smoke / 人工验收

## 已运行命令与结果

| 命令或操作 | 结果 | 证据 |
| --- | --- | --- |
| `command` | pass / fail / blocked / skipped | 日志摘要、截图、trace、request id 或 workflow 链接 |

## 未运行项

| 测试项 | 未运行原因 | 缺失环境 | 残余风险 | 后续归属 |
| --- | --- | --- | --- | --- |
| ... | ... | ... | ... | `#xxx` / `@owner` |

## 缺陷处理

| 问题 | 等级 | 处理结论 | 关联 issue / PR | 复现或验证 |
| --- | --- | --- | --- | --- |
| ... | 小问题 / 大问题 | 已修复 / 已转 issue / 暂不处理 | `#xxx` | 命令、请求、日志或 request id |

## 证据清单

- 测试报告（如适用）：`docs/testing/reports/YYYY-MM-DD/<scope>-test-report.md`
- PR：
- 截图 / trace：
- 日志：
- request id：

## 最终结论

选择一项并补充原因：

- 测试通过：
- 测试失败且已修复：
- 测试失败已转 issue：
- 因环境缺失未运行：
