package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/embedding"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/worker"
)

func TestIngestionHandlerRejectsInvalidA10PayloadWithoutTouchingState(t *testing.T) {
	handler, svc, repo, _ := newWorkerHarness(t, newSourceStore())
	handoff := seedIngestionJob(t, repo, "file_123")

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, map[string]string{
		"requestId": "req_worker",
		"jobId":     handoff.jobID,
		"userId":    "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v, want ack for permanent payload error", err)
	}
	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusQueued {
		t.Fatalf("job status = %s, want queued", job.Status)
	}
}

func TestIngestionHandlerProcessesA10PayloadFromFileServiceToReady(t *testing.T) {
	source := newSourceStore()
	source.Put("file_123", "# Intro\n\nThis is enough content for a text chunk.", "text/markdown")
	handler, svc, repo, vectors := newWorkerHarness(t, source)
	handoff := seedIngestionJob(t, repo, "file_123")

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v", err)
	}

	if source.lastRequest.UserID != "usr_123" || source.lastRequest.RequestID != "req_worker" || source.lastRequest.CallerService != "knowledge" {
		t.Fatalf("source request context = %+v", source.lastRequest)
	}
	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.ProgressPercent != 100 {
		t.Fatalf("job = %+v", job)
	}
	doc, err := svc.GetDocument(context.Background(), actorContext(), handoff.documentID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusReady || doc.ChunkCount != 1 {
		t.Fatalf("doc = %+v", doc)
	}
	if doc.ParserBackend == nil || *doc.ParserBackend != "parser-service" {
		t.Fatalf("parser backend = %v, want parser-service", doc.ParserBackend)
	}
	chunks, err := svc.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.documentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || chunks.Items[0].QdrantPointID == nil {
		t.Fatalf("chunks = %+v", chunks)
	}
	if chunks.Items[0].Metadata["page_start"] != 1 || chunks.Items[0].Metadata["page_end"] != 1 {
		t.Fatalf("chunk page metadata = %+v", chunks.Items[0].Metadata)
	}
	sourcePages, ok := chunks.Items[0].Metadata["source_pages"].([]int)
	if !ok || len(sourcePages) != 1 || sourcePages[0] != 1 {
		t.Fatalf("chunk source pages metadata = %+v", chunks.Items[0].Metadata["source_pages"])
	}
	if len(vectors.points) != 1 {
		t.Fatalf("vector points = %+v", vectors.points)
	}
	if chunks.Items[0].Metadata["parse_strategy"] != "ocr" ||
		chunks.Items[0].Metadata["text_layer_status"] != "broken" ||
		chunks.Items[0].Metadata["dpi"] != 180 {
		t.Fatalf("chunk parser metadata = %+v", chunks.Items[0].Metadata)
	}
	if confidence, ok := chunks.Items[0].Metadata["ocr_confidence"].(float64); !ok || confidence != 0.91 {
		t.Fatalf("chunk confidence metadata = %+v", chunks.Items[0].Metadata["ocr_confidence"])
	}
	warnings, ok := chunks.Items[0].Metadata["parse_warnings"].([]string)
	if !ok || len(warnings) != 1 || warnings[0] != "low_text_quality" {
		t.Fatalf("chunk warnings metadata = %+v", chunks.Items[0].Metadata["parse_warnings"])
	}
	assertMinimalVectorPayload(t, vectors.points[0].Payload)
}

func TestIngestionHandlerFailsSourceReadSafely(t *testing.T) {
	source := newSourceStore()
	source.err = service.NewError(service.CodeDependency, "file service content read failed", errors.New("secret object key"))
	handler, svc, repo, vectors := newWorkerHarness(t, source)
	handoff := seedIngestionJob(t, repo, "file_missing")

	err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	}))

	appErr := requireAppError(t, err, service.CodeDependency)
	if strings.Contains(appErr.Error(), "secret") {
		t.Fatalf("error leaked sensitive detail: %v", appErr)
	}
	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.ErrorCode == nil || *job.ErrorCode != string(service.CodeDependency) {
		t.Fatalf("job = %+v", job)
	}
	doc, err := svc.GetDocument(context.Background(), actorContext(), handoff.documentID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusFailed || doc.ErrorMessage == nil || strings.Contains(*doc.ErrorMessage, "secret") {
		t.Fatalf("doc = %+v", doc)
	}
	if len(vectors.points) != 0 {
		t.Fatalf("vector points = %+v", vectors.points)
	}
}

func TestIngestionHandlerDoesNotReprocessSucceededJob(t *testing.T) {
	source := newSourceStore()
	source.Put("file_123", "content for exactly one processing run", "text/plain")
	handler, svc, repo, vectors := newWorkerHarness(t, source)
	handoff := seedIngestionJob(t, repo, "file_123")
	payload := mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})

	if err := handler.HandleIngestionPayload(context.Background(), payload); err != nil {
		t.Fatalf("first HandleIngestionPayload() error = %v", err)
	}
	if err := handler.HandleIngestionPayload(context.Background(), payload); err != nil {
		t.Fatalf("second HandleIngestionPayload() error = %v, want ack for succeeded job redelivery", err)
	}
	chunks, err := svc.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.documentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || len(vectors.points) != 1 {
		t.Fatalf("chunks = %+v, vectors = %+v", chunks, vectors.points)
	}
	if source.readCount != 1 {
		t.Fatalf("source reads = %d, want 1", source.readCount)
	}
}

