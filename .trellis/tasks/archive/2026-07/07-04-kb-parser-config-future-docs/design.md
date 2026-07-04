# Design

## Boundary

Knowledge remains the owner of knowledge base parser configuration. Gateway only proxies public requests, while `services/knowledge-runtime` remains the implementation boundary for parser model references, document upload, parsing, chunking, embedding, and indexing.

The Go Knowledge adapter will not import runtime-specific parser libraries. It will only resolve parser config metadata from its own `parser_configs` table and forward sanitized runtime config plus protected credential payloads over the existing runtime HTTP API.

## API Shape

Extend the existing knowledge base update request with an optional `parserConfigId` field:

```json
{
  "parserConfigId": "parser_config_paddleocr_cloud_default"
}
```

This keeps the operation resource-oriented: `PATCH /knowledge-bases/{knowledgeBaseId}` updates the knowledge base resource. The field is optional and compatible with existing request bodies.

## Data Flow

1. Gateway receives `PATCH /api/v1/knowledge-bases/{knowledgeBaseId}` and forwards the body to Knowledge as it does for existing knowledge base updates.
2. Knowledge adapter decodes `parserConfigId` into `updateKnowledgeBaseRequest`.
3. When `parserConfigId` is present:
   - resolve the parser config snapshot through the existing parser config service,
   - convert the snapshot using `ragflowParserConfigFromSnapshot`,
   - pass sanitized `parser_config` and protected `parser_config_credentials` to runtime.
4. Runtime dataset update consumes `parser_config_credentials` before persistence, using the same PaddleOCR setup helper as dataset creation.
5. Runtime persists only the safe model reference in knowledge base `parser_config`.
6. Later uploads copy the updated knowledge base `parser_config` into new document rows.

## Compatibility

- Existing update requests with only `name`, `description`, `docType`, or `chunkStrategy` remain valid.
- `parserConfigId` is optional. Empty strings are rejected as validation errors when the field is present.
- `chunkStrategy` continues to mean explicit caller-supplied parser config overrides.
- If `parserConfigId` and `chunkStrategy` are both supplied, explicit `chunkStrategy` fields override non-sensitive fields from the resolved parser config.

## Runtime Change

Runtime creation already applies `parser_config_credentials`. Dataset update should do the same after request normalization and before `KnowledgebaseService.update_by_id`, so both creation and update produce the same persisted parser model reference.

## Security

- Do not place access tokens inside `parser_config`.
- Do not return access tokens in API responses.
- Do not log request bodies or credentials.
- Keep protected credentials under `parser_config_credentials`, and remove them before persistence.

## Rollback

Rollback is a code revert. Runtime data written by the feature is ordinary knowledge base `parser_config`; there is no schema migration. A knowledge base can be switched back to the builtin parser config through the same update path.
