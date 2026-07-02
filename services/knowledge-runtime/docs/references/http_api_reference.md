---
sidebar_position: 4
slug: /http_api_reference
sidebar_custom_props: {
  categoryIcon: LucideGlobe
}
---
# HTTP API

A complete reference for RAGFlow's RESTful API. Before proceeding, please ensure you [have your RAGFlow API key ready for authentication](https://ragflow.io/docs/dev/acquire_ragflow_api_key).

---

## ERROR CODES

---

| Code | Message               | Description                |
|------|-----------------------|----------------------------|
| 400  | Bad Request           | Invalid request parameters |
| 401  | Unauthorized          | Unauthorized access        |
| 403  | Forbidden             | Access denied              |
| 404  | Not Found             | Resource not found         |
| 500  | Internal Server Error | Server internal error      |
| 1001 | Invalid Chunk ID      | Invalid Chunk ID           |
| 1002 | Chunk Update Failed   | Chunk update failed        |

---

## DATASET MANAGEMENT

---

### Create dataset

**POST** `/api/v1/datasets`

Creates a dataset.

#### Request

- Method: POST
- URL: `/api/v1/datasets`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"name"`: `string`
  - `"avatar"`: `string`
  - `"description"`: `string`
  - `"embedding_model"`: `string`
  - `"permission"`: `string`
  - `"chunk_method"`: `string`
  - `"parser_config"`: `object`
  - `"parse_type"`: `int`
  - `"pipeline_id"`: `string`

##### A basic request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
      "name": "test_1"
      }'
```

##### A request example specifying ingestion pipeline

:::caution WARNING
You must *not* include `"chunk_method"` or `"parser_config"` when specifying an ingestion pipeline.
:::

```bash
curl --request POST \
  --url http://{address}/api/v1/datasets \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer <YOUR_API_KEY>' \
  --data '{
   "name": "test-sdk",
   "parse_type": <NUMBER_OF_PARSERS_IN_YOUR_PARSER_COMPONENT>,
   "pipeline_id": "<PIPELINE_ID_32_HEX>"
  }'
```

##### Request parameters

- `"name"`: (*Body parameter*), `string`, *Required*
  The unique name of the dataset to create. It must adhere to the following requirements:
  - Basic Multilingual Plane (BMP) only
  - Maximum 128 characters
  - Case-insensitive

- `"avatar"`: (*Body parameter*), `string`
  Base64 encoding of the avatar.
  - Maximum 65535 characters

- `"description"`: (*Body parameter*), `string`
  A brief description of the dataset to create.
  - Maximum 65535 characters

- `"embedding_model"`: (*Body parameter*), `string`
  The name of the embedding model to use. For example: `"BAAI/bge-large-zh-v1.5@BAAI"`
  - Maximum 255 characters
  - Must follow `model_name@model_factory` format

- `"permission"`: (*Body parameter*), `string`
  Specifies who can access the dataset to create. Available options:
  - `"me"`: (Default) Only you can manage the dataset.
  - `"team"`: All team members can manage the dataset.

- `"chunk_method"`: (*Body parameter*), `enum<string>`
  The default chunk method of the dataset to create. Mutually exclusive with `"parse_type"` and `"pipeline_id"`. If you set `"chunk_method"`, do not include `"parse_type"` or `"pipeline_id"`.
  Available options:
  - `"naive"`: General (default)
  - `"book"`: Book
  - `"email"`: Email
  - `"laws"`: Laws
  - `"manual"`: Manual
  - `"one"`: One
  - `"paper"`: Paper
  - `"picture"`: Picture
  - `"presentation"`: Presentation
  - `"qa"`: Q&A
  - `"table"`: Table
  - `"tag"`: Tag

- `"parser_config"`: (*Body parameter*), `object`
  The configuration settings for the dataset parser. The attributes in this JSON object vary with the selected `"chunk_method"`:
  - If `"chunk_method"` is `"naive"`, the `"parser_config"` object contains the following attributes:
    - `"auto_keywords"`: `int`
      - Defaults to `0`
      - Minimum: `0`
      - Maximum: `32`
    - `"auto_questions"`: `int`
      - Defaults to `0`
      - Minimum: `0`
      - Maximum: `10`
    - `"chunk_token_num"`: `int`
      - Defaults to `512`
      - Minimum: `1`
      - Maximum: `2048`
    - `"delimiter"`: `string`
      - Defaults to `"\n"`.
    - `"html4excel"`: `bool`
      - Whether to convert Excel documents into HTML format.
      - Defaults to `false`
    - `"layout_recognize"`: `string`
      - Defaults to `DeepDOC`
    - `"tag_kb_ids"`: `array<string>`
      - IDs of datasets to be parsed using the ​​Tag chunk method.
      - Before setting this, ensure a tag set is created and properly configured. For details, see [Use tag set](https://ragflow.io/docs/dev/use_tag_sets).
    - `"task_page_size"`: `int`
      - For PDFs only.
      - Defaults to `12`
      - Minimum: `1`
    - `"raptor"`: `object` RAPTOR-specific settings.
      - Defaults to: `{"use_raptor": false}`
    - `"graphrag"`: `object` GRAPHRAG-specific settings.
      - Defaults to: `{"use_graphrag": false}`
    - `"parent_child"`: `object` Parent-child chunking settings. When enabled, each chunk is further split into smaller child chunks using `children_delimiter`. At retrieval time, matched child chunks are replaced by their parent's full text before being passed to the LLM, giving precise vector matching with broader context.
      - `"use_parent_child"`: `bool` Whether to enable parent-child chunking. Defaults to `false`.
      - `"children_delimiter"`: `string` The delimiter used to split a parent chunk into child chunks. Only takes effect when `"use_parent_child"` is `true`. Defaults to `"\n"`.
  - If `"chunk_method"` is `"qa"`, `"manual"`, `"paper"`, `"book"`, `"laws"`, or `"presentation"`, the `"parser_config"` object contains the following attribute:
    - `"raptor"`: `object` RAPTOR-specific settings.
      - Defaults to: `{"use_raptor": false}`.
  - If `"chunk_method"` is `"table"`, `"picture"`, `"one"`, or `"email"`, `"parser_config"` is an empty JSON object.

- `"parse_type"`: (*Body parameter*), `int`
  The ingestion pipeline parse type identifier, i.e., the number of parsers in your **Parser** component.
  - Required (along with `"pipeline_id"`) if specifying an ingestion pipeline.
  - Must not be included when `"chunk_method"` is specified.

- `"pipeline_id"`: (*Body parameter*), `string`
  The ingestion pipeline ID. Can be found in the corresponding URL in the RAGFlow UI.
  - Required (along with `"parse_type"`) if specifying an ingestion pipeline.
  - Must be a 32-character lowercase hexadecimal string, e.g., `"d0bebe30ae2211f0970942010a8e0005"`.
  - Must not be included when `"chunk_method"` is specified.

:::caution WARNING
You can choose either of the following ingestion options when creating a dataset, but *not* both:

- Use a built-in chunk method -- specify `"chunk_method"` (optionally with `"parser_config"`).
- Use an ingestion pipeline -- specify both `"parse_type"` and `"pipeline_id"`.

If none of `"chunk_method"`, `"parse_type"`, or `"pipeline_id"` are provided, the system defaults to `chunk_method = "naive"`.
:::

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "avatar": null,
        "chunk_count": 0,
        "chunk_method": "naive",
        "create_date": "Mon, 28 Apr 2025 18:40:41 GMT",
        "create_time": 1745836841611,
        "created_by": "3af81804241d11f0a6a79f24fc270c7f",
        "description": null,
        "document_count": 0,
        "embedding_model": "BAAI/bge-large-zh-v1.5@BAAI",
        "id": "3b4de7d4241d11f0a6a79f24fc270c7f",
        "language": "English",
        "name": "RAGFlow example",
        "pagerank": 0,
        "parser_config": {
            "chunk_token_num": 128,
            "delimiter": "\\n!?;。；！？",
            "html4excel": false,
            "layout_recognize": "DeepDOC",
            "raptor": {
                "use_raptor": false
                }
            },
        "permission": "me",
        "similarity_threshold": 0.2,
        "status": "1",
        "tenant_id": "3af81804241d11f0a6a79f24fc270c7f",
        "token_num": 0,
        "update_date": "Mon, 28 Apr 2025 18:40:41 GMT",
        "update_time": 1745836841611,
        "vector_similarity_weight": 0.3,
    },
}
```

Failure:

```json
{
    "code": 101,
    "message": "Field: <name> - Message: <String should have at least 1 character> - Value: <>"
}
```

---

### Delete datasets

**DELETE** `/api/v1/datasets`

Deletes datasets by ID.

#### Request

- Method: DELETE
- URL: `/api/v1/datasets`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"ids"`: `list[string]` or `null`
  - `"delete_all"`: `boolean`

