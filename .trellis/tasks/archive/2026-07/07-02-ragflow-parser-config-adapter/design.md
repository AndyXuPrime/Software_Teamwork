# Design: RAGFlow Parser Config Adapter

## Boundary

The implementation belongs in PR #440's `services/knowledge` adapter layer and
`deploy/docker-compose.yml`.

`services/knowledge` remains a Go contract adapter. It should translate the
project's admin parser-config contract into RAGFlow vendor request payloads, not
reintroduce parser execution. `services/knowledge-runtime` remains the runtime
that executes parsing.

## Data Flow

1. Gateway calls Knowledge `POST /internal/v1/knowledge-bases`.
2. Adapter reads caller context and request body.
3. If the body includes `chunkStrategy`, adapter maps it to vendor
   `parser_config`, preserving explicit values.
4. If no `chunkStrategy` is provided and parser-config service is configured,
   adapter resolves the effective parser config for the requested document type
   or generic content type.
5. Adapter converts the parser config to RAGFlow `parser_config`:
   - `builtin` -> `layout_recognize: "DeepDOC"`
   - `local_ocr` -> `layout_recognize: "PaddleOCR"`
   - `remote_compatible` -> configurable `layout_recognize`, default
     `"PaddleOCR"`
   - `tika` / `unstructured` -> `layout_recognize: "Plain Text"`
6. Adapter forwards the create/update request to RAGFlow runtime.
7. Runtime uploads/parses documents using the configured parser_config.

## Contract Notes

Existing public/admin parser-config shapes remain unchanged. The bridge is
best-effort for RAGFlow:

- Preserve original parser config details in a metadata object so admins can
  trace which config shaped the KB.
- Do not expose endpoint credentials or provider secrets through parser_config.
- Do not make old `endpointUrl` a runtime network call from Go. Runtime-specific
  remote providers are configured by `knowledge-runtime` OCR env vars.

## Compose

Pass OCR provider env vars to both runtime processes because API and worker may
instantiate model/provider config:

- `PADDLEOCR_BASE_URL`
- `PADDLEOCR_API_URL`
- `PADDLEOCR_ACCESS_TOKEN`
- `PADDLEOCR_ALGORITHM`
- `MINERU_APISERVER`
- `MINERU_OUTPUT_DIR`
- `MINERU_BACKEND`
- `MINERU_SERVER_URL`
- `MINERU_DELETE_OUTPUT`
- `OPENDATALOADER_APISERVER`

## Compatibility

If `DATABASE_URL` is omitted, parser-config admin routes still return the
existing dependency error. Knowledge base creation should continue to work by
falling back to vendor defaults instead of failing solely because parser-config
storage is absent.

## Rollback

The adapter mapping is local to request payload construction. Rollback is to
remove the default parser_config merge and runtime OCR env passthrough, returning
PR #440 to vendor-default parsing.
