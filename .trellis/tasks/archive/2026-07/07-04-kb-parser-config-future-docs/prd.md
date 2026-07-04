# Make knowledge base parser changes affect future documents

## Goal

Let an existing knowledge base change its parser configuration so future document uploads and future first-time parse runs use the selected parser. Already parsed historical documents must keep their existing chunks, indexes, and document-level parser snapshots unless a separate reparse feature is built later.

## Background

- The Knowledge adapter already resolves the default parser config when creating a new knowledge base. `handleCreateKnowledgeBase` calls `resolveCreateParserConfig`, then `buildCreateDatasetBody` forwards sanitized `parser_config` plus protected `parser_config_credentials` to the runtime in `services/knowledge/internal/adapter/handlers.go:61` and `services/knowledge/internal/adapter/map.go:383`.
- The Knowledge adapter update path currently builds only name, description, doc type, and raw chunk strategy updates. It does not resolve a parser config by ID or send protected parser credentials in `services/knowledge/internal/adapter/handlers.go:114` and `services/knowledge/internal/adapter/map.go:436`.
- The runtime dataset update service can merge `parser_config` into the persisted knowledge base config in `services/knowledge-runtime/api/apps/services/dataset_api_service.py:337`.
- New document upload copies the knowledge base parser config into the new document row. `FileService.upload_document` builds `merged_parser_config` from `kb.parser_config` and inserts it as document `parser_config` in `services/knowledge-runtime/api/db/services/file_service.py:445` and `services/knowledge-runtime/api/db/services/file_service.py:517`.
- Existing documents have their own parser snapshots. Updating the knowledge base parser config must not mutate already parsed documents or force a reparse.

## Requirements

- Add a Knowledge adapter API path for updating an existing knowledge base's future parser selection by referencing a parser config ID.
- Preserve the current knowledge base update behavior for name, description, doc type, and caller-provided chunk strategy.
- When a parser config ID is provided, resolve it through the existing parser config service, convert it with the same sanitization and credential separation used by knowledge base creation, and forward it to the runtime dataset update API.
- The runtime dataset update path must apply protected parser credentials the same way creation does, so PaddleOCR cloud credentials become a persisted runtime model reference and tokens are not stored in dataset `parser_config`.
- Existing document rows, chunks, and indexes must not be changed by this feature.
- The implementation must not add PaddleOCR, PaddlePaddle, OpenCV, CUDA, or provider SDK dependencies to the Go Knowledge process.
- Response and error handling must continue to use the existing Knowledge adapter envelope and app error patterns.

## Acceptance Criteria

- [ ] Updating an existing knowledge base with a parser config ID changes the runtime knowledge base `parser_config.layout_recognize` for future uploads.
- [ ] Uploading a new PDF after the knowledge base parser switch creates a new document whose `parser_config` follows the updated knowledge base parser config.
- [ ] Already parsed documents are not updated by the knowledge base parser switch.
- [ ] PaddleOCR access tokens are carried only in protected credential payloads and do not appear in persisted runtime dataset `parser_config`, logs, tests, or API responses.
- [ ] Existing update behavior for name, description, doc type, and chunk strategy remains compatible.
- [ ] Unit or contract tests cover parser config switch, token redaction, and no-op behavior when no parser config ID is supplied.

## Out Of Scope

- No historical document reparse button, API, queue, or batch migration.
- No automatic migration of existing documents from DeepDOC to PaddleOCR.
- No frontend UI for this slice unless a later task explicitly requests it.
- No provider billing or remote dashboard integration beyond the existing runtime PaddleOCR call path.

## Implementation Notes

- Prefer extending the existing knowledge base `PATCH` contract rather than adding an action-style endpoint.
- `parserConfigId` should be optional. When omitted, existing update requests should behave as they do today.
- If both `chunkStrategy` and `parserConfigId` are provided, the implementation should merge the resolved parser config first and then apply explicit `chunkStrategy` fields as caller overrides, preserving the current API's ability to customize chunking.
