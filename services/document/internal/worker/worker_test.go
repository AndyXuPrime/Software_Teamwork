package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
	"github.com/hibiken/asynq"
)

func TestWorkerRecordsOperationLogsForJobStatusTransitions(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-worker",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-1",
		AttemptID: "attempt-1",
		UserID:    "user-1",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{}
	worker := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  manager,
		reportGenerationExecutor: &fakeReportGenerationExecutor{},
	}

	if err := worker.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, raw)); err != nil {
		t.Fatalf("handleReportJob() error = %v", err)
	}

	if len(manager.logs) != 2 {
		t.Fatalf("operation log count = %d, want 2", len(manager.logs))
	}
	if got := manager.logs[0]; got.OperationType != service.OperationReportJobRunning || got.TargetID != "job-1" || got.RequestSource != "worker" {
		t.Fatalf("running operation log = %+v", got)
	}
	if got := manager.logs[1]; got.OperationType != service.OperationReportJobSucceeded || got.TargetID != "job-1" || got.ParameterSummary["jobType"] != string(service.JobTypeContentGeneration) {
		t.Fatalf("succeeded operation log = %+v", got)
	}
}

func TestWorkerSanitizesOperationLogSummaries(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-worker",
		JobType:   "content_generation prompt=secret https://minio.local/bucket/object",
		JobID:     "job-1",
		AttemptID: "attempt-1",
		UserID:    "user-1",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{}
	worker := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  manager,
		reportGenerationExecutor: &fakeReportGenerationExecutor{},
	}

	if err := worker.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, raw)); err != nil {
		t.Fatalf("handleReportJob() error = %v", err)
	}

	if len(manager.logs) != 2 {
		t.Fatalf("operation log count = %d, want 2", len(manager.logs))
	}
	for _, log := range manager.logs {
		if got := log.ParameterSummary["jobType"]; got != "[redacted]" {
			t.Fatalf("operation log jobType summary was not sanitized: %+v", log.ParameterSummary)
		}
	}
}

func TestWorkerExecutesReportFileCreationJob(t *testing.T) {
	mgr := &fakeWorkerJobManager{}
	executor := &fakeReportFileExecutor{}
	w := &Worker{
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:            mgr,
		reportFileExecutor: executor,
	}
	payload := ReportJobPayload{
		RequestID: "req-1",
		JobType:   string(service.JobTypeReportFileCreation),
		JobID:     "job-1",
		AttemptID: "attempt-1",
		UserID:    "user-1",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := w.handleReportJob(context.Background(), asynq.NewTask(TaskReportFileCreation, body)); err != nil {
		t.Fatalf("handleReportJob() error = %v", err)
	}
	if executor.payload.JobID != "job-1" || executor.payload.UserID != "user-1" {
		t.Fatalf("executor payload = %+v", executor.payload)
	}
	if !mgr.jobRunning || !mgr.attemptRunning || !mgr.jobSucceeded || !mgr.attemptSucceeded {
		t.Fatalf("expected running and succeeded state transitions, got %+v", mgr)
	}
}

func TestWorkerExecutesReportGenerationJob(t *testing.T) {
	mgr := &fakeWorkerJobManager{}
	executor := &fakeReportGenerationExecutor{}
	w := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  mgr,
		reportGenerationExecutor: executor,
	}
	payload := ReportJobPayload{
		RequestID: "req-generation",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-generation",
		AttemptID: "attempt-generation",
		UserID:    "user-generation",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := w.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, body)); err != nil {
		t.Fatalf("handleReportJob() error = %v", err)
	}

	if executor.payload.JobID != "job-generation" || executor.payload.RequestID != "req-generation" || executor.payload.UserID != "user-generation" {
		t.Fatalf("generation executor payload = %+v", executor.payload)
	}
	if executor.payload.JobType != service.JobTypeContentGeneration {
		t.Fatalf("generation executor job type = %q, want %q", executor.payload.JobType, service.JobTypeContentGeneration)
	}
	if !mgr.jobRunning || !mgr.attemptRunning || !mgr.jobSucceeded || !mgr.attemptSucceeded {
		t.Fatalf("expected running and succeeded state transitions, got %+v", mgr)
	}
}

