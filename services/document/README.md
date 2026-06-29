# Document Service

`document` owns report types, templates, materials, reports, outlines, sections,
report jobs, job attempts, events, generated file metadata, statistics, and
operation logs.

The current implementation provides the service/data baseline, implemented
report type/template/material metadata APIs, and scaffold coverage for the
remaining active report-generation contract. It does not implement AI generation,
DOCX export execution, MCP tools, report workflow orchestration, or AI Gateway
calls yet.

## Local Configuration

Required environment variables:

| Variable | Example | Purpose |
| --- | --- | --- |
| `DOCUMENT_DATABASE_URL` | `postgres://document:document@localhost:5432/document?sslmode=disable` | PostgreSQL connection string. |
| `DOCUMENT_REDIS_ADDR` | `localhost:6379` | Redis/asynq queue endpoint. Redis is not the durable job state authority. |
| `DOCUMENT_FILE_SERVICE_URL` | `http://localhost:8082` | Internal file service base URL for later template/material/report-file bytes. |
| `DOCUMENT_AI_GATEWAY_URL` | `http://localhost:8086` | Internal AI Gateway base URL for later generation calls. |
| `DOCUMENT_AI_GATEWAY_PROFILE_ID` | `default-chat` | AI Gateway profile reference used by report settings/default generation. |

