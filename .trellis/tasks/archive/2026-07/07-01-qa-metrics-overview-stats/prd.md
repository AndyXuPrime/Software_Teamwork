# B-010 QA 指标接口与管理端统计数据

## 任务范围

实现 `/qa-metrics/overview`、`/trend`、`/top-queries`、`/intent-distribution` 接口的补齐工作：

1. `MetricsOverview` 补充 `knowledgeBaseCount` 和 `documentCount` 字段
2. 指标端点 `days` 参数校验对齐 OpenAPI（1-366）
3. 新增指标查询数据库索引
4. 测试覆盖空数据和有数据聚合

## 验收标准

- 管理端 QA 统计页面可用
- 支持 days、limit 等查询参数
- 指标含 requestId，错误 envelope 稳定
- 普通用户无权访问管理统计时返回 forbidden
- 指标响应不包含用户输入正文、prompt、token、API key

## 边界

- QA 专属指标，不实现全站 admin-metrics