##### Request example

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
     "ids": ["d94a8dc02c9711f0930f7fbc369eab6d", "e94a8dc02c9711f0930f7fbc369eab6e"]
     }'
```

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
     "delete_all": true
     }'
```

##### Request parameters

- `"ids"`: (*Body parameter*), `list[string]` or `null`
  Specifies the datasets to delete:
  - If omitted, or set to `null` or an empty array, no datasets are deleted.
  - If an array of IDs is provided, only the datasets matching those IDs are deleted.
- `"delete_all"`: (*Body parameter*), `boolean`
  Whether to delete all datasets owned by the current user when`"ids"` is omitted, or set to `null` or an empty array. Defaults to `false`.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code":108,
    "message":"User '<tenant_id>' lacks permission for datasets: '<dataset_ids>'"
}

```

---

### Update dataset

**PUT** `/api/v1/datasets/{dataset_id}`

Updates configurations for a specified dataset.

#### Request

- Method: PUT
- URL: `/api/v1/datasets/{dataset_id}`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"name"`: `string`
  - `"avatar"`: `string`
  - `"description"`: `string`
  - `"embedding_model"`: `string`
  - `"permission"`: `string`
  - `"chunk_method"`: `string`
  - `"pagerank"`: `int`
  - `"parser_config"`: `object`

##### Request example

```bash
curl --request PUT \
     --url http://{address}/api/v1/datasets/{dataset_id} \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "name": "updated_dataset"
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the dataset to update.
- `"name"`: (*Body parameter*), `string`
  The revised name of the dataset.
  - Basic Multilingual Plane (BMP) only
  - Maximum 128 characters
  - Case-insensitive
- `"avatar"`: (*Body parameter*), `string`
  The updated base64 encoding of the avatar.
  - Maximum 65535 characters
- `"embedding_model"`: (*Body parameter*), `string`
  The updated embedding model name.
  - Ensure that `"chunk_count"` is `0` before updating `"embedding_model"`.
  - Maximum 255 characters
  - Must follow `model_name@model_factory` format
- `"permission"`: (*Body parameter*), `string`
  The updated dataset permission. Available options:
  - `"me"`: (Default) Only you can manage the dataset.
  - `"team"`: All team members can manage the dataset.
- `"pagerank"`: (*Body parameter*), `int`
  refer to [Set page rank](https://ragflow.io/docs/dev/set_page_rank)
  - Default: `0`
  - Minimum: `0`
  - Maximum: `100`
- `"chunk_method"`: (*Body parameter*), `enum<string>`
  The chunking method for the dataset. Available options:
  - `"naive"`: General (default)
  - `"book"`: Book
  - `"email"`: Email
  - `"laws"`: Laws
  - `"manual"`: Manual
  - `"one"`: One
  - `"paper"`: Paper
  - `"picture"`: Picture
  - `"presentation"`: Presentation
  - `"qa"`: Q&A
  - `"table"`: Table
  - `"tag"`: Tag
- `"parser_config"`: (*Body parameter*), `object`
  The configuration settings for the dataset parser. The attributes in this JSON object vary with the selected `"chunk_method"`:
  - If `"chunk_method"` is `"naive"`, the `"parser_config"` object contains the following attributes:
    - `"auto_keywords"`: `int`
      - Defaults to `0`
      - Minimum: `0`
      - Maximum: `32`
    - `"auto_questions"`: `int`
      - Defaults to `0`
      - Minimum: `0`
      - Maximum: `10`
    - `"chunk_token_num"`: `int`
      - Defaults to `512`
      - Minimum: `1`
      - Maximum: `2048`
    - `"delimiter"`: `string`
      - Defaults to `"\n"`.
    - `"html4excel"`: `bool` Indicates whether to convert Excel documents into HTML format.
      - Defaults to `false`
    - `"layout_recognize"`: `string`
      - Defaults to `DeepDOC`
    - `"tag_kb_ids"`: `array<string>` refer to [Use tag set](https://ragflow.io/docs/dev/use_tag_sets)
      - Must include a list of dataset IDs, where each dataset is parsed using the ​​Tag Chunking Method
    - `"task_page_size"`: `int` For PDF only.
      - Defaults to `12`
      - Minimum: `1`
    - `"raptor"`: `object` RAPTOR-specific settings.
      - Defaults to: `{"use_raptor": false}`
    - `"graphrag"`: `object` GRAPHRAG-specific settings.
      - Defaults to: `{"use_graphrag": false}`
    - `"parent_child"`: `object` Parent-child chunking settings. When enabled, each chunk is further split into smaller child chunks using `children_delimiter`. At retrieval time, matched child chunks are replaced by their parent's full text before being passed to the LLM, giving precise vector matching with broader context.
      - `"use_parent_child"`: `bool` Whether to enable parent-child chunking. Defaults to `false`.
      - `"children_delimiter"`: `string` The delimiter used to split a parent chunk into child chunks. Only takes effect when `"use_parent_child"` is `true`. Defaults to `"\n"`.
  - If `"chunk_method"` is `"qa"`, `"manual"`, `"paper"`, `"book"`, `"laws"`, or `"presentation"`, the `"parser_config"` object contains the following attribute:
    - `"raptor"`: `object` RAPTOR-specific settings.
      - Defaults to: `{"use_raptor": false}`.
  - If `"chunk_method"` is `"table"`, `"picture"`, `"one"`, or `"email"`, `"parser_config"` is an empty JSON object.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code": 102,
    "message": "Can't change tenant_id."
}
```

---

### List datasets

**GET** `/api/v1/datasets?page={page}&page_size={page_size}&orderby={orderby}&desc={desc}&name={dataset_name}&id={dataset_id}&include_parsing_status={include_parsing_status}`

Lists datasets.

#### Request

- Method: GET
- URL: `/api/v1/datasets?page={page}&page_size={page_size}&orderby={orderby}&desc={desc}&name={dataset_name}&id={dataset_id}&include_parsing_status={include_parsing_status}`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets?page={page}&page_size={page_size}&orderby={orderby}&desc={desc}&name={dataset_name}&id={dataset_id} \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

