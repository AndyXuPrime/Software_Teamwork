# Knowledge MCP 工具接口与参数规范

## 1. 命名、版本与通用约定

本文是 Knowledge MCP 首期四个只读工具的字段级契约。Knowledge Server 在 MCP
`tools/list` 中发布原生工具名，QA 默认以 alias `knowledge` 注册后向模型暴露带前缀
名称：

| 原生工具名 | 模型侧默认名称 |
| --- | --- |
| `search` | `knowledge__search` |
| `list_documents` | `knowledge__list_documents` |
| `get_document` | `knowledge__get_document` |
| `get_chunk` | `knowledge__get_chunk` |

所有输入均为 JSON object，字段使用 lowerCamelCase，且
`additionalProperties=false`。未知字段、类型错误、缺少必填项或越界值都返回
`validation_error`，不做静默纠正。字符串在执行前去除首尾空白；仅含空白的必填
字符串视为缺失。

四个工具均为只读、幂等、封闭世界调用：

```json
{
  "readOnlyHint": true,
  "destructiveHint": false,
  "idempotentHint": true,
  "openWorldHint": false
}
```

## 2. 通用输出结构

成功和失败共享以下顶层 schema：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `requestId` | string | 是 | 调用链 ID；优先沿用 `X-Request-Id`。 |
| `toolName` | string | 是 | Knowledge 收到的原生工具名，不包含 QA alias。 |
| `status` | `succeeded \| failed` | 是 | 工具执行状态。 |
| `data` | object | 成功时 | 各工具的数据结构，失败时省略。 |
| `error` | object | 失败时 | 安全错误结构，成功时省略。 |

错误对象：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `code` | string | 是 | 稳定机器错误码。 |
| `message` | string | 是 | 可交给模型的脱敏摘要。 |
| `fields` | object<string,string> | 否 | 仅参数错误使用的字段提示。 |

服务器在四个工具上声明的通用 output JSON Schema 为：

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["requestId", "toolName", "status"],
  "properties": {
    "requestId": {"type": "string"},
    "toolName": {"type": "string"},
    "status": {"type": "string", "enum": ["succeeded", "failed"]},
    "data": {"type": "object", "additionalProperties": true},
    "error": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "code": {"type": "string"},
        "message": {"type": "string"},
        "fields": {
          "type": "object",
          "additionalProperties": {"type": "string"}
        }
      }
    }
  }
}
```

稳定错误码为 `validation_error`、`unauthorized`、`forbidden`、`not_found`、
`conflict_error`、`rate_limited`、`dependency_error`、`internal_error`。底层错误、凭据、SQL、内部 URL、
Qdrant/File/AI provider 原始响应不得进入输出。

失败示例：

```json
{
  "requestId": "req_01JZ8JH9N5",
  "toolName": "search",
  "status": "failed",
  "error": {
    "code": "validation_error",
    "message": "knowledge tool arguments are invalid",
    "fields": {
      "topK": "must be between 1 and 20"
    }
  }
}
```

## 3. `knowledge__search`

在当前用户可见的知识库中做语义检索，返回按相关度排序的安全 chunk 摘要和后续读取
所需的资源 ID。模型应优先调用此工具，再按需调用 `knowledge__get_chunk`。

### 3.1 输入参数

| 字段 | 类型 | 必填 | 默认值 | 约束与语义 |
| --- | --- | --- | --- | --- |
| `query` | string | 是 | - | 非空自然语言查询。 |
| `knowledgeBaseIds` | string[] | 否 | `[]` | 限定检索范围；空数组表示所有当前用户可见知识库。空 ID 被忽略，重复 ID 去重。 |
| `topK` | integer | 否 | `5` | `1..20`，最大返回候选数。 |
| `scoreThreshold` | number | 否 | `0` | `0..1`，低于阈值的结果不返回。 |
| `rerank` | boolean | 否 | `false` | 是否请求 Knowledge 使用已配置 reranker。 |
| `rerankTopN` | integer | 否 | 未设置 | `1..topK`；限制进入 rerank/最终候选的数量。 |

输入 JSON Schema：

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["query"],
  "properties": {
    "query": {"type": "string", "minLength": 1},
    "knowledgeBaseIds": {
      "type": "array",
      "items": {"type": "string", "minLength": 1}
    },
    "topK": {"type": "integer", "minimum": 1, "maximum": 20, "default": 5},
    "scoreThreshold": {"type": "number", "minimum": 0, "maximum": 1, "default": 0},
    "rerank": {"type": "boolean", "default": false},
    "rerankTopN": {"type": "integer", "minimum": 1, "maximum": 20}
  }
}
```

`rerankTopN <= topK` 是 JSON Schema 之外的跨字段运行时约束。

