package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestMemoryRepositorySoftDeleteKnowledgeBaseHidesDocuments(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}

	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	repo.SeedDocument(service.KnowledgeDocument{
		ID:              "doc_1",
		KnowledgeBaseID: "kb_1",
		Name:            "规程.pdf",
		Status:          service.DocumentStatusReady,
		CreatedBy:       "usr_1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	if err := repo.SoftDeleteKnowledgeBase(context.Background(), "kb_1", now.Add(time.Hour), scope); err != nil {
		t.Fatalf("SoftDeleteKnowledgeBase() error = %v", err)
	}
	if _, err := repo.GetKnowledgeBase(context.Background(), "kb_1", scope); err != service.ErrNotFound {
		t.Fatalf("GetKnowledgeBase() error = %v", err)
	}
	if _, err := repo.GetDocument(context.Background(), "doc_1", scope); err != service.ErrNotFound {
		t.Fatalf("GetDocument() error = %v", err)
	}
}

func TestMemoryRepositoryCreateDocumentWithJobAndMarkFailed(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 14, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})

	doc, job, err := repo.CreateDocumentWithJob(context.Background(), service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
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
	if doc.ID != "doc_1" || doc.CurrentJobID == nil || *doc.CurrentJobID != "job_1" {
		t.Fatalf("document = %+v", doc)
	}
	if job.ID != "job_1" || job.DocumentID == nil || *job.DocumentID != "doc_1" || job.Status != service.JobStatusQueued {
		t.Fatalf("job = %+v", job)
	}

	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_1", "job_1", nil, "dependency_error", "queue failed", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed() error = %v", err)
	}
	failedDoc, err := repo.GetDocument(context.Background(), "doc_1", scope)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if failedDoc.Status != service.DocumentStatusFailed || failedDoc.ErrorCode == nil || *failedDoc.ErrorCode != "dependency_error" {
		t.Fatalf("failed document = %+v", failedDoc)
	}
}

func TestMemoryRepositoryMarkFailedKeepsJobTerminalWhenDocumentWasDeleted(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 14, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	_, job, err := repo.CreateDocumentWithJob(context.Background(), service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		FileRef:         "file_1",
		Name:            "规程.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       9,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_1",
		CreatedBy:       "usr_1",
		JobID:           "job_1",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusRunning,
		JobStage:        "parsing",
		JobMessage:      "document ingestion in progress",
		MaxAttempts:     3,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, scope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob() error = %v", err)
	}
	if job.Status != service.JobStatusRunning {
		t.Fatalf("job status = %s, want running", job.Status)
	}
	if err := repo.SoftDeleteKnowledgeBase(context.Background(), "kb_1", now.Add(time.Minute), scope); err != nil {
		t.Fatalf("SoftDeleteKnowledgeBase() error = %v", err)
	}

	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_1", "job_1", nil, "dependency_error", "source content read failed", now.Add(2*time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed() error = %v", err)
	}
	failedJob, err := repo.GetProcessingJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if failedJob.Status != service.JobStatusFailed || failedJob.ErrorCode == nil || *failedJob.ErrorCode != "dependency_error" {
		t.Fatalf("failed job = %+v", failedJob)
	}
}

