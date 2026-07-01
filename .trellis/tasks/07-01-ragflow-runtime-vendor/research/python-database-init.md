# Research: Python database init (RAGFlow vendor)

- **Query**: Python database init in `api/db/db_models.py`, `common/settings.py` — Peewee connection, mysql vs postgres branches
- **Scope**: internal
- **Date**: 2026-07-01

## Findings

### Files Found

| File Path | Description |
|---|---|
| `services/knowledge/vendor/ragflow-runtime/common/settings.py` | `DATABASE_TYPE`, `DATABASE` from env + YAML |
| `services/knowledge/vendor/ragflow-runtime/common/config_utils.py` | `decrypt_database_config()` — loads YAML section by name |
| `services/knowledge/vendor/ragflow-runtime/api/db/db_models.py` | Peewee models, connection pool, locks, migrations |
| `services/knowledge/vendor/ragflow-runtime/api/db/init_data.py` | Calls `init_database_tables()` at startup |
| `services/knowledge/vendor/ragflow-runtime/api/ragflow_server.py` | Server startup invokes `init_database_tables()` |
| `services/knowledge/vendor/ragflow-runtime/docker/entrypoint.sh` | Boot: `init_database_tables()` via Python |
| `services/knowledge/vendor/ragflow-runtime/pyproject.toml` | `psycopg2-binary>=2.9.11` dependency present |

### Key Functions / Flow

#### Settings — `common/settings.py:69-70`, `182-185`

```python
DATABASE_TYPE = os.getenv("DB_TYPE", "mysql")
DATABASE = decrypt_database_config(name=DATABASE_TYPE)
```

- **`DB_TYPE` env must match YAML section name** (e.g. `postgres` → `postgres:` block in `service_conf.yaml`)
- Re-read on `init_settings()` call

#### Config load — `common/config_utils.py:139-144`

```python
def decrypt_database_config(database=None, passwd_key="password", name="database"):
    if not database:
        database = get_base_config(name, {})
    database[passwd_key] = decrypt_database_password(database[passwd_key])
    return database
```

- Expects keys: `name`, `user`, `password`, `host`, `port`, plus pool options (`max_connections`, `stale_timeout`)

#### Connection factory — `api/db/db_models.py:484-511`

```python
class PooledDatabase(Enum):
    MYSQL = RetryingPooledMySQLDatabase
    OCEANBASE = RetryingPooledOceanBaseDatabase
    POSTGRES = RetryingPooledPostgresqlDatabase

DB = BaseDataBase().database_connection  # PooledDatabase[DATABASE_TYPE.upper()]
```

- **`DATABASE_TYPE.upper()` must be `POSTGRES`** (not `POSTGRESQL`) to select postgres pool
- Peewee `PostgresqlDatabase` kwargs: `database=name`, `user`, `password`, `host`, `port`

#### Table init — `init_database_tables()` — `db_models.py:672-696`

- Discovers all `DataBaseModel` subclasses via `inspect.getmembers`
- `obj.create_table(safe=True)` if table missing
- Calls `migrate_db()` for incremental schema changes
- Wrapped with `@DB.lock("init_database_tables", 60)` — dialect-specific lock

#### Migrations — `migrate_db()` — `db_models.py:1327-1398`

- Uses `DatabaseMigrator[DATABASE_TYPE.upper()]` → `MySQLMigrator` or `PostgresqlMigrator`
- Most steps use playhouse `migrate()` (portable)
- **Dialect branches** in:
  - `migrate_add_unique_email()` — PG uses `pg_indexes`; MySQL uses `information_schema.statistics` + backtick DDL
  - `update_tenant_llm_to_id_primary_key()` — separate `_mysql` and `_postgres` implementations

### MySQL vs PostgreSQL Branches (existing)

| Feature | MySQL | PostgreSQL |
|---|---|---|
| Text columns | `LongTextField` → `LONGTEXT` | `LongTextField` → `TEXT` (`TextFieldType.POSTGRES`) |
| Connection pool | `RetryingPooledMySQLDatabase` | `RetryingPooledPostgresqlDatabase` |
| Migrator | `MySQLMigrator` | `PostgresqlMigrator` |
| Init lock | `GET_LOCK` / `RELEASE_LOCK` | `pg_try_advisory_lock` / `pg_advisory_unlock` |
| `tenant_llm` PK migration | User vars + `AUTO_INCREMENT` | `ROW_NUMBER()`, sequence `tenant_llm_id_seq` |
| Unique email migration | `information_schema.statistics`, `` `user` `` | `pg_indexes`, unquoted identifiers |

### Commented Config (ready to enable)

`conf/service_conf.yaml:60-67` and `docker/service_conf.yaml.template:68-75`:

```yaml
# postgres:
#   name: 'rag_flow'
#   user: 'rag_flow'
#   password: 'infini_rag_flow'
#   host: 'postgres'
#   port: 5432
```

Compose Phase 2 target (`services/knowledge/runtime/service_conf.compose.yaml:8-14`):

```yaml
# postgres:
#   name: knowledge_system
#   user: knowledge_app
#   password: knowledge_app_dev
#   host: postgres
#   port: 5432
```

### Startup entrypoints

| Entry | Calls |
|---|---|
| `api/ragflow_server.py` | `init_database_tables()` |
| `docker/entrypoint.sh:222` | `init_database_tables()` |
| `docker/launch_backend_service.sh:142` | `init_database_tables()` |
| `api/db/init_data.py` | `init_database_tables()` + seed `system_settings` |

## Caveats / Not Found

- **`DB_TYPE=postgres` is documented in config but not tested in Go path** — Python side is largely ready; Go server/ingestor still MySQL-only
- **`get_mysql_status()`** in `api/utils/health_utils.py:219` runs `SHOW PROCESSLIST` — MySQL-only diagnostic (may be unused in hot path)
- **`chunk_feedback_service.py`** does not use metadata DB — updates doc engine only
- No Python models for Go-only tables: `license`, `time_records`, `ingestion_task*`