func TestWorkerMarksReportGenerationPartialSucceeded(t *testing.T) {
	mgr := &fakeWorkerJobManager{}
	executor := &fakeReportGenerationExecutor{
		result: service.ReportGenerationExecutionResult{Status: service.JobStatusPartialSucceeded},
	}
	w := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  mgr,
		reportGenerationExecutor: executor,
	}
	payload := ReportJobPayload{
		RequestID: "req-partial",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-partial",
		AttemptID: "attempt-partial",
		UserID:    "user-partial",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := w.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, body)); err != nil {
		t.Fatalf("handleReportJob() error = %v", err)
	}

	if !mgr.jobPartialSucceeded || !mgr.attemptPartialSucceeded {
		t.Fatalf("expected partial succeeded state transitions, got %+v", mgr)
	}
	if mgr.jobSucceeded || mgr.attemptSucceeded {
		t.Fatalf("partial generation must not be marked fully succeeded: %+v", mgr)
	}
}

func TestWorkerSanitizesSensitiveLogValues(t *testing.T) {
	var logs bytes.Buffer
	mgr := &fakeWorkerJobManager{}
	executor := &fakeReportGenerationExecutor{
		err: errors.New("provider raw error sk-secret https://provider.internal/v1 prompt=secret"),
	}
	w := &Worker{
		logger:                   slog.New(slog.NewJSONHandler(&logs, nil)),
		jobsMgr:                  mgr,
		reportGenerationExecutor: executor,
	}
	payload := ReportJobPayload{
		RequestID: "req-log",
		JobType:   "content_generation prompt=secret https://minio.local/bucket/object",
		JobID:     "job-log",
		AttemptID: "attempt-log",
		UserID:    "user-log",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	err = w.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, body))
	if err == nil {
		t.Fatal("handleReportJob() error = nil, want execution failure")
	}

	output := logs.String()
	for _, forbidden := range []string{"sk-secret", "provider.internal", "prompt=secret", "minio.local"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("worker log leaked %q: %s", forbidden, output)
		}
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("worker returned error leaked %q: %s", forbidden, err.Error())
		}
	}
	if !strings.Contains(output, "[redacted]") {
		t.Fatalf("worker log did not include a redacted marker: %s", output)
	}
}

func TestWorkerRecordsFailedOperationLogWhenJobUpdateFails(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-worker-failed",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-1",
		AttemptID: "attempt-1",
		UserID:    "user-1",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{succeededErr: errors.New("postgres://user:pass@db.internal/document unavailable")}
	worker := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  manager,
		reportGenerationExecutor: &fakeReportGenerationExecutor{},
	}

	if err := worker.handleReportJob(context.Background(), asynq.NewTask(TaskContentGeneration, raw)); err == nil {
		t.Fatal("handleReportJob() error = nil, want state update error")
	}

	if len(manager.logs) != 2 {
		t.Fatalf("operation log count = %d, want running and failed", len(manager.logs))
	}
	if got := manager.logs[1]; got.OperationType != service.OperationReportJobFailed || got.OperationResult != service.OperationResultFailed || got.TargetID != "job-1" {
		t.Fatalf("failed operation log = %+v", got)
	}
	if strings.Contains(manager.attemptFailedMessage, "user:pass") || strings.Contains(manager.attemptFailedMessage, "db.internal") {
		t.Fatalf("attempt failure message leaked dependency details: %q", manager.attemptFailedMessage)
	}
}

