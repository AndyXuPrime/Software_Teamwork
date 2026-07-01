package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

const (
	DocumentIngestionTaskType     = "knowledge:document:ingest"
	DocumentDeleteCleanupTaskType = "knowledge:document:delete_cleanup"
)

type AsynqQueue struct {
	client *asynq.Client
}

func NewAsynqQueue(client *asynq.Client) *AsynqQueue {
	return &AsynqQueue{client: client}
}

func (q *AsynqQueue) EnqueueDocumentIngestion(ctx context.Context, task service.DocumentIngestionTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return service.NewError(service.CodeInternal, "ingestion task payload is invalid", err)
	}
	if q == nil || q.client == nil {
		return service.NewError(service.CodeDependency, "ingestion queue is not configured", nil)
	}
	maxRetries := int(service.DefaultIngestionMaxAttempts - 1)
	if maxRetries < 0 {
		maxRetries = 0
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(DocumentIngestionTaskType, payload), asynq.MaxRetry(maxRetries))
	if err != nil {
		return service.NewError(service.CodeDependency, "ingestion queue handoff failed", err)
	}
	return nil
}

func (q *AsynqQueue) EnqueueDocumentDeleteCleanup(ctx context.Context, task service.DocumentDeleteCleanupTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return service.NewError(service.CodeInternal, "delete cleanup task payload is invalid", err)
	}
	if q == nil || q.client == nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue is not configured", nil)
	}
	maxRetries := int(service.DefaultIngestionMaxAttempts - 1)
	if maxRetries < 0 {
		maxRetries = 0
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(DocumentDeleteCleanupTaskType, payload),
		asynq.MaxRetry(maxRetries),
		asynq.TaskID(deleteCleanupTaskID(task.JobID)),
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	if err != nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue handoff failed", err)
	}
	return nil
}

func deleteCleanupTaskID(jobID string) string {
	return "knowledge-delete-cleanup-" + strings.TrimSpace(jobID)
}
