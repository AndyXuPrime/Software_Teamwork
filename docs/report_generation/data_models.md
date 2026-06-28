# 报告生成数据模型文档

## 1. 文档说明

本文定义报告生成模块的核心数据模型，用于支撑后端接口、MCP 工具、数据库持久化和 MinIO 文件引用。

本文只描述逻辑数据模型，不提供具体 SQL 建表语句。后续实现可根据 Go 服务和数据库规范转换为 PostgreSQL migration。

## 2. 存储边界

### 2.1 数据库

数据库保存结构化业务数据：

- 报告基础信息。
- 报告大纲。
- 报告章节内容。
- 报告模板元数据。
- 素材元数据。
- 生成任务。
- 导出文件记录。
- 操作日志。
- 统计聚合数据或查询视图。

### 2.2 MinIO

MinIO 保存文件类对象：

- 报告模板文件。
- 专业素材文件。
- 导出的 DOCX 文件。
- 后续可能保存的生成结果快照文件。

数据库中只保存 MinIO 对象引用，不直接保存文件二进制内容。

## 3. 实体关系概览

```text
ReportType 1 ── N ReportTemplate
ReportType 1 ── N Report

ReportTemplate 1 ── N Report
ReportTemplate N ── N Material

Report 1 ── 1 ReportOutline
Report 1 ── N ReportSection
Report 1 ── N GenerationTask
Report 1 ── N ExportFile

GenerationTask 1 ── N OperationLog
Report 1 ── N OperationLog
```

## 4. 通用字段约定

| 字段 | 说明 |
|---|---|
| `id` | 主键，建议 UUID |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |
| `deleted_at` | 软删除时间，可选 |
| `created_by` | 创建人标识，可由调用方传入 |
| `updated_by` | 更新人标识，可由调用方传入 |

本模块不做用户认证，用户相关字段只用于记录来源和追溯。

## 5. 核心实体

### 5.1 ReportType

报告类型可以作为固定枚举，也可以落库方便后续扩展。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `code` | string | 类型编码，唯一 |
| `name` | string | 类型名称 |
| `description` | string | 类型描述 |
| `enabled` | boolean | 是否启用 |
| `default_template_id` | uuid | 默认模板 ID，可选 |

初始枚举：

| code | name |
|---|---|
| `summer_peak_inspection` | 迎峰度夏检查报告 |
| `coal_inventory_audit` | 煤库存审计报告 |

### 5.2 Report

报告主记录。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 报告 ID |
| `report_name` | string | 报告名称 |
| `report_type` | string | 报告类型编码 |
| `template_id` | uuid | 当前使用模板 |
| `topic` | string | 报告主题 |
| `specialty` | string | 专业 |
| `plant_or_business_object` | string | 电厂或业务对象 |
| `year` | int | 年份 |
| `status` | string | 报告状态 |
| `extra_context_json` | json | 扩展上下文 |
| `creator_id` | string | 创建人标识 |
| `creator_name` | string | 创建人名称 |
| `source` | string | 来源，例如 `frontend`、`admin`、`mcp`、`backend` |
| `latest_generation_task_id` | uuid | 最新生成任务 |
| `latest_export_file_id` | uuid | 最新导出文件 |
| `generated_at` | datetime | 正文生成完成时间 |
| `exported_at` | datetime | 最近导出完成时间 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `deleted_at` | datetime | 软删除时间 |

状态枚举建议：

| status | 说明 |
|---|---|
| `draft` | 草稿 |
| `outline_generating` | 大纲生成中 |
| `outline_generated` | 大纲已生成 |
| `content_generating` | 正文生成中 |
| `generated` | 正文已生成 |
| `exporting` | 导出中 |
| `exported` | 已导出 |
| `failed` | 失败 |
| `deleted` | 已删除 |

### 5.3 ReportOutline

报告大纲。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 大纲 ID |
| `report_id` | uuid | 所属报告 |
| `outline_json` | json | 多级大纲树 |
| `version` | int | 大纲版本 |
| `source_task_id` | uuid | 生成或重新生成任务 ID |
| `manual_edited` | boolean | 是否发生过手工编辑 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

说明：

- AI 重新生成大纲时，应提升 `version`。
- 是否保留旧版本由后续实现决定，但生成任务中必须保留请求和响应快照。

### 5.4 ReportSection

报告章节内容。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 章节 ID |
| `report_id` | uuid | 所属报告 |
| `parent_id` | uuid | 父章节 ID，可空 |
| `outline_node_id` | string | 对应大纲节点 ID |
| `title` | string | 章节标题 |
| `level` | int | 层级 |
| `sort_order` | int | 排序 |
| `numbering` | string | 编号 |
| `section_type` | string | 章节类型 |
| `content` | text | 正文内容 |
| `tables_json` | json | 表格内容 |
| `images_json` | json | 图片引用，后续扩展 |
| `generation_status` | string | 章节生成状态 |
| `content_source` | string | 内容来源 |
| `manual_edited` | boolean | 是否手工编辑 |
| `version` | int | 内容版本 |
| `last_generation_task_id` | uuid | 最近生成或重新生成任务 |
| `generated_at` | datetime | 生成时间 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

章节类型枚举建议：

| section_type | 说明 |
|---|---|
| `text` | 正文 |
| `table` | 表格 |
| `image` | 图片 |
| `mixed` | 混合内容 |

内容来源枚举建议：

| content_source | 说明 |
|---|---|
| `ai` | AI 生成 |
| `manual` | 手工编辑 |
| `mixed` | AI 生成后手工编辑 |

### 5.5 ReportTemplate

