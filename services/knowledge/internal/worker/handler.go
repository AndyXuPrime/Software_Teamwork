package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

const IngestionTaskType = "knowledge:document:ingest"

type IngestionPayload = service.DocumentIngestionTask
type DeleteCleanupPayload = service.DocumentDeleteCleanupTask

type IngestionHandler struct {
	knowledge *service.Service
	logger    *slog.Logger
}

type DeleteCleanupHandler struct {
	knowledge *service.Service
	logger    *slog.Logger
}

type IngestionHandlerOption func(*IngestionHandler)

func NewIngestionHandler(knowledge *service.Service, opts ...IngestionHandlerOption) *IngestionHandler {
	h := &IngestionHandler{
		knowledge: knowledge,
		logger:    slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func NewDeleteCleanupHandler(knowledge *service.Service, loggers ...*slog.Logger) *DeleteCleanupHandler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &DeleteCleanupHandler{knowledge: knowledge, logger: logger}
}

func WithLogger(logger *slog.Logger) IngestionHandlerOption {
	return func(h *IngestionHandler) {
		if logger != nil {
			h.logger = logger
		}
	}
}

func (h *IngestionHandler) HandleIngestionPayload(ctx context.Context, payload []byte) error {
	parsed, err := DecodeIngestionPayload(payload)
	if err != nil {
		h.logFailure(ctx, IngestionPayload{}, err)
		// Malformed task payloads are permanent; retrying cannot make the JSON valid.
		return nil
	}
	if h.knowledge == nil {
		return service.DependencyError("knowledge service is not configured", nil)
	}
	reqCtx := service.RequestContext{
		RequestID:     parsed.RequestID,
		UserID:        parsed.UserID,
		CallerService: "knowledge",
	}
	_, err = h.knowledge.ProcessIngestionTask(ctx, reqCtx, parsed)
	if err == nil {
		return nil
	}
	h.logFailure(ctx, parsed, err)
	if shouldAckIngestionError(err) || h.jobReachedMaxAttempts(ctx, reqCtx, parsed.JobID) {
		return nil
	}
	return err
}

func (h *DeleteCleanupHandler) HandleDeleteCleanupPayload(ctx context.Context, payload []byte) error {
	parsed, err := DecodeDeleteCleanupPayload(payload)
	if err != nil {
		h.logFailure(ctx, DeleteCleanupPayload{}, err)
		// Malformed task payloads are permanent; retrying cannot make the JSON valid.
		return nil
	}
	if h.knowledge == nil {
		return service.DependencyError("knowledge service is not configured", nil)
	}
	reqCtx := service.RequestContext{
		RequestID:     parsed.RequestID,
		UserID:        parsed.UserID,
		CallerService: "knowledge",
	}
	_, err = h.knowledge.ProcessDeleteCleanupTask(ctx, reqCtx, parsed)
	if err == nil {
		return nil
	}
	h.logFailure(ctx, parsed, err)
	if shouldAckDeleteCleanupError(err) {
		return nil
	}
	return err
}

func (h *IngestionHandler) logFailure(ctx context.Context, payload IngestionPayload, err error) {
	if h.logger == nil || err == nil {
		return
	}
	code := "unknown"
	if appErr, ok := service.Classify(err); ok {
		code = string(appErr.Code)
	}
	// Keep worker logs to identifiers and normalized error codes only.
	h.logger.WarnContext(ctx, "knowledge ingestion job failed",
		"service", "knowledge",
		"request_id", payload.RequestID,
		"user_id", payload.UserID,
		"job_id", payload.JobID,
		"document_id", payload.DocumentID,
		"knowledge_base_id", payload.KnowledgeBaseID,
		"operation", "knowledge_ingestion_worker",
		"status", "failed",
		"error_code", code,
	)
}

func (h *DeleteCleanupHandler) logFailure(ctx context.Context, payload DeleteCleanupPayload, err error) {
	if h.logger == nil || err == nil {
		return
	}
	code := "unknown"
	if appErr, ok := service.Classify(err); ok {
		code = string(appErr.Code)
	}
	// Keep worker logs to identifiers and normalized error codes only.
	h.logger.WarnContext(ctx, "knowledge delete cleanup job failed",
		"service", "knowledge",
		"request_id", payload.RequestID,
		"user_id", payload.UserID,
		"job_id", payload.JobID,
		"document_id", payload.DocumentID,
		"knowledge_base_id", payload.KnowledgeBaseID,
		"operation", "knowledge_delete_cleanup_worker",
		"status", "failed",
		"error_code", code,
	)
}

func shouldAckIngestionError(err error) bool {
	appErr, ok := service.Classify(err)
	if !ok {
		return false
	}
	switch appErr.Code {
	case service.CodeValidation, service.CodeUnauthorized, service.CodeForbidden, service.CodeNotFound, service.CodeConflict:
		return true
	default:
		return false
	}
}

func shouldAckDeleteCleanupError(err error) bool {
	appErr, ok := service.Classify(err)
	if !ok {
		return false
	}
	switch appErr.Code {
	case service.CodeValidation, service.CodeNotFound, service.CodeConflict:
		return true
	default:
		return false
	}
}

func (h *IngestionHandler) jobReachedMaxAttempts(ctx context.Context, reqCtx service.RequestContext, jobID string) bool {
	if h.knowledge == nil || strings.TrimSpace(jobID) == "" {
		return false
	}
	job, err := h.knowledge.GetJob(ctx, reqCtx, jobID)
	if err != nil {
		return false
	}
	return job.Status == service.JobStatusFailed && job.MaxAttempts > 0 && job.Attempts >= job.MaxAttempts
}

func DecodeIngestionPayload(payload []byte) (IngestionPayload, error) {
	var parsed IngestionPayload
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return IngestionPayload{}, service.ValidationError("worker payload validation failed", map[string]string{"body": "must be a valid ingestion payload"})
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return IngestionPayload{}, service.ValidationError("worker payload validation failed", map[string]string{"body": "must contain only one JSON object"})
	}
	parsed.RequestID = strings.TrimSpace(parsed.RequestID)
	parsed.JobID = strings.TrimSpace(parsed.JobID)
	parsed.DocumentID = strings.TrimSpace(parsed.DocumentID)
	parsed.KnowledgeBaseID = strings.TrimSpace(parsed.KnowledgeBaseID)
	parsed.UserID = strings.TrimSpace(parsed.UserID)

	fields := map[string]string{}
	if parsed.RequestID == "" {
		fields["requestId"] = "is required"
	}
	if parsed.JobID == "" {
		fields["jobId"] = "is required"
	}
	if parsed.DocumentID == "" {
		fields["documentId"] = "is required"
	}
	if parsed.KnowledgeBaseID == "" {
		fields["knowledgeBaseId"] = "is required"
	}
	if parsed.UserID == "" {
		fields["userId"] = "is required"
	}
	if len(fields) > 0 {
		return IngestionPayload{}, service.ValidationError("worker payload validation failed", fields)
	}
	return parsed, nil
}

