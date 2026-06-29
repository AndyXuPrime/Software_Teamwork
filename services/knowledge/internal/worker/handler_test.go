package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/embedding"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/platform/parser"
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
	chunks, err := svc.ListChunks(context.Background(), actorContext(), service.ListChunksInput{DocumentID: handoff.documentID})
	if err != nil {
		t.Fatalf("ListChunks() error = %v", err)
	}
	if chunks.Page.Total != 1 || chunks.Items[0].QdrantPointID == nil {
		t.Fatalf("chunks = %+v", chunks)
	}
	if len(vectors.points) != 1 {
		t.Fatalf("vector points = %+v", vectors.points)
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

func TestIngestionHandlerAcksPermanentParsingFailure(t *testing.T) {
	source := newSourceStore()
	source.Put("file_empty", "", "text/plain")
	handler, svc, repo, vectors := newWorkerHarness(t, source)
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

func TestDecodeIngestionPayloadRejectsUnknownFields(t *testing.T) {
	_, err := worker.DecodeIngestionPayload([]byte(`{"requestId":"req","jobId":"job","documentId":"doc","knowledgeBaseId":"kb","userId":"usr","fileRef":"secret"}`))
	requireAppError(t, err, service.CodeValidation)
}

func newWorkerHarness(t *testing.T, source service.SourceReader) (*worker.IngestionHandler, *service.Service, *repository.MemoryRepository, *recordingVectorIndex) {
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
		service.WithProcessingPipeline(source, parser.NewRouter(), parser.NewFixedChunker()),
		service.WithVectorIndex(embedding.NewLocalHasher("local_hashing", "local_hashing", 16), vectors),
	)
	return worker.NewIngestionHandler(svc), svc, repo, vectors
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

func newSourceStore() *sourceStore {
	return &sourceStore{docs: map[string]sourceDoc{}}
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

type recordingVectorIndex struct {
	points []service.VectorPoint
}

func (i *recordingVectorIndex) Upsert(ctx context.Context, points []service.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.points = append(i.points, points...)
	return nil
}

func (i *recordingVectorIndex) DeleteByDocument(ctx context.Context, documentID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	filtered := i.points[:0]
	for _, point := range i.points {
		if point.Payload["document_id"] != documentID {
			filtered = append(filtered, point)
		}
	}
	i.points = filtered
	return nil
}

func assertMinimalVectorPayload(t *testing.T, payload map[string]any) {
	t.Helper()
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	want := []string{"chunk_id", "chunk_index", "chunk_type", "document_id", "knowledge_base_id", "metadata", "section_path", "tags"}
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

func sequenceIDs() func(prefix string) string {
	counter := 0
	return func(prefix string) string {
		counter++
		return prefix + "_" + strconv.Itoa(counter)
	}
}