func TestIngestionHandlerAtomicallyClaimsDuplicateDeliveries(t *testing.T) {
	source := newBlockingSourceStore("content for exactly one concurrent processing run", "text/plain")
	handler, svc, repo, vectors := newWorkerHarness(t, source)
	handoff := seedIngestionJob(t, repo, "file_123")
	payload := mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	errCh := make(chan error, 2)
	go func() {
		errCh <- handler.HandleIngestionPayload(ctx, payload)
	}()
	select {
	case <-source.entered:
	case <-time.After(time.Second):
		t.Fatal("first worker did not reach source read")
	}

	go func() {
		errCh <- handler.HandleIngestionPayload(ctx, payload)
	}()
	select {
	case err := <-errCh:
		requireAppError(t, err, service.CodeDependency)
	case <-time.After(300 * time.Millisecond):
		close(source.release)
		t.Fatal("duplicate delivery blocked instead of returning retryable running state")
	}

	close(source.release)
	if err := <-errCh; err != nil {
		t.Fatalf("claimed HandleIngestionPayload() error = %v", err)
	}
	if err := handler.HandleIngestionPayload(context.Background(), payload); err != nil {
		t.Fatalf("completed redelivery error = %v, want ack for succeeded job", err)
	}
	if reads := source.reads.Load(); reads != 1 {
		t.Fatalf("source reads = %d, want 1", reads)
	}
	chunks, err := svc.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.documentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || len(vectors.points) != 1 {
		t.Fatalf("chunks = %+v, vectors = %+v", chunks, vectors.points)
	}
}

func TestIngestionHandlerReclaimsStaleRunningJob(t *testing.T) {
	source := newSourceStore()
	source.Put("file_123", "content for a stale running job redelivery", "text/plain")
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	vectors := &recordingVectorIndex{}
	now := fixedNow().Add(time.Hour)
	svc := service.NewWithDependencies(
		repo,
		nil,
		nil,
		func() time.Time { return now },
		sequenceIDs(),
		service.WithProcessingPipeline(source, fakeParser{}, service.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
		service.WithIngestionRunningLease(5*time.Minute),
	)
	handler := worker.NewIngestionHandler(svc)
	handoff := seedRunningIngestionJob(t, repo, "file_123", fixedNow())

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v", err)
	}

	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.Attempts != 2 {
		t.Fatalf("job = %+v", job)
	}
	doc, err := svc.GetDocument(context.Background(), actorContext(), handoff.documentID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusReady || doc.ParserBackend == nil || *doc.ParserBackend != "parser-service" {
		t.Fatalf("doc = %+v", doc)
	}
	if source.readCount != 1 || len(vectors.points) != 1 {
		t.Fatalf("source reads = %d, vector points = %+v", source.readCount, vectors.points)
	}
}

func TestIngestionAttemptFencingRejectsStaleCompletionAndFailure(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedRunningIngestionJob(t, repo, "file_123", fixedNow())
	attemptOne := int32(1)
	claimAt := fixedNow().Add(time.Hour)
	staleBefore := claimAt.Add(-5 * time.Minute)
	stage := "parsing"

	claimed, err := repo.ClaimProcessingJob(context.Background(), handoff.jobID, service.JobStateUpdate{
		Status:             service.JobStatusRunning,
		CurrentStage:       &stage,
		ProgressPercent:    20,
		StartedAt:          &claimAt,
		UpdatedAt:          claimAt,
		StaleRunningBefore: &staleBefore,
	})
	if err != nil {
		t.Fatalf("ClaimProcessingJob() error = %v", err)
	}
	if claimed.Attempts != 2 {
		t.Fatalf("claimed attempts = %d, want 2", claimed.Attempts)
	}

	parserBackend := "parser-service"
	finishedAt := claimAt.Add(time.Minute)
	if _, err := repo.CompleteIngestion(context.Background(), service.CompleteIngestionRecord{
		DocumentID:       handoff.documentID,
		JobID:            handoff.jobID,
		ExpectedAttempts: &claimed.Attempts,
		ParserBackend:    &parserBackend,
		Chunks: []service.DocumentChunk{{
			ID:              "chunk_attempt_2",
			KnowledgeBaseID: handoff.knowledgeBaseID,
			DocumentID:      handoff.documentID,
			ChunkIndex:      0,
			Content:         "new attempt content",
			CreatedAt:       finishedAt,
		}},
		UpdatedAt:  finishedAt,
		FinishedAt: finishedAt,
	}); err != nil {
		t.Fatalf("CompleteIngestion() for active attempt error = %v", err)
	}

	err = repo.MarkDocumentJobFailed(context.Background(), handoff.documentID, handoff.jobID, &attemptOne, "dependency_error", "old worker failure", finishedAt.Add(time.Minute))
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("MarkDocumentJobFailed() stale attempt error = %v, want ErrConflict", err)
	}
	_, err = repo.CompleteIngestion(context.Background(), service.CompleteIngestionRecord{
		DocumentID:       handoff.documentID,
		JobID:            handoff.jobID,
		ExpectedAttempts: &attemptOne,
		Chunks: []service.DocumentChunk{{
			ID:              "chunk_attempt_1",
			KnowledgeBaseID: handoff.knowledgeBaseID,
			DocumentID:      handoff.documentID,
			ChunkIndex:      0,
			Content:         "stale attempt content",
			CreatedAt:       finishedAt.Add(time.Minute),
		}},
		UpdatedAt:  finishedAt.Add(time.Minute),
		FinishedAt: finishedAt.Add(time.Minute),
	})
	if !errors.Is(err, service.ErrConflict) {
		t.Fatalf("CompleteIngestion() stale attempt error = %v, want ErrConflict", err)
	}

	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.Attempts != 2 {
		t.Fatalf("job = %+v", job)
	}
	doc, err := repo.GetDocument(context.Background(), handoff.documentID, service.AccessScope{CanReadAll: true})
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusReady || doc.ParserBackend == nil || *doc.ParserBackend != "parser-service" {
		t.Fatalf("doc = %+v", doc)
	}
	chunks, err := repo.ListChunks(context.Background(), handoff.documentID, service.AccessScope{CanReadAll: true}, service.PageInput{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || chunks.Items[0].ID != "chunk_attempt_2" {
		t.Fatalf("chunks = %+v", chunks)
	}
}

func TestIngestionHandlerCleansStaleAttemptVectorsAfterReclaim(t *testing.T) {
	source := newSourceStore()
	source.Put("file_123", "content for a stale vector write race", "text/plain")
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	vectors := &reclaimingVectorIndex{repo: repo}
	svc := service.NewWithDependencies(
		repo,
		nil,
		nil,
		fixedClock(),
		sequenceIDs(),
		service.WithProcessingPipeline(source, fakeParser{}, service.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
		service.WithIngestionRunningLease(5*time.Minute),
	)
	handler := worker.NewIngestionHandler(svc)
	handoff := seedIngestionJob(t, repo, "file_123")
	vectors.handoff = handoff

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v", err)
	}

	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.Attempts != 2 {
		t.Fatalf("job = %+v", job)
	}
	chunks, err := svc.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.documentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || chunks.Items[0].ID != "chunk_attempt_2" {
		t.Fatalf("chunks = %+v", chunks)
	}
	if len(vectors.points) != 1 {
		t.Fatalf("vector points = %+v", vectors.points)
	}
	if vectors.points[0].Payload[service.VectorPayloadIngestionAttempt] != "job_1:2" {
		t.Fatalf("remaining vector payload = %+v, want attempt 2 only", vectors.points[0].Payload)
	}
	if vectors.points[0].Payload["chunk_id"] != "chunk_attempt_2" {
		t.Fatalf("remaining vector payload = %+v, want attempt 2 chunk", vectors.points[0].Payload)
	}
}

func TestIngestionHandlerRetriesWhenFailureStateCannotPersist(t *testing.T) {
	source := newSourceStore()
	source.Put("file_empty", "", "text/plain")
	baseRepo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, baseRepo)
	repo := &markFailureRepository{
		MemoryRepository: baseRepo,
		err:              errors.New("temporary postgres outage"),
	}
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(
		repo,
		nil,
		nil,
		fixedClock(),
		sequenceIDs(),
		service.WithProcessingPipeline(source, failingParser{err: service.ValidationError("document parsing failed", map[string]string{"file": "could not be parsed"})}, service.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewIngestionHandler(svc)
	handoff := seedIngestionJob(t, baseRepo, "file_empty")

	err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	}))

	requireAppError(t, err, service.CodeDependency)
	job, getErr := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if getErr != nil {
		t.Fatalf("GetJob() error = %v", getErr)
	}
	if job.Status != service.JobStatusRunning {
		t.Fatalf("job status = %s, want running until failure state can be persisted", job.Status)
	}
}

