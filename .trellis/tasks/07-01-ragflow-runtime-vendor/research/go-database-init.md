# Research: Go database init (RAGFlow vendor)

- **Query**: Go database init in `internal/dao/database.go`, `internal/server/config.go` — driver/DSN build, MySQL-specific code
- **Scope**: internal
- **Date**: 2026-07-01

## Findings

### Files Found

| File Path | Description |
|---|---|
| `services/knowledge/vendor/ragflow-runtime/internal/dao/database.go` | `InitDB()`, `autoMigrateSafely()`, AutoMigrate model list |
| `services/knowledge/vendor/ragflow-runtime/internal/dao/migration.go` | Manual MySQL-only migrations via raw SQL |
| `services/knowledge/vendor/ragflow-runtime/internal/server/config.go` | `DatabaseConfig`, YAML/env loading, MySQL-only driver switch |
| `services/knowledge/vendor/ragflow-runtime/go.mod` | `gorm.io/driver/mysql v1.5.2`; `github.com/lib/pq v1.10.9` present but unused for GORM |
| `services/knowledge/vendor/ragflow-runtime/conf/service_conf.yaml` | Active `mysql:` block; commented `# postgres:` block |
| `services/knowledge/runtime/service_conf.compose.yaml` | Phase 2 target: `knowledge_system` on compose postgres |

### Key Functions

#### `InitDB()` — `internal/dao/database.go`

- Reads `server.GetConfig().Database`
- Builds **MySQL-only DSN**:
  ```go
  dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", ...)
  ```
- Opens with `gorm.Open(mysql.Open(dsn), ...)`
- No branch on `Database.Driver`; always MySQL
- Calls `autoMigrateSafely()` for 27 entity types, then `RunMigrations()`

#### `autoMigrateSafely()` — `internal/dao/database.go`

- Wraps `db.AutoMigrate(model)`
- Ignores errors matching **MySQL error codes only**: 1061, 1060, 1050, 1091, 1138
- No PostgreSQL duplicate-object error handling

#### `DatabaseConfig` — `internal/server/config.go:86-95`

```go
type DatabaseConfig struct {
    Driver   string `mapstructure:"driver"` // mysql
    Host, Port, Database, Username, Password, Charset
}
```

#### `FromEnvironments()` — `internal/server/config.go:419-431`

- Reads `DB_TYPE` env var
- Accepts **`mysql` only** (or empty → default mysql)
- Any other value: `return fmt.Errorf("invalid database type: %s", databaseType)`

#### `FromConfigFile()` — `internal/server/config.go:530-544`

- If `Database.Host` empty, maps **`mysql:` YAML section** into `DatabaseConfig`
- Sets `Driver = "mysql"`, `Charset = "utf8mb4"`
- **No mapping from commented `postgres:` section**

#### `convertServiceConfToAdminFormat()` — `internal/server/config.go:333-352`

- Admin export treats DB as `meta_type: "mysql"` only

### Code Patterns

| Pattern | Location | PostgreSQL impact |
|---|---|---|
| Hardcoded MySQL DSN + driver | `database.go:72-91` | Must add postgres DSN branch + `gorm.io/driver/postgres` |
| MySQL error swallowing | `database.go:222-247` | Need PG equivalents (`already exists`, `42701`, etc.) |
| `INFORMATION_SCHEMA` + `AUTO_INCREMENT` | `migration.go` entire file | All manual migrations are MySQL dialect |
| `MODIFY COLUMN ... LONGTEXT` | `migration.go:202-210` | Invalid on PostgreSQL |
| `ADD UNIQUE INDEX` syntax | `migration.go:134,176` | PG prefers `CREATE UNIQUE INDEX` |
| `DATE_FORMAT()` in SELECT | `internal/dao/user_tenant.go:149` | Must use `TO_CHAR()` or Go-side formatting |
| GORM `type:longtext` tags | Multiple `internal/entity/*.go` | PG has no `LONGTEXT`; use `text` |
| Reserved table name `user` | `internal/entity/user.go:45` | GORM/Peewee must quote; verify on PG |

### Related Specs

- `services/knowledge/runtime/README.md` — Phase 2: metadata target is PostgreSQL `knowledge_system`
- `deploy/postgres/init/001-create-databases.sql` — `knowledge_system` DB owned by `knowledge_app`

## Caveats / Not Found

- **`gorm.io/driver/postgres` not in go.mod** — only `gorm.io/driver/mysql`
- **`lib/pq` is a transitive/direct dep** but not wired into `InitDB`
- Go-only AutoMigrate models (no Python Peewee counterpart): `License`, `TimeRecord`, `IngestionTask*`, `IngestionTasklet*` (4 tables)
- Project `services/knowledge/migrations/*.sql` defines **legacy Knowledge schema** (`knowledge_bases`, etc.), **not** RAGFlow vendor tables — separate schema namespace from vendor `user`/`knowledgebase`/`document`