func DecodeDeleteCleanupPayload(payload []byte) (DeleteCleanupPayload, error) {
	var parsed DeleteCleanupPayload
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return DeleteCleanupPayload{}, service.ValidationError("worker payload validation failed", map[string]string{"body": "must be a valid delete cleanup payload"})
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return DeleteCleanupPayload{}, service.ValidationError("worker payload validation failed", map[string]string{"body": "must contain only one JSON object"})
	}
	parsed.RequestID = strings.TrimSpace(parsed.RequestID)
	parsed.JobID = strings.TrimSpace(parsed.JobID)
	parsed.DocumentID = strings.TrimSpace(parsed.DocumentID)
	parsed.KnowledgeBaseID = strings.TrimSpace(parsed.KnowledgeBaseID)
	parsed.UserID = strings.TrimSpace(parsed.UserID)

	fields := map[string]string{}
	if parsed.RequestID == "" {
		fields["requestId"] = "is required"
	}
	if parsed.JobID == "" {
		fields["jobId"] = "is required"
	}
	if parsed.DocumentID == "" {
		fields["documentId"] = "is required"
	}
	if parsed.KnowledgeBaseID == "" {
		fields["knowledgeBaseId"] = "is required"
	}
	if parsed.UserID == "" {
		fields["userId"] = "is required"
	}
	if len(fields) > 0 {
		return DeleteCleanupPayload{}, service.ValidationError("worker payload validation failed", fields)
	}
	return parsed, nil
}