```bash
# List datasets with parsing status
curl --request GET \
     --url 'http://{address}/api/v1/datasets?include_parsing_status=true' \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `page`: (*Filter parameter*)
  Specifies the page on which the datasets will be displayed. Defaults to `1`.
- `page_size`: (*Filter parameter*)
  The number of datasets on each page. Defaults to `30`.
- `orderby`: (*Filter parameter*)
  The field by which datasets should be sorted. Available options:
  - `create_time` (default)
  - `update_time`
- `desc`: (*Filter parameter*)
  Indicates whether the retrieved datasets should be sorted in descending order. Defaults to `true`.
- `name`: (*Filter parameter*)
  The name of the dataset to retrieve.
- `id`: (*Filter parameter*)
  The ID of the dataset to retrieve.
- `include_parsing_status`: (*Filter parameter*)
  Whether to include document parsing status counts in the response. Defaults to `false`. When set to `true`, each dataset object in the response will include the following additional fields:
  - `unstart_count`: Number of documents not yet started parsing.
  - `running_count`: Number of documents currently being parsed.
  - `cancel_count`: Number of documents whose parsing was cancelled.
  - `done_count`: Number of documents that have been successfully parsed.
  - `fail_count`: Number of documents whose parsing failed.

#### Response

Success:

```json
{
    "code": 0,
    "data": [
        {
            "avatar": "",
            "chunk_count": 59,
            "create_date": "Sat, 14 Sep 2024 01:12:37 GMT",
            "create_time": 1726276357324,
            "created_by": "69736c5e723611efb51b0242ac120007",
            "description": null,
            "document_count": 1,
            "embedding_model": "BAAI/bge-large-zh-v1.5",
            "id": "6e211ee0723611efa10a0242ac120007",
            "language": "English",
            "name": "mysql",
            "chunk_method": "naive",
            "parser_config": {
                "chunk_token_num": 8192,
                "delimiter": "\\n",
                "entity_types": [
                    "organization",
                    "person",
                    "location",
                    "event",
                    "time"
                ]
            },
            "permission": "me",
            "similarity_threshold": 0.2,
            "status": "1",
            "tenant_id": "69736c5e723611efb51b0242ac120007",
            "token_num": 12744,
            "update_date": "Thu, 10 Oct 2024 04:07:23 GMT",
            "update_time": 1728533243536,
            "vector_similarity_weight": 0.3
        }
    ],
    "total_datasets": 1
}
```

Success (with `include_parsing_status=true`):

```json
{
    "code": 0,
    "data": [
        {
            "avatar": null,
            "cancel_count": 0,
            "chunk_count": 30,
            "chunk_method": "qa",
            "create_date": "2026-03-09T18:57:13",
            "create_time": 1773053833094,
            "created_by": "928f92a210b911f1ac4cc39e0b8fa3ad",
            "description": null,
            "document_count": 1,
            "done_count": 1,
            "embedding_model": "text-embedding-v2@Tongyi-Qianwen",
            "fail_count": 0,
            "id": "ba6586c21ba611f1a3dc476f0709e75e",
            "language": "English",
            "name": "Test Dataset",
            "parser_config": {
                "graphrag": { "use_graphrag": false },
                "llm_id": "deepseek-chat@DeepSeek",
                "raptor": { "use_raptor": false }
            },
            "permission": "me",
            "running_count": 0,
            "similarity_threshold": 0.2,
            "status": "1",
            "tenant_id": "928f92a210b911f1ac4cc39e0b8fa3ad",
            "token_num": 1746,
            "unstart_count": 0,
            "update_date": "2026-03-09T18:59:32",
            "update_time": 1773053972723,
            "vector_similarity_weight": 0.3
        }
    ],
    "total_datasets": 1
}
```

Failure:

```json
{
    "code": 102,
    "message": "The dataset doesn't exist"
}
```

 ---

### Get knowledge graph

**GET** `/api/v1/datasets/{dataset_id}/knowledge_graph`

Retrieves the knowledge graph of a specified dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/knowledge_graph`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/knowledge_graph \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "graph": {
            "directed": false,
            "edges": [
                {
                    "description": "The notice is a document issued to convey risk warnings and operational alerts.<SEP>The notice is a specific instance of a notification document issued under the risk warning framework.",
                    "keywords": ["9", "8"],
                    "source": "notice",
                    "source_id": ["8a46cdfe4b5c11f0a5281a58e595aa1c"],
                    "src_id": "xxx",
                    "target": "xxx",
                    "tgt_id": "xxx",
                    "weight": 17.0
                }
            ],
            "graph": {
                "source_id": ["8a46cdfe4b5c11f0a5281a58e595aa1c", "8a7eb6424b5c11f0a5281a58e595aa1c"]
            },
            "multigraph": false,
            "nodes": [
                {
                    "description": "xxx",
                    "entity_name": "xxx",
                    "entity_type": "ORGANIZATION",
                    "id": "xxx",
                    "pagerank": 0.10804906590624092,
                    "rank": 3,
                    "source_id": ["8a7eb6424b5c11f0a5281a58e595aa1c"]
                }
            ]
        },
        "mind_map": {}
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "The dataset doesn't exist"
}
```

---

### Delete knowledge graph

**DELETE** `/api/v1/datasets/{dataset_id}/knowledge_graph`

Removes the knowledge graph of a specified dataset.

#### Request

- Method: DELETE
- URL: `/api/v1/datasets/{dataset_id}/knowledge_graph`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/knowledge_graph \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code": 0,
    "data": true
}
```

Failure:

```json
{
    "code": 102,
    "message": "The dataset doesn't exist"
}
```

---

### Construct knowledge graph

**POST** `/api/v1/datasets/{dataset_id}/run_graphrag`

Constructs a knowledge graph from a specified dataset.

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/run_graphrag`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/run_graphrag \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code":0,
    "data":{
      "graphrag_task_id":"e498de54bfbb11f0ba028f704583b57b"
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "Invalid Dataset ID"
}
```

---

### Get knowledge graph construction status

**GET** `/api/v1/datasets/{dataset_id}/trace_graphrag`

Retrieves the knowledge graph construction status for a specified dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/trace_graphrag`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/trace_graphrag \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code":0,
    "data":{
        "begin_at":"Wed, 12 Nov 2025 19:36:56 GMT",
        "chunk_ids":"",
        "create_date":"Wed, 12 Nov 2025 19:36:56 GMT",
        "create_time":1762947416350,
        "digest":"39e43572e3dcd84f",
        "doc_id":"44661c10bde211f0bc93c164a47ffc40",
        "from_page":100000000,
        "id":"e498de54bfbb11f0ba028f704583b57b",
        "priority":0,
        "process_duration":2.45419,
        "progress":1.0,
        "progress_msg":"19:36:56 created task graphrag\n19:36:57 Task has been received.\n19:36:58 [GraphRAG] doc:083661febe2411f0bc79456921e5745f has no available chunks, skip generation.\n19:36:58 [GraphRAG] build_subgraph doc:44661c10bde211f0bc93c164a47ffc40 start (chunks=1, timeout=10000000000s)\n19:36:58 Graph already contains 44661c10bde211f0bc93c164a47ffc40\n19:36:58 [GraphRAG] build_subgraph doc:44661c10bde211f0bc93c164a47ffc40 empty\n19:36:58 [GraphRAG] kb:33137ed0bde211f0bc93c164a47ffc40 no subgraphs generated successfully, end.\n19:36:58 Knowledge Graph done (0.72s)","retry_count":1,
        "task_type":"graphrag",
        "to_page":100000000,
        "update_date":"Wed, 12 Nov 2025 19:36:58 GMT",
        "update_time":1762947418454
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "Invalid Dataset ID"
}
```

---

### Construct RAPTOR

**POST** `/api/v1/datasets/{dataset_id}/run_raptor`

Construct a RAPTOR from a specified dataset.

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/run_raptor`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/run_raptor \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code":0,
    "data":{
        "raptor_task_id":"50d3c31cbfbd11f0ba028f704583b57b"
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "Invalid Dataset ID"
}
```

---

### Get RAPTOR construction status