func TestWorkerRecordsFailedOperationLogWhenReportFileExecutionFails(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-file-failed",
		JobType:   string(service.JobTypeReportFileCreation),
		JobID:     "job-1",
		AttemptID: "attempt-1",
		UserID:    "user-1",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{}
	executor := &fakeReportFileExecutor{err: errors.New("file service unavailable")}
	worker := &Worker{
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:            manager,
		reportFileExecutor: executor,
	}

	if err := worker.handleReportJob(context.Background(), asynq.NewTask(TaskReportFileCreation, raw)); err == nil {
		t.Fatal("handleReportJob() error = nil, want execution error")
	}

	if len(manager.logs) != 2 {
		t.Fatalf("operation log count = %d, want running and failed", len(manager.logs))
	}
	if !manager.jobFailed || !manager.attemptFailed {
		t.Fatalf("expected failed state transitions, got %+v", manager)
	}
	if got := manager.logs[1]; got.OperationType != service.OperationReportJobFailed || got.OperationResult != service.OperationResultFailed || got.TargetID != "job-1" {
		t.Fatalf("failed operation log = %+v", got)
	}
}

type fakeWorkerJobManager struct {
	jobRunning              bool
	jobSucceeded            bool
	jobFailed               bool
	attemptRunning          bool
	attemptSucceeded        bool
	attemptFailed           bool
	attemptFailedMessage    string
	jobPartialSucceeded     bool
	attemptPartialSucceeded bool
	logs                    []service.OperationLog
	succeededErr            error
	setJobSucceededCount    int
}

func (f *fakeWorkerJobManager) SetJobRunning(context.Context, string) error {
	f.jobRunning = true
	return nil
}

func (f *fakeWorkerJobManager) SetJobSucceeded(context.Context, string) error {
	if f.succeededErr != nil {
		return f.succeededErr
	}
	// mirrors real repository: writing succeeded overwrites any prior failed state.
	f.jobSucceeded = true
	f.jobFailed = false
	f.setJobSucceededCount++
	return nil
}

func (f *fakeWorkerJobManager) SetJobPartialSucceeded(context.Context, string) error {
	f.jobPartialSucceeded = true
	return nil
}

func (f *fakeWorkerJobManager) SetJobFailed(context.Context, string, string, string) error {
	f.jobFailed = true
	return nil
}

func (f *fakeWorkerJobManager) SetAttemptRunning(context.Context, string) error {
	f.attemptRunning = true
	return nil
}

func (f *fakeWorkerJobManager) SetAttemptSucceeded(context.Context, string) error {
	f.attemptSucceeded = true
	return nil
}

func (f *fakeWorkerJobManager) SetAttemptPartialSucceeded(context.Context, string) error {
	f.attemptPartialSucceeded = true
	return nil
}

func (f *fakeWorkerJobManager) SetAttemptFailed(_ context.Context, _, _, message string) error {
	f.attemptFailed = true
	f.attemptFailedMessage = message
	return nil
}

func (f *fakeWorkerJobManager) CreateOperationLog(_ context.Context, log service.OperationLog) (service.OperationLog, error) {
	f.logs = append(f.logs, log)
	return log, nil
}

type fakeReportFileExecutor struct {
	payload service.ReportFileExecutionPayload
	err     error
}

func (f *fakeReportFileExecutor) ExecuteReportFileCreation(_ context.Context, payload service.ReportFileExecutionPayload) error {
	f.payload = payload
	return f.err
}

type fakeReportGenerationExecutor struct {
	payload service.ReportGenerationExecutionPayload
	result  service.ReportGenerationExecutionResult
	err     error
}

func (f *fakeReportGenerationExecutor) ExecuteReportGeneration(_ context.Context, payload service.ReportGenerationExecutionPayload) (service.ReportGenerationExecutionResult, error) {
	f.payload = payload
	if f.result.Status == "" {
		f.result.Status = service.JobStatusSucceeded
	}
	return f.result, f.err
}

