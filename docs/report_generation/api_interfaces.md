# 报告生成后端接口文档

## 1. 文档说明

本文定义报告生成模块需要提供的后端接口能力。接口用于支撑大项目前端、管理端、其他后端模块和 MCP 工具调用。

本模块不负责用户认证、角色权限校验和前端页面。调用方应在调用本模块前完成认证和权限控制。本模块只记录调用方传入的操作者信息和请求来源。

## 2. 通用约定

### 2.1 Base Path

建议基础路径：

```text
/api/v1/report-generation
```

### 2.2 通用请求头

| Header | 必填 | 说明 |
|---|---:|---|
| `X-Request-Id` | 否 | 调用方请求链路 ID，不传时由服务端生成 |
| `X-Operator-Id` | 否 | 操作者 ID，仅用于记录 |
| `X-Operator-Name` | 否 | 操作者名称，仅用于记录 |
| `X-Request-Source` | 否 | 请求来源，例如 `frontend`、`admin`、`mcp`、`backend` |

### 2.3 通用响应结构

```json
{
  "code": "ok",
  "message": "success",
  "data": {},
  "request_id": "req-uuid"
}
```

### 2.4 通用错误结构

```json
{
  "code": "validation_error",
  "message": "report_type is required",
  "details": {
    "field": "report_type"
  },
  "request_id": "req-uuid"
}
```

### 2.5 常用错误码

| code | 说明 |
|---|---|
| `validation_error` | 请求参数不合法 |
| `not_found` | 资源不存在 |
| `conflict` | 当前状态不允许执行该操作 |
| `generation_failed` | AI 生成失败 |
| `export_failed` | DOCX 导出失败 |
| `storage_error` | 数据库或 MinIO 存储失败 |
| `internal_error` | 未分类服务端错误 |

### 2.6 接口总览表

下表中的路径均相对于 Base Path `/api/v1/report-generation`。

| 分组 | 方法 | 路径 | 说明 |
|---|---|---|---|
| 报告类型 | `GET` | `/report-types` | 查询支持的报告类型列表 |
| 报告类型 | `GET` | `/report-types/{report_type}/templates` | 查询指定报告类型可用模板 |
| 报告 | `POST` | `/reports` | 创建报告草稿 |
| 报告 | `GET` | `/reports/{report_id}` | 查询报告详情 |
| 报告 | `PATCH` | `/reports/{report_id}` | 更新报告基础信息 |
| 报告 | `DELETE` | `/reports/{report_id}` | 删除报告记录 |
| 报告 | `GET` | `/reports` | 分页查询报告记录 |
| 大纲 | `POST` | `/reports/{report_id}/outline/generate` | 生成报告大纲 |
| 大纲 | `POST` | `/reports/{report_id}/outline/regenerate` | AI 重新生成报告大纲 |
| 大纲 | `GET` | `/reports/{report_id}/outline` | 查询报告大纲 |
| 大纲 | `PUT` | `/reports/{report_id}/outline` | 保存完整大纲 |
| 大纲 | `PATCH` | `/reports/{report_id}/outline/sections/{section_id}` | 调整大纲章节 |
| 大纲 | `DELETE` | `/reports/{report_id}/outline/sections/{section_id}` | 删除大纲章节 |
| 大纲 | `POST` | `/reports/{report_id}/outline/renumber` | 对大纲重新编号 |
| 正文 | `POST` | `/reports/{report_id}/content/generate` | 生成完整正文 |
| 正文 | `POST` | `/reports/{report_id}/content/regenerate` | AI 重新生成完整正文 |
| 正文 | `POST` | `/reports/{report_id}/sections/{section_id}/regenerate` | AI 重新生成指定章节 |
| 正文 | `GET` | `/reports/{report_id}/sections` | 查询章节内容 |
| 正文 | `PATCH` | `/reports/{report_id}/sections/{section_id}` | 更新章节正文或表格 |
| 生成任务 | `GET` | `/generation-tasks/{task_id}` | 查询生成任务状态 |
| 生成任务 | `GET` | `/reports/{report_id}/generation-tasks` | 查询报告生成任务列表 |
| 生成任务 | `POST` | `/generation-tasks/{task_id}/retry` | 重试失败任务 |
| 导出 | `POST` | `/reports/{report_id}/exports` | 创建 DOCX 导出任务 |
| 导出 | `GET` | `/exports/{export_file_id}` | 查询导出状态 |
| 导出 | `GET` | `/exports/{export_file_id}/file` | 获取导出文件引用 |
| 模板 | `POST` | `/templates` | 上传报告模板 |
| 模板 | `GET` | `/templates` | 查询模板列表 |
| 模板 | `GET` | `/templates/{template_id}` | 查询模板详情 |
| 模板 | `PATCH` | `/templates/{template_id}` | 更新模板元数据 |
| 模板 | `DELETE` | `/templates/{template_id}` | 删除模板 |
| 模板 | `GET` | `/templates/{template_id}/structure` | 查询模板结构 |
| 模板 | `PUT` | `/templates/{template_id}/structure` | 保存模板结构 |
| 素材 | `POST` | `/materials` | 上传素材 |
| 素材 | `GET` | `/materials` | 查询素材列表 |
| 素材 | `GET` | `/materials/{material_id}` | 查询素材详情 |
| 素材 | `DELETE` | `/materials/{material_id}` | 删除素材 |
| 统计与日志 | `GET` | `/statistics/overview` | 查询统计概览 |
| 统计与日志 | `GET` | `/statistics/report-generation-trend` | 查询近 30 天生成趋势 |
| 统计与日志 | `GET` | `/operation-logs` | 查询操作日志 |

