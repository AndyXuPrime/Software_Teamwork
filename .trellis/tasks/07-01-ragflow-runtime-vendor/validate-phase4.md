# Phase 4 Validation (2026-07-01)

## Automated checks

| Check | Result | Notes |
| --- | --- | --- |
| `go test ./internal/adapter/...` | PASS | Contract tests including upload→parse |
| `go test ./internal/adapterconfig/...` | PASS | AutoStartIngestion env defaults |
| `go build ./cmd/adapter` | BLOCKED | pgx download timeout in sandbox (proxy.golang.org); not a code defect |
| `go test -tags=integration ...` | SKIP | Requires `KNOWLEDGE_VENDOR_INTEGRATION_URL` + `KNOWLEDGE_INTEGRATION_USER_ID` |

## Contract coverage

- `TestAdapterDocumentUploadStartsVendorIngestion` — fake vendor receives parse call after upload
- `TestAdapterDocumentUploadSkipsIngestionWhenDisabled` — `KNOWLEDGE_AUTO_START_INGESTION=false` skips parse
- Existing adapter RBAC/route tests unchanged and passing

## Live vendor E2E (manual)

Run when vendor Python API (:9380) and task executor are up:

```bash
KNOWLEDGE_VENDOR_INTEGRATION_URL=http://127.0.0.1:9380 \
KNOWLEDGE_INTEGRATION_USER_ID=usr_local_admin \
go test -tags=integration ./internal/adapter/... -run Integration -count=1
```

Gateway identity (`X-User-Id`) must match a vendor tenant user seeded in vendor PG metadata.

## Conclusion

Phase 4 adapter ingestion wiring is verified at contract-test level. Proceed to Phase 5 legacy cleanup.
