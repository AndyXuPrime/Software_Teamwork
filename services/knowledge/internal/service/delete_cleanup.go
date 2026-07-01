package service

import (
	"context"
	"errors"
	"strings"
)

const defaultDeleteCleanupRequeueLimit = 50

func (s *Service) ProcessDeleteCleanupTask(ctx context.Context, reqCtx RequestContext, task DocumentDeleteCleanupTask) (ProcessingJob, error) {
	normalized, err := normalizeDeleteCleanupTask(task)
	if err != nil {
		return ProcessingJob{}, err
	}
	reqCtx.RequestID = strings.TrimSpace(firstNonEmpty(reqCtx.RequestID, normalized.RequestID))
	reqCtx.UserID = strings.TrimSpace(firstNonEmpty(reqCtx.UserID, normalized.UserID))
	if strings.TrimSpace(reqCtx.CallerService) == "" {
		reqCtx.CallerService = "knowledge"
	}
	if reqCtx.UserID == "" {
		return ProcessingJob{}, UnauthorizedError()
	}

	job, err := s.repo.GetProcessingJob(ctx, normalized.JobID)
	if err != nil {
		return ProcessingJob{}, repositoryError(err)
	}
	if strings.TrimSpace(job.JobType) != JobTypeDeleteCleanup {
		return ProcessingJob{}, ConflictError("job type is not supported by delete cleanup pipeline", nil)
	}
	if job.DocumentID == nil || strings.TrimSpace(*job.DocumentID) == "" {
		return ProcessingJob{}, ConflictError("job has no document", nil)
	}
	if job.KnowledgeBaseID != normalized.KnowledgeBaseID || strings.TrimSpace(*job.DocumentID) != normalized.DocumentID {
		return ProcessingJob{}, ConflictError("worker payload does not match job", nil)
	}

	now := s.now()
	staleRunningBefore := runningStaleBefore(now, s.runningLease)
	if job.Status == JobStatusSucceeded {
		return job, nil
	}
	if job.Status == JobStatusFailed && hasExhaustedJobAttempts(job) {
		return job, ConflictError("job has reached max attempts", nil)
	}
	if job.Status == JobStatusRunning && isStaleRunningJob(job, staleRunningBefore) && hasExhaustedJobAttempts(job) {
		failed, failErr := s.failDeleteCleanup(ctx, job, normalized.DocumentID, string(CodeDependency), "delete cleanup job reached max attempts")
		if failErr != nil {
			return failed, failErr
		}
		return failed, ConflictError("job has reached max attempts", nil)
	}
	if job.Status == JobStatusRunning && !isStaleRunningJob(job, staleRunningBefore) {
		return job, DependencyError("job is already running", nil)
	}
	if job.Status != JobStatusQueued && job.Status != JobStatusFailed && job.Status != JobStatusRunning {
		return job, ConflictError("job is not ready to run", nil)
	}

	startedAt := now
	cleanupStage := "delete_cleanup"
	runningMessage := "document delete cleanup running"
	job, err = s.repo.ClaimProcessingJob(ctx, job.ID, JobStateUpdate{
		Status:             JobStatusRunning,
		CurrentStage:       &cleanupStage,
		ProgressPercent:    20,
		Message:            &runningMessage,
		StartedAt:          &startedAt,
		UpdatedAt:          startedAt,
		StaleRunningBefore: staleRunningBefore,
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			latest, latestErr := s.repo.GetProcessingJob(ctx, normalized.JobID)
			if latestErr != nil {
				return job, DependencyError("job state update failed", latestErr)
			}
			if latest.Status == JobStatusSucceeded {
				return latest, nil
			}
			if latest.Status == JobStatusFailed && hasExhaustedJobAttempts(latest) {
				return latest, ConflictError("job has reached max attempts", err)
			}
			if latest.Status == JobStatusRunning {
				return latest, DependencyError("job is already running", err)
			}
			return latest, ConflictError("job is not ready to run", err)
		}
		return ProcessingJob{}, DependencyError("job state update failed", err)
	}

	target, err := s.repo.GetDeletedDocumentCleanupTarget(ctx, job.ID)
	if err != nil {
		return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, string(CodeDependency), "delete cleanup target lookup failed",
			DependencyError("delete cleanup target lookup failed", err))
	}
	if target.DocumentID != normalized.DocumentID || target.KnowledgeBaseID != normalized.KnowledgeBaseID {
		return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, string(CodeConflict), "delete cleanup target mismatch",
			ConflictError("delete cleanup target mismatch", nil))
	}

	if target.FileRef != nil && strings.TrimSpace(*target.FileRef) != "" && s.files == nil {
		return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, string(CodeDependency), "file cleanup client is not configured",
			DependencyError("file cleanup client is not configured", nil))
	}
	if s.files != nil && target.FileRef != nil && strings.TrimSpace(*target.FileRef) != "" {
		if err := s.files.DeleteFile(ctx, reqCtx, strings.TrimSpace(*target.FileRef)); err != nil {
			return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, cleanupFailureCode(err), "file cleanup failed",
				cleanupFailureError(err, "file cleanup failed"))
		}
	}

	vectorAt := s.now()
	vectorStage := "vector_cleanup"
	vectorMessage := "document vector cleanup running"
	job, err = s.repo.UpdateJobState(ctx, job.ID, JobStateUpdate{
		Status:           JobStatusRunning,
		CurrentStage:     &vectorStage,
		ProgressPercent:  70,
		Message:          &vectorMessage,
		UpdatedAt:        vectorAt,
		ExpectedAttempts: &job.Attempts,
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return job, ConflictError("job attempt is no longer active", err)
		}
		return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, string(CodeDependency), "job state update failed",
			DependencyError("job state update failed", err))
	}
	if s.vectorIndex != nil {
		if err := s.vectorIndex.DeleteByDocument(ctx, normalized.DocumentID); err != nil {
			return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, cleanupFailureCode(err), "vector cleanup failed",
				cleanupFailureError(err, "vector cleanup failed"))
		}
	}

	finishedAt := s.now()
	completedStage := "completed"
	completedMessage := "document delete cleanup completed"
	completed, err := s.repo.UpdateJobState(ctx, job.ID, JobStateUpdate{
		Status:           JobStatusSucceeded,
		CurrentStage:     &completedStage,
		ProgressPercent:  100,
		Message:          &completedMessage,
		FinishedAt:       &finishedAt,
		UpdatedAt:        finishedAt,
		ExpectedAttempts: &job.Attempts,
	})
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return job, ConflictError("job attempt is no longer active", err)
		}
		return s.failDeleteCleanupAndReturn(ctx, job, normalized.DocumentID, string(CodeDependency), "delete cleanup completion failed",
			DependencyError("delete cleanup completion failed", err))
	}
	return completed, nil
}

