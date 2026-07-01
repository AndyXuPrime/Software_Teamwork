package dao

import (
	"fmt"
	"ragflow/internal/common"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func runPostgresMigrations(db *gorm.DB) error {
	if err := renameColumnIfExists(db, "task", "process_duation", "process_duration"); err != nil {
		return fmt.Errorf("failed to rename task.process_duation: %w", err)
	}
	if err := renameColumnIfExists(db, "document", "process_duation", "process_duration"); err != nil {
		return fmt.Errorf("failed to rename document.process_duation: %w", err)
	}
	if err := migrateAddUniqueEmailPostgres(db); err != nil {
		return fmt.Errorf("failed to add unique index on user.email: %w", err)
	}
	if err := migrateTenantLLMPrimaryKeyPostgres(db); err != nil {
		return fmt.Errorf("failed to migrate tenant_llm primary key: %w", err)
	}

	common.Info("All PostgreSQL manual migrations completed successfully")
	return nil
}

func migrateAddUniqueEmailPostgres(db *gorm.DB) error {
	if !db.Migrator().HasTable("user") {
		return nil
	}

	var duplicateCount int64
	err := db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT email FROM "user" GROUP BY email HAVING COUNT(*) > 1
		) AS duplicates
	`).Scan(&duplicateCount).Error
	if err != nil {
		return err
	}
	if duplicateCount > 0 {
		common.Warn("Found duplicate emails in user table, cannot add unique index", zap.Int64("count", duplicateCount))
		return nil
	}

	common.Info("Ensuring unique index on user.email for PostgreSQL...")
	err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email_unique ON "user" (email)`).Error
	if err != nil && !isBenignMigrationError(err) {
		return fmt.Errorf("failed to add unique index on email: %w", err)
	}
	return nil
}

func migrateTenantLLMPrimaryKeyPostgres(db *gorm.DB) error {
	if !db.Migrator().HasTable("tenant_llm") {
		return nil
	}

	var idColumnExists int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_catalog = current_database()
		  AND table_name = 'tenant_llm'
		  AND column_name = 'id'
	`).Scan(&idColumnExists).Error
	if err != nil {
		return err
	}
	if idColumnExists > 0 {
		return nil
	}

	common.Info("Migrating tenant_llm to use ID primary key on PostgreSQL...")

	return db.Transaction(func(tx *gorm.DB) error {
		var tempIDExists int64
		tx.Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_catalog = current_database()
			  AND table_name = 'tenant_llm'
			  AND column_name = 'temp_id'
		`).Scan(&tempIDExists)
		if tempIDExists > 0 {
			if err := tx.Exec(`ALTER TABLE tenant_llm DROP COLUMN temp_id`).Error; err != nil && !isBenignMigrationError(err) {
				common.Warn("Failed to drop temp_id column", zap.Error(err))
			}
		}

		if err := tx.Exec(`ALTER TABLE tenant_llm ADD COLUMN temp_id INTEGER NULL`).Error; err != nil {
			return fmt.Errorf("failed to add temp_id column: %w", err)
		}
		if err := tx.Exec(`
			UPDATE tenant_llm
			SET temp_id = subq.rn
			FROM (
				SELECT ctid,
				       ROW_NUMBER() OVER (ORDER BY tenant_id, llm_factory, llm_name) AS rn
				FROM tenant_llm
			) AS subq
			WHERE tenant_llm.ctid = subq.ctid
		`).Error; err != nil {
			return fmt.Errorf("failed to populate temp_id: %w", err)
		}

		var pkName *string
		tx.Raw(`
			SELECT constraint_name
			FROM information_schema.table_constraints
			WHERE table_catalog = current_database()
			  AND table_name = 'tenant_llm'
			  AND constraint_type = 'PRIMARY KEY'
		`).Scan(&pkName)
		if pkName != nil && strings.TrimSpace(*pkName) != "" {
			if err := tx.Exec(fmt.Sprintf(`ALTER TABLE tenant_llm DROP CONSTRAINT "%s"`, *pkName)).Error; err != nil {
				return fmt.Errorf("failed to drop tenant_llm primary key: %w", err)
			}
		}

		statements := []string{
			`ALTER TABLE tenant_llm ALTER COLUMN temp_id SET NOT NULL`,
			`CREATE SEQUENCE IF NOT EXISTS tenant_llm_id_seq`,
			`SELECT setval('tenant_llm_id_seq', COALESCE((SELECT MAX(temp_id) FROM tenant_llm), 0))`,
			`ALTER TABLE tenant_llm ALTER COLUMN temp_id SET DEFAULT nextval('tenant_llm_id_seq')`,
			`ALTER SEQUENCE tenant_llm_id_seq OWNED BY tenant_llm.temp_id`,
			`ALTER TABLE tenant_llm ADD PRIMARY KEY (temp_id)`,
			`ALTER TABLE tenant_llm ADD CONSTRAINT uk_tenant_llm UNIQUE (tenant_id, llm_factory, llm_name)`,
			`ALTER TABLE tenant_llm RENAME COLUMN temp_id TO id`,
		}
		for _, stmt := range statements {
			if err := tx.Exec(stmt).Error; err != nil && !isBenignMigrationError(err) {
				return fmt.Errorf("failed running tenant_llm migration (%s): %w", stmt, err)
			}
		}

		common.Info("tenant_llm primary key migration completed on PostgreSQL")
		return nil
	})
}
