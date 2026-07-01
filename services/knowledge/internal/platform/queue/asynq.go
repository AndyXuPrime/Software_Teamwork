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
	defaultAsynqQueue             = "default"
)

type asynqEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type asynqTaskInspector interface {
	GetTaskInfo(queue string, id string) (*asynq.TaskInfo, error)
	RunTask(queue string, id string) error
}

type AsynqQueue struct {
	client    asynqEnqueuer
	inspector asynqTaskInspector
	queueName string
}

func NewAsynqQueue(client *asynq.Client) *AsynqQueue {
	return &AsynqQueue{client: client, queueName: defaultAsynqQueue}
}

func NewAsynqQueueWithInspector(client *asynq.Client, inspector *asynq.Inspector) *AsynqQueue {
	return &AsynqQueue{client: client, inspector: inspector, queueName: defaultAsynqQueue}
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
	taskID := deleteCleanupTaskID(task.JobID)
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(DocumentDeleteCleanupTaskType, payload),
		asynq.MaxRetry(maxRetries),
		asynq.TaskID(taskID),
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return q.handleDeleteCleanupTaskIDConflict(taskID)
	}
	if errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	if err != nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue handoff failed", err)
	}
	return nil
}

func (q *AsynqQueue) handleDeleteCleanupTaskIDConflict(taskID string) error {
	if q == nil || q.inspector == nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue task conflict could not be inspected", nil)
	}
	queueName := strings.TrimSpace(q.queueName)
	if queueName == "" {
		queueName = defaultAsynqQueue
	}
	info, err := q.inspector.GetTaskInfo(queueName, taskID)
	if err != nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue task inspection failed", err)
	}
	if info == nil {
		return service.NewError(service.CodeDependency, "delete cleanup queue task inspection failed", nil)
	}
	switch info.State {
	case asynq.TaskStateArchived:
		if err := q.inspector.RunTask(queueName, taskID); err != nil {
			return service.NewError(service.CodeDependency, "delete cleanup archived queue task requeue failed", err)
		}
		return nil
	case asynq.TaskStateActive, asynq.TaskStatePending, asynq.TaskStateScheduled, asynq.TaskStateRetry, asynq.TaskStateAggregating:
		return nil
	default:
		return service.NewError(service.CodeDependency, "delete cleanup queue task is not retryable", nil)
	}
}

func deleteCleanupTaskID(jobID string) string {
	return "knowledge-delete-cleanup-" + strings.TrimSpace(jobID)
}