func TestMemoryRepositoryMarkFailedDoesNotOverwriteSucceededJob(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 14, 30, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	_, job, err := repo.CreateDocumentWithJob(context.Background(), service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		FileRef:         "file_1",
		Name:            "规程.pdf",
		ContentType:     "application/pdf",
		SizeBytes:       9,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_1",
		CreatedBy:       "usr_1",
		JobID:           "job_1",
		JobType:         service.JobTypeDeleteCleanup,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "delete_cleanup",
		JobMessage:      "document marked deleted; cleanup is pending",
		MaxAttempts:     3,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, scope)
	if err != nil {
		t.Fatalf("CreateDocumentWithJob() error = %v", err)
	}
	finishedAt := now.Add(time.Minute)
	stage := "completed"
	message := "document delete cleanup completed"
	if _, err := repo.UpdateJobState(context.Background(), job.ID, service.JobStateUpdate{
		Status:          service.JobStatusSucceeded,
		CurrentStage:    &stage,
		ProgressPercent: 100,
		Message:         &message,
		FinishedAt:      &finishedAt,
		UpdatedAt:       finishedAt,
	}); err != nil {
		t.Fatalf("UpdateJobState(succeeded) error = %v", err)
	}

	err = repo.MarkDocumentJobFailed(context.Background(), "doc_1", "job_1", nil, string(service.CodeDependency), "delete cleanup queue handoff failed", now.Add(2*time.Minute))
	if err != service.ErrConflict {
		t.Fatalf("MarkDocumentJobFailed() error = %v, want ErrConflict", err)
	}
	loaded, err := repo.GetProcessingJob(context.Background(), "job_1")
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if loaded.Status != service.JobStatusSucceeded || loaded.ErrorCode != nil || loaded.ErrorMessage != nil {
		t.Fatalf("terminal job was overwritten: %+v", loaded)
	}
}

func TestMemoryRepositoryDeletedDocumentCleanupTargetReadsSoftDeletedDocumentOnly(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 15, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}
	fileRef := "file_1"
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	repo.SeedDocument(service.KnowledgeDocument{
		ID:              "doc_1",
		KnowledgeBaseID: "kb_1",
		FileRef:         &fileRef,
		Name:            "规程.pdf",
		Status:          service.DocumentStatusReady,
		CreatedBy:       "usr_1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if _, err := repo.GetDeletedDocumentCleanupTarget(context.Background(), "job_cleanup"); err != service.ErrNotFound {
		t.Fatalf("GetDeletedDocumentCleanupTarget() before delete error = %v, want not found", err)
	}
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_cleanup",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: 3,
		DeletedAt:   now.Add(time.Minute),
		CreatedAt:   now.Add(time.Minute),
		UpdatedAt:   now.Add(time.Minute),
	}, scope); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}

	target, err := repo.GetDeletedDocumentCleanupTarget(context.Background(), "job_cleanup")
	if err != nil {
		t.Fatalf("GetDeletedDocumentCleanupTarget() error = %v", err)
	}
	if target.DocumentID != "doc_1" || target.KnowledgeBaseID != "kb_1" || target.FileRef == nil || *target.FileRef != "file_1" {
		t.Fatalf("cleanup target = %+v", target)
	}
	if _, err := repo.GetDocument(context.Background(), "doc_1", scope); err != service.ErrNotFound {
		t.Fatalf("GetDocument() after delete error = %v, want not found", err)
	}
}

