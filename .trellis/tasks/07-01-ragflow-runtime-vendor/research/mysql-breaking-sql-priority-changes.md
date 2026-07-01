# Research: MySQL-specific SQL and minimal PostgreSQL change list

- **Query**: MySQL-specific SQL/migrations that break on PostgreSQL; prioritized change list for minimal PG support on `knowledge_system` DB
- **Scope**: internal
- **Date**: 2026-07-01

## Findings

### MySQL-Specific SQL / Code That Breaks on PostgreSQL

#### Go — blocking

| Location | MySQL-specific construct | PG failure mode |
|---|---|---|
| `internal/dao/database.go:72-91` | `@tcp(...)`, `charset=utf8mb4`, `mysql.Open` | Cannot connect |
| `internal/dao/database.go:222-247` | Error codes 1061, 1060, 1050, 1091, 1138 | Unhandled duplicate-object errors |
| `internal/dao/migration.go:68-84` | `INFORMATION_SCHEMA.COLUMNS`, `EXTRA LIKE '%auto_increment%'` | Different catalog columns |
| `internal/dao/migration.go:111-122` | `AUTO_INCREMENT PRIMARY KEY`, `ADD COLUMN ... FIRST` | Invalid syntax |
| `internal/dao/migration.go:134,176` | `ADD UNIQUE INDEX idx_... (...)` | Use `CREATE UNIQUE INDEX` |
| `internal/dao/migration.go:202-210` | `MODIFY COLUMN ... LONGTEXT` | Use `ALTER COLUMN ... TYPE text` |
| `internal/dao/migration.go:216-224` | `MODIFY COLUMN ... DATETIME` | Use `timestamp` |
| `internal/dao/user_tenant.go:149` | `DATE_FORMAT(user.update_date, '%Y-%m-%dT%H:%i:%s')` | Use `TO_CHAR` or app-side format |
| `internal/server/config.go:422-430` | `DB_TYPE` rejects `postgres` | Startup error |
| `internal/server/config.go:533-543` | Maps only `mysql:` YAML section | PG config ignored |
| Multiple `internal/entity/*.go` | GORM `type:longtext` | Invalid PG type name |

#### Python — mostly handled; remaining gaps

| Location | Issue | Severity |
|---|---|---|
| `api/utils/health_utils.py:219-221` | `SHOW PROCESSLIST` | Low — diagnostic only |
| `db_models.py:1084-1090` | Duplicate column error code `1060` only | Low — PG errors caught generically in some paths |
| `db_models.go` N/A | `alter_db_add_column` MySQL-centric error codes | Low |
| `db_models.py:1148` | `` ALTER TABLE `user` DROP INDEX `` with backticks | N/A on PG branch (guarded) |

#### Not metadata DB (out of scope for Phase 2 PG port)

- `rag/utils/ob_conn.py` — OceanBase/MySQL doc engine SQL
- `rag/utils/opendal_conn.py` — `ON UPDATE CURRENT_TIMESTAMP` in opendal schema
- `common/data_source/rdbms_connector.py` — external connector; already supports PG

### Prioritized Change List (minimal PostgreSQL on `knowledge_system`)

#### P0 — Must have (both stacks can start)

1. **Config wiring**
   - Uncomment `postgres:` in `services/knowledge/runtime/service_conf.compose.yaml` (or runtime config) pointing to `knowledge_system` / `knowledge_app`
   - Set `DB_TYPE=postgres` for Python processes
   - Extend Go `FromConfigFile()` to map `postgres:` YAML → `DatabaseConfig` (mirror mysql block)
   - Extend Go `FromEnvironments()` to accept `DB_TYPE=postgres`

2. **Go driver + DSN**
   - Add `gorm.io/driver/postgres` to `go.mod`
   - Branch `InitDB()` on `cfg.Database.Driver`:
     - `postgres`: `host=... user=... password=... dbname=... port=... sslmode=disable TimeZone=Local`
     - `mysql`: existing DSN
   - Remove or gate `charset` for postgres

3. **Go entity tags**
   - Replace `type:longtext` with `type:text` (or dialect-agnostic omission) on ~15 entity fields
   - Verify `type:json` vs `longtext` for JSON columns (`parser_config`, etc.)

#### P1 — Migrations must not crash on fresh PG DB

4. **Go `RunMigrations()` dialect split**
   - Port patterns from Python `_update_tenant_llm_to_id_primary_key_postgres()` and `migrate_add_unique_email()` PG branch
   - Replace all `INFORMATION_SCHEMA` queries with PG `information_schema` / `pg_catalog` equivalents
   - Replace `MODIFY COLUMN`, `AUTO_INCREMENT`, `ADD UNIQUE INDEX` with PG DDL

5. **Go `autoMigrateSafely()`**
   - Add PG duplicate errors: `already exists`, SQLSTATE `42701`, `42P07`, etc.

6. **Go query portability**
   - Fix `DATE_FORMAT` in `internal/dao/user_tenant.go:149`

#### P2 — Python polish (mostly done)

7. **Enable postgres config + `DB_TYPE=postgres`** — Python path should work with existing code
8. **Verify `init_database_tables()` + `migrate_db()`** against empty `knowledge_system`
9. **Optional:** PG-safe replacement for `get_mysql_status()` or guard by `DATABASE_TYPE`

#### P3 — Schema coexistence / deployment

10. **Decide namespace** for vendor tables vs legacy goose tables in same `knowledge_system` DB (no collision today — different table names)
11. **Go-only tables** (`license`, `time_records`, `ingestion_task*`) — created only by Go AutoMigrate; ensure Python startup doesn't assume absence
12. **Reserved identifier `user`** — verify GORM AutoMigrate quotes correctly on PostgreSQL

### Suggested implementation order

```
Config (P0) → Go driver/DSN (P0) → Go entity types (P0)
    → Python smoke test (P2) with DB_TYPE=postgres
    → Go migration.go PG branch (P1)
    → Go autoMigrateSafely + DATE_FORMAT (P1)
    → Integration test both Python + Go against knowledge_system (P3)
```

### Key file paths for implementer

| Priority | Files |
|---|---|
| P0 | `internal/dao/database.go`, `internal/server/config.go`, `go.mod`, `service_conf.compose.yaml` |
| P1 | `internal/dao/migration.go`, `internal/dao/user_tenant.go`, `internal/entity/*.go` (longtext tags) |
| P2 | `common/settings.py` (verify only), `conf/service_conf.yaml`, docker entrypoint env |
| Reference | `api/db/db_models.py` (PG migration patterns already implemented) |

### External References

- [GORM PostgreSQL driver](https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL) — DSN format for `postgres.Open`
- [Peewee PostgresqlDatabase](http://docs.peewee-orm.com/en/latest/peewee/database.html#postgresql) — existing vendor usage

## Caveats / Not Found

- No automated test suite found that runs vendor metadata against PostgreSQL end-to-end
- **`DB_TYPE=postgresql`** would fail Python enum lookup (`KeyError: POSTGRESQL`) — must use `postgres`
- Fresh PG database has **no vendor seed migration SQL** — schema relies entirely on runtime `create_table` / `AutoMigrate`