## 3. 报告类型接口

### 3.1 查询报告类型列表

```http
GET /api/v1/report-generation/report-types
```

响应数据：

```json
{
  "items": [
    {
      "code": "summer_peak_inspection",
      "name": "迎峰度夏检查报告",
      "description": "用于迎峰度夏检查场景",
      "enabled": true
    },
    {
      "code": "coal_inventory_audit",
      "name": "煤库存审计报告",
      "description": "用于煤库存审计场景",
      "enabled": true
    }
  ]
}
```

### 3.2 查询报告类型可用模板

```http
GET /api/v1/report-generation/report-types/{report_type}/templates
```

响应数据：

```json
{
  "items": [
    {
      "template_id": "tpl-uuid",
      "template_name": "迎峰度夏检查报告默认模板",
      "version": 1,
      "enabled": true
    }
  ]
}
```

## 4. 报告接口

### 4.1 创建报告草稿

```http
POST /api/v1/report-generation/reports
```

请求体：

```json
{
  "report_name": "2026年迎峰度夏检查报告",
  "report_type": "summer_peak_inspection",
  "template_id": "tpl-uuid",
  "topic": "2026年迎峰度夏检查",
  "specialty": "电气",
  "plant_or_business_object": "某电厂",
  "year": 2026,
  "extra_context": {
    "region": "华东",
    "notes": "重点关注设备隐患"
  },
  "operator": {
    "id": "user-001",
    "name": "张三"
  }
}
```

响应数据：

```json
{
  "report_id": "report-uuid",
  "status": "draft"
}
```

### 4.2 查询报告详情

```http
GET /api/v1/report-generation/reports/{report_id}
```

响应数据应包含报告基础信息、当前大纲、章节摘要、最新生成任务和最新导出文件引用。

### 4.3 更新报告基础信息

```http
PATCH /api/v1/report-generation/reports/{report_id}
```

请求体可包含创建报告时的任意可编辑字段。

### 4.4 删除报告记录

```http
DELETE /api/v1/report-generation/reports/{report_id}
```

建议默认采用软删除。若需要物理删除，应由后续设计明确。

### 4.5 分页查询报告记录

```http
GET /api/v1/report-generation/reports?page=1&page_size=20&report_type=summer_peak_inspection&status=generated
```

支持筛选字段：

- `report_name`
- `report_type`
- `year`
- `status`
- `creator_id`
- `created_from`
- `created_to`

## 5. 大纲接口

### 5.1 生成大纲

