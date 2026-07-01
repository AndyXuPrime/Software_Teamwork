package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestEnqueueDocumentDeleteCleanupRunsArchivedTaskIDConflict(t *testing.T) {
	inspector := &fakeTaskInspector{
		info: &asynq.TaskInfo{
			ID:    "knowledge-delete-cleanup-job_1",
			Queue: defaultAsynqQueue,
			State: asynq.TaskStateArchived,
		},
	}
	queue := &AsynqQueue{
		client:    &fakeAsynqClient{err: asynq.ErrTaskIDConflict},
		inspector: inspector,
	}

	err := queue.EnqueueDocumentDeleteCleanup(context.Background(), service.DocumentDeleteCleanupTask{
		RequestID:       "req_cleanup",
		JobID:           "job_1",
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		UserID:          "usr_1",
	})
	if err != nil {
		t.Fatalf("EnqueueDocumentDeleteCleanup() error = %v", err)
	}
	if inspector.runQueue != defaultAsynqQueue || inspector.runID != "knowledge-delete-cleanup-job_1" {
		t.Fatalf("RunTask(queue=%q, id=%q)", inspector.runQueue, inspector.runID)
	}
}

func TestEnqueueDocumentDeleteCleanupKeepsLiveTaskIDConflict(t *testing.T) {
	inspector := &fakeTaskInspector{
		info: &asynq.TaskInfo{
			ID:    "knowledge-delete-cleanup-job_1",
			Queue: defaultAsynqQueue,
			State: asynq.TaskStateRetry,
		},
	}
	queue := &AsynqQueue{
		client:    &fakeAsynqClient{err: asynq.ErrTaskIDConflict},
		inspector: inspector,
	}

	err := queue.EnqueueDocumentDeleteCleanup(context.Background(), service.DocumentDeleteCleanupTask{
		RequestID:       "req_cleanup",
		JobID:           "job_1",
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		UserID:          "usr_1",
	})
	if err != nil {
		t.Fatalf("EnqueueDocumentDeleteCleanup() error = %v", err)
	}
	if inspector.runID != "" {
		t.Fatalf("RunTask was called for live retry task: %q", inspector.runID)
	}
}

func TestEnqueueDocumentDeleteCleanupReportsUninspectableTaskIDConflict(t *testing.T) {
	queue := &AsynqQueue{
		client: &fakeAsynqClient{err: asynq.ErrTaskIDConflict},
	}

	err := queue.EnqueueDocumentDeleteCleanup(context.Background(), service.DocumentDeleteCleanupTask{
		RequestID:       "req_cleanup",
		JobID:           "job_1",
		DocumentID:      "doc_1",
		KnowledgeBaseID: "kb_1",
		UserID:          "usr_1",
	})
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("EnqueueDocumentDeleteCleanup() error = %v", err)
	}
}

type fakeAsynqClient struct {
	err error
}

func (c *fakeAsynqClient) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &asynq.TaskInfo{}, nil
}

type fakeTaskInspector struct {
	info     *asynq.TaskInfo
	infoErr  error
	runErr   error
	runQueue string
	runID    string
}

func (i *fakeTaskInspector) GetTaskInfo(queue string, id string) (*asynq.TaskInfo, error) {
	if i.infoErr != nil {
		return nil, i.infoErr
	}
	return i.info, nil
}

func (i *fakeTaskInspector) RunTask(queue string, id string) error {
	i.runQueue = queue
	i.runID = id
	return i.runErr
}

func hasCode(err error, code service.Code) bool {
	var appErr *service.AppError
	return errors.As(err, &appErr) && appErr.Code == code
}