**GET** `/api/v1/datasets/{dataset_id}/trace_raptor`

Retrieves the RAPTOR construction status for a specified dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/trace_raptor`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/trace_raptor \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the target dataset.

#### Response

Success:

```json
{
    "code":0,
    "data":{
        "begin_at":"Wed, 12 Nov 2025 19:47:07 GMT",
        "chunk_ids":"",
        "create_date":"Wed, 12 Nov 2025 19:47:07 GMT",
        "create_time":1762948027427,
        "digest":"8b279a6248cb8fc6",
        "doc_id":"44661c10bde211f0bc93c164a47ffc40",
        "from_page":100000000,
        "id":"50d3c31cbfbd11f0ba028f704583b57b",
        "priority":0,
        "process_duration":0.948244,
        "progress":1.0,
        "progress_msg":"19:47:07 created task raptor\n19:47:07 Task has been received.\n19:47:07 Processing...\n19:47:07 Processing...\n19:47:07 Indexing done (0.01s).\n19:47:07 Task done (0.29s)",
        "retry_count":1,
        "task_type":"raptor",
        "to_page":100000000,
        "update_date":"Wed, 12 Nov 2025 19:47:07 GMT",
        "update_time":1762948027948
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "Invalid Dataset ID"
}
```

---

## FILE MANAGEMENT WITHIN DATASET

---

### Upload documents

**POST** `/api/v1/datasets/{dataset_id}/documents`

Uploads documents to a specified dataset.

This endpoint supports three creation modes via the optional `type` query parameter:

- `type=local` or omitted: Upload one or more local files using `multipart/form-data`.
- `type=web`: Crawl a web page and save it as a document.
- `type=empty`: Create an empty virtual document by name.

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/documents`
- Query:
  - `type`: Optional. One of `local`, `web`, or `empty`. Defaults to `local`.
- Headers:
  - `'Content-Type: multipart/form-data'` for `type=local` and `type=web`
  - `'Content-Type: application/json'` for `type=empty`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - For `type=local`: form field `'file=@{FILE_PATH}'`
  - For `type=web`: form fields `'name'` and `'url'`
  - For `type=empty`: JSON body with `'name'`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents \
     --header 'Content-Type: multipart/form-data' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --form 'file=@./test1.txt' \
     --form 'file=@./test2.pdf'
```

```bash
curl --request POST \
     --url 'http://{address}/api/v1/datasets/{dataset_id}/documents?type=web' \
     --header 'Content-Type: multipart/form-data' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --form 'name=example-page' \
     --form 'url=https://example.com'
```

```bash
curl --request POST \
     --url 'http://{address}/api/v1/datasets/{dataset_id}/documents?type=empty' \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{"name":"blank.txt"}'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the dataset to which the documents will be uploaded.
- `type`: (*Query parameter*)
  Controls how the document is created:
  - `local`: Upload files.
  - `web`: Crawl a URL into a document.
  - `empty`: Create an empty document without file upload.
- `'file'`: (*Body parameter*)
  A document to upload. Required when `type=local`.
- `'name'`: (*Body parameter*)
  The document name. Required when `type=web` or `type=empty`.
- `'url'`: (*Body parameter*)
  The source URL to crawl. Required when `type=web`.

#### Response

Success:

```json
{
    "code": 0,
    "data": [
        {
            "chunk_method": "naive",
            "created_by": "69736c5e723611efb51b0242ac120007",
            "dataset_id": "527fa74891e811ef9c650242ac120006",
            "id": "b330ec2e91ec11efbc510242ac120004",
            "location": "1.txt",
            "name": "1.txt",
            "parser_config": {
                "chunk_token_num": 128,
                "delimiter": "\\n",
                "html4excel": false,
                "layout_recognize": true,
                "raptor": {
                    "use_raptor": false
                }
            },
            "run": "UNSTART",
            "size": 17966,
            "thumbnail": "",
            "type": "doc"
        }
    ]
}
```

Failure:

```json
{
    "code": 101,
    "message": "No file part!"
}
```

---

### Update document

**PUT** `/api/v1/datasets/{dataset_id}/documents/{document_id}`

Updates configurations for a specified document.

#### Request

- Method: PUT
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"name"`:`string`
  - `"meta_fields"`:`object`
  - `"chunk_method"`:`string`
  - `"parser_config"`:`object`

##### Request example

```bash
curl --request PUT \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id} \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --header 'Content-Type: application/json' \
     --data '
     {
          "name": "manual.txt",
          "chunk_method": "manual",
          "parser_config": {"chunk_token_num": 128}
     }'

```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the associated dataset.
- `document_id`: (*Path parameter*)
  The ID of the document to update.
- `"name"`: (*Body parameter*), `string`
- `"meta_fields"`: (*Body parameter*), `dict[str, Any]` The meta fields of the document.
- `"chunk_method"`: (*Body parameter*), `string`
  The parsing method to apply to the document:
  - `"naive"`: General
  - `"manual`: Manual
  - `"qa"`: Q&A
  - `"table"`: Table
  - `"paper"`: Paper
  - `"book"`: Book
  - `"laws"`: Laws
  - `"presentation"`: Presentation
  - `"picture"`: Picture
  - `"one"`: One
  - `"email"`: Email
- `"parser_config"`: (*Body parameter*), `object`
  The configuration settings for the dataset parser. The attributes in this JSON object vary with the selected `"chunk_method"`:
  - If `"chunk_method"` is `"naive"`, the `"parser_config"` object contains the following attributes:
    - `"chunk_token_num"`: Defaults to `256`.
    - `"layout_recognize"`: Defaults to `true`.
    - `"html4excel"`: Indicates whether to convert Excel documents into HTML format. Defaults to `false`.
    - `"delimiter"`: Defaults to `"\n"`.
    - `"task_page_size"`: Defaults to `12`. For PDF only.
    - `"raptor"`: RAPTOR-specific settings. Defaults to: `{"use_raptor": false}`.
  - If `"chunk_method"` is `"qa"`, `"manual"`, `"paper"`, `"book"`, `"laws"`, or `"presentation"`, the `"parser_config"` object contains the following attribute:
    - `"raptor"`: RAPTOR-specific settings. Defaults to: `{"use_raptor": false}`.
  - If `"chunk_method"` is `"table"`, `"picture"`, `"one"`, or `"email"`, `"parser_config"` is an empty JSON object.
- `"enabled"`: (*Body parameter*), `integer`
  Whether the document should be **available** in the knowledge base.
  - `1` → （available）
  - `0` → （unavailable）

#### Response

Success:

```json
{
  "code": 0,
  "data": {
    "id": "cd38dd72d4a611f0af9c71de94a988ef",
    "name": "large.md",
    "type": "doc",
    "suffix": "md",
    "size": 2306906,
    "location": "large.md",
    "source_type": "local",
    "status": "1",
    "run": "DONE",
    "dataset_id": "5f546a1ad4a611f0af9c71de94a988ef",

    "chunk_method": "naive",
    "chunk_count": 2,
    "token_count": 8126,

    "created_by": "eab7f446cb5a11f0ab334fbc3aa38f35",
    "create_date": "Tue, 09 Dec 2025 10:28:52 GMT",
    "create_time": 1765247332122,
    "update_date": "Wed, 17 Dec 2025 10:51:16 GMT",
    "update_time": 1765939876819,

    "process_begin_at": "Wed, 17 Dec 2025 10:33:55 GMT",
    "process_duration": 14.8615,
    "progress": 1.0,

    "progress_msg": [
      "10:33:58 Task has been received.",
      "10:33:59 Page(1~100000001): Start to parse.",
      "10:33:59 Page(1~100000001): Finish parsing.",
      "10:34:07 Page(1~100000001): Generate 2 chunks",
      "10:34:09 Page(1~100000001): Embedding chunks (2.13s)",
      "10:34:09 Page(1~100000001): Indexing done (0.31s).",
      "10:34:09 Page(1~100000001): Task done (11.68s)"
    ],

    "parser_config": {
      "chunk_token_num": 512,
      "delimiter": "\n",
      "auto_keywords": 0,
      "auto_questions": 0,
      "topn_tags": 3,

      "layout_recognize": "DeepDOC",
      "html4excel": false,
      "image_context_size": 0,
      "table_context_size": 0,

      "graphrag": {
        "use_graphrag": true,
        "method": "light",
        "entity_types": [
          "organization",
          "person",
          "geo",
          "event",
          "category"
        ]
      },

      "raptor": {
        "use_raptor": true,
        "max_cluster": 64,
        "max_token": 256,
        "threshold": 0.1,
        "random_seed": 0,
        "prompt": "Please summarize the following paragraphs. Be careful with the numbers, do not make things up. Paragraphs as following:\n      {cluster_content}\nThe above is the content you need to summarize."
      }
    },

    "meta_fields": {},
    "pipeline_id": "",
    "thumbnail": ""
  }
}

```

