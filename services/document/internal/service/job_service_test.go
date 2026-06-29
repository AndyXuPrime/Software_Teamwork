package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestJobServiceCreateJobAcceptsDocumentJobTypes(t *testing.T) {
	ctx := context.Background()
	repo := &fakeJobRepository{
		report: Report{
			ID:        "report-1",
			CreatorID: "user-1",
		},
	}
	enqueuer := &fakeTaskEnqueuer{}
	svc := NewJobService(repo, enqueuer)

	job, err := svc.CreateJob(ctx, RequestContext{UserID: "user-1"}, CreateJobInput{
		RequestID: "req-1",
		UserID:    "user-1",
		ReportID:  "report-1",
		JobType:   JobTypeContentGeneration,
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if job.JobType != JobTypeContentGeneration {
		t.Fatalf("JobType = %q, want %q", job.JobType, JobTypeContentGeneration)
	}
	if enqueuer.jobType != JobTypeContentGeneration {
		t.Fatalf("enqueued job type = %q, want %q", enqueuer.jobType, JobTypeContentGeneration)
	}
}

func TestJobServiceCreateJobRejectsUnknownJobType(t *testing.T) {
	ctx := context.Background()
	svc := NewJobService(&fakeJobRepository{
		report: Report{ID: "report-1", CreatorID: "user-1"},
	}, &fakeTaskEnqueuer{})

	_, err := svc.CreateJob(ctx, RequestContext{UserID: "user-1"}, CreateJobInput{
		RequestID: "req-1",
		UserID:    "user-1",
		ReportID:  "report-1",
		JobType:   JobType("unknown"),
	})
	if err == nil {
		t.Fatal("CreateJob() error = nil, want validation error")
	}
	appErr, ok := Classify(err)
	if !ok || appErr.Code != CodeValidation {
		t.Fatalf("CreateJob() error = %v, want validation_error", err)
	}
}

func TestJobServiceCreateJobRecordsOperationLog(t *testing.T) {
	ctx := context.Background()
	repo := &fakeJobRepository{
		report: Report{ID: "report-1", CreatorID: "user-1"},
	}
	svc := NewJobService(repo, &fakeTaskEnqueuer{})

	job, err := svc.CreateJob(ctx, RequestContext{UserID: "user-1", RequestID: "req-job-audit"}, CreateJobInput{
		RequestID: "req-job-audit",
		UserID:    "user-1",
		ReportID:  "report-1",
		JobType:   JobTypeReportFileCreation,
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if len(repo.operationLogs) != 1 {
		t.Fatalf("operation log count = %d, want 1", len(repo.operationLogs))
	}
	if got := repo.operationLogs[0]; got.OperationType != OperationReportFileCreation || got.TargetID != job.ID || got.Metadata["reportId"] != "report-1" {
		t.Fatalf("unexpected job operation log: %+v", got)
	}
}

func TestJobServiceCreateJobRecordsFailedOperationLogWhenEnqueueFails(t *testing.T) {
	ctx := context.Background()
	repo := &fakeJobRepository{
		report: Report{ID: "report-1", CreatorID: "user-1"},
	}
	svc := NewJobService(repo, &fakeTaskEnqueuer{err: errors.New("redis unavailable")})

	_, err := svc.CreateJob(ctx, RequestContext{UserID: "user-1", RequestID: "req-job-failed"}, CreateJobInput{
		RequestID: "req-job-failed",
		UserID:    "user-1",
		ReportID:  "report-1",
		JobType:   JobTypeContentGeneration,
	})
	if err == nil {
		t.Fatal("CreateJob() error = nil, want enqueue error")
	}
	if len(repo.operationLogs) != 1 {
		t.Fatalf("operation log count = %d, want 1", len(repo.operationLogs))
	}
	if got := repo.operationLogs[0]; got.OperationType != OperationReportJobFailed || got.OperationResult != OperationResultFailed || got.TargetType != "job" || got.RequestID != "req-job-failed" {
		t.Fatalf("unexpected failed job operation log: %+v", got)
	}
}

type fakeJobRepository struct {
	report        Report
	operationLogs []OperationLog
}

func (f *fakeJobRepository) GetReportByID(context.Context, string) (Report, error) {
	return f.report, nil
}

func (f *fakeJobRepository) FindReportJobByID(context.Context, string) (ReportJob, error) {
	return ReportJob{}, nil
}

func (f *fakeJobRepository) ListReportJobsByReportID(context.Context, string) ([]ReportJob, error) {
	return nil, nil
}

func (f *fakeJobRepository) CreateReportJob(_ context.Context, value ReportJob) (ReportJob, error) {
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	return value, nil
}

func (f *fakeJobRepository) UpdateReportJobStatus(context.Context, string, JobStatus, string, string, *time.Time, *time.Time) (ReportJob, error) {
	return ReportJob{}, nil
}

func (f *fakeJobRepository) UpdateJobAsynqTaskID(context.Context, string, string) error {
	return nil
}

func (f *fakeJobRepository) CreateReportJobAttempt(_ context.Context, value ReportJobAttempt) (ReportJobAttempt, error) {
	return value, nil
}

func (f *fakeJobRepository) UpdateAttemptAsynqTaskID(context.Context, string, string) error {
	return nil
}

func (f *fakeJobRepository) SetAttemptFailed(context.Context, string, string, string) error {
	return nil
}

func (f *fakeJobRepository) ClaimRetry(context.Context, string, string, string, string) (ReportJobAttempt, error) {
	return ReportJobAttempt{}, nil
}

func (f *fakeJobRepository) ListReportJobAttemptsByJobID(context.Context, string) ([]ReportJobAttempt, error) {
	return nil, nil
}

func (f *fakeJobRepository) ListReportEventsByReportID(context.Context, string) ([]ReportEvent, error) {
	return nil, nil
}

func (f *fakeJobRepository) CreateOperationLog(_ context.Context, log OperationLog) (OperationLog, error) {
	f.operationLogs = append(f.operationLogs, log)
	return log, nil
}

type fakeTaskEnqueuer struct {
	jobType JobType
	err     error
}

func (f *fakeTaskEnqueuer) EnqueueReportJob(_ context.Context, jobType JobType, _, _, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.jobType = jobType
	return "task-1", nil
}
