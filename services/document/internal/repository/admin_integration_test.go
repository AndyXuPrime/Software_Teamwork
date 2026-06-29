package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

func TestPostgresRepositoryAdminSettingsLogsAndStats(t *testing.T) {
	databaseURL := os.Getenv("DOCUMENT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DOCUMENT_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool := newTestPool(t, ctx, databaseURL)
	defer pool.Close()
	applyMigration(t, ctx, pool)

	repo := NewPostgresRepository(pool)
	now := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	reportType, err := repo.UpsertReportType(ctx, service.ReportType{
		Code:      "admin_report",
		Name:      "Admin Report",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("UpsertReportType() error = %v", err)
	}
	template, err := repo.CreateReportTemplate(ctx, service.ReportTemplate{
		ID:           "00000000-0000-0000-0000-000000002101",
		TemplateName: "admin template",
		ReportType:   reportType.Code,
		Version:      1,
		Filename:     "template.docx",
		FileSize:     10,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, service.ReportTemplateStructure{})
	if err != nil {
		t.Fatalf("CreateReportTemplate() error = %v", err)
	}
	if _, err := repo.CreateReportMaterial(ctx, service.ReportMaterial{
		ID:           "00000000-0000-0000-0000-000000002102",
		MaterialName: "material",
		MaterialType: "doc",
		Filename:     "material.docx",
		FileSize:     20,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("CreateReportMaterial() error = %v", err)
	}
	report, err := repo.CreateReport(ctx, service.Report{
		ID:         "00000000-0000-0000-0000-000000002103",
		Name:       "admin report",
		ReportType: reportType.Code,
		TemplateID: template.ID,
		Topic:      "admin",
		Status:     service.ReportStatusGenerated,
		Source:     "backend",
		GeneratedAt: func() *time.Time {
			value := now
			return &value
		}(),
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	if _, err := repo.CreateReportJob(ctx, service.ReportJob{
		ID:          "00000000-0000-0000-0000-000000002104",
		RequestID:   "req-admin",
		Source:      "api",
		JobType:     service.JobTypeContentGeneration,
		TargetType:  "report",
		TargetID:    report.ID,
		QueueName:   "document",
		ReportID:    report.ID,
		Status:      service.JobStatusSucceeded,
		MaxAttempts: 3,
		FinishedAt:  &now,
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("CreateReportJob() error = %v", err)
	}

	settings, err := repo.SaveReportSettings(ctx, service.ReportSettings{
		LLM:              service.ReportSettingsModelConfig{Provider: "ai-gateway", ProfileID: "mp-chat", Model: "gpt-test", TimeoutSeconds: 60},
		DefaultTemplates: map[string]string{reportType.Code: template.ID},
		File:             service.ReportSettingsFileDefaults{DefaultFormat: "docx", DefaultNumberingMode: "global"},
		UpdatedAt:        now,
	})
	if err != nil {
		t.Fatalf("SaveReportSettings() error = %v", err)
	}
	if settings.DefaultTemplates[reportType.Code] != template.ID {
		t.Fatalf("settings = %+v", settings)
	}
	reloaded, err := repo.GetReportSettings(ctx)
	if err != nil {
		t.Fatalf("GetReportSettings() error = %v", err)
	}
	if reloaded.LLM.ProfileID != "mp-chat" {
		t.Fatalf("reloaded settings = %+v", reloaded)
	}

	if _, err := repo.CreateOperationLog(ctx, service.OperationLog{
		ID:              "00000000-0000-0000-0000-000000002105",
		OperatorID:      "admin-1",
		OperationType:   service.OperationCreateReport,
		TargetType:      "report",
		TargetID:        report.ID,
		RequestID:       "req-admin",
		RequestSource:   "mcp",
		ToolName:        "generate_report_outline",
		OperationResult: service.OperationResultSucceeded,
		ParameterSummary: map[string]any{
			"reportType": reportType.Code,
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateOperationLog() error = %v", err)
	}
	logs, err := repo.ListOperationLogs(ctx, service.OperationLogListFilter{
		Page:          1,
		PageSize:      10,
		TargetType:    "report",
		RequestSource: "mcp",
	})
	if err != nil {
		t.Fatalf("ListOperationLogs() error = %v", err)
	}
	if logs.Page.Total != 1 || len(logs.Items) != 1 || logs.Items[0].ToolName != "generate_report_outline" {
		t.Fatalf("logs = %+v", logs)
	}

	overview, err := repo.GetReportStatisticsOverview(ctx, 30)
	if err != nil {
		t.Fatalf("GetReportStatisticsOverview() error = %v", err)
	}
	if overview.ReportCount != 1 || overview.TemplateCount != 1 || overview.MaterialCount != 1 || overview.JobStatusCounts[string(service.JobStatusSucceeded)] != 1 {
		t.Fatalf("overview = %+v", overview)
	}
	daily, err := repo.ListReportDailyStatistics(ctx, 30)
	if err != nil {
		t.Fatalf("ListReportDailyStatistics() error = %v", err)
	}
	if len(daily) == 0 {
		t.Fatalf("daily stats should not be empty")
	}
}