Failure:

```json
{
    "code": 102,
    "message": "The dataset does not have the document."
}
```

---

### Download document

**GET** `/api/v1/datasets/{dataset_id}/documents/{document_id}`

Downloads a document from a specified dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Output:
  - `'{PATH_TO_THE_FILE}'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id} \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --output ./ragflow.txt
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `documents_id`: (*Path parameter*)
  The ID of the document to download.

#### Response

Success:

```json
This is a test to verify the file download feature.
```

Failure:

```json
{
    "code": 102,
    "message": "You do not own the dataset 7898da028a0511efbf750242ac1220005."
}
```

---

### List documents

**GET** `/api/v1/datasets/{dataset_id}/documents?page={page}&page_size={page_size}&orderby={orderby}&desc={desc}&keywords={keywords}&id={document_id}&name={document_name}&create_time_from={timestamp}&create_time_to={timestamp}&suffix={file_suffix}&run={run_status}&metadata_condition={json}`

Lists documents in a specified dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/documents?page={page}&page_size={page_size}&orderby={orderby}&desc={desc}&keywords={keywords}&id={document_id}&name={document_name}&create_time_from={timestamp}&create_time_to={timestamp}&suffix={file_suffix}&run={run_status}`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request examples

**A basic request with pagination:**

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents?page=1&page_size=10 \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `keywords`: (*Filter parameter*), `string`
  The keywords used to match document titles.
- `page`: (*Filter parameter*), `integer`
  Specifies the page on which the documents will be displayed. Defaults to `1`.
- `page_size`: (*Filter parameter*), `integer`
  The maximum number of documents on each page. Defaults to `30`.
- `orderby`: (*Filter parameter*), `string`
  The field by which documents should be sorted. Available options:
  - `create_time` (default)
  - `update_time`
- `desc`: (*Filter parameter*), `boolean`
  Indicates whether the retrieved documents should be sorted in descending order. Defaults to `true`.
- `id`: (*Filter parameter*), `string`
  The ID of the document to retrieve.
- `create_time_from`: (*Filter parameter*), `integer`
  Unix timestamp for filtering documents created after this time. 0 means no filter. Defaults to `0`.
- `create_time_to`: (*Filter parameter*), `integer`
  Unix timestamp for filtering documents created before this time. 0 means no filter. Defaults to `0`.
- `suffix`: (*Filter parameter*), `array[string]`
  Filter by file suffix. Supports multiple values, e.g., `pdf`, `txt`, and `docx`. Defaults to all suffixes.
- `run`: (*Filter parameter*), `array[string]`
  Filter by document processing status. Supports numeric, text, and mixed formats:
  - Numeric format: `["0", "1", "2", "3", "4"]`
  - Text format: `[UNSTART, RUNNING, CANCEL, DONE, FAIL]`
  - Mixed format: `[UNSTART, 1, DONE]` (mixing numeric and text formats)
  - Status mapping:
    - `0` / `UNSTART`: Document not yet processed
    - `1` / `RUNNING`: Document is currently being processed
    - `2` / `CANCEL`: Document processing was cancelled
    - `3` / `DONE`: Document processing completed successfully
    - `4` / `FAIL`: Document processing failed
  Defaults to all statuses.
- `metadata_condition`: (*Filter parameter*), `object` (JSON in query)
  Optional metadata filter applied to documents when `document_ids` is not provided. Uses the same structure as retrieval:
  - `logic`: `"and"` (default) or `"or"`
  - `conditions`: array of `{ "name": string, "comparison_operator": string, "value": string }`
    - `comparison_operator` supports: `is`, `not is`, `contains`, `not contains`, `in`, `not in`, `start with`, `end with`, `>`, `<`, `≥`, `≤`, `empty`, `not empty`

##### Usage examples

**A request with multiple filtering parameters**

```bash
curl --request GET \
     --url 'http://{address}/api/v1/datasets/{dataset_id}/documents?suffix=pdf&run=DONE&page=1&page_size=10' \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

**Filter by metadata (query JSON):**

```bash
curl -G \
  --url "http://localhost:9222/api/v1/datasets/{{KB_ID}}/documents" \
  --header 'Authorization: Bearer <YOUR_API_KEY>' \
  --data-urlencode 'metadata_condition={"logic":"and","conditions":[{"name":"tags","comparison_operator":"is","value":"bar"},{"name":"author","comparison_operator":"is","value":"alice"}]}'
```

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "docs": [
            {
                "chunk_count": 0,
                "create_date": "Mon, 14 Oct 2024 09:11:01 GMT",
                "create_time": 1728897061948,
                "created_by": "69736c5e723611efb51b0242ac120007",
                "id": "3bcfbf8a8a0c11ef8aba0242ac120006",
                "knowledgebase_id": "7898da028a0511efbf750242ac120005",
                "location": "Test_2.txt",
                "name": "Test_2.txt",
                "parser_config": {
                    "chunk_token_count": 128,
                    "delimiter": "\n",
                    "layout_recognize": true,
                    "task_page_size": 12
                },
                "chunk_method": "naive",
                "process_begin_at": null,
                "process_duration": 0.0,
                "progress": 0.0,
                "progress_msg": "",
                "run": "UNSTART",
                "size": 7,
                "source_type": "local",
                "status": "1",
                "thumbnail": null,
                "token_count": 0,
                "type": "doc",
                "update_date": "Mon, 14 Oct 2024 09:11:01 GMT",
                "update_time": 1728897061948
            }
        ],
        "total_datasets": 1
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "You don't own the dataset 7898da028a0511efbf750242ac1220005. "
}
```

---

### Delete documents

**DELETE** `/api/v1/datasets/{dataset_id}/documents`

Deletes documents by ID.

#### Request

- Method: DELETE
- URL: `/api/v1/datasets/{dataset_id}/documents`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"ids"`: `list[string]`
  - `"delete_all"`: `boolean`

##### Request example

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "ids": ["id_1","id_2"]
     }'
```

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
          "delete_all": true
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `"ids"`: (*Body parameter*), `list[string]`
  The IDs of the documents to delete.
  - If omitted, or set to `null` or an empty array, no documents are deleted.
  - If an array of IDs is provided, only the documents matching those IDs are deleted.
- `"delete_all"`: (*Body parameter*), `boolean`
  Whether to delete all documents in the specified dataset when `"ids"` is omitted, or set to `null` or an empty array. Defaults to `false`.

#### Response

Success:

```json
{
    "code": 0
}.
```

Failure:

```json
{
    "code": 102,
    "message": "You do not own the dataset 7898da028a0511efbf750242ac1220005."
}
```