func TestMemoryRepositoryListRetryableDeleteCleanupTasksFiltersTerminalAndFreshJobs(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Date(2026, 6, 29, 16, 0, 0, 0, time.UTC)
	scope := service.AccessScope{UserID: "usr_1", CanWrite: true}
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "规程库",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	seedDeletedCleanupJob := func(t *testing.T, docID string, jobID string, createdAt time.Time) {
		t.Helper()
		fileRef := "file_" + docID
		repo.SeedDocument(service.KnowledgeDocument{
			ID:              docID,
			KnowledgeBaseID: "kb_1",
			FileRef:         &fileRef,
			Name:            docID + ".pdf",
			Status:          service.DocumentStatusReady,
			CreatedBy:       "usr_1",
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		})
		if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
			DocumentID:  docID,
			JobID:       jobID,
			JobType:     service.JobTypeDeleteCleanup,
			JobStatus:   service.JobStatusQueued,
			JobStage:    "delete_cleanup",
			JobMessage:  "document marked deleted; cleanup is pending",
			MaxAttempts: 3,
			DeletedAt:   createdAt,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		}, scope); err != nil {
			t.Fatalf("SoftDeleteDocument(%s) error = %v", docID, err)
		}
	}

	seedDeletedCleanupJob(t, "doc_queued", "job_queued", now.Add(-6*time.Minute))
	seedDeletedCleanupJob(t, "doc_failed_dependency", "job_failed_dependency", now.Add(-5*time.Minute))
	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_failed_dependency", "job_failed_dependency", nil, string(service.CodeDependency), "delete cleanup queue handoff failed", now.Add(-5*time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed(dependency) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_failed_unauthorized", "job_failed_unauthorized", now.Add(-4*time.Minute))
	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_failed_unauthorized", "job_failed_unauthorized", nil, string(service.CodeUnauthorized), "file service rejected knowledge request", now.Add(-4*time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed(unauthorized) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_failed_conflict", "job_failed_conflict", now.Add(-4*time.Minute))
	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_failed_conflict", "job_failed_conflict", nil, string(service.CodeConflict), "delete cleanup target mismatch", now.Add(-4*time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed(conflict) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_running_stale", "job_running_stale", now.Add(-3*time.Minute))
	staleStage := "delete_cleanup"
	staleAttempts := int32(1)
	if _, err := repo.UpdateJobState(context.Background(), "job_running_stale", service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &staleStage,
		ProgressPercent: 20,
		Attempts:        &staleAttempts,
		StartedAt:       ptrTime(now.Add(-30 * time.Minute)),
		UpdatedAt:       now.Add(-30 * time.Minute),
	}); err != nil {
		t.Fatalf("UpdateJobState(stale running) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_running_fresh", "job_running_fresh", now.Add(-2*time.Minute))
	freshAttempts := int32(1)
	if _, err := repo.UpdateJobState(context.Background(), "job_running_fresh", service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &staleStage,
		ProgressPercent: 20,
		Attempts:        &freshAttempts,
		StartedAt:       ptrTime(now),
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("UpdateJobState(fresh running) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_running_exhausted", "job_running_exhausted", now.Add(-90*time.Minute))
	exhaustedRunningAttempts := int32(3)
	if _, err := repo.UpdateJobState(context.Background(), "job_running_exhausted", service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &staleStage,
		ProgressPercent: 20,
		Attempts:        &exhaustedRunningAttempts,
		StartedAt:       ptrTime(now.Add(-90 * time.Minute)),
		UpdatedAt:       now.Add(-90 * time.Minute),
	}); err != nil {
		t.Fatalf("UpdateJobState(exhausted stale running) error = %v", err)
	}
	seedDeletedCleanupJob(t, "doc_exhausted", "job_exhausted", now.Add(-time.Minute))
	exhaustedAttempts := int32(3)
	errorCode := string(service.CodeDependency)
	errorMessage := "file cleanup failed"
	if _, err := repo.UpdateJobState(context.Background(), "job_exhausted", service.JobStateUpdate{
		Status:          service.JobStatusFailed,
		CurrentStage:    &staleStage,
		ProgressPercent: 20,
		Attempts:        &exhaustedAttempts,
		ErrorCode:       &errorCode,
		ErrorMessage:    &errorMessage,
		UpdatedAt:       now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("UpdateJobState(exhausted) error = %v", err)
	}

	tasks, err := repo.ListRetryableDeleteCleanupTasks(context.Background(), service.DeleteCleanupTaskListInput{
		RequestID:          "req_reconcile",
		Limit:              10,
		StaleRunningBefore: ptrTime(now.Add(-5 * time.Minute)),
	})
	if err != nil {
		t.Fatalf("ListRetryableDeleteCleanupTasks() error = %v", err)
	}
	got := map[string]service.DocumentDeleteCleanupTask{}
	for _, task := range tasks {
		got[task.JobID] = task
	}
	for _, jobID := range []string{"job_queued", "job_failed_dependency", "job_failed_unauthorized", "job_running_stale", "job_running_exhausted"} {
		task, exists := got[jobID]
		if !exists {
			t.Fatalf("missing retryable job %s in tasks %+v", jobID, tasks)
		}
		if task.RequestID != "req_reconcile" || task.UserID != "usr_1" || task.KnowledgeBaseID != "kb_1" {
			t.Fatalf("task for %s = %+v", jobID, task)
		}
	}
	for _, jobID := range []string{"job_failed_conflict", "job_running_fresh", "job_exhausted"} {
		if _, exists := got[jobID]; exists {
			t.Fatalf("non-retryable job %s appeared in tasks %+v", jobID, tasks)
		}
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
