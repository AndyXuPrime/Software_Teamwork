package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

const DocumentIngestionTaskType = "knowledge:document:ingest"

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
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(DocumentIngestionTaskType, payload))
	if err != nil {
		return service.NewError(service.CodeDependency, "ingestion queue handoff failed", err)
	}
	return nil
}