---

### Parse documents

**POST** `/api/v1/datasets/{dataset_id}/chunks`

Parses documents in a specified dataset using the built-in chunking pipeline.

:::note
This endpoint only supports datasets that use the built-in chunking pipeline. For datasets configured with an ingestion pipeline, use `POST /api/v1/documents/ingest` instead.
:::

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/chunks`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"document_ids"`: `list[string]`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "document_ids": ["97a5f1c2759811efaa500242ac120004","97ad64b6759811ef9fc30242ac120004"]
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The dataset ID.
- `"document_ids"`: (*Body parameter*), `list[string]`, *Required*
  The IDs of the documents to parse.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code": 102,
    "message": "`document_ids` is required"
}
```

---

### Ingest documents

**POST** `/api/v1/documents/ingest`

Starts, cancels, or reruns ingestion for documents. Use this endpoint for documents in datasets configured with an ingestion pipeline.

#### Request

- Method: POST
- URL: `/api/v1/documents/ingest`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"doc_ids"`: `list[string]`
  - `"run"`: `string`
  - `"delete"`: `boolean`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/documents/ingest \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "doc_ids": ["97a5f1c2759811efaa500242ac120004"],
          "run": "1",
          "delete": true
     }'
```

##### Request parameters

- `"doc_ids"`: (*Body parameter*), `list[string]`, *Required*
  The IDs of the documents to ingest.
- `"run"`: (*Body parameter*), `string`, *Required*
  The ingestion action. Use `"1"` to start ingestion and `"2"` to cancel ingestion.
- `"delete"`: (*Body parameter*), `boolean`
  Whether to delete existing tasks and chunks before rerunning. Defaults to `false`.

#### Response

Success:

```json
{
    "code": 0,
    "data": true
}
```

Failure:

```json
{
    "code": 102,
    "message": "Document not found!"
}
```

---

### Stop parsing documents

**DELETE** `/api/v1/datasets/{dataset_id}/chunks`

Stops parsing specified documents.

#### Request

- Method: DELETE
- URL: `/api/v1/datasets/{dataset_id}/chunks`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"document_ids"`: `list[string]`

##### Request example

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "document_ids": ["97a5f1c2759811efaa500242ac120004","97ad64b6759811ef9fc30242ac120004"]
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `"document_ids"`: (*Body parameter*), `list[string]`, *Required*
  The IDs of the documents for which the parsing should be stopped.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code": 102,
    "message": "`document_ids` is required"
}
```

---

## CHUNK MANAGEMENT WITHIN DATASET

---

### Add chunk

**POST** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`

Adds a chunk to a specified document in a specified dataset.

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"content"`: `string`
  - `"important_keywords"`: `list[string]`
  - `"tag_kwd"`: `list[string]`
  - `"questions"`: `list[string]`
  - `"image_base64"`: `string`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "content": "<CHUNK_CONTENT_HERE>",
          "image_base64": "<BASE64_ENCODED_IMAGE>"
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `document_id`: (*Path parameter*)
  The associated document ID.
- `"content"`: (*Body parameter*), `string`, *Required*
  The text content of the chunk.
- `"important_keywords"`: (*Body parameter*), `list[string]`
  The key terms or phrases to tag with the chunk.
- `"tag_kwd"`: (*Body parameter*), `list[string]`
  Tag keywords to associate with the chunk.
- `"questions"`: (*Body parameter*), `list[string]`
  Optional questions to use when embedding the chunk.
- `"image_base64"`: (*Body parameter*), `string`
  A base64-encoded image to associate with the chunk.

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "chunk": {
            "content": "who are you",
            "create_time": "2024-12-30 16:59:55",
            "create_timestamp": 1735549195.969164,
            "dataset_id": "72f36e1ebdf411efb7250242ac120006",
            "document_id": "61d68474be0111ef98dd0242ac120006",
            "id": "12ccdc56e59837e5",
            "image_id": "",
            "important_keywords": [],
            "tag_kwd": [],
            "questions": []
        }
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "`content` is required"
}
```

---

### List chunks

**GET** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks?keywords={keywords}&page={page}&page_size={page_size}&id={id}`

Lists chunks in a specified document.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks?keywords={keywords}&page={page}&page_size={page_size}&id={chunk_id}`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks?keywords={keywords}&page={page}&page_size={page_size}&id={chunk_id} \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `document_id`: (*Path parameter*)
  The associated document ID.
- `keywords`(*Filter parameter*), `string`
  The keywords used to match chunk content.
- `page`(*Filter parameter*), `integer`
  Specifies the page on which the chunks will be displayed. Defaults to `1`.
- `page_size`(*Filter parameter*), `integer`
  The maximum number of chunks on each page. Defaults to `30`.
- `id`(*Filter parameter*), `string`
  The ID of the chunk to retrieve. You can also use `GET /api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}` to retrieve one chunk.

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "chunks": [
            {
                "available": true,
                "content": "This is a test content.",
                "docnm_kwd": "1.txt",
                "document_id": "b330ec2e91ec11efbc510242ac120004",
                "id": "b48c170e90f70af998485c1065490726",
                "image_id": "",
                "important_keywords": [],
                "tag_kwd": [],
                "positions": []
            }
        ],
        "doc": {
            "chunk_count": 1,
            "chunk_method": "naive",
            "create_date": "Thu, 24 Oct 2024 09:45:27 GMT",
            "create_time": 1729763127646,
            "created_by": "69736c5e723611efb51b0242ac120007",
            "dataset_id": "527fa74891e811ef9c650242ac120006",
            "id": "b330ec2e91ec11efbc510242ac120004",
            "location": "1.txt",
            "name": "1.txt",
            "parser_config": {
                "chunk_token_num": 128,
                "delimiter": "\\n",
                "html4excel": false,
                "layout_recognize": true,
                "raptor": {
                    "use_raptor": false
                }
            },
            "process_begin_at": "Thu, 24 Oct 2024 09:56:44 GMT",
            "process_duration": 0.54213,
            "progress": 0.0,
            "progress_msg": "Task dispatched...",
            "run": "2",
            "size": 17966,
            "source_type": "local",
            "status": "1",
            "thumbnail": "",
            "token_count": 8,
            "type": "doc",
            "update_date": "Thu, 24 Oct 2024 11:03:15 GMT",
            "update_time": 1729767795721
        },
        "total": 1
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "You don't own the document 5c5999ec7be811ef9cab0242ac12000e5."
}
```

---

### Get chunk

**GET** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}`

Retrieves a specified chunk in a specified document. Runtime fields such as vector and token fields are not returned.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Request example

```bash
curl --request GET \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id} \
     --header 'Authorization: Bearer <YOUR_API_KEY>'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `document_id`: (*Path parameter*)
  The associated document ID.
- `chunk_id`: (*Path parameter*)
  The ID of the chunk to retrieve.

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "available_int": 1,
        "content_with_weight": "This is a test content.",
        "doc_id": "b330ec2e91ec11efbc510242ac120004",
        "docnm_kwd": "1.txt",
        "id": "b48c170e90f70af998485c1065490726",
        "img_id": "",
        "important_kwd": [],
        "question_kwd": [],
        "tag_kwd": []
    }
}
```

Failure:

```json
{
    "code": 100,
    "message": "Chunk not found"
}
```

---

### Delete chunks

**DELETE** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`

Deletes chunks by ID.

#### Request

- Method: DELETE
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"chunk_ids"`: `list[string]`
  - `"delete_all"`: `boolean`

##### Request example

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "chunk_ids": ["test_1", "test_2"]
     }'
```

```bash
curl --request DELETE \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
          "delete_all": true
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `document_id`: (*Path parameter*)
  The associated document ID.
