package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/worker"
)

func TestKnowledgeBaseCRUDAndSoftDelete(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
	svc := service.NewWithOptions(repo, func() time.Time { return now }, func(prefix string) string {
		return prefix + "_test"
	})
	reqCtx := writeContext("usr_1")

	kb, err := svc.CreateKnowledgeBase(context.Background(), reqCtx, service.CreateKnowledgeBaseInput{
		Name: "规程库",
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
	if kb.ID != "kb_test" || kb.CreatedBy != "usr_1" || kb.DocType != "GENERAL" {
		t.Fatalf("created knowledge base = %+v", kb)
	}
	if !json.Valid(kb.ChunkStrategy) || !json.Valid(kb.RetrievalStrategy) {
		t.Fatalf("default strategies are not valid JSON")
	}

	newName := "规程库 2026"
	updated, err := svc.UpdateKnowledgeBase(context.Background(), reqCtx, service.UpdateKnowledgeBaseInput{
		ID:   kb.ID,
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateKnowledgeBase() error = %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("updated name = %q", updated.Name)
	}

	list, err := svc.ListKnowledgeBases(context.Background(), readContext("usr_1"), service.ListKnowledgeBasesInput{})
	if err != nil {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
	if list.Page.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("list = %+v", list)
	}

	if err := svc.DeleteKnowledgeBase(context.Background(), reqCtx, kb.ID); err != nil {
		t.Fatalf("DeleteKnowledgeBase() error = %v", err)
	}
	_, err = svc.GetKnowledgeBase(context.Background(), reqCtx, kb.ID)
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetKnowledgeBase() after delete error = %v", err)
	}
}

func TestCreateRequiresWritePermission(t *testing.T) {
	svc := service.New(repository.NewMemoryRepository())

	_, err := svc.CreateKnowledgeBase(context.Background(), readContext("usr_1"), service.CreateKnowledgeBaseInput{Name: "规程库"})
	if !hasCode(err, service.CodeForbidden) {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
}

func TestMissingUserReturnsUnauthorized(t *testing.T) {
	svc := service.New(repository.NewMemoryRepository())

	_, err := svc.ListKnowledgeBases(context.Background(), service.RequestContext{}, service.ListKnowledgeBasesInput{})
	if !hasCode(err, service.CodeUnauthorized) {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
}

func TestOwnerFilteringHidesOtherUsersResources(t *testing.T) {
	repo := repository.NewMemoryRepository()
	svc := service.NewWithOptions(repo, fixedClock(), func(prefix string) string { return prefix + "_owner" })
	kb, err := svc.CreateKnowledgeBase(context.Background(), writeContext("owner"), service.CreateKnowledgeBaseInput{Name: "owner kb"})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}

	_, err = svc.GetKnowledgeBase(context.Background(), readContext("other"), kb.ID)
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetKnowledgeBase() for other user error = %v", err)
	}

	got, err := svc.GetKnowledgeBase(context.Background(), service.RequestContext{UserID: "reader", Permissions: []string{service.PermissionKnowledgeRead}}, kb.ID)
	if err != nil {
		t.Fatalf("GetKnowledgeBase() with read permission error = %v", err)
	}
	if got.ID != kb.ID {
		t.Fatalf("knowledge base id = %q", got.ID)
	}
}

func TestDocumentListAndDetailExcludeDeletedKnowledgeBase(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
		ChunkCount:      3,
		Tags:            []string{"锅炉"},
		CreatedBy:       "usr_1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	svc := service.NewWithOptions(repo, func() time.Time { return now.Add(time.Hour) }, nil)

	status := service.DocumentStatusReady
	list, err := svc.ListDocuments(context.Background(), readContext("usr_1"), service.ListDocumentsInput{
		KnowledgeBaseID: "kb_1",
		Status:          &status,
	})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if list.Page.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != "doc_1" {
		t.Fatalf("document list = %+v", list)
	}

	if err := svc.DeleteKnowledgeBase(context.Background(), writeContext("usr_1"), "kb_1"); err != nil {
		t.Fatalf("DeleteKnowledgeBase() error = %v", err)
	}
	_, err = svc.GetDocument(context.Background(), readContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocument() after kb delete error = %v", err)
	}
	_, err = svc.ListDocuments(context.Background(), readContext("usr_1"), service.ListDocumentsInput{KnowledgeBaseID: "kb_1"})
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("ListDocuments() after kb delete error = %v", err)
	}
}

func TestInvalidPatchAndPagination(t *testing.T) {
	svc := service.New(repository.NewMemoryRepository())

	blank := " "
	_, err := svc.UpdateKnowledgeBase(context.Background(), writeContext("usr_1"), service.UpdateKnowledgeBaseInput{
		ID:   "kb_1",
		Name: &blank,
	})
	if !hasCode(err, service.CodeValidation) {
		t.Fatalf("UpdateKnowledgeBase() error = %v", err)
	}

	_, err = svc.ListKnowledgeBases(context.Background(), readContext("usr_1"), service.ListKnowledgeBasesInput{
		Page: service.PageInput{Page: 0, PageSize: 201},
	})
	if !hasCode(err, service.CodeValidation) {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
}

func TestInvalidDocumentStatus(t *testing.T) {
	svc := service.New(repository.NewMemoryRepository())
	status := service.DocumentStatus("deleted")

	_, err := svc.ListDocuments(context.Background(), readContext("usr_1"), service.ListDocumentsInput{
		KnowledgeBaseID: "kb_1",
		Status:          &status,
	})
	if !hasCode(err, service.CodeValidation) {
		t.Fatalf("ListDocuments() error = %v", err)
	}
}

func TestUploadDocumentCreatesDocumentJobAndQueuesIngestion(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	seedUploadParserConfig(repo, now)

	files := &uploadFileClient{
		createFn: func(ctx context.Context, reqCtx service.RequestContext, file service.UploadedFile) (service.FileObject, error) {
			if reqCtx.RequestID != "req_upload" || reqCtx.UserID != "usr_1" {
				t.Fatalf("request context = %+v", reqCtx)
			}
			if file.Filename != "knowledge-guide.pdf" {
				t.Fatalf("file filename = %q", file.Filename)
			}
			body, err := io.ReadAll(file.Content)
			if err != nil {
				t.Fatalf("read file content: %v", err)
			}
			if string(body) != "pdf-bytes" {
				t.Fatalf("file content = %q", string(body))
			}
			return service.FileObject{
				ID:             "file_1",
				Filename:       "knowledge-guide.pdf",
				ContentType:    "application/pdf",
				SizeBytes:      9,
				ChecksumSHA256: "abc123",
				CreatedAt:      now,
			}, nil
		},
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, files, queue, func() time.Time { return now }, func(prefix string) string {
		return prefix + "_test"
	})

	doc, err := svc.UploadDocument(context.Background(), service.RequestContext{
		RequestID:     "req_upload",
		UserID:        "usr_1",
		Permissions:   []string{service.PermissionKnowledgeWrite},
		CallerService: "gateway",
	}, service.UploadDocumentInput{
		KnowledgeBaseID: "kb_1",
		File: service.UploadedFile{
			Filename:    "reports/knowledge-guide.pdf",
			ContentType: "application/pdf",
			SizeBytes:   9,
			Content:     bytes.NewReader([]byte("pdf-bytes")),
		},
		Tags: []string{"锅炉", " 锅炉 ", "规程"},
	})
	if err != nil {
		t.Fatalf("UploadDocument() error = %v", err)
	}
	if doc.ID != "doc_test" || doc.KnowledgeBaseID != "kb_1" || doc.Status != service.DocumentStatusUploaded {
		t.Fatalf("uploaded doc = %+v", doc)
	}
	if doc.Name != "knowledge-guide.pdf" {
		t.Fatalf("document name = %q", doc.Name)
	}
	if doc.FileRef == nil || *doc.FileRef != "file_1" {
		t.Fatalf("file ref = %+v", doc.FileRef)
	}
	if doc.CurrentJobID == nil || *doc.CurrentJobID != "job_test" {
		t.Fatalf("current job id = %+v", doc.CurrentJobID)
	}
	if len(doc.Tags) != 2 {
		t.Fatalf("tags = %+v", doc.Tags)
	}
	if queue.calls != 1 {
		t.Fatalf("queue calls = %d", queue.calls)
	}
	if queue.task.JobID != "job_test" || queue.task.DocumentID != "doc_test" || queue.task.KnowledgeBaseID != "kb_1" || queue.task.RequestID != "req_upload" || queue.task.UserID != "usr_1" {
		t.Fatalf("queue task = %+v", queue.task)
	}
}

func TestUploadDocumentUsesBuiltinFallbackWhenParserConfigsEmpty(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 15, 0, 0, time.UTC)
	repo := &uploadRepository{
		MemoryRepository: repository.NewMemoryRepository(),
	}
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "knowledge base",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{}`),
		RetrievalStrategy: json.RawMessage(`{}`),
		CreatedBy:         "usr_1",
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	files := &uploadFileClient{
		createFn: func(context.Context, service.RequestContext, service.UploadedFile) (service.FileObject, error) {
			return service.FileObject{
				ID:             "file_1",
				Filename:       "knowledge-guide.pdf",
				ContentType:    "application/pdf",
				SizeBytes:      9,
				ChecksumSHA256: "abc123",
				CreatedAt:      now,
			}, nil
		},
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, files, queue, func() time.Time { return now }, func(prefix string) string {
		return prefix + "_test"
	})

	doc, err := svc.UploadDocument(context.Background(), service.RequestContext{
		RequestID:     "req_upload",
		UserID:        "usr_1",
		Permissions:   []string{service.PermissionKnowledgeWrite},
		CallerService: "gateway",
	}, service.UploadDocumentInput{
		KnowledgeBaseID: "kb_1",
		File: service.UploadedFile{
			Filename:    "knowledge-guide.pdf",
			ContentType: "application/pdf",
			SizeBytes:   9,
			Content:     bytes.NewReader([]byte("pdf-bytes")),
		},
	})
	if err != nil {
		t.Fatalf("UploadDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusUploaded || queue.calls != 1 {
		t.Fatalf("doc = %+v queue calls = %d", doc, queue.calls)
	}
	if repo.lastCreate.ParserConfigID != "" {
		t.Fatalf("fallback parser config id = %q", repo.lastCreate.ParserConfigID)
	}
	var snapshot service.ParserConfigSnapshot
	if err := json.Unmarshal(repo.lastCreate.ParserConfigSnapshot, &snapshot); err != nil {
		t.Fatalf("unmarshal parser snapshot: %v", err)
	}
	if snapshot.Backend != service.ParserBackendBuiltin || snapshot.Concurrency != 4 {
		t.Fatalf("fallback snapshot = %+v", snapshot)
	}
}

func TestUploadDocumentCompensatesWhenRepositoryFails(t *testing.T) {
	now := time.Date(2026, 6, 29, 11, 30, 0, 0, time.UTC)
	repo := &uploadRepository{
		MemoryRepository: repository.NewMemoryRepository(),
		createErr:        service.ErrConflict,
	}
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
	seedUploadParserConfig(repo.MemoryRepository, now)
	files := &uploadFileClient{
		createFn: func(context.Context, service.RequestContext, service.UploadedFile) (service.FileObject, error) {
			return service.FileObject{
				ID:             "file_1",
				Filename:       "knowledge-guide.pdf",
				ContentType:    "application/pdf",
				SizeBytes:      9,
				ChecksumSHA256: "abc123",
				CreatedAt:      now,
			}, nil
		},
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, files, queue, func() time.Time { return now }, func(prefix string) string {
		return prefix + "_test"
	})

	_, err := svc.UploadDocument(context.Background(), service.RequestContext{
		RequestID:     "req_upload",
		UserID:        "usr_1",
		Permissions:   []string{service.PermissionKnowledgeWrite},
		CallerService: "gateway",
	}, service.UploadDocumentInput{
		KnowledgeBaseID: "kb_1",
		File: service.UploadedFile{
			Filename:    "knowledge-guide.pdf",
			ContentType: "application/pdf",
			SizeBytes:   9,
			Content:     bytes.NewReader([]byte("pdf-bytes")),
		},
	})
	if !hasCode(err, service.CodeConflict) {
		t.Fatalf("UploadDocument() error = %v", err)
	}
	if len(files.deleted) != 1 || files.deleted[0] != "file_1" {
		t.Fatalf("delete calls = %+v", files.deleted)
	}
	if queue.calls != 0 {
		t.Fatalf("queue calls = %d", queue.calls)
	}
	if repo.lastCreate.ParserConfigID != "parser_default" || !json.Valid(repo.lastCreate.ParserConfigSnapshot) {
		t.Fatalf("parser config snapshot = id:%q body:%s", repo.lastCreate.ParserConfigID, string(repo.lastCreate.ParserConfigSnapshot))
	}
}

func TestUploadDocumentValidatesKnowledgeBaseBeforeFileWrite(t *testing.T) {
	files := &uploadFileClient{
		createFn: func(context.Context, service.RequestContext, service.UploadedFile) (service.FileObject, error) {
			t.Fatal("file service should not be called when knowledge base is not visible")
			return service.FileObject{}, nil
		},
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repository.NewMemoryRepository(), files, queue, fixedClock(), func(prefix string) string {
		return prefix + "_test"
	})

	_, err := svc.UploadDocument(context.Background(), service.RequestContext{
		RequestID:     "req_upload",
		UserID:        "usr_1",
		Permissions:   []string{service.PermissionKnowledgeWrite},
		CallerService: "gateway",
	}, service.UploadDocumentInput{
		KnowledgeBaseID: "kb_missing",
		File: service.UploadedFile{
			Filename:    "knowledge-guide.pdf",
			ContentType: "application/pdf",
			SizeBytes:   9,
			Content:     bytes.NewReader([]byte("pdf-bytes")),
		},
	})
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("UploadDocument() error = %v", err)
	}
	if queue.calls != 0 {
		t.Fatalf("queue calls = %d", queue.calls)
	}
}

func TestUploadDocumentMarksFailureWhenQueueHandoffFails(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	repo := &uploadRepository{
		MemoryRepository: repository.NewMemoryRepository(),
	}
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
	seedUploadParserConfig(repo.MemoryRepository, now)
	files := &uploadFileClient{
		createFn: func(context.Context, service.RequestContext, service.UploadedFile) (service.FileObject, error) {
			return service.FileObject{
				ID:             "file_1",
				Filename:       "knowledge-guide.pdf",
				ContentType:    "application/pdf",
				SizeBytes:      9,
				ChecksumSHA256: "abc123",
				CreatedAt:      now,
			}, nil
		},
	}
	queue := &uploadQueue{err: errors.New("redis unavailable")}
	svc := service.NewWithDependencies(repo, files, queue, func() time.Time { return now }, func(prefix string) string {
		return prefix + "_test"
	})

	_, err := svc.UploadDocument(context.Background(), service.RequestContext{
		RequestID:     "req_upload",
		UserID:        "usr_1",
		Permissions:   []string{service.PermissionKnowledgeWrite},
		CallerService: "gateway",
	}, service.UploadDocumentInput{
		KnowledgeBaseID: "kb_1",
		File: service.UploadedFile{
			Filename:    "knowledge-guide.pdf",
			ContentType: "application/pdf",
			SizeBytes:   9,
			Content:     bytes.NewReader([]byte("pdf-bytes")),
		},
	})
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("UploadDocument() error = %v", err)
	}
	if len(repo.markFailedCalls) != 1 {
		t.Fatalf("mark failed calls = %+v", repo.markFailedCalls)
	}
	if repo.markFailedCalls[0].DocumentID != "doc_test" || repo.markFailedCalls[0].JobID != "job_test" {
		t.Fatalf("mark failed call = %+v", repo.markFailedCalls[0])
	}
	doc, err := repo.GetDocument(context.Background(), "doc_test", service.AccessScope{UserID: "usr_1", CanWrite: true})
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusFailed || doc.ErrorCode == nil || *doc.ErrorCode != string(service.CodeDependency) {
		t.Fatalf("failed doc = %+v", doc)
	}
}

func TestDeleteDocumentKeepsDocumentHiddenWhenCleanupQueueHandoffFails(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 30, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	queue := &uploadQueue{err: errors.New("redis unavailable with secret file_1")}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_test"
	})

	err := svc.DeleteDocument(context.Background(), writeContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	if queue.cleanupCalls != 1 || queue.cleanupTask.DocumentID != "doc_1" {
		t.Fatalf("cleanup task = %+v calls=%d", queue.cleanupTask, queue.cleanupCalls)
	}
	_, err = svc.GetDocument(context.Background(), readContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocument() after queue failure error = %v", err)
	}
	job, err := repo.GetProcessingJob(context.Background(), "job_test")
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.ErrorMessage == nil || *job.ErrorMessage != "delete cleanup queue handoff failed" {
		t.Fatalf("job = %+v", job)
	}
	if strings.Contains(*job.ErrorMessage, "file_1") || strings.Contains(*job.ErrorMessage, "secret") {
		t.Fatalf("job error leaked sensitive detail: %q", *job.ErrorMessage)
	}
}

func TestDeleteCleanupReconcilerRequeuesQueueHandoffFailure(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 30, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	queue := &uploadQueue{err: errors.New("redis unavailable with secret file_1")}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_test"
	})

	err := svc.DeleteDocument(context.Background(), writeContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	queue.err = nil

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if err != nil {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v", err)
	}
	if result.Scanned != 1 || result.Enqueued != 1 || result.Failed != 0 {
		t.Fatalf("requeue result = %+v", result)
	}
	if queue.cleanupCalls != 2 {
		t.Fatalf("cleanup queue calls = %d, want initial failed call plus reconciler call", queue.cleanupCalls)
	}
	task := queue.cleanupTask
	if task.RequestID != "req_reconcile" ||
		task.JobID != "job_test" ||
		task.DocumentID != "doc_1" ||
		task.KnowledgeBaseID != "kb_1" ||
		task.UserID != "usr_1" {
		t.Fatalf("cleanup task = %+v", task)
	}
	_, err = svc.GetDocument(context.Background(), readContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocument() after requeue error = %v", err)
	}
}

func TestDeleteDocumentQueuesDecodableCleanupTaskWithoutRequestID(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 40, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_test"
	})

	if err := svc.DeleteDocument(context.Background(), writeContext("usr_1"), "doc_1"); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	payload, err := json.Marshal(queue.cleanupTask)
	if err != nil {
		t.Fatalf("marshal cleanup task: %v", err)
	}
	decoded, err := worker.DecodeDeleteCleanupPayload(payload)
	if err != nil {
		t.Fatalf("DecodeDeleteCleanupPayload(%s) error = %v", string(payload), err)
	}
	if decoded.RequestID == "" || decoded.JobID != "job_test" || decoded.DocumentID != "doc_1" || decoded.KnowledgeBaseID != "kb_1" || decoded.UserID != "usr_1" {
		t.Fatalf("decoded cleanup task = %+v", decoded)
	}
}

func TestDeleteCleanupReconcilerSkipsPermanentConflictFailure(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 30, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_conflict",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: service.DefaultIngestionMaxAttempts,
		DeletedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, service.AccessScope{UserID: "usr_1", CanWrite: true}); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}
	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_1", "job_conflict", nil, string(service.CodeConflict), "delete cleanup target mismatch", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed() error = %v", err)
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_test"
	})

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if err != nil {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v", err)
	}
	if result.Scanned != 0 || result.Enqueued != 0 || result.Failed != 0 || queue.cleanupCalls != 0 {
		t.Fatalf("result = %+v cleanupCalls=%d", result, queue.cleanupCalls)
	}
}

func TestDeleteCleanupReconcilerRequeuesRetryableAuthFailures(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 45, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_unauthorized",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: service.DefaultIngestionMaxAttempts,
		DeletedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, service.AccessScope{UserID: "usr_1", CanWrite: true}); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}
	if err := repo.MarkDocumentJobFailed(context.Background(), "doc_1", "job_unauthorized", nil, string(service.CodeUnauthorized), "file service rejected knowledge request", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkDocumentJobFailed() error = %v", err)
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_test"
	})

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if err != nil {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v", err)
	}
	if result.Scanned != 1 || result.Enqueued != 1 || result.Failed != 0 || queue.cleanupCalls != 1 {
		t.Fatalf("result = %+v cleanupCalls=%d", result, queue.cleanupCalls)
	}
	if queue.cleanupTask.JobID != "job_unauthorized" || queue.cleanupTask.RequestID != "req_reconcile" {
		t.Fatalf("cleanup task = %+v", queue.cleanupTask)
	}
}

func TestDeleteCleanupReconcilerRequeuesExhaustedStaleRunningJobForTerminalization(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 45, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_stale_running",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: 1,
		DeletedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, service.AccessScope{UserID: "usr_1", CanWrite: true}); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}
	stage := "delete_cleanup"
	attempts := int32(1)
	startedAt := now.Add(-time.Hour)
	if _, err := repo.UpdateJobState(context.Background(), "job_stale_running", service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &stage,
		ProgressPercent: 20,
		Attempts:        &attempts,
		StartedAt:       &startedAt,
		UpdatedAt:       now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("UpdateJobState() error = %v", err)
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now
	}, func(prefix string) string {
		return prefix + "_unused"
	}, service.WithIngestionRunningLease(5*time.Minute))

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if err != nil {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v", err)
	}
	if result.Scanned != 1 || result.Enqueued != 1 || result.Failed != 0 || queue.cleanupCalls != 1 {
		t.Fatalf("result = %+v cleanupCalls=%d", result, queue.cleanupCalls)
	}
	if queue.cleanupTask.JobID != "job_stale_running" ||
		queue.cleanupTask.DocumentID != "doc_1" ||
		queue.cleanupTask.RequestID != "req_reconcile" {
		t.Fatalf("cleanup task = %+v", queue.cleanupTask)
	}
}

func TestDeleteCleanupReconcilerDoesNotOverwriteSucceededJobWhenQueueHandoffFails(t *testing.T) {
	now := time.Date(2026, 6, 30, 8, 50, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_cleanup",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: service.DefaultIngestionMaxAttempts,
		DeletedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, service.AccessScope{UserID: "usr_1", CanWrite: true}); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}
	queue := &uploadQueue{
		err: errors.New("redis unavailable"),
		cleanupHook: func(ctx context.Context, task service.DocumentDeleteCleanupTask) {
			finishedAt := now.Add(time.Minute)
			stage := "completed"
			message := "document delete cleanup completed"
			if _, err := repo.UpdateJobState(ctx, task.JobID, service.JobStateUpdate{
				Status:          service.JobStatusSucceeded,
				CurrentStage:    &stage,
				ProgressPercent: 100,
				Message:         &message,
				FinishedAt:      &finishedAt,
				UpdatedAt:       finishedAt,
			}); err != nil {
				t.Fatalf("UpdateJobState(succeeded) error = %v", err)
			}
		},
	}
	svc := service.NewWithDependencies(repo, nil, queue, func() time.Time {
		return now.Add(2 * time.Minute)
	}, func(prefix string) string {
		return prefix + "_unused"
	})

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v, want dependency", err)
	}
	if result.Scanned != 1 || result.Enqueued != 0 || result.Failed != 1 || queue.cleanupCalls != 1 {
		t.Fatalf("result = %+v cleanupCalls=%d", result, queue.cleanupCalls)
	}
	job, err := repo.GetProcessingJob(context.Background(), "job_cleanup")
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.ErrorCode != nil || job.ErrorMessage != nil {
		t.Fatalf("succeeded cleanup job was overwritten: %+v", job)
	}
}

func TestDeleteCleanupReconcilerReportsRepositoryDependency(t *testing.T) {
	repo := &uploadRepository{
		MemoryRepository: repository.NewMemoryRepository(),
		listRetryableErr: errors.New("postgres unavailable"),
	}
	queue := &uploadQueue{}
	svc := service.NewWithDependencies(repo, nil, queue, fixedClock(), func(prefix string) string {
		return prefix + "_test"
	})

	result, err := svc.RequeueDeleteCleanupTasks(context.Background(), service.RequestContext{RequestID: "req_reconcile"}, 10)
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("RequeueDeleteCleanupTasks() error = %v", err)
	}
	if result.FailedDependency != "postgres" {
		t.Fatalf("failed dependency = %q", result.FailedDependency)
	}
}

func TestDocumentLifecycleUpdateDeleteChunksAndContent(t *testing.T) {
	now := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	repo := repository.NewMemoryRepository()
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
		Status:          service.DocumentStatusUploaded,
		Tags:            []string{"旧标签"},
		CreatedBy:       "usr_1",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	tokenCount := int32(12)
	repo.SeedDocumentChunk(service.DocumentChunk{
		ID:              "chunk_1",
		KnowledgeBaseID: "kb_1",
		DocumentID:      "doc_1",
		ChunkIndex:      0,
		Content:         "第一段",
		TokenCount:      &tokenCount,
		Metadata:        map[string]any{"source": "parser"},
		CreatedAt:       now,
	})
	files := &uploadFileClient{
		contentFn: func(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.FileContent, error) {
			if fileID != "file_1" || reqCtx.RequestID != "req_content" {
				t.Fatalf("content call fileID=%q reqCtx=%+v", fileID, reqCtx)
			}
			return service.FileContent{
				Content:     io.NopCloser(strings.NewReader("pdf-bytes")),
				ContentType: "application/pdf",
				SizeBytes:   9,
			}, nil
		},
	}
	queue := &uploadQueue{}
	vector := &lifecycleVectorIndex{hits: []service.VectorSearchHit{{
		ID:      "point_1",
		Score:   1,
		Payload: map[string]any{"chunk_id": "chunk_1"},
	}}}
	svc := service.NewWithDependencies(repo, files, queue, func() time.Time {
		return now.Add(time.Hour)
	}, func(prefix string) string {
		return prefix + "_cleanup"
	}, service.WithProcessingPipeline(files, nil, nil), service.WithVectorIndex(lifecycleEmbedder{}, vector))

	updatedTags := []string{"锅炉", "锅炉", " 规程 "}
	updated, err := svc.UpdateDocument(context.Background(), writeContext("usr_1"), service.UpdateDocumentInput{
		ID:   "doc_1",
		Tags: &updatedTags,
	})
	if err != nil {
		t.Fatalf("UpdateDocument() error = %v", err)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "锅炉" || updated.Tags[1] != "规程" {
		t.Fatalf("updated tags = %+v", updated.Tags)
	}

	chunks, err := svc.ListDocumentChunks(context.Background(), readContext("usr_1"), service.ListDocumentChunksInput{DocumentID: "doc_1"})
	if err != nil {
		t.Fatalf("ListDocumentChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || len(chunks.Items) != 1 || chunks.Items[0].Content != "第一段" {
		t.Fatalf("chunks = %+v", chunks)
	}

	content, err := svc.GetDocumentContent(context.Background(), service.RequestContext{
		RequestID: "req_content",
		UserID:    "usr_1",
	}, "doc_1")
	if err != nil {
		t.Fatalf("GetDocumentContent() error = %v", err)
	}
	body, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	_ = content.Body.Close()
	if string(body) != "pdf-bytes" || content.ContentType != "application/pdf" {
		t.Fatalf("content = %q type=%q", string(body), content.ContentType)
	}

	if err := svc.DeleteDocument(context.Background(), writeContext("usr_1"), "doc_1"); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}
	if queue.cleanupCalls != 1 ||
		queue.cleanupTask.JobID != "job_cleanup" ||
		queue.cleanupTask.DocumentID != "doc_1" ||
		queue.cleanupTask.KnowledgeBaseID != "kb_1" ||
		queue.cleanupTask.UserID != "usr_1" {
		t.Fatalf("cleanup queue task = %+v calls=%d", queue.cleanupTask, queue.cleanupCalls)
	}
	_, err = svc.GetDocument(context.Background(), readContext("usr_1"), "doc_1")
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocument() after delete error = %v", err)
	}
	list, err := svc.ListDocuments(context.Background(), readContext("usr_1"), service.ListDocumentsInput{KnowledgeBaseID: "kb_1"})
	if err != nil {
		t.Fatalf("ListDocuments() after delete error = %v", err)
	}
	if list.Page.Total != 0 || len(list.Items) != 0 {
		t.Fatalf("ListDocuments() after delete = %+v, want empty", list)
	}
	_, err = svc.ListDocumentChunks(context.Background(), readContext("usr_1"), service.ListDocumentChunksInput{DocumentID: "doc_1"})
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("ListDocumentChunks() after delete error = %v", err)
	}
	_, err = svc.GetDocumentContent(context.Background(), service.RequestContext{RequestID: "req_content", UserID: "usr_1"}, "doc_1")
	if !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocumentContent() after delete error = %v", err)
	}
	query, err := svc.CreateKnowledgeQuery(context.Background(), readContext("usr_1"), service.KnowledgeQueryInput{
		Query:            "第一段",
		KnowledgeBaseIDs: []string{"kb_1"},
		TopK:             1,
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeQuery() after delete error = %v", err)
	}
	if len(query.Results) != 0 {
		t.Fatalf("query results after delete = %+v, want empty", query.Results)
	}
}

type uploadFileClient struct {
	createFn  func(context.Context, service.RequestContext, service.UploadedFile) (service.FileObject, error)
	deleteFn  func(context.Context, service.RequestContext, string) error
	contentFn func(context.Context, service.RequestContext, string) (service.FileContent, error)
	deleted   []string
}

func (f *uploadFileClient) CreateFile(ctx context.Context, reqCtx service.RequestContext, file service.UploadedFile) (service.FileObject, error) {
	if f.createFn != nil {
		return f.createFn(ctx, reqCtx, file)
	}
	return service.FileObject{}, nil
}

func (f *uploadFileClient) DeleteFile(ctx context.Context, reqCtx service.RequestContext, fileID string) error {
	f.deleted = append(f.deleted, fileID)
	if f.deleteFn != nil {
		return f.deleteFn(ctx, reqCtx, fileID)
	}
	return nil
}

func (f *uploadFileClient) GetFileContent(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.FileContent, error) {
	if f.contentFn != nil {
		return f.contentFn(ctx, reqCtx, fileID)
	}
	return service.FileContent{}, service.NotFoundError("file content not found")
}

func (f *uploadFileClient) ReadSource(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.SourceDocument, error) {
	content, err := f.GetFileContent(ctx, reqCtx, fileID)
	if err != nil {
		return service.SourceDocument{}, err
	}
	return service.SourceDocument{
		Body:        content.Content,
		ContentType: content.ContentType,
		SizeBytes:   content.SizeBytes,
	}, nil
}

type uploadQueue struct {
	task         service.DocumentIngestionTask
	cleanupTask  service.DocumentDeleteCleanupTask
	calls        int
	cleanupCalls int
	err          error
	cleanupHook  func(context.Context, service.DocumentDeleteCleanupTask)
}

func (q *uploadQueue) EnqueueDocumentIngestion(ctx context.Context, task service.DocumentIngestionTask) error {
	q.calls++
	q.task = task
	return q.err
}

func (q *uploadQueue) EnqueueDocumentDeleteCleanup(ctx context.Context, task service.DocumentDeleteCleanupTask) error {
	q.cleanupCalls++
	q.cleanupTask = task
	if q.cleanupHook != nil {
		q.cleanupHook(ctx, task)
	}
	return q.err
}

type lifecycleEmbedder struct{}

func (lifecycleEmbedder) Embed(context.Context, service.EmbeddingRequest) (service.EmbeddingResult, error) {
	return service.EmbeddingResult{Vectors: [][]float32{{1, 0}}, Provider: "fake", Model: "fake", Dimension: 2}, nil
}

type lifecycleVectorIndex struct {
	hits []service.VectorSearchHit
}

func (*lifecycleVectorIndex) Upsert(context.Context, []service.VectorPoint) error { return nil }
func (*lifecycleVectorIndex) DeleteByDocument(context.Context, string) error      { return nil }
func (*lifecycleVectorIndex) DeleteByDocumentIngestionAttempt(context.Context, string, string) error {
	return nil
}
func (*lifecycleVectorIndex) DeleteStaleDocumentPoints(context.Context, string, string) error {
	return nil
}
func (i *lifecycleVectorIndex) Search(context.Context, service.VectorSearchRequest) ([]service.VectorSearchHit, error) {
	return append([]service.VectorSearchHit(nil), i.hits...), nil
}

type uploadRepository struct {
	*repository.MemoryRepository
	createErr        error
	listRetryableErr error
	lastCreate       service.CreateDocumentWithJobRecord
	markFailedCalls  []markFailedCall
}

type markFailedCall struct {
	DocumentID string
	JobID      string
	Code       string
	Message    string
}

func (r *uploadRepository) CreateDocumentWithJob(ctx context.Context, input service.CreateDocumentWithJobRecord, scope service.AccessScope) (service.KnowledgeDocument, service.ProcessingJob, error) {
	r.lastCreate = input
	if r.createErr != nil {
		return service.KnowledgeDocument{}, service.ProcessingJob{}, r.createErr
	}
	return r.MemoryRepository.CreateDocumentWithJob(ctx, input, scope)
}

func (r *uploadRepository) MarkDocumentJobFailed(ctx context.Context, documentID string, jobID string, expectedAttempts *int32, code string, message string, failedAt time.Time) error {
	r.markFailedCalls = append(r.markFailedCalls, markFailedCall{
		DocumentID: documentID,
		JobID:      jobID,
		Code:       code,
		Message:    message,
	})
	return r.MemoryRepository.MarkDocumentJobFailed(ctx, documentID, jobID, expectedAttempts, code, message, failedAt)
}

func (r *uploadRepository) ListRetryableDeleteCleanupTasks(ctx context.Context, input service.DeleteCleanupTaskListInput) ([]service.DocumentDeleteCleanupTask, error) {
	if r.listRetryableErr != nil {
		return nil, r.listRetryableErr
	}
	return r.MemoryRepository.ListRetryableDeleteCleanupTasks(ctx, input)
}

func writeContext(userID string) service.RequestContext {
	return service.RequestContext{UserID: userID, Permissions: []string{service.PermissionKnowledgeWrite}}
}

func readContext(userID string) service.RequestContext {
	return service.RequestContext{UserID: userID}
}

func fixedClock() service.Clock {
	return func() time.Time {
		return time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	}
}

func seedUploadParserConfig(repo *repository.MemoryRepository, now time.Time) {
	repo.SeedParserConfig(service.ParserConfig{
		ID:                    "parser_default",
		Name:                  "Default builtin parser",
		Backend:               service.ParserBackendBuiltin,
		Enabled:               true,
		IsDefault:             true,
		Concurrency:           4,
		SupportedContentTypes: []string{"application/pdf"},
		DefaultParameters:     json.RawMessage(`{}`),
		CreatedAt:             now,
		UpdatedAt:             now,
	})
}

func hasCode(err error, code service.Code) bool {
	var appErr *service.AppError
	return errors.As(err, &appErr) && appErr.Code == code
}
