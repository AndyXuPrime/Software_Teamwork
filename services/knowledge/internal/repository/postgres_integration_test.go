package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestPostgresRepositoryDocumentUploadLifecycle(t *testing.T) {
	repo, pool, cleanup := newPostgresRepositoryForTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Date(2026, 6, 29, 14, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}

	kb, err := repo.CreateKnowledgeBase(ctx, service.CreateKnowledgeBaseRecord{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{"type":"fixed"}`),
		RetrievalStrategy: json.RawMessage(`{"mode":"vector"}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
	if kb.ID != "kb_1" || kb.CreatedBy != "usr_1" {
		t.Fatalf("knowledge base = %+v", kb)
	}

	doc, job, err := repo.CreateDocumentWithJob(ctx, service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_1",
		KnowledgeBaseID: kb.ID,
		FileRef:         "file_1",
		Name:            "规程.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       9,
		Status:          service.DocumentStatusUploaded,
		Tags:            []string{"锅炉"},
		CurrentJobID:    "job_1",
		CreatedBy:       "usr_1",
		JobID:           "job_1",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "uploaded",
		JobMessage:      "document uploaded and queued for ingestion",
		MaxAttempts:     3,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, scope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob() error = %v", err)
	}
	if doc.CurrentJobID == nil || *doc.CurrentJobID != job.ID {
		t.Fatalf("document/job link = %+v / %+v", doc, job)
	}
	if job.DocumentID == nil || *job.DocumentID != doc.ID || job.Status != service.JobStatusQueued {
		t.Fatalf("job = %+v", job)
	}

	list, err := repo.ListDocumentsByKnowledgeBase(ctx, kb.ID, nil, scope, service.PageInput{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListDocumentsByKnowledgeBase() error = %v", err)
	}
	if list.Page.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != doc.ID {
		t.Fatalf("document list = %+v", list)
	}

	failedAt := now.Add(time.Minute)
	if err := repo.MarkDocumentJobFailed(ctx, doc.ID, job.ID, nil, "dependency_error", "queue failed", failedAt); err != nil {
		t.Fatalf("MarkDocumentJobFailed() error = %v", err)
	}
	failedDoc, err := repo.GetDocument(ctx, doc.ID, scope)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if failedDoc.Status != service.DocumentStatusFailed ||
		failedDoc.ErrorCode == nil || *failedDoc.ErrorCode != "dependency_error" ||
		failedDoc.ErrorMessage == nil || *failedDoc.ErrorMessage != "queue failed" {
		t.Fatalf("failed document = %+v", failedDoc)
	}

	var jobStatus, jobErrorCode, jobErrorMessage string
	var jobFinishedAt, jobUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT status, COALESCE(error_code, ''), COALESCE(error_message, ''), finished_at, updated_at
		FROM processing_jobs
		WHERE id = $1
	`, job.ID).Scan(&jobStatus, &jobErrorCode, &jobErrorMessage, &jobFinishedAt, &jobUpdatedAt); err != nil {
		t.Fatalf("query failed processing job: %v", err)
	}
	if jobStatus != string(service.JobStatusFailed) ||
		jobErrorCode != "dependency_error" ||
		jobErrorMessage != "queue failed" ||
		!jobFinishedAt.Equal(failedAt) ||
		!jobUpdatedAt.Equal(failedAt) {
		t.Fatalf("failed job status = %q errorCode = %q errorMessage = %q finishedAt = %s updatedAt = %s",
			jobStatus, jobErrorCode, jobErrorMessage, jobFinishedAt, jobUpdatedAt)
	}

	_, succeededJob, err := repo.CreateDocumentWithJob(ctx, service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_terminal",
		KnowledgeBaseID: kb.ID,
		FileRef:         "file_terminal",
		Name:            "terminal.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       9,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_terminal",
		CreatedBy:       "usr_1",
		JobID:           "job_terminal",
		JobType:         service.JobTypeDeleteCleanup,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "delete_cleanup",
		JobMessage:      "document marked deleted; cleanup is pending",
		MaxAttempts:     3,
		CreatedAt:       now.Add(2 * time.Minute),
		UpdatedAt:       now.Add(2 * time.Minute),
	}, scope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob(terminal) error = %v", err)
	}
	completedStage := "completed"
	completedMessage := "document delete cleanup completed"
	completedAt := now.Add(3 * time.Minute)
	if _, err := repo.UpdateJobState(ctx, succeededJob.ID, service.JobStateUpdate{
		Status:          service.JobStatusSucceeded,
		CurrentStage:    &completedStage,
		ProgressPercent: 100,
		Message:         &completedMessage,
		FinishedAt:      &completedAt,
		UpdatedAt:       completedAt,
	}); err != nil {
		t.Fatalf("UpdateJobState(succeeded) error = %v", err)
	}
	if err := repo.MarkDocumentJobFailed(ctx, "doc_terminal", succeededJob.ID, nil, string(service.CodeDependency), "delete cleanup queue handoff failed", now.Add(4*time.Minute)); err != service.ErrConflict {
		t.Fatalf("MarkDocumentJobFailed(succeeded) error = %v, want ErrConflict", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status, COALESCE(error_code, ''), COALESCE(error_message, '')
		FROM processing_jobs
		WHERE id = $1
	`, succeededJob.ID).Scan(&jobStatus, &jobErrorCode, &jobErrorMessage); err != nil {
		t.Fatalf("query succeeded processing job: %v", err)
	}
	if jobStatus != string(service.JobStatusSucceeded) || jobErrorCode != "" || jobErrorMessage != "" {
		t.Fatalf("terminal job was overwritten: status=%q errorCode=%q errorMessage=%q", jobStatus, jobErrorCode, jobErrorMessage)
	}
}

func TestPostgresRepositoryDocumentLifecycleUpdateAndDelete(t *testing.T) {
	repo, pool, cleanup := newPostgresRepositoryForTest(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	ownerScope := service.AccessScope{UserID: "usr_owner", CanWrite: true}
	otherScope := service.AccessScope{UserID: "usr_other", CanWrite: true}

	kb, err := repo.CreateKnowledgeBase(ctx, service.CreateKnowledgeBaseRecord{
		ID:                "kb_lifecycle",
		Name:              "生命周期库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{"type":"fixed"}`),
		RetrievalStrategy: json.RawMessage(`{"mode":"vector"}`),
		CreatedBy:         "usr_owner",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}

	doc, _, err := repo.CreateDocumentWithJob(ctx, service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_lifecycle",
		KnowledgeBaseID: kb.ID,
		FileRef:         "file_lifecycle",
		Name:            "生命周期.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       32,
		Status:          service.DocumentStatusReady,
		Tags:            []string{"old"},
		CurrentJobID:    "job_ingest_lifecycle",
		CreatedBy:       "usr_owner",
		JobID:           "job_ingest_lifecycle",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusSucceeded,
		JobStage:        "ready",
		JobMessage:      "document ready",
		MaxAttempts:     3,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, ownerScope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob() error = %v", err)
	}

	if _, err := repo.UpdateDocument(ctx, service.UpdateDocumentRecord{
		ID:        doc.ID,
		Tags:      []string{"new", "reviewed"},
		UpdatedAt: now.Add(time.Minute),
	}, otherScope); err == nil {
		t.Fatal("UpdateDocument() by unrelated user succeeded, want not found")
	}

	updated, err := repo.UpdateDocument(ctx, service.UpdateDocumentRecord{
		ID:        doc.ID,
		Tags:      []string{"new", "reviewed"},
		UpdatedAt: now.Add(2 * time.Minute),
	}, ownerScope)
	if err != nil {
		t.Fatalf("UpdateDocument() error = %v", err)
	}
	if got, want := strings.Join(updated.Tags, ","), "new,reviewed"; got != want {
		t.Fatalf("updated tags = %q, want %q", got, want)
	}

	deleteAt := now.Add(3 * time.Minute)
	if err := repo.SoftDeleteDocument(ctx, service.DeleteDocumentRecord{
		DocumentID:  doc.ID,
		JobID:       "job_delete_cleanup_lifecycle",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document queued for delete cleanup",
		MaxAttempts: 1,
		DeletedAt:   deleteAt,
		CreatedAt:   deleteAt,
		UpdatedAt:   deleteAt,
	}, otherScope); err == nil {
		t.Fatal("SoftDeleteDocument() by unrelated user succeeded, want not found")
	}

	if err := repo.SoftDeleteDocument(ctx, service.DeleteDocumentRecord{
		DocumentID:  doc.ID,
		JobID:       "job_delete_cleanup_lifecycle",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document queued for delete cleanup",
		MaxAttempts: 1,
		DeletedAt:   deleteAt,
		CreatedAt:   deleteAt,
		UpdatedAt:   deleteAt,
	}, ownerScope); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}

	if _, err := repo.GetDocument(ctx, doc.ID, ownerScope); err == nil {
		t.Fatal("GetDocument() after delete succeeded, want not found")
	}
	target, err := repo.GetDeletedDocumentCleanupTarget(ctx, "job_delete_cleanup_lifecycle")
	if err != nil {
		t.Fatalf("GetDeletedDocumentCleanupTarget() error = %v", err)
	}
	if target.DocumentID != doc.ID || target.KnowledgeBaseID != kb.ID || target.FileRef == nil || *target.FileRef != "file_lifecycle" {
		t.Fatalf("cleanup target = %+v", target)
	}
	retryableTasks, err := repo.ListRetryableDeleteCleanupTasks(ctx, service.DeleteCleanupTaskListInput{
		RequestID: "req_reconcile",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListRetryableDeleteCleanupTasks() error = %v", err)
	}
	if len(retryableTasks) != 1 ||
		retryableTasks[0].RequestID != "req_reconcile" ||
		retryableTasks[0].JobID != "job_delete_cleanup_lifecycle" ||
		retryableTasks[0].DocumentID != doc.ID ||
		retryableTasks[0].KnowledgeBaseID != kb.ID ||
		retryableTasks[0].UserID != ownerScope.UserID {
		t.Fatalf("retryable cleanup tasks = %+v", retryableTasks)
	}
	authDoc, _, err := repo.CreateDocumentWithJob(ctx, service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_delete_cleanup_auth",
		KnowledgeBaseID: kb.ID,
		FileRef:         "file_auth",
		Name:            "auth.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       12,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_ingest_auth",
		CreatedBy:       ownerScope.UserID,
		JobID:           "job_ingest_auth",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "uploaded",
		JobMessage:      "document uploaded and queued for ingestion",
		MaxAttempts:     3,
		CreatedAt:       now.Add(4 * time.Minute),
		UpdatedAt:       now.Add(4 * time.Minute),
	}, ownerScope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob(auth cleanup) error = %v", err)
	}
	if err := repo.SoftDeleteDocument(ctx, service.DeleteDocumentRecord{
		DocumentID:  authDoc.ID,
		JobID:       "job_delete_cleanup_auth",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document queued for delete cleanup",
		MaxAttempts: 3,
		DeletedAt:   now.Add(5 * time.Minute),
		CreatedAt:   now.Add(5 * time.Minute),
		UpdatedAt:   now.Add(5 * time.Minute),
	}, ownerScope); err != nil {
		t.Fatalf("SoftDeleteDocument(auth cleanup) error = %v", err)
	}
	if err := repo.MarkDocumentJobFailed(ctx, authDoc.ID, "job_delete_cleanup_auth", nil, string(service.CodeUnauthorized), "file service rejected knowledge request", now.Add(6*time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed(auth cleanup) error = %v", err)
	}
	retryableTasks, err = repo.ListRetryableDeleteCleanupTasks(ctx, service.DeleteCleanupTaskListInput{
		RequestID: "req_reconcile",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListRetryableDeleteCleanupTasks(auth failure) error = %v", err)
	}
	foundAuth := false
	for _, task := range retryableTasks {
		if task.JobID == "job_delete_cleanup_auth" {
			foundAuth = true
			if task.DocumentID != authDoc.ID || task.UserID != ownerScope.UserID {
				t.Fatalf("auth retryable task = %+v", task)
			}
		}
	}
	if !foundAuth {
		t.Fatalf("retryable auth cleanup task missing from %+v", retryableTasks)
	}

	staleRunningDoc, _, err := repo.CreateDocumentWithJob(ctx, service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_delete_cleanup_stale_running",
		KnowledgeBaseID: kb.ID,
		FileRef:         "file_stale_running",
		Name:            "stale-running.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       12,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_ingest_stale_running",
		CreatedBy:       ownerScope.UserID,
		JobID:           "job_ingest_stale_running",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "uploaded",
		JobMessage:      "document uploaded and queued for ingestion",
		MaxAttempts:     3,
		CreatedAt:       now.Add(7 * time.Minute),
		UpdatedAt:       now.Add(7 * time.Minute),
	}, ownerScope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob(stale running cleanup) error = %v", err)
	}
	if err := repo.SoftDeleteDocument(ctx, service.DeleteDocumentRecord{
		DocumentID:  staleRunningDoc.ID,
		JobID:       "job_delete_cleanup_stale_running",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document queued for delete cleanup",
		MaxAttempts: 1,
		DeletedAt:   now.Add(8 * time.Minute),
		CreatedAt:   now.Add(8 * time.Minute),
		UpdatedAt:   now.Add(8 * time.Minute),
	}, ownerScope); err != nil {
		t.Fatalf("SoftDeleteDocument(stale running cleanup) error = %v", err)
	}
	staleStage := "delete_cleanup"
	staleAttempts := int32(1)
	staleStartedAt := now.Add(-time.Hour)
	if _, err := repo.UpdateJobState(ctx, "job_delete_cleanup_stale_running", service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &staleStage,
		ProgressPercent: 20,
		Attempts:        &staleAttempts,
		StartedAt:       &staleStartedAt,
		UpdatedAt:       staleStartedAt,
	}); err != nil {
		t.Fatalf("UpdateJobState(stale running cleanup) error = %v", err)
	}
	retryableTasks, err = repo.ListRetryableDeleteCleanupTasks(ctx, service.DeleteCleanupTaskListInput{
		RequestID:          "req_reconcile",
		Limit:              20,
		StaleRunningBefore: &deleteAt,
	})
	if err != nil {
		t.Fatalf("ListRetryableDeleteCleanupTasks(stale running) error = %v", err)
	}
	foundStaleRunning := false
	for _, task := range retryableTasks {
		if task.JobID == "job_delete_cleanup_stale_running" {
			foundStaleRunning = true
			if task.DocumentID != staleRunningDoc.ID || task.UserID != ownerScope.UserID {
				t.Fatalf("stale running retryable task = %+v", task)
			}
		}
	}
	if !foundStaleRunning {
		t.Fatalf("stale running cleanup task missing from %+v", retryableTasks)
	}

	list, err := repo.ListDocumentsByKnowledgeBase(ctx, kb.ID, nil, ownerScope, service.PageInput{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListDocumentsByKnowledgeBase() after delete error = %v", err)
	}
	if list.Page.Total != 0 || len(list.Items) != 0 {
		t.Fatalf("deleted document still visible in list: %+v", list)
	}

	var jobType, jobStatus, jobStage, jobMessage string
	var maxAttempts int32
	var deletedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT j.job_type, j.status, COALESCE(j.current_stage, ''), COALESCE(j.message, ''), j.max_attempts, d.deleted_at
		FROM processing_jobs j
		JOIN knowledge_documents d ON d.id = j.document_id
		WHERE j.id = $1
	`, "job_delete_cleanup_lifecycle").Scan(&jobType, &jobStatus, &jobStage, &jobMessage, &maxAttempts, &deletedAt); err != nil {
		t.Fatalf("query delete cleanup job: %v", err)
	}
	if jobType != service.JobTypeDeleteCleanup ||
		jobStatus != service.JobStatusQueued ||
		jobStage != "delete_cleanup" ||
		jobMessage != "document queued for delete cleanup" ||
		maxAttempts != 1 ||
		!deletedAt.Equal(deleteAt) {
		t.Fatalf("cleanup job = type:%q status:%q stage:%q message:%q maxAttempts:%d deletedAt:%s",
			jobType, jobStatus, jobStage, jobMessage, maxAttempts, deletedAt)
	}
}

func newPostgresRepositoryForTest(t *testing.T) (*repository.PostgresRepository, *pgxpool.Pool, func()) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("KNOWLEDGE_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("set KNOWLEDGE_TEST_DATABASE_URL to run Postgres repository integration tests")
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	schema := fmt.Sprintf("knowledge_test_%d", time.Now().UnixNano())
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		adminPool.Close()
		t.Fatalf("create test schema: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		t.Fatalf("parse test database url: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
		t.Fatalf("connect isolated test schema: %v", err)
	}

	applyKnowledgeMigration(t, ctx, pool)
	cleanup := func() {
		pool.Close()
		_, _ = adminPool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		adminPool.Close()
	}
	return repository.NewPostgresRepository(pool), pool, cleanup
}

func applyKnowledgeMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, migration := range []string{
		"../../migrations/0001_create_knowledge_core_tables.sql",
		"../../migrations/0002_create_parser_configs.sql",
	} {
		contents, err := os.ReadFile(migration)
		if err != nil {
			t.Fatalf("read knowledge migration %s: %v", migration, err)
		}
		upSQL, _, _ := strings.Cut(string(contents), "-- +goose Down")
		upSQL = strings.ReplaceAll(upSQL, "-- +goose Up", "")

		for _, statement := range strings.Split(upSQL, ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := pool.Exec(ctx, statement); err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migration, statement, err)
			}
		}
	}
}