### 3.2 成功数据

`data` 字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `query` | string | 是 | 规范化后的查询。 |
| `results` | SearchHit[] | 是 | 排序结果；无命中时为空数组。 |
| `hitCount` | integer | 是 | 本次返回的结果数。 |

`SearchHit`：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `rank` | integer | 是 | 从 1 开始的返回顺序。 |
| `score` | number | 是 | 最终相关度分数。 |
| `knowledgeBaseId` | string | 是 | 知识库 ID。 |
| `documentId` | string | 是 | 文档 ID。 |
| `documentName` | string | 是 | 用户可见文档名。 |
| `chunkId` | string | 是 | 可传给 `knowledge__get_chunk`。 |
| `chunkIndex` | integer | 否 | chunk 在文档中的序号。 |
| `chunkType` | string | 否 | chunk 类型，例如 `text`。 |
| `sectionPath` | string | 否 | 标题/章节路径。 |
| `contentPreview` | string | 是 | 脱敏的命中摘要。 |
| `tags` | string[] | 否 | 文档标签。 |

成功示例：

```json
{
  "requestId": "req_01JZ8JH9N5",
  "toolName": "search",
  "status": "succeeded",
  "data": {
    "query": "断路器合闸前检查什么",
    "results": [
      {
        "rank": 1,
        "score": 0.92,
        "knowledgeBaseId": "kb_safety",
        "documentId": "doc_manual",
        "documentName": "高压设备操作手册.pdf",
        "chunkId": "chunk_0042",
        "chunkIndex": 42,
        "chunkType": "text",
        "sectionPath": "第三章/合闸前检查",
        "contentPreview": "合闸前应确认保护装置、接地开关和指示状态……",
        "tags": ["安全规程"]
      }
    ],
    "hitCount": 1
  }
}
```

## 4. `knowledge__list_documents`

分页列出某个当前用户可见知识库中的文档，可按处理状态过滤。

### 4.1 输入参数

| 字段 | 类型 | 必填 | 默认值 | 约束与语义 |
| --- | --- | --- | --- | --- |
| `knowledgeBaseId` | string | 是 | - | 要查看的知识库 ID。 |
| `status` | string | 否 | 未设置 | `uploaded`、`parsing`、`chunking`、`embedding`、`ready`、`failed`。 |
| `page` | integer | 否 | `1` | 最小值 `1`。 |
| `pageSize` | integer | 否 | `20` | `1..100`。 |

输入 JSON Schema：

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["knowledgeBaseId"],
  "properties": {
    "knowledgeBaseId": {"type": "string", "minLength": 1},
    "status": {
      "type": "string",
      "enum": ["uploaded", "parsing", "chunking", "embedding", "ready", "failed"]
    },
    "page": {"type": "integer", "minimum": 1, "default": 1},
    "pageSize": {"type": "integer", "minimum": 1, "maximum": 100, "default": 20}
  }
}
```

### 4.2 成功数据

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `knowledgeBaseId` | string | 是 | 本次查询的知识库。 |
| `documents` | DocumentSummary[] | 是 | 当前页文档；无结果时为空数组。 |
| `totalCount` | integer | 是 | 过滤后的文档总数。 |
| `page` | integer | 是 | 当前页。 |
| `pageSize` | integer | 是 | 页大小。 |

成功示例：

```json
{
  "requestId": "req_01JZ8K3D1A",
  "toolName": "list_documents",
  "status": "succeeded",
  "data": {
    "knowledgeBaseId": "kb_safety",
    "documents": [
      {
        "id": "doc_manual",
        "knowledgeBaseId": "kb_safety",
        "name": "高压设备操作手册.pdf",
        "status": "ready",
        "tags": ["安全规程"],
        "chunkCount": 126,
        "createdAt": "2026-07-03T09:30:00Z",
        "updatedAt": "2026-07-03T09:32:10Z"
      }
    ],
    "totalCount": 1,
    "page": 1,
    "pageSize": 20
  }
}
```

## 5. `knowledge__get_document`

读取单篇当前用户可见文档的安全元数据与处理状态。

### 5.1 输入参数与 schema

| 字段 | 类型 | 必填 | 约束与语义 |
| --- | --- | --- | --- |
| `documentId` | string | 是 | 非空 Knowledge 文档 ID。 |

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["documentId"],
  "properties": {
    "documentId": {
      "type": "string",
      "minLength": 1,
      "description": "Knowledge document ID."
    }
  }
}
```

### 5.2 `DocumentSummary`