// sequentialReportGenerationExecutor returns errs[i] on the i-th call, then succeeds.
type sequentialReportGenerationExecutor struct {
	errs []error
	call int
}

func (f *sequentialReportGenerationExecutor) ExecuteReportGeneration(_ context.Context, _ service.ReportGenerationExecutionPayload) (service.ReportGenerationExecutionResult, error) {
	i := f.call
	f.call++
	if i < len(f.errs) && f.errs[i] != nil {
		return service.ReportGenerationExecutionResult{}, f.errs[i]
	}
	return service.ReportGenerationExecutionResult{Status: service.JobStatusSucceeded}, nil
}

// TestWorkerHandlesIdempotentJobRedeliveryAfterSuccess verifies that delivering the same
// asynq task a second time after it already succeeded does not produce a contradictory
// failed state in the job manager (at-least-once delivery, scene A).
func TestWorkerHandlesIdempotentJobRedeliveryAfterSuccess(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-idempotent-a",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-idempotent-a",
		AttemptID: "attempt-idempotent-a",
		UserID:    "user-idempotent-a",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{}
	worker := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  manager,
		reportGenerationExecutor: &fakeReportGenerationExecutor{},
	}

	task := asynq.NewTask(TaskContentGeneration, raw)
	if err := worker.handleReportJob(context.Background(), task); err != nil {
		t.Fatalf("first delivery: handleReportJob() error = %v", err)
	}
	if err := worker.handleReportJob(context.Background(), task); err != nil {
		t.Fatalf("second delivery: handleReportJob() error = %v", err)
	}

	if manager.jobFailed {
		t.Fatalf("jobFailed must remain false after idempotent redelivery, manager = %+v", manager)
	}
	if !manager.jobSucceeded {
		t.Fatalf("jobSucceeded must be true, manager = %+v", manager)
	}
	// Two deliveries produce at most 4 operation logs (running+succeeded × 2).
	if len(manager.logs) > 4 {
		t.Fatalf("operation log count = %d, want ≤ 4 for two successful deliveries", len(manager.logs))
	}
}

// TestWorkerHandlesIdempotentJobRedeliveryAfterFirstFailure verifies that a job that
// fails on first delivery can succeed on a second delivery, and that SetJobSucceeded is
// called exactly once (at-least-once delivery, scene B).
func TestWorkerHandlesIdempotentJobRedeliveryAfterFirstFailure(t *testing.T) {
	payload := ReportJobPayload{
		RequestID: "req-idempotent-b",
		JobType:   string(service.JobTypeContentGeneration),
		JobID:     "job-idempotent-b",
		AttemptID: "attempt-idempotent-b",
		UserID:    "user-idempotent-b",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	manager := &fakeWorkerJobManager{}
	executor := &sequentialReportGenerationExecutor{
		errs: []error{errors.New("transient executor failure"), nil},
	}
	worker := &Worker{
		logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		jobsMgr:                  manager,
		reportGenerationExecutor: executor,
	}

	task := asynq.NewTask(TaskContentGeneration, raw)
	if err := worker.handleReportJob(context.Background(), task); err == nil {
		t.Fatalf("first delivery: handleReportJob() error = nil, want execution failure")
	}
	if !manager.jobFailed {
		t.Fatalf("jobFailed must be true after first failed delivery, manager = %+v", manager)
	}

	if err := worker.handleReportJob(context.Background(), task); err != nil {
		t.Fatalf("second delivery: handleReportJob() error = %v", err)
	}
	if !manager.jobSucceeded {
		t.Fatalf("jobSucceeded must be true after second successful delivery, manager = %+v", manager)
	}
	if manager.setJobSucceededCount != 1 {
		t.Fatalf("SetJobSucceeded call count = %d, want exactly 1", manager.setJobSucceededCount)
	}
	if manager.jobFailed {
		t.Fatalf("jobFailed must be false after second successful delivery converges state to succeeded, manager = %+v", manager)
	}
}
