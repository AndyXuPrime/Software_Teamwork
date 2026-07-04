# Design

## Boundaries

The durable fix belongs in `services/knowledge/internal/adapter`. Runtime `Document.type` is not the public `DocumentSummary.contentType` contract, so the adapter must translate runtime metadata into the Gateway-facing contract.

The frontend remains a defensive consumer. It should not require backend bugs to be fixed before showing useful file labels, but it should still prefer API `contentType` when the value is valid.

## Backend Data Flow

Current flow:

1. Knowledge runtime returns document maps through `vendorclient.ListDocuments`.
2. `documentFromVendor` maps each raw document map into `documentSummary`.
3. Gateway exposes `documentSummary.contentType` to the frontend.

New flow:

1. `documentFromVendor` calls a helper such as `normalizedDocumentContentType(raw)`.
2. The helper chooses the first reliable source:
   - real MIME fields: `content_type`, `contentType`, `mime_type`, `mimeType`
   - filename/suffix fields: `name`, `filename`, `suffix`
   - conservative runtime type fallback only for extension-like common values such as `pdf`, `docx`, `pptx`, `csv`, `png`
3. Broad runtime type `doc` is never returned directly as a MIME content type. Without a usable filename/suffix or real MIME field, the adapter omits `contentType`.

## MIME Mapping

Use a small explicit extension map for stable behavior across environments. The scope is intentionally limited to common user-uploaded files:

- `.pdf` -> `application/pdf`
- `.txt` -> `text/plain`
- `.md`, `.markdown`, `.mdx` -> `text/markdown`
- `.doc` -> `application/msword`
- `.docx` -> `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- `.ppt` -> `application/vnd.ms-powerpoint`
- `.pptx` -> `application/vnd.openxmlformats-officedocument.presentationml.presentation`
- `.xls` -> `application/vnd.ms-excel`
- `.xlsx` -> `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `.csv` -> `text/csv`
- common images -> image MIME types

Do not add a general `mime.TypeByExtension` fallback or code-file MIME catalogue for this bug. Unknown extensions remain unknown.

## Upload Forwarding

`vendorclient.UploadDocument` already accepts a `contentType` argument, but `CreateFormFile` does not preserve it. Replace it with `CreatePart` and set:

- `Content-Disposition: form-data; name="file"; filename="<escaped>"`
- `Content-Type: <contentType>` when non-empty

If `contentType` is empty or the generic `application/octet-stream` in adapter upload handling, infer one from the filename before forwarding when the filename is in the common-type allowlist.

## Frontend Display

Add or adjust a helper near the document table:

- Normalize valid MIME values into display labels.
- Treat broad legacy values such as `doc` as insufficient when a filename extension exists.
- Fall back to filename extension for display.
- Fall back to the raw content type only when neither MIME nor filename gives a useful label.

This keeps the UI correct against old local data while preserving the backend contract fix.

## Compatibility

No API schema change is required. Existing clients reading `contentType` receive a better value for the same field.

Runtime database rows do not need migration because the adapter derives MIME at read time from already available `name`/`suffix`/runtime fields.

## Rollback

The backend change is localized to adapter mapping and vendor upload multipart construction. If issues appear, revert the helper usage and tests. The frontend fallback is display-only and can be reverted independently.