func TestIngestionHandlerAcksPermanentParsingFailure(t *testing.T) {
	source := newSourceStore()
	source.Put("file_empty", "", "text/plain")
	handler, svc, repo, vectors := newWorkerHarnessWithParser(t, source, failingParser{err: service.ValidationError("document parsing failed", map[string]string{"file": "could not be parsed"})})
	handoff := seedIngestionJob(t, repo, "file_empty")

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v, want ack for permanent parsing failure", err)
	}

	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.ErrorCode == nil || *job.ErrorCode != "parse_failed" {
		t.Fatalf("job = %+v", job)
	}
	doc, err := svc.GetDocument(context.Background(), actorContext(), handoff.documentID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusFailed || doc.ErrorMessage == nil || strings.Contains(*doc.ErrorMessage, "file_empty") {
		t.Fatalf("doc = %+v", doc)
	}
	if len(vectors.points) != 0 {
		t.Fatalf("vector points = %+v", vectors.points)
	}
}

func TestIngestionHandlerAcksDependencyFailureAfterMaxAttempts(t *testing.T) {
	source := newSourceStore()
	source.err = service.NewError(service.CodeDependency, "file service content read failed", nil)
	handler, svc, repo, _ := newWorkerHarness(t, source)
	handoff := seedIngestionJobWithMaxAttempts(t, repo, "file_missing", 1)

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID:       "req_worker",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v, want ack once max attempts is reached", err)
	}

	job, err := svc.GetJob(context.Background(), actorContext(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.Attempts != 1 || job.MaxAttempts != 1 {
		t.Fatalf("job = %+v", job)
	}
}

func TestDeleteCleanupHandlerDeletesFileAndVectorsThenCompletesJob(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedDeleteCleanupJob(t, repo, "file_123")
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{points: []service.VectorPoint{
		{
			ID: "point_doc_1",
			Payload: map[string]any{
				service.VectorPayloadDocumentID:       handoff.documentID,
				service.VectorPayloadIngestionAttempt: "job_ingest:1",
			},
		},
		{
			ID: "point_other_doc",
			Payload: map[string]any{
				service.VectorPayloadDocumentID: "doc_other",
			},
		},
	}}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleDeleteCleanupPayload() error = %v", err)
	}

	if len(files.deleted) != 1 || files.deleted[0] != "file_123" {
		t.Fatalf("file delete calls = %+v", files.deleted)
	}
	if files.lastRequest.RequestID != "req_cleanup" || files.lastRequest.UserID != "usr_123" || files.lastRequest.CallerService != "knowledge" {
		t.Fatalf("file request context = %+v", files.lastRequest)
	}
	if len(vectors.deletedDocuments) != 1 || vectors.deletedDocuments[0] != handoff.documentID {
		t.Fatalf("vector delete documents = %+v", vectors.deletedDocuments)
	}
	if len(vectors.points) != 1 || vectors.points[0].ID != "point_other_doc" {
		t.Fatalf("remaining vector points = %+v", vectors.points)
	}
	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.ProgressPercent != 100 {
		t.Fatalf("job = %+v", job)
	}
	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("redelivered HandleDeleteCleanupPayload() error = %v", err)
	}
	if len(files.deleted) != 1 || len(vectors.deletedDocuments) != 1 {
		t.Fatalf("duplicate delivery repeated cleanup: file=%+v vector=%+v", files.deleted, vectors.deletedDocuments)
	}
	_, err = svc.GetDocument(context.Background(), actorContext(), handoff.documentID)
	if !errors.Is(err, service.ErrNotFound) && !hasAppCode(err, service.CodeNotFound) {
		t.Fatalf("GetDocument() after cleanup error = %v, want not found", err)
	}
}

