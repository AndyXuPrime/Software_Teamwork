package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/embedding"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/parser"
	sourceplatform "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/source"
	vectorplatform "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/vector"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/worker"
)

func TestIngestionHandlerRejectsInvalidPayloadWithoutTouchingState(t *testing.T) {
	handler, knowledge, repo := newWorkerTestHarness(t, missingSourceReader{})
	seedKnowledgeBase(t, repo, "kb_jobs", "usr_123")
	handoff := createIngestionJob(t, knowledge, "kb_jobs", "file_123")

	err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, map[string]string{
		"requestId": "req_worker",
		"jobId":     handoff.JobID,
	}))

	var appErr *service.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want AppError", err)
	}
	if appErr.Code != service.CodeValidation || appErr.Fields["userId"] == "" {
		t.Fatalf("appErr = %+v", appErr)
	}
	job, err := knowledge.GetJob(context.Background(), actorContext(), handoff.JobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusQueued {
		t.Fatalf("job status = %s, want queued", job.Status)
	}
}

func TestIngestionHandlerProcessesQueuedJobToReady(t *testing.T) {
	sourceReader := sourceplatform.NewMemorySourceReader()
	sourceReader.Put("file_123", "# Intro\n\nThis is enough content for a text chunk.", "text/markdown")
	handler, knowledge, repo := newWorkerTestHarness(t, sourceReader)
	seedKnowledgeBase(t, repo, "kb_jobs", "usr_123")
	handoff := createIngestionJob(t, knowledge, "kb_jobs", "file_123")

	if err := handler.HandleIngestionPayload(context.Background(), mustJSON(t, worker.IngestionPayload{
		RequestID: "req_worker",
		JobID:     handoff.JobID,
		UserID:    "usr_123",
	})); err != nil {
		t.Fatalf("HandleIngestionPayload() error = %v", err)
	}

	job, err := knowledge.GetJob(context.Background(), actorContext(), handoff.JobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Status != service.JobStatusSucceeded || job.ProgressPercent != 100 {
		t.Fatalf("job = %+v", job)
	}
	doc, err := knowledge.GetDocument(context.Background(), actorContext(), handoff.DocumentID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}
	if doc.Status != service.DocumentStatusReady || doc.ChunkCount == 0 {
		t.Fatalf("doc = %+v", doc)
	}
	chunks, err := knowledge.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.DocumentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total == 0 || chunks.Items[0].QdrantPointID == nil {
		t.Fatalf("chunks = %+v", chunks)
	}
}

func TestIngestionHandlerDoesNotReprocessSucceededJob(t *testing.T) {
	sourceReader := sourceplatform.NewMemorySourceReader()
	sourceReader.Put("file_123", "content for exactly one processing run", "text/plain")
	handler, knowledge, repo := newWorkerTestHarness(t, sourceReader)
	seedKnowledgeBase(t, repo, "kb_jobs", "usr_123")
	handoff := createIngestionJob(t, knowledge, "kb_jobs", "file_123")
	payload := mustJSON(t, worker.IngestionPayload{RequestID: "req_worker", JobID: handoff.JobID, UserID: "usr_123"})

	if err := handler.HandleIngestionPayload(context.Background(), payload); err != nil {
		t.Fatalf("first HandleIngestionPayload() error = %v", err)
	}
	err := handler.HandleIngestionPayload(context.Background(), payload)

	var appErr *service.AppError
	if !errors.As(err, &appErr) || appErr.Code != service.CodeConflict {
		t.Fatalf("second error = %v, want conflict", err)
	}
	chunks, err := knowledge.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.DocumentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 {
		t.Fatalf("chunk total = %d, want 1", chunks.Page.Total)
	}
}

func newWorkerTestHarness(t *testing.T, sourceReader service.SourceReader) (*worker.IngestionHandler, *service.KnowledgeService, *repository.MemoryRepository) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	knowledge := service.NewKnowledgeService(
		repo,
		service.WithClock(func() time.Time { return fixedNow() }),
		service.WithIDGenerator(sequenceIDs()),
		service.WithPipeline(missingSourceReader{}, parser.NewTextParser(), parser.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectorplatform.NewMemoryIndex()),
	)
	if sourceReader != nil {
		knowledge = service.NewKnowledgeService(
			repo,
			service.WithClock(func() time.Time { return fixedNow() }),
			service.WithIDGenerator(sequenceIDs()),
			service.WithPipeline(sourceReader, parser.NewTextParser(), parser.NewFixedChunker()),
			service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectorplatform.NewMemoryIndex()),
		)
	}
	return worker.NewIngestionHandler(knowledge), knowledge, repo
}

type missingSourceReader struct{}

func (missingSourceReader) ReadSource(ctx context.Context, fileID string) (service.SourceDocument, error) {
	return service.SourceDocument{}, errors.New("missing source")
}

func seedKnowledgeBase(t *testing.T, repo *repository.MemoryRepository, id string, owner string) {
	t.Helper()
	_, err := repo.CreateKnowledgeBase(context.Background(), service.KnowledgeBase{
		ID:                id,
		Name:              "Jobs",
		DocType:           "GENERAL",
		ChunkStrategy:     service.ChunkStrategy{"type": "SEMANTIC_TEXT", "chunkSize": 64, "overlap": 0},
		RetrievalStrategy: service.RetrievalStrategy{"mode": "VECTOR"},
		CreatedBy:         owner,
		CreatedAt:         fixedNow(),
		UpdatedAt:         fixedNow(),
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
}

func createIngestionJob(t *testing.T, knowledge *service.KnowledgeService, kbID string, fileID string) service.HandoffResult {
	t.Helper()
	handoff, err := knowledge.CreateIngestionJob(context.Background(), actorContext(), service.HandoffInput{
		KnowledgeBaseID: kbID,
		FileID:          fileID,
		Name:            "manual.md",
	})
	if err != nil {
		t.Fatalf("CreateIngestionJob() error = %v", err)
	}
	return handoff
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

func actorContext() service.RequestContext {
	return service.RequestContext{RequestID: "req_test", UserID: "usr_123"}
}

func sequenceIDs() func(prefix string) (string, error) {
	counter := 0
	return func(prefix string) (string, error) {
		counter++
		return prefix + "_" + strconv.Itoa(counter), nil
	}
}