```http
POST /api/v1/report-generation/reports/{report_id}/outline/generate
```

请求体：

```json
{
  "requirements": "请生成适合迎峰度夏检查场景的专业报告大纲",
  "material_ids": ["mat-uuid"],
  "save_result": true
}
```

响应数据：

```json
{
  "task_id": "task-uuid",
  "request_id": "gen-uuid",
  "status": "pending"
}
```

### 5.2 AI 重新生成大纲

```http
POST /api/v1/report-generation/reports/{report_id}/outline/regenerate
```

请求体：

```json
{
  "requirements": "按最新补充要求重新生成大纲",
  "material_ids": ["mat-uuid"],
  "preserve_manual_edits": false,
  "save_result": true
}
```

说明：

- `preserve_manual_edits = true` 时，服务应尽量保留现有手工编辑章节，并在任务结果中返回保留或冲突说明。
- 重新生成应创建新的生成任务记录。

### 5.3 查询大纲

```http
GET /api/v1/report-generation/reports/{report_id}/outline
```

### 5.4 保存完整大纲

```http
PUT /api/v1/report-generation/reports/{report_id}/outline
```

请求体：

```json
{
  "outline": [
    {
      "section_id": "sec-1",
      "title": "概述",
      "level": 1,
      "numbering": "1",
      "children": []
    }
  ]
}
```

### 5.5 调整大纲章节

```http
PATCH /api/v1/report-generation/reports/{report_id}/outline/sections/{section_id}
```

支持修改标题、层级、排序等字段。

### 5.6 删除大纲章节

```http
DELETE /api/v1/report-generation/reports/{report_id}/outline/sections/{section_id}
```

### 5.7 重新编号

```http
POST /api/v1/report-generation/reports/{report_id}/outline/renumber
```

## 6. 正文接口

### 6.1 生成完整正文

```http
POST /api/v1/report-generation/reports/{report_id}/content/generate
```

请求体：

```json
{
  "requirements": "请基于当前大纲逐章节生成正文",
  "material_ids": ["mat-uuid"],
  "save_result": true
}
```

响应数据：

```json
{
  "task_id": "task-uuid",
  "request_id": "gen-uuid",
  "status": "pending"
}
```

### 6.2 AI 重新生成完整正文

```http
POST /api/v1/report-generation/reports/{report_id}/content/regenerate
```

请求体：

```json
{
  "requirements": "请结合最新大纲和补充要求重新生成正文",
  "material_ids": ["mat-uuid"],
  "preserve_manual_edits": false,
  "save_result": true
}
```

### 6.3 AI 重新生成指定章节

```http
POST /api/v1/report-generation/reports/{report_id}/sections/{section_id}/regenerate
```

请求体：

```json
{
  "requirements": "请强化该章节的风险分析内容",
  "material_ids": ["mat-uuid"],
  "preserve_manual_edits": false,
  "save_result": true
}
```

### 6.4 查询章节内容

```http
GET /api/v1/report-generation/reports/{report_id}/sections
```

### 6.5 更新章节正文

```http
PATCH /api/v1/report-generation/reports/{report_id}/sections/{section_id}
```

请求体：

```json
{
  "content": "更新后的章节正文",
  "tables": []
}
```

## 7. 生成任务接口

### 7.1 查询任务状态

```http
GET /api/v1/report-generation/generation-tasks/{task_id}
```

响应数据：

```json
{
  "task_id": "task-uuid",
  "request_id": "gen-uuid",
  "task_type": "regenerate_section",
  "status": "running",
  "progress": {
    "completed_sections": 3,
    "total_sections": 10,
    "percent": 30
  },
  "error": null
}
```

### 7.2 查询报告生成任务列表

```http
GET /api/v1/report-generation/reports/{report_id}/generation-tasks
```

### 7.3 重试失败任务

```http
POST /api/v1/report-generation/generation-tasks/{task_id}/retry
```

## 8. 导出接口

### 8.1 创建导出任务

```http
POST /api/v1/report-generation/reports/{report_id}/exports
```

请求体：