- `"chunk_ids"`: (*Body parameter*), `list[string]`
  The IDs of the chunks to delete.
  - If omitted, or set to `null` or an empty array, no chunks are deleted.
  - If an array of IDs is provided, only the chunks matching those IDs are deleted.
- `"delete_all"`: (*Body parameter*), `boolean`
  Whether to delete all chunks of the specified document when `"chunk_ids"` is omitted, or set to `null` or an empty array. Defaults to `false`.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code": 102,
    "message": "rm_chunk deleted chunks 0, expect 1"
}
```

---

### Update chunk

**PATCH** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}`

Updates content or configurations for a specified chunk.

:::caution DEPRECATED
`PUT /api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}` is deprecated. Use this endpoint instead.
:::

#### Request

- Method: PATCH
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id}`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"content"`: `string`
  - `"important_keywords"`: `list[string]`
  - `"questions"`: `list[string]`
  - `"positions"`: `list`
  - `"tag_kwd"`: `list[string]`
  - `"available"`: `boolean`
  - `"image_base64"`: `string`

##### Request example

```bash
curl --request PATCH \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks/{chunk_id} \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "content": "ragflow123",
          "important_keywords": []
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `document_id`: (*Path parameter*)
  The associated document ID.
- `chunk_id`: (*Path parameter*)
  The ID of the chunk to update.
- `"content"`: (*Body parameter*), `string`
  The text content of the chunk.
- `"important_keywords"`: (*Body parameter*), `list[string]`
  A list of key terms or phrases to tag with the chunk.
- `"questions"`: (*Body parameter*), `list[string]`
  Optional questions to use when embedding the chunk.
- `"positions"`: (*Body parameter*), `list`
  Updated source positions for the chunk.
- `"tag_kwd"`: (*Body parameter*), `list[string]`
  Updated tag keywords.
- `"available"`: (*Body parameter*) `boolean`
  The chunk's availability status in the dataset. Value options:
  - `true`: Available (default)
  - `false`: Unavailable
- `"image_base64"`: (*Body parameter*), `string`
  Base64-encoded image content to associate with the chunk.

#### Response

Success:

```json
{
    "code": 0
}
```

Failure:

```json
{
    "code": 102,
    "message": "Can't find this chunk 29a2d9987e16ba331fb4d7d30d99b71d2"
}
```

---

### Update chunk availability

**PATCH** `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`

Updates or switches the availability status of specified chunks, controlling whether they are available for retrieval.

#### Request

- Method: PATCH
- URL: `/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks`
- Headers:
  - `'Content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"chunk_ids"`: `list[string]` (*Required*)
  - `"available_int"`: `integer` (*Optional*)
  - `"available"`: `boolean` (*Optional*)

##### Request example

```bash
curl --request PATCH \
     --url http://{address}/api/v1/datasets/{dataset_id}/documents/{document_id}/chunks \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "chunk_ids": ["chunk_id_1", "chunk_id_2"],
          "available_int": 1
     }'
```

##### Request parameters

- `dataset_id`: (*Path parameter*)
  The ID of the dataset.
- `document_id`: (*Path parameter*)
  The ID of the document.
- `"chunk_ids"`: (*Body parameter*), `list[string]` (*Required*)
  IDs of the chunks whose availability status is to be updated.
- `"available_int"`: (*Body parameter*), `integer` (*Optional*)
  Availability status for the specified chunks. You must provide either `"available_int"` or `"available"`. If both are provided, `"available_int"` is used.
  - `1`: Available,
  - `0`: Unavailable.
- `"available"`: (*Body parameter*), `boolean` (*Optional*)
  Availability status of the specified chunks. Used when `"available_int"` is not provided.
  - `true`: Available,
  - `false`: Unavailable.

#### Response

Success:

```json
{
    "code": 0,
    "data": true
}
```

Failure:

```json
{
    "code": 102,
    "message": "You don't own the dataset {dataset_id}."
}
```

```json
{
    "code": 102,
    "message": "`chunk_ids` is required."
}
```

```json
{
    "code": 102,
    "message": "`available_int` or `available` is required."
}
```

```json
{
    "code": 102,
    "message": "Document not found!"
}
```

```json
{
    "code": 102,
    "message": "Index updating failure"
}
```

---

### Retrieve a metadata summary from a dataset

**GET** `/api/v1/datasets/{dataset_id}/metadata/summary`

Aggregates metadata values across all documents in a dataset.

#### Request

- Method: GET
- URL: `/api/v1/datasets/{dataset_id}/metadata/summary`
- Headers:
  - `'Authorization: Bearer <YOUR_API_KEY>'`

##### Response

Success:

```json
{
  "code": 0,
  "data": {
    "summary": {
      "tags": {
        "type": "string",
        "values": [["bar", 2], ["foo", 1], ["baz", 1]]
      },
      "author": {
        "type": "string",
        "values": [["alice", 2], ["bob", 1]]
      }
    }
  }
}
```

---

### Update or delete metadata

**POST** `/api/v1/datasets/{dataset_id}/metadata/update`

Batch update or delete document-level metadata within a specified dataset. If both `document_ids` and `metadata_condition` are omitted, all documents within that dataset are selected. When both are provided, the intersection is used.

#### Request

- Method: POST
- URL: `/api/v1/datasets/{dataset_id}/metadata/update`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `selector`: `object`
  - `updates`: `list[object]`
  - `deletes`: `list[object]`

#### Request parameters

- `dataset_id`: (*Path parameter*)
  The associated dataset ID.
- `"selector"`: (*Body parameter*), `object`, *optional*
  A document selector:
  - `"document_ids"`: `list[string]` *optional*
    The associated document ID.
  - `"metadata_condition"`: `object`, *optional*
    - `"logic"`: Defines the logic relation between conditions if multiple conditions are provided. Options:
      - `"and"` (default)
      - `"or"`
    - `"conditions"`: `list[object]` *optional*
      Each object: `{ "name": string, "comparison_operator": string, "value": string }`
      - `"name"`: `string` The key name to search by.
      - `"comparison_operator"`: `string` Available options:
        - `"is"`
        - `"not is"`
        - `"contains"`
        - `"not contains"`
        - `"in"`
        - `"not in"`
        - `"start with"`
        - `"end with"`
        - `">"`
        - `"<"`
        - `"≥"`
        - `"≤"`
        - `"empty"`
        - `"not empty"`
      - `"value"`: `string` The key value to search by.
- `"updates"`: (*Body parameter*), `list[object]`, *optional*
  Replaces metadata of the retrieved documents. Each object: `{ "key": string, "match": string, "value": string }`.
  - `"key"`: `string` The name of the key to update.
  - `"match"`: `string` *optional* The current value of the key to update. When omitted, the corresponding keys are updated to `"value"` regardless of their current values.
  - `"value"`: `string` The new value to set for the specified keys.
- `"deletes"`: (*Body parameter*), `list[object]`, *optional*
  Deletes metadata of the retrieved documents. Each object: `{ "key": string, "value": string }`.
  - `"key"`: `string` The name of the key to delete.
  - `"value"`: `string` *Optional* The value of the key to delete.
    - When provided, only keys with a matching value are deleted.
    - When omitted, all specified keys are deleted.

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/datasets/{dataset_id}/metadata/update \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '{
       "selector": {
         "metadata_condition": {
           "logic": "and",
           "conditions": [
             {"name": "author", "comparison_operator": "is", "value": "alice"}
           ]
         }
       },
       "updates": [
         {"key": "tags", "match": "foo", "value": "foo_new"}
       ],
       "deletes": [
         {"key": "obsolete_key"},
         {"key": "author", "value": "alice"}
       ]
     }'
