# 最终验收汇总模板

> 使用方式：测试组最终汇报或演示前，从 `docs/testing/test-matrix.md` 和各测试报告汇总本文件。不要把未运行项写成通过。

## 0. 基本信息

| 项目 | 记录 |
| --- | --- |
| 汇总日期 | `YYYY-MM-DD` |
| 汇总负责人 | `@github-user` |
| Base branch | `upstream/develop @ <commit sha>` |
| 汇总范围 | Batch / 模块 / 演示范围 |
| 关联矩阵 | `docs/testing/test-matrix.md` |
| 最终结论 | 可验收 / 有条件验收 / 不建议验收 / 因环境缺失未完成 |

## 1. 覆盖范围

| 测试层级 | 覆盖 issue | 已验证范围 | 证据链接 | 结论 |
| --- | --- | --- | --- | --- |
| 前端本地自动化 / E2E | `#xxx` | ... | 报告、PR、截图或 trace | pass / fail / blocked / skipped |
| 后端单元 / 服务测试 | `#xxx` | ... | ... | ... |
| 契约 / OpenAPI | `#xxx` | ... | ... | ... |
| 集成 / env-gated smoke | `#xxx` | ... | ... | ... |
| 真实 provider smoke | `#xxx` | ... | ... | ... |
| 部署 / Seed / Reset | `#xxx` | ... | ... | ... |
| 人工验收 | `#xxx` | ... | ... | ... |

## 2. 命令汇总

| 命令或操作 | 运行环境 | 结果 | 证据 |
| --- | --- | --- | --- |
| `command` | 本地 / CI / Compose / 外部环境 | pass / fail / blocked / skipped | 日志摘要、workflow、request id、截图或 trace |

## 3. 未运行项与环境依赖

| 未运行项 | 原因 | 缺失环境 | 残余风险 | 后续归属 |
| --- | --- | --- | --- | --- |
| ... | ... | ... | ... | `#xxx` / `@owner` |

## 4. 缺陷与跟踪

| 缺陷 | 等级 | 当前状态 | 归属 | 证据 |
| --- | --- | --- | --- | --- |
| ... | 小问题 / 大问题 | 已修复 / 已转 issue / 暂不处理 | `#xxx` / `@owner` | 复现步骤、日志、request id 或 PR |

## 5. 风险结论

- 已覆盖：
- 已知缺口：
- 环境依赖：
- 真实 provider 或人工凭据风险：
- 不应写成 required check 的内容：

## 6. 签收清单

- [ ] 每个测试 issue 都有报告或证据块链接。
- [ ] 所有未运行项都写明原因、缺失环境和残余风险。
- [ ] mock/fake、轻量容器、env-gated、真实 provider 和人工验收已分开标注。
- [ ] 失败项已判断小问题或大问题，并已修复或转 owner issue。
- [ ] 汇总中没有 token、API key、object key、内部 URL、完整 prompt、provider raw body 或生产凭据。
- [ ] 最终结论可以直接用于测试组周报或验收汇报。