func (s *Service) ProcessDeleteCleanupJob(ctx context.Context, reqCtx RequestContext, jobID string) (ProcessingJob, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return ProcessingJob{}, ValidationError("worker payload validation failed", map[string]string{"jobId": "is required"})
	}
	job, err := s.repo.GetProcessingJob(ctx, jobID)
	if err != nil {
		return ProcessingJob{}, repositoryError(err)
	}
	if job.DocumentID == nil || strings.TrimSpace(*job.DocumentID) == "" {
		return ProcessingJob{}, ConflictError("job has no document", nil)
	}
	return s.ProcessDeleteCleanupTask(ctx, reqCtx, DocumentDeleteCleanupTask{
		RequestID:       reqCtx.RequestID,
		JobID:           job.ID,
		DocumentID:      strings.TrimSpace(*job.DocumentID),
		KnowledgeBaseID: job.KnowledgeBaseID,
		UserID:          reqCtx.UserID,
	})
}

func (s *Service) RequeueDeleteCleanupTasks(ctx context.Context, reqCtx RequestContext, limit int) (DeleteCleanupRequeueResult, error) {
	if s.queue == nil {
		return DeleteCleanupRequeueResult{FailedDependency: "redis"}, DependencyError("delete cleanup queue is not configured", nil)
	}
	requestID := strings.TrimSpace(reqCtx.RequestID)
	if requestID == "" {
		requestID = "delete_cleanup_reconciler"
	}
	tasks, err := s.repo.ListRetryableDeleteCleanupTasks(ctx, DeleteCleanupTaskListInput{
		RequestID:          requestID,
		Limit:              normalizeDeleteCleanupRequeueLimit(limit),
		StaleRunningBefore: runningStaleBefore(s.now(), s.runningLease),
	})
	if err != nil {
		return DeleteCleanupRequeueResult{FailedDependency: "postgres"}, repositoryError(err)
	}

	result := DeleteCleanupRequeueResult{Scanned: len(tasks)}
	for _, task := range tasks {
		task.RequestID = requestID
		normalized, err := normalizeDeleteCleanupTask(task)
		if err != nil {
			result.Failed++
			result.FailedDependency = "postgres"
			_ = s.repo.MarkDocumentJobFailed(ctx, task.DocumentID, task.JobID, nil, string(CodeDependency), "delete cleanup queue handoff failed", s.now())
			return result, DependencyError("delete cleanup task payload is invalid", err)
		}
		if err := s.queue.EnqueueDocumentDeleteCleanup(ctx, normalized); err != nil {
			result.Failed++
			result.FailedDependency = "redis"
			_ = s.repo.MarkDocumentJobFailed(ctx, task.DocumentID, task.JobID, nil, string(CodeDependency), "delete cleanup queue handoff failed", s.now())
			return result, DependencyError("delete cleanup queue handoff failed", err)
		}
		result.Enqueued++
	}
	return result, nil
}

