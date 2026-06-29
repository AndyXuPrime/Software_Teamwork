package repository

import (
	"os"
	"strings"
	"testing"
)

func TestAdminMigrationIncludesOperationLogFilterIndexes(t *testing.T) {
	raw, err := os.ReadFile("../../migrations/0002_create_report_settings_indexes.sql")
	if err != nil {
		t.Fatalf("read admin migration: %v", err)
	}
	migration := string(raw)
	for _, indexName := range []string{
		"idx_report_operation_logs_target_created_at",
		"idx_report_operation_logs_operation_created_at",
		"idx_report_operation_logs_request_id_created_at",
		"idx_report_operation_logs_source_tool_created_at",
	} {
		if !strings.Contains(migration, indexName) {
			t.Fatalf("migration missing operation log filter index %q", indexName)
		}
	}
}
