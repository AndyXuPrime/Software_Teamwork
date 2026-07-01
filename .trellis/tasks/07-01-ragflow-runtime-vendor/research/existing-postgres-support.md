# Research: Existing PostgreSQL support in RAGFlow vendor

- **Query**: Any existing postgres support (commented config, postgres migrations, dialect switches)
- **Scope**: internal
- **Date**: 2026-07-01

## Findings

### Python metadata DB — substantial existing support

The Python Peewee layer already implements a **dual-dialect design** keyed off `DB_TYPE`:

| Component | File | Status |
|---|---|---|
| Postgres connection pool | `api/db/db_models.py:333-401` | Implemented |
| Postgres migrator | `api/db/db_models.py:493` | Implemented |
| Postgres advisory locks | `api/db/db_models.py:555-591` | Implemented |
| `tenant_llm` PK migration (PG) | `api/db/db_models.py:1251-1324` | Implemented |
| Unique email migration (PG branch) | `api/db/db_models.py:1118-1128` | Implemented |
| `LongTextField` → TEXT on PG | `api/db/db_models.py:66-73` | Implemented |
| `psycopg2-binary` dependency | `pyproject.toml:55` | Present |

**Activation path (Python only):**

1. Set `DB_TYPE=postgres`
2. Uncomment/configure `postgres:` section in `service_conf.yaml`
3. Ensure database exists (e.g. `knowledge_system` on compose postgres)

### Go metadata DB — no PostgreSQL support

| Component | Status |
|---|---|
| `gorm.io/driver/postgres` | **Not in go.mod** |
| `InitDB()` driver switch | **MySQL only** |
| `FromEnvironments()` `DB_TYPE` | **Rejects non-mysql** |
| `FromConfigFile()` postgres mapping | **Not implemented** |
| `RunMigrations()` | **MySQL INFORMATION_SCHEMA + DDL only** |

### Config artifacts (commented, not active)

| File | Content |
|---|---|
| `conf/service_conf.yaml:60-67` | Commented `postgres:` block |
| `docker/service_conf.yaml.template:68-75` | Commented `postgres:` with env var placeholders |
| `services/knowledge/runtime/service_conf.compose.yaml:8-14` | Phase 2 target for `knowledge_system` |

### PostgreSQL references unrelated to metadata DB

These use PostgreSQL protocol for **other subsystems**, not RAGFlow metadata:

| File | Purpose |
|---|---|
| `infinity.postgres_port` in `conf/service_conf.yaml:33` | Infinity doc engine PG wire protocol |
| `internal/engine/infinity/sql.go:286-323` | Infinity client psql host/port |
| `common/doc_store/infinity_conn_pool.py` | Infinity connection URI |
| `common/data_source/rdbms_connector.py` | External data import from PG/MySQL/MSSQL |
| `Dockerfile:49` | Installs `postgresql-client` CLI |

### Project-level PostgreSQL (parent repo)

| File | Notes |
|---|---|
| `deploy/docker-compose.yml:291` | `DATABASE_URL=postgres://knowledge_app@postgres:5432/knowledge_system` for Knowledge service |
| `deploy/postgres/init/001-create-databases.sql` | Creates `knowledge_system` DB |
| `services/knowledge/migrations/0001_*.sql`, `0002_*.sql` | **Legacy Knowledge schema** (goose), not RAGFlow vendor tables |

### Dialect switch env var contract

| Variable | Values | Consumers |
|---|---|---|
| `DB_TYPE` | `mysql` (default), **`postgres`** for Python | `common/settings.py`, Go `FromEnvironments()` (mysql only) |

**Important:** Python enum key is `POSTGRES` — use `DB_TYPE=postgres`, not `postgresql`.

## Caveats / Not Found

- **No dedicated postgres migration SQL files** for RAGFlow vendor schema under vendor tree — schema created via Peewee `create_table` + `migrate_db()` at runtime
- **No goose/flyway migrations** for vendor tables in `services/knowledge/migrations/` — those migrations target different table names
- **Coexistence question unresolved in code:** legacy `knowledge_bases` tables vs vendor `knowledgebase` table would share `knowledge_system` DB unless namespaced/separated