func (s *Service) failDeleteCleanupAndReturn(ctx context.Context, job ProcessingJob, documentID string, code string, message string, cleanupErr error) (ProcessingJob, error) {
	failed, err := s.failDeleteCleanup(ctx, job, documentID, code, message)
	if err != nil {
		return failed, err
	}
	// Once the retry budget is exhausted, the durable job state is already failed;
	// returning a conflict lets the worker ack the final delivery without hiding the failure.
	if hasExhaustedJobAttempts(failed) {
		return failed, ConflictError("job has reached max attempts", cleanupErr)
	}
	return failed, cleanupErr
}

func (s *Service) failDeleteCleanup(ctx context.Context, job ProcessingJob, documentID string, code string, message string) (ProcessingJob, error) {
	now := s.now()
	if err := s.repo.MarkDocumentJobFailed(ctx, documentID, job.ID, &job.Attempts, code, message, now); err != nil {
		if errors.Is(err, ErrConflict) {
			return job, ConflictError("job attempt is no longer active", err)
		}
		return job, DependencyError("failed to persist delete cleanup failure state", err)
	}
	failed, err := s.repo.GetProcessingJob(ctx, job.ID)
	if err != nil {
		return job, DependencyError("failed to reload delete cleanup failure state", err)
	}
	return failed, nil
}

func normalizeDeleteCleanupTask(task DocumentDeleteCleanupTask) (DocumentDeleteCleanupTask, error) {
	task.RequestID = strings.TrimSpace(task.RequestID)
	task.JobID = strings.TrimSpace(task.JobID)
	task.DocumentID = strings.TrimSpace(task.DocumentID)
	task.KnowledgeBaseID = strings.TrimSpace(task.KnowledgeBaseID)
	task.UserID = strings.TrimSpace(task.UserID)
	fields := map[string]string{}
	if task.RequestID == "" {
		fields["requestId"] = "is required"
	}
	if task.JobID == "" {
		fields["jobId"] = "is required"
	}
	if task.DocumentID == "" {
		fields["documentId"] = "is required"
	}
	if task.KnowledgeBaseID == "" {
		fields["knowledgeBaseId"] = "is required"
	}
	if task.UserID == "" {
		fields["userId"] = "is required"
	}
	if len(fields) > 0 {
		return DocumentDeleteCleanupTask{}, ValidationError("worker payload validation failed", fields)
	}
	return task, nil
}

func normalizeDeleteCleanupRequeueLimit(limit int) int {
	if limit <= 0 {
		return defaultDeleteCleanupRequeueLimit
	}
	if limit > maxPageSize {
		return maxPageSize
	}
	return limit
}

func cleanupFailureCode(err error) string {
	if appErr, ok := Classify(err); ok && appErr.Code != "" {
		return string(appErr.Code)
	}
	return string(CodeDependency)
}

func cleanupFailureError(err error, message string) error {
	if appErr, ok := Classify(err); ok {
		return NewError(appErr.Code, message, err)
	}
	return DependencyError(message, err)
}
