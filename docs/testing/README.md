# 测试资料目录

本目录是测试组的统一资料入口，用于存放测试策略、测试报告模板、测试执行记录和按日期归档的测试报告。

## 目录结构

| 路径 | 用途 |
| --- | --- |
| `docs/testing/strategy.md` | 仓库当前测试策略、CI 覆盖和本地验证矩阵。 |
| `docs/testing/test-matrix.md` | 测试组总测试矩阵、issue 证据追踪和阶段性汇总口径。 |
| `docs/testing/templates/test-report-template.md` | 测试报告标准模板。每次测试任务都应以此为基础生成报告。 |
| `docs/testing/templates/test-evidence-record-template.md` | 测试 issue 或 PR 中使用的快速证据块模板。 |
| `docs/testing/templates/final-acceptance-summary-template.md` | 周报或最终验收汇总模板。 |
| `docs/testing/reports/YYYY-MM-DD/` | 按测试执行日期归档的测试报告。 |

旧的 `docs/tests/` 目录已迁移到 `docs/testing/reports/`。后续不要再向 `docs/tests/` 新增报告。

## 测试报告规则

- 每个 `T-*` 测试任务都必须留下可复核证据；只提交测试代码、测试清单或口头结论不算完成。
- 纯单元测试、组件测试和静态检查默认并入自动化；可以在 issue/PR 中保留轻量执行记录，不强制生成完整测试报告。
- 集成测试、E2E、权限/安全边界、文件/Parser 边界、migration、环境验收、人工验收、回归测试或缺陷复现必须生成完整测试报告。
- 完整测试报告必须基于 `docs/testing/templates/test-report-template.md`，并保留“执行命令与结果”“缺陷处理”“证据清单”“最终结论”等关键章节。
- 报告按实际执行日期放入 `docs/testing/reports/YYYY-MM-DD/`。例如：`docs/testing/reports/2026-07-01/auth-gateway-test-report.md`。
- 文件名使用小写英文、数字和连字符，建议格式为 `<module-or-flow>-test-report.md`。
- 如果同一天同一模块有多轮测试，可以在文件名追加范围或轮次，例如 `knowledge-rerank-regression-test-report.md`。
- 报告中的测试结论必须区分：测试通过、测试失败且已修复、测试失败已转 issue、因环境缺失未运行。
- 未运行的测试不能写成通过，必须在轻量记录或完整报告中记录缺失环境、跳过条件、残余风险和后续归属。
- 测试组总进度和证据位置统一维护在 `docs/testing/test-matrix.md`。矩阵状态落后时以 GitHub Issue / Project 为准，并在 PR 前补齐报告路径、未运行原因和残余风险。

## 缺陷处理规则

测试主责人需要先判断测试发现的问题等级：

- 小问题：可以在当前测试任务 PR 中顺手修复，但报告中必须说明修复范围、验证命令和风险。
- 大问题：不要在测试任务中扩大修改范围；新建独立 issue 指派给对应 owner 小组，并在测试报告、测试 issue 和 PR 中互相链接。

大问题包括但不限于：跨服务契约变更、数据模型或 migration 变更、权限或安全边界缺陷、需要 owner service 重构、需要产品或架构决策、会影响多个模块的行为变更。

对于发现但暂不修复的问题，报告中必须记录复现步骤、实际结果、预期结果、相关日志或 request id、影响范围、建议归属小组和阻塞关系。

## 提交流程

1. 在测试 issue 中确认范围、依据和预期交付物。
2. 按 `docs/testing/strategy.md` 选择需要运行的命令和人工验证项。
3. 实际运行测试，保留命令、环境、结果和失败证据。
4. 如果是纯单元/组件自动化或静态检查，在 issue/PR 中留下轻量执行记录。
5. 如果是复杂测试，复制 `docs/testing/templates/test-report-template.md` 生成当日测试报告，并放入 `docs/testing/reports/YYYY-MM-DD/`。
6. 在测试 issue 和 PR 中链接报告路径或轻量记录；轻量记录可以复制 `docs/testing/templates/test-evidence-record-template.md` 填写。
7. 更新 `docs/testing/test-matrix.md` 中对应 issue 的状态、证据位置、未运行项和残余风险。