func TestDeleteCleanupHandlerTreatsEmptyFileRefAsAlreadyCleaned(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedDeleteCleanupJob(t, repo, "")
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleDeleteCleanupPayload() error = %v", err)
	}
	if len(files.deleted) != 0 {
		t.Fatalf("file delete calls = %+v, want none", files.deleted)
	}
	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded {
		t.Fatalf("job status = %s, want succeeded", job.Status)
	}
}

func TestDeleteCleanupHandlerPersistsSanitizedFileFailureForRetry(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedDeleteCleanupJob(t, repo, "file_123")
	files := &cleanupFileClient{
		err: service.NewError(service.CodeDependency, "file service failed with bucket secret-bucket object secret-key", errors.New("service token secret")),
	}
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	}))
	requireAppError(t, err, service.CodeDependency)
	job, getErr := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if getErr != nil {
		t.Fatalf("GetProcessingJob() error = %v", getErr)
	}
	if job.Status != service.JobStatusFailed || job.ErrorMessage == nil || *job.ErrorMessage != "file cleanup failed" {
		t.Fatalf("job = %+v", job)
	}
	for _, forbidden := range []string{"file_123", "secret-bucket", "secret-key", "service token"} {
		if strings.Contains(*job.ErrorMessage, forbidden) {
			t.Fatalf("job error leaked %q: %q", forbidden, *job.ErrorMessage)
		}
	}
	if len(vectors.deletedDocuments) != 0 {
		t.Fatalf("vector cleanup should not run after file failure: %+v", vectors.deletedDocuments)
	}
}

func TestDeleteCleanupHandlerPersistsVectorFailureForRetry(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedDeleteCleanupJob(t, repo, "file_123")
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{
		points: []service.VectorPoint{{
			ID: "point_doc_1",
			Payload: map[string]any{
				service.VectorPayloadDocumentID: handoff.documentID,
			},
		}},
		deleteErr: service.NewError(service.CodeDependency, "qdrant failed with raw vector payload", errors.New("secret vector payload")),
	}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	}))
	requireAppError(t, err, service.CodeDependency)
	if len(files.deleted) != 1 {
		t.Fatalf("file delete calls = %+v, want one before vector retry", files.deleted)
	}
	job, getErr := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if getErr != nil {
		t.Fatalf("GetProcessingJob() error = %v", getErr)
	}
	if job.Status != service.JobStatusFailed || job.ErrorMessage == nil || *job.ErrorMessage != "vector cleanup failed" {
		t.Fatalf("job = %+v", job)
	}
	if strings.Contains(*job.ErrorMessage, "raw vector") || strings.Contains(*job.ErrorMessage, "secret") {
		t.Fatalf("job error leaked sensitive detail: %q", *job.ErrorMessage)
	}
}

func TestDeleteCleanupHandlerRetriesWhenJobIsAlreadyRunning(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedRunningDeleteCleanupJob(t, repo, "file_123", fixedNow())
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
		service.WithIngestionRunningLease(5*time.Minute),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	}))

	requireAppError(t, err, service.CodeDependency)
	if len(files.deleted) != 0 || len(vectors.deletedDocuments) != 0 {
		t.Fatalf("running job should not perform cleanup: file=%+v vector=%+v", files.deleted, vectors.deletedDocuments)
	}
	job, getErr := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if getErr != nil {
		t.Fatalf("GetProcessingJob() error = %v", getErr)
	}
	if job.Status != service.JobStatusRunning || job.Attempts != 1 {
		t.Fatalf("job = %+v, want running attempt 1", job)
	}
}