Optional variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `DOCUMENT_HTTP_ADDR` | `:8085` | HTTP listen address. |
| `DOCUMENT_PANDOC_PATH` | `pandoc` | DOCX toolchain command path reserved for worker usage. |
| `DOCUMENT_LIBREOFFICE_PATH` | `soffice` | LibreOffice command path reserved for worker usage. |
| `DOCUMENT_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout. |

## Run

```powershell
$env:DOCUMENT_DATABASE_URL = "postgres://document:document@localhost:5432/document?sslmode=disable"
$env:DOCUMENT_REDIS_ADDR = "localhost:6379"
$env:DOCUMENT_FILE_SERVICE_URL = "http://localhost:8082"
$env:DOCUMENT_AI_GATEWAY_URL = "http://localhost:8086"
$env:DOCUMENT_AI_GATEWAY_PROFILE_ID = "default-chat"
go run ./cmd/server
```

Operational routes:

```text
GET /healthz
GET /readyz
```

Both JSON responses use the project envelope: `{ "data": ..., "requestId": "..." }`.
The service-local operational contract is documented in [`api/openapi.yaml`](api/openapi.yaml).

## Active Report Route Coverage

Gateway exposes these document-owned report routes under `/api/v1`. The service
local paths below omit that prefix. Implemented routes call the document service
layer. Scaffold routes are registered and return the standard error envelope with
`error.code=not_implemented` and HTTP `501` until their business workflows land.

| Method | Local path | Operation ID | Status |
| --- | --- | --- | --- |
| `GET` | `/report-types` | `listReportTypes` | Implemented |
| `GET` | `/report-templates` | `listReportTemplates` | Implemented |
| `POST` | `/report-templates` | `createReportTemplate` | Implemented |
| `GET` | `/report-templates/{reportTemplateId}` | `getReportTemplate` | Implemented |
| `PATCH` | `/report-templates/{reportTemplateId}` | `updateReportTemplate` | Implemented |
| `DELETE` | `/report-templates/{reportTemplateId}` | `deleteReportTemplate` | Implemented |
| `GET` | `/report-templates/{reportTemplateId}/structure` | `getReportTemplateStructure` | Implemented |
| `PATCH` | `/report-templates/{reportTemplateId}/structure` | `updateReportTemplateStructure` | Implemented |
| `GET` | `/report-materials` | `listReportMaterials` | Implemented |
| `POST` | `/report-materials` | `createReportMaterial` | Implemented |
| `GET` | `/report-materials/{materialId}` | `getReportMaterial` | Implemented |
| `DELETE` | `/report-materials/{materialId}` | `deleteReportMaterial` | Implemented |
| `GET` | `/reports` | `listReports` | Scaffold |
| `POST` | `/reports` | `createReport` | Scaffold |
| `GET` | `/reports/{reportId}` | `getReport` | Scaffold |
| `PATCH` | `/reports/{reportId}` | `updateReport` | Scaffold |
| `DELETE` | `/reports/{reportId}` | `deleteReport` | Scaffold |
| `GET` | `/reports/{reportId}/outlines` | `listReportOutlines` | Scaffold |
| `POST` | `/reports/{reportId}/outlines` | `createReportOutline` | Scaffold |
| `GET` | `/reports/{reportId}/outlines/{outlineId}` | `getReportOutline` | Scaffold |
| `PATCH` | `/reports/{reportId}/outlines/{outlineId}` | `updateReportOutline` | Scaffold |
| `DELETE` | `/reports/{reportId}/outlines/{outlineId}/sections/{sectionId}` | `deleteReportOutlineSection` | Scaffold |
| `GET` | `/reports/{reportId}/sections` | `listReportSections` | Scaffold |
| `POST` | `/reports/{reportId}/sections` | `createReportSection` | Scaffold |
| `GET` | `/reports/{reportId}/sections/{sectionId}` | `getReportSection` | Scaffold |
| `PATCH` | `/reports/{reportId}/sections/{sectionId}` | `updateReportSection` | Scaffold |
| `GET` | `/reports/{reportId}/sections/{sectionId}/versions` | `listReportSectionVersions` | Scaffold |
| `POST` | `/reports/{reportId}/sections/{sectionId}/versions` | `createReportSectionVersion` | Scaffold |
| `GET` | `/reports/{reportId}/jobs` | `listReportJobs` | Scaffold |
| `POST` | `/reports/{reportId}/jobs` | `createReportJob` | Scaffold |
| `GET` | `/report-jobs/{jobId}` | `getReportJob` | Scaffold |
| `GET` | `/report-jobs/{jobId}/attempts` | `listReportJobAttempts` | Scaffold |
| `POST` | `/report-jobs/{jobId}/attempts` | `createReportJobAttempt` | Scaffold |
| `GET` | `/reports/{reportId}/events` | `listReportEvents` | Scaffold |
| `GET` | `/report-files` | `listReportFiles` | Scaffold |
| `POST` | `/report-files` | `createReportFile` | Scaffold |
| `GET` | `/report-files/{reportFileId}` | `getReportFile` | Scaffold |
| `GET` | `/report-files/{reportFileId}/content` | `getReportFileContent` | Scaffold |
| `GET` | `/report-statistics/overview` | `getReportStatisticsOverview` | Scaffold |
| `GET` | `/report-statistics/daily` | `listDailyReportStatistics` | Scaffold |
| `GET` | `/report-operation-logs` | `listReportOperationLogs` | Scaffold |
| `GET` | `/report-settings` | `getReportSettings` | Scaffold |
| `PATCH` | `/report-settings` | `updateReportSettings` | Scaffold |

## Migrations

Migration files live in `migrations/` and are applied with the project-pinned `goose@v3.27.1` command.

```powershell
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$env:DOCUMENT_DATABASE_URL" up
```

The first migration creates the report generation tables and seeds the initial
report types:

- `summer_peak_inspection`
- `coal_inventory_audit`

`report_jobs`, `report_job_attempts`, and `report_events` are PostgreSQL
business-state tables. Redis/asynq should only carry queue payloads and task
execution coordination.

## SQLC

SQL queries live under `internal/repository/queries/`, and generated code lives
under `internal/repository/sqlc/`.

```powershell
sqlc generate
```

## Tests

```powershell
go test ./...
go build ./cmd/server
```

Repository integration tests are skipped unless `DOCUMENT_TEST_DATABASE_URL` is
set:

```powershell
$env:DOCUMENT_TEST_DATABASE_URL = "postgres://document:document@localhost:5432/document_test?sslmode=disable"
go test ./internal/repository
```