```json
{
  "format": "docx",
  "template_id": "tpl-uuid",
  "style_options": {
    "numbering_mode": "global"
  }
}
```

### 8.2 查询导出状态

```http
GET /api/v1/report-generation/exports/{export_file_id}
```

### 8.3 获取导出文件引用

```http
GET /api/v1/report-generation/exports/{export_file_id}/file
```

响应数据：

```json
{
  "export_file_id": "file-uuid",
  "file_name": "2026年迎峰度夏检查报告.docx",
  "file_reference": "minio-object-key-or-signed-url",
  "expires_at": "2026-06-28T12:00:00Z"
}
```

## 9. 模板接口

### 9.1 上传模板

```http
POST /api/v1/report-generation/templates
```

请求类型建议使用 `multipart/form-data`。

字段：

- `file`
- `template_name`
- `report_type`
- `description`

### 9.2 查询模板列表

```http
GET /api/v1/report-generation/templates?report_type=summer_peak_inspection&enabled=true
```

### 9.3 查询模板详情

```http
GET /api/v1/report-generation/templates/{template_id}
```

### 9.4 更新模板元数据

```http
PATCH /api/v1/report-generation/templates/{template_id}
```

### 9.5 删除模板

```http
DELETE /api/v1/report-generation/templates/{template_id}
```

### 9.6 查询模板结构

```http
GET /api/v1/report-generation/templates/{template_id}/structure
```

响应数据：

```json
{
  "template_id": "template-uuid",
  "outline_schema": [],
  "style_config": {}
}
```

### 9.7 保存模板结构

```http
PUT /api/v1/report-generation/templates/{template_id}/structure
```

请求体：

```json
{
  "outline_schema": [],
  "style_config": {}
}
```

## 10. 素材接口

### 10.1 上传素材

```http
POST /api/v1/report-generation/materials
```

请求类型建议使用 `multipart/form-data`。

字段：

- `file`
- `material_name`
- `category`
- `description`

### 10.2 查询素材列表

```http
GET /api/v1/report-generation/materials?category=检查报告
```

### 10.3 查询素材详情

```http
GET /api/v1/report-generation/materials/{material_id}
```

### 10.4 删除素材

```http
DELETE /api/v1/report-generation/materials/{material_id}
```

## 11. 统计与日志接口

### 11.1 查询统计概览

```http
GET /api/v1/report-generation/statistics/overview
```

响应数据：

```json
{
  "template_count": 2,
  "report_count": 18,
  "material_count": 5,
  "generation": {
    "running": 1,
    "succeeded": 15,
    "failed": 2
  }
}
```

### 11.2 查询近 30 天生成趋势

```http
GET /api/v1/report-generation/statistics/report-generation-trend?days=30
```

### 11.3 查询操作日志

```http
GET /api/v1/report-generation/operation-logs?target_type=report&target_id=report-uuid
```

## 12. MCP 工具映射

| MCP 工具 | 对应后端能力 |
|---|---|
| `generate_report_outline` | `POST /reports/{report_id}/outline/generate` |
| `regenerate_report_outline` | `POST /reports/{report_id}/outline/regenerate` |
| `generate_report_text` | `POST /reports/{report_id}/content/generate` |
| `regenerate_report_text` | `POST /reports/{report_id}/content/regenerate` |
| `regenerate_report_section` | `POST /reports/{report_id}/sections/{section_id}/regenerate` |
| `get_generation_status` | `GET /generation-tasks/{task_id}` |
| `get_report_result` | `GET /reports/{report_id}` |
| `export_report_docx` | `POST /reports/{report_id}/exports` |
| `get_template_schema` | `GET /templates/{template_id}/structure` |

## 13. 接口验收要求

- 所有接口返回统一响应结构。
- 所有失败场景返回明确错误码和错误说明。
- 生成、重新生成和导出类接口必须创建任务记录。
- 重新生成接口不得丢失报告基础信息。
- 编辑接口保存后，导出接口必须使用最新内容。
- 文件上传和导出文件应保存到 MinIO。
- 数据库记录应保存 MinIO 对象引用。
- MCP 工具参数应具备明确必填字段和结构化错误返回。