func TestDeleteCleanupHandlerReclaimsStaleRunningJob(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedRunningDeleteCleanupJob(t, repo, "file_123", fixedNow())
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{}
	now := fixedNow().Add(time.Hour)
	svc := service.NewWithDependencies(repo, files, nil, func() time.Time { return now }, sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
		service.WithIngestionRunningLease(5*time.Minute),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleDeleteCleanupPayload() error = %v", err)
	}

	if len(files.deleted) != 1 || files.deleted[0] != "file_123" {
		t.Fatalf("file delete calls = %+v", files.deleted)
	}
	if len(vectors.deletedDocuments) != 1 || vectors.deletedDocuments[0] != handoff.documentID {
		t.Fatalf("vector delete documents = %+v", vectors.deletedDocuments)
	}
	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.Attempts != 2 {
		t.Fatalf("job = %+v, want succeeded attempt 2", job)
	}
}

func TestDeleteCleanupHandlerAcksDependencyFailureAfterMaxAttempts(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedDeleteCleanupJobWithMaxAttempts(t, repo, "file_123", 1)
	files := &cleanupFileClient{
		err: service.NewError(service.CodeDependency, "file service failed with bucket secret-bucket object secret-key", errors.New("service token secret")),
	}
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(repo, files, nil, fixedClock(), sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleDeleteCleanupPayload() error = %v, want ack once max attempts is reached", err)
	}

	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.Attempts != 1 || job.MaxAttempts != 1 {
		t.Fatalf("job = %+v", job)
	}
	if job.ErrorCode == nil || *job.ErrorCode != string(service.CodeDependency) {
		t.Fatalf("job error code = %v, want dependency_error", job.ErrorCode)
	}
	if job.ErrorMessage == nil || *job.ErrorMessage != "file cleanup failed" || strings.Contains(*job.ErrorMessage, "secret") {
		t.Fatalf("job error message = %v", job.ErrorMessage)
	}
	if len(vectors.deletedDocuments) != 0 {
		t.Fatalf("vector cleanup should not run after file failure: %+v", vectors.deletedDocuments)
	}
}

func TestDeleteCleanupHandlerTerminalizesExhaustedStaleRunningJob(t *testing.T) {
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	handoff := seedRunningDeleteCleanupJob(t, repo, "file_123", fixedNow())
	files := &cleanupFileClient{}
	vectors := &recordingVectorIndex{}
	exhaustedAttempts := service.DefaultIngestionMaxAttempts
	stage := "delete_cleanup"
	startedAt := fixedNow().Add(-time.Hour)
	if _, err := repo.UpdateJobState(context.Background(), handoff.jobID, service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &stage,
		ProgressPercent: 20,
		Attempts:        &exhaustedAttempts,
		StartedAt:       &startedAt,
		UpdatedAt:       startedAt,
	}); err != nil {
		t.Fatalf("UpdateJobState() error = %v", err)
	}
	svc := service.NewWithDependencies(repo, files, nil, func() time.Time {
		return fixedNow()
	}, sequenceIDs(),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
		service.WithIngestionRunningLease(5*time.Minute),
	)
	handler := worker.NewDeleteCleanupHandler(svc)

	if err := handler.HandleDeleteCleanupPayload(context.Background(), mustJSON(t, worker.DeleteCleanupPayload{
		RequestID:       "req_cleanup",
		JobID:           handoff.jobID,
		DocumentID:      handoff.documentID,
		KnowledgeBaseID: handoff.knowledgeBaseID,
		UserID:          "usr_123",
	})); err != nil {
		t.Fatalf("HandleDeleteCleanupPayload() error = %v, want ack once stale running job is exhausted", err)
	}
	if len(files.deleted) != 0 || len(vectors.deletedDocuments) != 0 {
		t.Fatalf("exhausted stale running job should not perform cleanup: file=%+v vector=%+v", files.deleted, vectors.deletedDocuments)
	}
	job, err := repo.GetProcessingJob(context.Background(), handoff.jobID)
	if err != nil {
		t.Fatalf("GetProcessingJob() error = %v", err)
	}
	if job.Status != service.JobStatusFailed || job.Attempts != service.DefaultIngestionMaxAttempts {
		t.Fatalf("job = %+v", job)
	}
	if job.ErrorMessage == nil || *job.ErrorMessage != "delete cleanup job reached max attempts" {
		t.Fatalf("job error message = %v", job.ErrorMessage)
	}
}

func TestDecodeIngestionPayloadRejectsUnknownFields(t *testing.T) {
	_, err := worker.DecodeIngestionPayload([]byte(`{"requestId":"req","jobId":"job","documentId":"doc","knowledgeBaseId":"kb","userId":"usr","fileRef":"secret"}`))
	requireAppError(t, err, service.CodeValidation)
}

func TestDecodeDeleteCleanupPayloadRejectsUnknownFields(t *testing.T) {
	_, err := worker.DecodeDeleteCleanupPayload([]byte(`{"requestId":"req","jobId":"job","documentId":"doc","knowledgeBaseId":"kb","userId":"usr","fileRef":"secret"}`))
	requireAppError(t, err, service.CodeValidation)
}

func newWorkerHarness(t *testing.T, source service.SourceReader) (*worker.IngestionHandler, *service.Service, *repository.MemoryRepository, *recordingVectorIndex) {
	t.Helper()
	return newWorkerHarnessWithParser(t, source, fakeParser{})
}