报告模板。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 模板 ID |
| `template_name` | string | 模板名称 |
| `report_type` | string | 绑定报告类型 |
| `version` | int | 模板版本 |
| `file_object_key` | string | MinIO 模板文件对象引用 |
| `structure_json` | json | 大纲结构配置 |
| `style_config_json` | json | DOCX 样式配置 |
| `description` | string | 描述 |
| `enabled` | boolean | 是否启用 |
| `created_by` | string | 创建人 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `deleted_at` | datetime | 删除时间 |

### 5.6 Material

专业素材。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 素材 ID |
| `material_name` | string | 素材名称 |
| `material_type` | string | 文件类型 |
| `category` | string | 分类 |
| `file_object_key` | string | MinIO 素材文件对象引用 |
| `description` | string | 描述 |
| `tags_json` | json | 标签 |
| `enabled` | boolean | 是否启用 |
| `created_by` | string | 创建人 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `deleted_at` | datetime | 删除时间 |

### 5.7 TemplateMaterialLink

模板与素材的关联关系。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 关联 ID |
| `template_id` | uuid | 模板 ID |
| `material_id` | uuid | 素材 ID |
| `usage_type` | string | 用途，例如 `outline`、`content`、`export` |
| `created_at` | datetime | 创建时间 |

## 6. 任务与文件实体

### 6.1 GenerationTask

生成任务记录，覆盖生成和重新生成。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 任务 ID |
| `request_id` | string | 请求标识 |
| `source` | string | 来源，例如 `api`、`mcp` |
| `task_type` | string | 任务类型 |
| `target_type` | string | 目标类型 |
| `target_id` | string | 目标 ID |
| `report_id` | uuid | 报告 ID |
| `template_id` | uuid | 模板 ID |
| `request_payload_json` | json | 请求快照 |
| `response_payload_json` | json | 响应快照 |
| `input_snapshot_json` | json | 生成前报告状态快照 |
| `status` | string | 任务状态 |
| `progress_json` | json | 进度 |
| `error_code` | string | 错误码 |
| `error_message` | string | 错误信息 |
| `started_at` | datetime | 开始时间 |
| `finished_at` | datetime | 结束时间 |
| `created_at` | datetime | 创建时间 |

任务类型枚举建议：

| task_type | 说明 |
|---|---|
| `generate_outline` | 首次生成大纲 |
| `regenerate_outline` | 重新生成大纲 |
| `generate_content` | 首次生成完整正文 |
| `regenerate_content` | 重新生成完整正文 |
| `regenerate_section` | 重新生成指定章节 |
| `export_docx` | 导出 DOCX |

任务状态枚举建议：

| status | 说明 |
|---|---|
| `pending` | 待执行 |
| `running` | 执行中 |
| `succeeded` | 成功 |
| `partial_succeeded` | 部分成功 |
| `failed` | 失败 |
| `canceled` | 已取消 |

### 6.2 ExportFile

导出文件记录。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 导出文件 ID |
| `report_id` | uuid | 报告 ID |
| `generation_task_id` | uuid | 导出任务 ID |
| `file_name` | string | 文件名 |
| `file_type` | string | 文件类型，例如 `docx` |
| `object_key` | string | MinIO 对象引用 |
| `file_size` | int64 | 文件大小 |
| `export_status` | string | 导出状态 |
| `created_by` | string | 创建人 |
| `created_at` | datetime | 创建时间 |

导出状态枚举建议：

| export_status | 说明 |
|---|---|
| `pending` | 待导出 |
| `running` | 导出中 |
| `succeeded` | 导出成功 |
| `failed` | 导出失败 |

### 6.3 OperationLog

操作日志。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `id` | uuid | 日志 ID |
| `operator_id` | string | 操作者 ID |
| `operator_name` | string | 操作者名称 |
| `operation_type` | string | 操作类型 |
| `target_type` | string | 目标类型 |
| `target_id` | string | 目标 ID |
| `request_source` | string | 请求来源 |
| `operation_result` | string | 操作结果 |
| `error_message` | string | 错误信息 |
| `created_at` | datetime | 创建时间 |

操作类型建议：

- `create_report`
- `update_report`
- `generate_outline`
- `regenerate_outline`
- `save_outline`
- `generate_content`
- `regenerate_content`
- `regenerate_section`
- `update_section`
- `export_docx`
- `upload_template`
- `update_template`
- `upload_material`
- `delete_material`
- `mcp_call`

## 7. 统计数据

统计数据可通过实时聚合查询，也可以后续增加聚合表。

### 7.1 ReportDailyStatistic

可选聚合模型。

| 字段 | 类型建议 | 说明 |
|---|---|---|
| `stat_date` | date | 日期 |
| `report_type` | string | 报告类型 |
| `created_count` | int | 新建报告数 |
| `generated_count` | int | 生成成功数 |
| `failed_count` | int | 生成失败数 |
| `exported_count` | int | 导出成功数 |
| `updated_at` | datetime | 更新时间 |

第一阶段可以不建聚合表，直接从报告、任务和导出记录聚合。

## 8. 关键约束

- `Report.report_type` 必须是支持的报告类型。
- `Report.template_id` 应引用启用状态的模板。
- `ReportOutline.report_id` 与 `ReportSection.report_id` 必须属于同一报告。
- AI 重新生成必须创建新的 `GenerationTask`。
- 重新生成不得删除报告基础信息。
- 重新生成正文或章节时，应更新对应章节的 `last_generation_task_id`。
- 删除模板时，如果已有报告使用该模板，建议只允许停用或软删除。
- 删除素材时，如果已有任务引用该素材，建议只允许软删除。
- 导出文件的 `object_key` 必须能在 MinIO 中定位文件。
- 操作日志不得记录密钥、完整下载签名等敏感信息。
