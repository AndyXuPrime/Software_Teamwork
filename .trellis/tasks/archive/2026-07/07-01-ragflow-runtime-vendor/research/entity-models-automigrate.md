# Research: Entity/table models requiring auto-migrate

- **Query**: List of entity/table models that need auto-migrate for knowledge_system DB
- **Scope**: internal
- **Date**: 2026-07-01

## Findings

### Go AutoMigrate list — `internal/dao/database.go:114-142`

| # | Go Entity | Table Name | Python Peewee Model | Notes |
|---|---|---|---|---|
| 1 | `entity.User` | `user` | `User` | PG reserved word |
| 2 | `entity.Tenant` | `tenant` | `Tenant` | |
| 3 | `entity.UserTenant` | `user_tenant` | `UserTenant` | |
| 4 | `entity.File` | `file` | `File` | |
| 5 | `entity.File2Document` | `file2document` | `File2Document` | |
| 6 | `entity.TenantLLM` | `tenant_llm` | `TenantLLM` | Complex PK migration |
| 7 | `entity.Task` | `task` | `Task` | Python: no explicit `Meta.db_table` → defaults to `task` |
| 8 | `entity.APIToken` | `api_token` | `APIToken` | Composite PK |
| 9 | `entity.Knowledgebase` | `knowledgebase` | `Knowledgebase` | Not same as legacy `knowledge_bases` |
| 10 | `entity.InvitationCode` | `invitation_code` | `InvitationCode` | |
| 11 | `entity.Document` | `document` | `Document` | Not same as legacy `knowledge_documents` |
| 12 | `entity.LLMFactories` | `llm_factories` | `LLMFactories` | |
| 13 | `entity.LLM` | `llm` | `LLM` | Composite PK `(fid, llm_name)` |
| 14 | `entity.SystemSettings` | `system_settings` | `SystemSettings` | Seeded from `conf/system_settings.json` |
| 15 | `entity.MCPServer` | `mcp_server` | `MCPServer` | |
| 16 | `entity.PipelineOperationLog` | `pipeline_operation_log` | `PipelineOperationLog` | |
| 17 | `entity.TimeRecord` | `time_records` | **None** | Go-only |
| 18 | `entity.License` | `license` | **None** | Go-only |
| 19 | `entity.TenantModelInstance` | `tenant_model_instance` | `TenantModelInstance` | |
| 20 | `entity.TenantModel` | `tenant_model` | `TenantModel` | |
| 21 | `entity.TenantModelGroupMapping` | `tenant_model_group_mapping` | `TenantModelGroupMapping` | Composite PK |
| 22 | `entity.TenantModelProvider` | `tenant_model_provider` | `TenantModelProvider` | |
| 23 | `entity.TenantModelGroup` | `tenant_model_group` | `TenantModelGroup` | |
| 24 | `entity.IngestionTask` | `ingestion_task` | **None** | Go-only |
| 25 | `entity.IngestionTaskLog` | `ingestion_task_log` | **None** | Go-only |
| 26 | `entity.IngestionTasklet` | `ingestion_tasklet` | **None** | Go-only |
| 27 | `entity.IngestionTaskletLog` | `ingestion_tasklet_log` | **None** | Go-only |

**Total: 27 Go models, 23 Python models, 6 Go-only tables**

### Python-only discovery — `init_database_tables()`

Python auto-discovers all `DataBaseModel` subclasses in `db_models.py` (same 23 tables as above minus Go-only 6).

Classes confirmed in `api/db/db_models.py`:

`User`, `Tenant`, `UserTenant`, `InvitationCode`, `LLMFactories`, `LLM`, `TenantLLM`, `Knowledgebase`, `Document`, `File`, `File2Document`, `Task`, `APIToken`, `MCPServer`, `PipelineOperationLog`, `SystemSettings`, `TenantModelProvider`, `TenantModelInstance`, `TenantModel`, `TenantModelGroup`, `TenantModelGroupMapping`

### Init order at runtime

1. **Python** (`init_database_tables`): create missing tables → `migrate_db()` incremental alters
2. **Go** (`InitDB`): `AutoMigrate` all 27 models → `RunMigrations()` manual MySQL DDL

Both stacks may run against the same DB; `autoMigrateSafely()` explicitly tolerates duplicate index errors from Python-created schema.

### Column type hotspots (cross-dialect)

| Column pattern | Go tag | Python field | PG note |
|---|---|---|---|
| Large JSON/text blobs | `type:longtext` | `LongTextField` / `JSONField` | PG: `text` |
| Parser config | Go: mixed `json` and `longtext` | `JSONField` | Align types |
| Timestamps | `BaseModel` bigint + datetime pairs | `BigIntegerField` + `DateTimeField` | Compatible |
| Boolean flags | Go `bool`, Python `CharField(1)` for some | Mixed representation | Existing parity issue |

### Tables NOT in vendor schema (legacy Knowledge service)

From `services/knowledge/migrations/` — separate product schema:

- `knowledge_bases`, `knowledge_documents`, `processing_jobs`, `document_chunks`, `parser_configs`, `parser_config_audits`

These coexist in `knowledge_system` DB today via goose migrations but are **orthogonal** to RAGFlow vendor metadata tables.

## Caveats / Not Found

- **`Task` Python model** lacks explicit `class Meta: db_table` — Peewee default table name is `task` (lowercase class name)
- **No `chunk_feedback` DB table** — feature uses doc engine fields only
- Schema collision risk if legacy and vendor tables share one PostgreSQL database without migration plan