func newWorkerHarnessWithParser(t *testing.T, source service.SourceReader, parser service.Parser) (*worker.IngestionHandler, *service.Service, *repository.MemoryRepository, *recordingVectorIndex) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	seedKnowledgeBase(t, repo)
	vectors := &recordingVectorIndex{}
	svc := service.NewWithDependencies(
		repo,
		nil,
		nil,
		fixedClock(),
		sequenceIDs(),
		service.WithProcessingPipeline(source, parser, service.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	return worker.NewIngestionHandler(svc), svc, repo, vectors
}

type fakeParser struct{}

func (p fakeParser) Parse(ctx context.Context, input service.ParseInput) (service.ParsedDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.ParsedDocument{}, err
	}
	return service.ParsedDocument{
		Content: "Parsed document content from parser service",
		Title:   "Parsed document",
		Backend: "parser-service",
		Pages: []service.ParsedPage{
			{
				PageNumber:      1,
				Content:         "Parsed document content from parser service",
				ParseStrategy:   "ocr",
				TextLayerStatus: "broken",
				OCRConfidence:   ptrFloat64(0.91),
				DPI:             ptrInt(180),
				Warnings:        []string{"low_text_quality"},
			},
		},
	}, nil
}

type failingParser struct {
	err error
}

func (p failingParser) Parse(ctx context.Context, input service.ParseInput) (service.ParsedDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.ParsedDocument{}, err
	}
	return service.ParsedDocument{}, p.err
}

type ingestionHandoff struct {
	knowledgeBaseID string
	documentID      string
	jobID           string
}

func seedIngestionJob(t *testing.T, repo *repository.MemoryRepository, fileID string) ingestionHandoff {
	t.Helper()
	return seedIngestionJobWithMaxAttempts(t, repo, fileID, service.DefaultIngestionMaxAttempts)
}

func seedIngestionJobWithMaxAttempts(t *testing.T, repo *repository.MemoryRepository, fileID string, maxAttempts int32) ingestionHandoff {
	t.Helper()
	now := fixedNow()
	doc, job, err := repo.CreateDocumentWithJob(context.Background(), service.CreateDocumentWithJobRecord{
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		FileRef:         fileID,
		Name:            "manual.md",
		ContentType:     "text/markdown",
		SizeBytes:       48,
		Status:          service.DocumentStatusUploaded,
		CurrentJobID:    "job_1",
		CreatedBy:       "usr_123",
		JobID:           "job_1",
		JobType:         service.JobTypeDocumentIngestion,
		JobStatus:       service.JobStatusQueued,
		JobStage:        "uploaded",
		JobMessage:      "document uploaded and queued for ingestion",
		MaxAttempts:     maxAttempts,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, service.AccessScope{UserID: "usr_123", CanWrite: true})
	if err != nil {
		t.Fatalf("CreateDocumentWithJob() error = %v", err)
	}
	return ingestionHandoff{knowledgeBaseID: doc.KnowledgeBaseID, documentID: doc.ID, jobID: job.ID}
}

func seedRunningIngestionJob(t *testing.T, repo *repository.MemoryRepository, fileID string, runningAt time.Time) ingestionHandoff {
	t.Helper()
	handoff := seedIngestionJob(t, repo, fileID)
	stage := "parsing"
	attempts := int32(1)
	if _, err := repo.UpdateJobState(context.Background(), handoff.jobID, service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &stage,
		ProgressPercent: 20,
		Attempts:        &attempts,
		StartedAt:       &runningAt,
		UpdatedAt:       runningAt,
	}); err != nil {
		t.Fatalf("UpdateJobState() error = %v", err)
	}
	if _, err := repo.UpdateDocumentProcessingState(context.Background(), handoff.documentID, service.DocumentStateUpdate{
		Status:    service.DocumentStatusParsing,
		UpdatedAt: runningAt,
	}); err != nil {
		t.Fatalf("UpdateDocumentProcessingState() error = %v", err)
	}
	return handoff
}

func seedDeleteCleanupJob(t *testing.T, repo *repository.MemoryRepository, fileID string) ingestionHandoff {
	t.Helper()
	return seedDeleteCleanupJobWithMaxAttempts(t, repo, fileID, service.DefaultIngestionMaxAttempts)
}