`get_document.data` 以及 `list_documents.data.documents[]` 使用同一结构：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 文档 ID。 |
| `knowledgeBaseId` | string | 是 | 所属知识库 ID。 |
| `name` | string | 是 | 用户可见文档名。 |
| `status` | string | 是 | 文档处理状态。 |
| `tags` | string[] | 否 | 文档标签。 |
| `chunkCount` | integer | 是 | 已持久化 chunk 数。 |
| `errorCode` | string | 否 | 处理失败时的稳定错误码。 |
| `errorMessage` | string | 否 | 仅在 `failed` 状态返回固定脱敏摘要。 |
| `createdAt` | RFC 3339 string | 是 | 创建时间，UTC 输出。 |
| `updatedAt` | RFC 3339 string | 是 | 最后更新时间，UTC 输出。 |

成功示例：

```json
{
  "requestId": "req_01JZ8M1N8Q",
  "toolName": "get_document",
  "status": "succeeded",
  "data": {
    "id": "doc_manual",
    "knowledgeBaseId": "kb_safety",
    "name": "高压设备操作手册.pdf",
    "status": "ready",
    "tags": ["安全规程"],
    "chunkCount": 126,
    "createdAt": "2026-07-03T09:30:00Z",
    "updatedAt": "2026-07-03T09:32:10Z"
  }
}
```

输出不包含 `fileRef`、content type、对象存储信息、parser 内部配置或当前 job 内部字段。

## 6. `knowledge__get_chunk`

读取一个当前用户可见 chunk 的完整文本。通常只应使用
`knowledge__search.results[].chunkId` 作为输入，避免模型猜测 ID。

### 6.1 输入参数与 schema

| 字段 | 类型 | 必填 | 约束与语义 |
| --- | --- | --- | --- |
| `chunkId` | string | 是 | 非空 Knowledge chunk ID。 |

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["chunkId"],
  "properties": {
    "chunkId": {
      "type": "string",
      "minLength": 1,
      "description": "Knowledge chunk ID from a search result."
    }
  }
}
```

### 6.2 成功数据

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `chunkId` | string | 是 | chunk ID。 |
| `documentId` | string | 是 | 所属文档 ID。 |
| `documentName` | string | 是 | 用户可见文档名。 |
| `knowledgeBaseId` | string | 是 | 所属知识库 ID。 |
| `chunkIndex` | integer | 是 | chunk 在文档中的序号。 |
| `chunkType` | string | 否 | chunk 类型。 |
| `sectionPath` | string | 否 | 标题/章节路径。 |
| `content` | string | 是 | 完整 chunk 文本。 |
| `tags` | string[] | 否 | 所属文档标签。 |

成功示例：

```json
{
  "requestId": "req_01JZ8N0V4R",
  "toolName": "get_chunk",
  "status": "succeeded",
  "data": {
    "chunkId": "chunk_0042",
    "documentId": "doc_manual",
    "documentName": "高压设备操作手册.pdf",
    "knowledgeBaseId": "kb_safety",
    "chunkIndex": 42,
    "chunkType": "text",
    "sectionPath": "第三章/合闸前检查",
    "content": "合闸前应确认保护装置、接地开关和指示状态均符合操作票要求。",
    "tags": ["安全规程"]
  }
}
```

输出不包含 token count、embedding provider/model/dimension、向量、Qdrant point ID、
内部 metadata 或文件存储引用。

## 7. Agent 选用规则

| 用户意图 | 首选工具 | 后续动作 |
| --- | --- | --- |
| 基于知识库回答事实问题 | `knowledge__search` | 摘要不足时用命中的 `chunkId` 调 `knowledge__get_chunk`。 |
| 查看某知识库有哪些文档 | `knowledge__list_documents` | 按需用 `documentId` 查看详情。 |
| 查看文档是否已处理完成 | `knowledge__get_document` | `status=ready` 后再检索；失败时只陈述安全摘要。 |
| 展开某条检索命中的完整上下文 | `knowledge__get_chunk` | 将内容用于回答并保留 document/chunk citation。 |

Agent 不应枚举或猜测资源 ID，不应把 `get_chunk` 当作无界全文下载工具，也不应把
模型生成的用户身份或权限放入工具参数；身份只能来自 QA 注入的受信任请求上下文。

## 8. 兼容性检查清单

- `tools/list` 必须同时返回四个原生工具及 input/output schema。
- QA 默认 alias 后的工具名必须与本文一致。
- 未提供可选参数时，默认值必须与本文一致。
- 所有 object 输入拒绝未知字段。
- 搜索和列表的空结果必须返回空数组而不是 `null`。
- 成功输出不得出现 `error`，失败输出不得出现 `data`。
- `status=failed` 时 MCP `isError` 必须为 `true`。
- 私有/不可见资源必须在 Knowledge 服务层完成鉴权，不能依赖 Agent 自律。
- schema 或字段语义发生破坏性变化时，必须同步 QA 消费方测试并引入版本迁移。