```

##### Response

Success:

```json
{
  "code": 0,
  "data": {
    "updated": 1,
    "matched_docs": 2
  }
}
```

---

### Retrieve chunks

**POST** `/api/v1/retrieval`

Retrieves chunks from specified datasets.

#### Request

- Method: POST
- URL: `/api/v1/retrieval`
- Headers:
  - `'content-Type: application/json'`
  - `'Authorization: Bearer <YOUR_API_KEY>'`
- Body:
  - `"question"`: `string`
  - `"dataset_ids"`: `list[string]`
  - `"document_ids"`: `list[string]`
  - `"page"`: `integer`
  - `"page_size"`: `integer`
  - `"similarity_threshold"`: `float`
  - `"vector_similarity_weight"`: `float`
  - `"top_k"`: `integer`
  - `"rerank_id"`: `string`
  - `"keyword"`: `boolean`
  - `"highlight"`: `boolean`
  - `"cross_languages"`: `list[string]`
  - `"metadata_condition"`: `object`
  - `"use_kg"`: `boolean`
  - `"toc_enhance"`: `boolean`

##### Request example

```bash
curl --request POST \
     --url http://{address}/api/v1/retrieval \
     --header 'Content-Type: application/json' \
     --header 'Authorization: Bearer <YOUR_API_KEY>' \
     --data '
     {
          "question": "What is advantage of ragflow?",
          "dataset_ids": ["b2a62730759d11ef987d0242ac120004"],
          "document_ids": ["77df9ef4759a11ef8bdd0242ac120004"],
          "metadata_condition": {
            "logic": "and",
            "conditions": [
              {
                "name": "author",
                "comparison_operator": "=",
                "value": "Toby"
              },
              {
                "name": "url",
                "comparison_operator": "not contains",
                "value": "amd"
              }
            ]
          }
     }'
```

##### Request parameter

- `"question"`: (*Body parameter*), `string`, *Required*
  The user query or query keywords.
- `"dataset_ids"`: (*Body parameter*) `list[string]`
  The IDs of the datasets to search. If you do not set this argument, ensure that you set `"document_ids"`.
- `"document_ids"`: (*Body parameter*), `list[string]`
  The IDs of the documents to search. Ensure that all selected documents use the same embedding model. Otherwise, an error will occur. If you do not set this argument, ensure that you set `"dataset_ids"`.
- `"page"`: (*Body parameter*), `integer`
  Specifies the page on which the chunks will be displayed. Defaults to `1`.
- `"page_size"`: (*Body parameter*)
  The maximum number of chunks on each page. Defaults to `30`.
- `"similarity_threshold"`: (*Body parameter*)
  The minimum similarity score. Defaults to `0.2`.
- `"vector_similarity_weight"`: (*Body parameter*), `float`
  The weight of vector cosine similarity. Defaults to `0.3`. If x represents the weight of vector cosine similarity, then (1 - x) is the term similarity weight.
- `"top_k"`: (*Body parameter*), `integer`
  The number of chunks engaged in vector cosine computation. Defaults to `1024`.
- `"use_kg"`: (*Body parameter*), `boolean`
  Whether to search chunks related to the generated knowledge graph for multi-hop queries. Defaults to `False`. Before enabling this, ensure you have successfully constructed a knowledge graph for the specified datasets. See [here](../guides/dataset/advanced/construct_knowledge_graph.md) for details.
- `"toc_enhance"`: (*Body parameter*), `boolean`
  Whether to search chunks with extracted table of content. Defaults to `False`. Before enabling this, ensure you have enabled `TOC_Enhance` and successfully extracted table of contents for the specified datasets. See [here](https://ragflow.io/docs/dev/enable_table_of_contents) for details.
- `"rerank_id"`: (*Body parameter*), `string`
  The ID of the rerank model.
- `"keyword"`: (*Body parameter*), `boolean`
  Indicates whether to enable keyword-based matching:
  - `true`: Enable keyword-based matching.
  - `false`: Disable keyword-based matching (default).
- `"highlight"`: (*Body parameter*), `boolean`
  Specifies whether to enable highlighting of matched terms in the results:
  - `true`: Enable highlighting of matched terms.
  - `false`: Disable highlighting of matched terms (default).
- `"cross_languages"`: (*Body parameter*) `list[string]`
  The languages that should be translated into, in order to achieve keywords retrievals in different languages.
- `"metadata_condition"`: (*Body parameter*), `object`
  The metadata condition used for filtering chunks:
  - `"logic"`: (*Body parameter*), `string`
    - `"and"`: Return only results that satisfy *every* condition (default).
    - `"or"`: Return results that satisfy *any* condition.
  - `"conditions"`: (*Body parameter*), `array`
    A list of metadata filter conditions.
    - `"name"`: `string` - The metadata field name to filter by, e.g., `"author"`, `"company"`, `"url"`. Ensure this parameter before use.
    - `comparison_operator`: `string` - The comparison operator. Can be one of:
      - `"contains"`
      - `"not contains"`
      - `"start with"`
      - `"empty"`
      - `"not empty"`
      - `"="`
      - `"≠"`
      - `">"`
      - `"<"`
      - `"≥"`
      - `"≤"`
    - `"value"`: `string` - The value to compare.

#### Response

Success:

```json
{
    "code": 0,
    "data": {
        "chunks": [
            {
                "content": "ragflow content",
                "content_ltks": "ragflow content",
                "document_id": "5c5999ec7be811ef9cab0242ac120005",
                "document_keyword": "1.txt",
                "highlight": "<em>ragflow</em> content",
                "id": "d78435d142bd5cf6704da62c778795c5",
                "image_id": "",
                "important_keywords": [
                    ""
                ],
                "tag_kwd": [],
                "kb_id": "c7ee74067a2c11efb21c0242ac120006",
                "positions": [
                    ""
                ],
                "similarity": 0.9669436601210759,
                "term_similarity": 1.0,
                "vector_similarity": 0.8898122004035864
            }
        ],
        "doc_aggs": [
            {
                "count": 1,
                "doc_id": "5c5999ec7be811ef9cab0242ac120005",
                "doc_name": "1.txt"
            }
        ],
        "total": 1
    }
}
```

Failure:

```json
{
    "code": 102,
    "message": "`datasets` is required."
}
```

---

## System

---

### Check system health

**GET** `/api/v1/system/healthz`

Check the health status of RAGFlow's dependencies (database, Redis, document engine, object storage).

:::caution DEPRECATED
`GET /v1/system/healthz` is deprecated. Use this endpoint instead.
:::

#### Request

- Method: GET
- URL: `/api/v1/system/healthz`
- Headers:
  - 'Content-Type: application/json'
  (no Authorization required)

##### Request example

```bash
curl --request GET
     --url http://{address}/api/v1/system/healthz
     --header 'Content-Type: application/json'
```

##### Request parameters

- `address`: (*Path parameter*), string
  The host and port of the backend service (e.g., `localhost:7897`).

---

#### Responses

- **200 OK** – All services healthy

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "db": "ok",
  "redis": "ok",
  "doc_engine": "ok",
  "storage": "ok",
  "status": "ok"
}
```

- **500 Internal Server Error** – At least one service unhealthy

```http
HTTP/1.1 500 INTERNAL SERVER ERROR
Content-Type: application/json

{
  "db": "ok",
  "redis": "nok",
  "doc_engine": "ok",
  "storage": "ok",
  "status": "nok",
  "_meta": {
    "redis": {
      "elapsed": "5.2",
      "error": "Lost connection!"
    }
  }
}
```

Explanation:

- Each service is reported as "ok" or "nok".
- The top-level `status` reflects overall health.
- If any service is "nok", detailed error info appears in `_meta`.

---