func seedDeleteCleanupJobWithMaxAttempts(t *testing.T, repo *repository.MemoryRepository, fileID string, maxAttempts int32) ingestionHandoff {
	t.Helper()
	now := fixedNow()
	fileRef := fileID
	repo.SeedDocument(service.KnowledgeDocument{
		ID:              "doc_1",
		KnowledgeBaseID: "kb_1",
		FileRef:         &fileRef,
		Name:            "manual.md",
		Status:          service.DocumentStatusReady,
		CreatedBy:       "usr_123",
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err := repo.SoftDeleteDocument(context.Background(), service.DeleteDocumentRecord{
		DocumentID:  "doc_1",
		JobID:       "job_1",
		JobType:     service.JobTypeDeleteCleanup,
		JobStatus:   service.JobStatusQueued,
		JobStage:    "delete_cleanup",
		JobMessage:  "document marked deleted; cleanup is pending",
		MaxAttempts: maxAttempts,
		DeletedAt:   now.Add(time.Minute),
		CreatedAt:   now.Add(time.Minute),
		UpdatedAt:   now.Add(time.Minute),
	}, service.AccessScope{UserID: "usr_123", CanWrite: true}); err != nil {
		t.Fatalf("SoftDeleteDocument() error = %v", err)
	}
	return ingestionHandoff{knowledgeBaseID: "kb_1", documentID: "doc_1", jobID: "job_1"}
}

func seedRunningDeleteCleanupJob(t *testing.T, repo *repository.MemoryRepository, fileID string, runningAt time.Time) ingestionHandoff {
	t.Helper()
	handoff := seedDeleteCleanupJob(t, repo, fileID)
	stage := "delete_cleanup"
	attempts := int32(1)
	if _, err := repo.UpdateJobState(context.Background(), handoff.jobID, service.JobStateUpdate{
		Status:          service.JobStatusRunning,
		CurrentStage:    &stage,
		ProgressPercent: 20,
		Attempts:        &attempts,
		StartedAt:       &runningAt,
		UpdatedAt:       runningAt,
	}); err != nil {
		t.Fatalf("UpdateJobState() error = %v", err)
	}
	return handoff
}

func seedKnowledgeBase(t *testing.T, repo *repository.MemoryRepository) {
	t.Helper()
	repo.SeedKnowledgeBase(service.KnowledgeBase{
		ID:                "kb_1",
		Name:              "Jobs",
		Description:       "",
		DocType:           "GENERAL",
		ChunkStrategy:     json.RawMessage(`{"size":64,"overlap":0}`),
		RetrievalStrategy: json.RawMessage(`{"mode":"VECTOR"}`),
		CreatedBy:         "usr_123",
		CreatedAt:         fixedNow(),
		UpdatedAt:         fixedNow(),
	})
}

type sourceStore struct {
	docs        map[string]sourceDoc
	err         error
	lastRequest service.RequestContext
	readCount   int
}

type sourceDoc struct {
	body        string
	contentType string
}

type blockingSourceStore struct {
	body        string
	contentType string
	entered     chan struct{}
	release     chan struct{}
	enterOnce   sync.Once
	reads       atomic.Int32
}

func newSourceStore() *sourceStore {
	return &sourceStore{docs: map[string]sourceDoc{}}
}

func newBlockingSourceStore(body string, contentType string) *blockingSourceStore {
	return &blockingSourceStore{
		body:        body,
		contentType: contentType,
		entered:     make(chan struct{}),
		release:     make(chan struct{}),
	}
}

func (s *sourceStore) Put(fileID string, body string, contentType string) {
	s.docs[fileID] = sourceDoc{body: body, contentType: contentType}
}

func (s *sourceStore) ReadSource(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.SourceDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.SourceDocument{}, err
	}
	s.readCount++
	s.lastRequest = reqCtx
	if s.err != nil {
		return service.SourceDocument{}, s.err
	}
	doc, exists := s.docs[fileID]
	if !exists {
		return service.SourceDocument{}, service.NewError(service.CodeDependency, "file service content read failed", nil)
	}
	return service.SourceDocument{
		Body:        io.NopCloser(strings.NewReader(doc.body)),
		ContentType: doc.contentType,
		SizeBytes:   int64(len(doc.body)),
	}, nil
}

func (s *blockingSourceStore) ReadSource(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.SourceDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.SourceDocument{}, err
	}
	s.reads.Add(1)
	s.enterOnce.Do(func() {
		close(s.entered)
	})
	select {
	case <-s.release:
	case <-ctx.Done():
		return service.SourceDocument{}, ctx.Err()
	}
	return service.SourceDocument{
		Body:        io.NopCloser(strings.NewReader(s.body)),
		ContentType: s.contentType,
		SizeBytes:   int64(len(s.body)),
	}, nil
}

type recordingVectorIndex struct {
	mu               sync.Mutex
	points           []service.VectorPoint
	deletedDocuments []string
	deleteErr        error
}

type cleanupFileClient struct {
	deleted     []string
	lastRequest service.RequestContext
	err         error
}

type markFailureRepository struct {
	*repository.MemoryRepository
	err error
}

func (r *markFailureRepository) MarkDocumentJobFailed(ctx context.Context, documentID string, jobID string, expectedAttempts *int32, code string, message string, failedAt time.Time) error {
	return r.err
}

func (i *recordingVectorIndex) Upsert(ctx context.Context, points []service.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.points = append(i.points, points...)
	return nil
}

func (i *recordingVectorIndex) DeleteByDocumentIngestionAttempt(ctx context.Context, documentID string, ingestionAttempt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	filtered := i.points[:0]
	for _, point := range i.points {
		if point.Payload[service.VectorPayloadDocumentID] != documentID ||
			point.Payload[service.VectorPayloadIngestionAttempt] != ingestionAttempt {
			filtered = append(filtered, point)
		}
	}
	i.points = filtered
	return nil
}

func (i *recordingVectorIndex) DeleteStaleDocumentPoints(ctx context.Context, documentID string, activeIngestionAttempt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	filtered := i.points[:0]
	for _, point := range i.points {
		if point.Payload[service.VectorPayloadDocumentID] != documentID ||
			point.Payload[service.VectorPayloadIngestionAttempt] == activeIngestionAttempt {
			filtered = append(filtered, point)
		}
	}
	i.points = filtered
	return nil
}

func (i *recordingVectorIndex) DeleteByDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.deletedDocuments = append(i.deletedDocuments, documentID)
	if i.deleteErr != nil {
		return i.deleteErr
	}
	filtered := i.points[:0]
	for _, point := range i.points {
		if point.Payload[service.VectorPayloadDocumentID] != documentID {
			filtered = append(filtered, point)
		}
	}
	i.points = filtered
	return nil
}

func (i *recordingVectorIndex) Search(ctx context.Context, request service.VectorSearchRequest) ([]service.VectorSearchHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []service.VectorSearchHit{}, nil
}

func (f *cleanupFileClient) CreateFile(ctx context.Context, reqCtx service.RequestContext, file service.UploadedFile) (service.FileObject, error) {
	return service.FileObject{}, service.NewError(service.CodeDependency, "file upload is not configured", nil)
}

func (f *cleanupFileClient) DeleteFile(ctx context.Context, reqCtx service.RequestContext, fileID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.deleted = append(f.deleted, fileID)
	f.lastRequest = reqCtx
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *cleanupFileClient) GetFileContent(ctx context.Context, reqCtx service.RequestContext, fileID string) (service.FileContent, error) {
	return service.FileContent{}, service.NewError(service.CodeDependency, "file content read is not configured", nil)
}

func hasAppCode(err error, code service.Code) bool {
	appErr, ok := service.Classify(err)
	return ok && appErr.Code == code
}

type reclaimingVectorIndex struct {
	recordingVectorIndex
	repo      *repository.MemoryRepository
	handoff   ingestionHandoff
	reclaimed bool
}

func (i *reclaimingVectorIndex) Upsert(ctx context.Context, points []service.VectorPoint) error {
	if err := i.recordingVectorIndex.Upsert(ctx, points); err != nil {
		return err
	}
	if len(points) == 0 || i.reclaimed || points[0].Payload[service.VectorPayloadIngestionAttempt] != "job_1:1" {
		return nil
	}
	i.reclaimed = true
	claimAt := fixedNow().Add(time.Hour)
	staleBefore := claimAt.Add(-5 * time.Minute)
	stage := "embedding"
	claimed, err := i.repo.ClaimProcessingJob(ctx, i.handoff.jobID, service.JobStateUpdate{
		Status:             service.JobStatusRunning,
		CurrentStage:       &stage,
		ProgressPercent:    80,
		StartedAt:          &claimAt,
		UpdatedAt:          claimAt,
		StaleRunningBefore: &staleBefore,
	})
	if err != nil {
		return err
	}
	parserBackend := "parser-service"
	pointID := "point_attempt_2"
	chunk := service.DocumentChunk{
		ID:              "chunk_attempt_2",
		KnowledgeBaseID: i.handoff.knowledgeBaseID,
		DocumentID:      i.handoff.documentID,
		ChunkIndex:      0,
		Content:         "new attempt content",
		QdrantPointID:   &pointID,
		CreatedAt:       claimAt,
	}
	if _, err := i.repo.CompleteIngestion(ctx, service.CompleteIngestionRecord{
		DocumentID:       i.handoff.documentID,
		JobID:            i.handoff.jobID,
		ExpectedAttempts: &claimed.Attempts,
		ParserBackend:    &parserBackend,
		Chunks:           []service.DocumentChunk{chunk},
		UpdatedAt:        claimAt,
		FinishedAt:       claimAt,
	}); err != nil {
		return err
	}
	i.recordingVectorIndex.mu.Lock()
	defer i.recordingVectorIndex.mu.Unlock()
	i.recordingVectorIndex.points = append(i.recordingVectorIndex.points, service.VectorPoint{
		ID:     pointID,
		Vector: []float32{0.2, 0.3},
		Payload: map[string]any{
			"knowledge_base_id":                   i.handoff.knowledgeBaseID,
			service.VectorPayloadDocumentID:       i.handoff.documentID,
			"chunk_id":                            "chunk_attempt_2",
			"chunk_index":                         int32(0),
			"chunk_type":                          "",
			"section_path":                        "",
			"tags":                                []string(nil),
			"metadata":                            map[string]any{},
			"job_id":                              i.handoff.jobID,
			"job_attempt":                         claimed.Attempts,
			service.VectorPayloadIngestionAttempt: "job_1:2",
		},
	})
	return nil
}

func assertMinimalVectorPayload(t *testing.T, payload map[string]any) {
	t.Helper()
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	want := []string{"chunk_id", "chunk_index", "chunk_type", "document_id", "ingestion_attempt", "job_attempt", "job_id", "knowledge_base_id", "metadata", "section_path", "tags"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("payload keys = %v", keys)
	}
	for _, forbidden := range []string{"content", "file_ref", "fileId", "object_key", "url", "token", "prompt", "provider_body"} {
		if _, exists := payload[forbidden]; exists {
			t.Fatalf("payload leaked %q: %+v", forbidden, payload)
		}
	}
}

func requireAppError(t *testing.T, err error, code service.Code) *service.AppError {
	t.Helper()
	var appErr *service.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Code != code {
		t.Fatalf("error code = %s, want %s; error = %v", appErr.Code, code, err)
	}
	return appErr
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return data
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
}

func fixedClock() func() time.Time {
	return fixedNow
}

func actorContext() service.RequestContext {
	return service.RequestContext{RequestID: "req_test", UserID: "usr_123"}
}

func ptrFloat64(value float64) *float64 {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func sequenceIDs() func(prefix string) string {
	var mu sync.Mutex
	counter := 0
	return func(prefix string) string {
		mu.Lock()
		defer mu.Unlock()
		counter++
		return prefix + "_" + strconv.Itoa(counter)
	}
}
