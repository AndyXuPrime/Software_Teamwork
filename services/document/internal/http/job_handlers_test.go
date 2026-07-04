package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// mockJobSvc implements JobSvc for testing.
type mockJobSvc struct {
	createJobFn    func(ctx context.Context, rctx service.RequestContext, input service.CreateJobInput) (service.ReportJob, error)
	listJobsFn     func(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportJob, error)
	getJobFn       func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error)
	cancelJobFn    func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error)
	retryJobFn     func(ctx context.Context, rctx service.RequestContext, id, reason string) (service.ReportJobAttempt, error)
	listAttemptsFn func(ctx context.Context, rctx service.RequestContext, jobID string) ([]service.ReportJobAttempt, error)
	listEventsFn   func(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportEvent, error)
}

func (m *mockJobSvc) CreateJob(ctx context.Context, rctx service.RequestContext, input service.CreateJobInput) (service.ReportJob, error) {
	return m.createJobFn(ctx, rctx, input)
}

func (m *mockJobSvc) ListJobs(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportJob, error) {
	return m.listJobsFn(ctx, rctx, reportID)
}

func (m *mockJobSvc) GetJob(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
	return m.getJobFn(ctx, rctx, id)
}

func (m *mockJobSvc) CancelJob(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
	return m.cancelJobFn(ctx, rctx, id)
}

func (m *mockJobSvc) RetryJob(ctx context.Context, rctx service.RequestContext, id, reason string) (service.ReportJobAttempt, error) {
	return m.retryJobFn(ctx, rctx, id, reason)
}

func (m *mockJobSvc) ListAttempts(ctx context.Context, rctx service.RequestContext, jobID string) ([]service.ReportJobAttempt, error) {
	return m.listAttemptsFn(ctx, rctx, jobID)
}

func (m *mockJobSvc) ListEvents(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportEvent, error) {
	return m.listEventsFn(ctx, rctx, reportID)
}

func newTestServerWithJobSvc(svc JobSvc) *Server {
	return NewServer(Config{JobSvc: svc})
}

func TestCreateJobAcceptsGenerationPayload(t *testing.T) {
	var captured service.CreateJobInput
	mock := &mockJobSvc{
		createJobFn: func(ctx context.Context, rctx service.RequestContext, input service.CreateJobInput) (service.ReportJob, error) {
			captured = input
			return service.ReportJob{
				ID:          "job-1",
				ReportID:    input.ReportID,
				JobType:     input.JobType,
				TargetType:  input.TargetScope,
				TargetID:    input.SectionID,
				Status:      service.JobStatusPending,
				MaxAttempts: 3,
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)
	body := `{
		"jobType": "section_regeneration",
		"target": {"scope": "section", "sectionId": "section-1"},
		"requirements": "focus on overload risks",
		"materialIds": ["material-1"],
		"options": {"knowledgeBaseIds": ["kb-1"], "topK": 3}
	}`

	req := httptest.NewRequest(http.MethodPost, "/reports/report-1/jobs", strings.NewReader(body))
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if captured.JobType != service.JobTypeSectionRegeneration || captured.TargetScope != "section" || captured.SectionID != "section-1" {
		t.Fatalf("captured input = %+v", captured)
	}
	if captured.Requirements != "focus on overload risks" {
		t.Fatalf("captured requirements = %q", captured.Requirements)
	}
	if len(captured.MaterialIDs) != 1 || captured.MaterialIDs[0] != "material-1" {
		t.Fatalf("captured material IDs = %#v", captured.MaterialIDs)
	}
	if captured.Options["topK"] != float64(3) {
		t.Fatalf("captured options = %#v", captured.Options)
	}
}

func TestGetJobResponseMatchesGatewayReportJobContract(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	mock := &mockJobSvc{
		getJobFn: func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
			return service.ReportJob{
				ID:         id,
				ReportID:   "report-1",
				TemplateID: "template-1",
				JobType:    service.JobTypeContentGeneration,
				TargetType: "report",
				TargetID:   "report-1",
				Status:     service.JobStatusSucceeded,
				Progress: map[string]any{
					"completed": 2,
					"total":     4,
				},
				ErrorCode:    "execution_failed",
				ErrorMessage: "report job execution failed",
				StartedAt:    &now,
				FinishedAt:   &now,
				CreatedAt:    now,
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/report-jobs/job-1", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, field := range []string{"id", "reportId", "templateId", "jobType", "targetType", "targetId", "status", "progress", "resultSummary", "error", "createdAt"} {
		if _, ok := body.Data[field]; !ok {
			t.Fatalf("response missing %s: %#v", field, body.Data)
		}
	}
	for _, legacyField := range []string{"attemptCount", "maxAttempts", "errorCode", "errorMessage"} {
		if _, ok := body.Data[legacyField]; ok {
			t.Fatalf("response contains legacy field %s: %#v", legacyField, body.Data)
		}
	}
	if body.Data["resultSummary"] != "已生成 2 / 4 个章节" {
		t.Fatalf("resultSummary = %#v", body.Data["resultSummary"])
	}
	errObj, ok := body.Data["error"].(map[string]any)
	if !ok || errObj["code"] != "execution_failed" || errObj["message"] != "report job execution failed" {
		t.Fatalf("error = %#v", body.Data["error"])
	}
}

func TestListJobsEmptyList(t *testing.T) {
	mock := &mockJobSvc{
		listJobsFn: func(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportJob, error) {
			return []service.ReportJob{}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/550e8400-e29b-41d4-a716-446655440000/jobs", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Data []jobResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("expected empty list, got %d items", len(body.Data))
	}
}

func TestGetJobNotFound(t *testing.T) {
	mock := &mockJobSvc{
		getJobFn: func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
			return service.ReportJob{}, service.NewError(service.CodeNotFound, "report job not found", nil)
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/report-jobs/550e8400-e29b-41d4-a716-446655440001", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetJobForbidden(t *testing.T) {
	mock := &mockJobSvc{
		getJobFn: func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
			if rctx.UserID != "usr_owner" {
				return service.ReportJob{}, service.NewError(service.CodeForbidden, "you do not have access to this report", nil)
			}
			return service.ReportJob{ID: id}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/report-jobs/550e8400-e29b-41d4-a716-446655440001", nil)
	req.Header.Set("X-User-Id", "usr_other")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 Forbidden", rec.Code)
	}
}

func TestCancelJobUpdatesRunningJobStatus(t *testing.T) {
	now := time.Now().UTC()
	var capturedID string
	mock := &mockJobSvc{
		cancelJobFn: func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
			capturedID = id
			if rctx.UserID != "usr_owner" {
				return service.ReportJob{}, service.NewError(service.CodeForbidden, "you do not have access to this report", nil)
			}
			return service.ReportJob{
				ID:         id,
				ReportID:   "report-1",
				JobType:    service.JobTypeContentGeneration,
				Status:     service.JobStatusCanceled,
				FinishedAt: &now,
				CreatedAt:  now.Add(-time.Minute),
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodPatch, "/report-jobs/job-1", strings.NewReader(`{"status":"canceled"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if capturedID != "job-1" {
		t.Fatalf("captured job id = %q, want job-1", capturedID)
	}
	var body struct {
		Data jobResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Status != string(service.JobStatusCanceled) {
		t.Fatalf("status = %q, want canceled", body.Data.Status)
	}
}

func TestCancelJobRejectsUnsupportedPatchStatus(t *testing.T) {
	mock := &mockJobSvc{
		cancelJobFn: func(ctx context.Context, rctx service.RequestContext, id string) (service.ReportJob, error) {
			t.Fatal("CancelJob should not be called for unsupported status")
			return service.ReportJob{}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodPatch, "/report-jobs/job-1", strings.NewReader(`{"status":"running"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestRetryJobMaxAttemptsReached(t *testing.T) {
	mock := &mockJobSvc{
		retryJobFn: func(ctx context.Context, rctx service.RequestContext, id, reason string) (service.ReportJobAttempt, error) {
			return service.ReportJobAttempt{}, service.NewError(service.CodeValidation, "max retry attempts reached", nil)
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodPost, "/report-jobs/550e8400-e29b-41d4-a716-446655440001/attempts", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRetryJobAttemptResponseMatchesGatewayContract(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
	mock := &mockJobSvc{
		retryJobFn: func(ctx context.Context, rctx service.RequestContext, id, reason string) (service.ReportJobAttempt, error) {
			return service.ReportJobAttempt{
				ID:            "attempt-2",
				JobID:         id,
				AttemptNumber: 2,
				TriggerSource: "manual",
				Status:        service.JobStatusFailed,
				ErrorCode:     "execution_failed",
				ErrorMessage:  "report job execution failed",
				StartedAt:     &now,
				FinishedAt:    &now,
				CreatedAt:     now,
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodPost, "/report-jobs/job-1/attempts", strings.NewReader(`{"reason":"retry"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertAttemptResponseMatchesGatewayContract(t, body.Data)
}

func TestListAttempts(t *testing.T) {
	now := time.Now().UTC()
	mock := &mockJobSvc{
		listAttemptsFn: func(ctx context.Context, rctx service.RequestContext, jobID string) ([]service.ReportJobAttempt, error) {
			return []service.ReportJobAttempt{
				{
					ID:            "attempt-1",
					JobID:         jobID,
					AttemptNumber: 1,
					TriggerSource: "system",
					Status:        service.JobStatusSucceeded,
					ErrorCode:     "execution_failed",
					ErrorMessage:  "report job execution failed",
					CreatedAt:     now,
				},
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/report-jobs/550e8400-e29b-41d4-a716-446655440001/attempts", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("attempt count = %d, want 1", len(body.Data))
	}
	assertAttemptResponseMatchesGatewayContract(t, body.Data[0])
}

func TestListEvents(t *testing.T) {
	mock := &mockJobSvc{
		listEventsFn: func(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportEvent, error) {
			return []service.ReportEvent{}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/550e8400-e29b-41d4-a716-446655440000/events", nil)
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func assertAttemptResponseMatchesGatewayContract(t *testing.T, data map[string]any) {
	t.Helper()

	for _, field := range []string{"id", "jobId", "attemptNumber", "status", "error", "createdAt"} {
		if _, ok := data[field]; !ok {
			t.Fatalf("attempt response missing %s: %#v", field, data)
		}
	}
	for _, legacyField := range []string{"triggerSource", "errorCode", "errorMessage"} {
		if _, ok := data[legacyField]; ok {
			t.Fatalf("attempt response contains legacy field %s: %#v", legacyField, data)
		}
	}
	errObj, ok := data["error"].(map[string]any)
	if !ok || errObj["code"] != "execution_failed" || errObj["message"] != "report job execution failed" {
		t.Fatalf("error = %#v", data["error"])
	}
}

func TestStreamEventsEmitsPersistedReportEventsAsSSE(t *testing.T) {
	now := time.Date(2026, 7, 4, 7, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mock := &mockJobSvc{
		listEventsFn: func(ctx context.Context, rctx service.RequestContext, reportID string) ([]service.ReportEvent, error) {
			if reportID != "550e8400-e29b-41d4-a716-446655440000" {
				t.Fatalf("reportID = %q", reportID)
			}
			cancel()
			return []service.ReportEvent{
				{
					ID:        "evt-1",
					ReportID:  reportID,
					JobID:     "job-1",
					EventType: "outline.delta",
					Message:   "{\"title\":\"Live outline\"}",
					CreatedAt: now,
				},
			}, nil
		},
	}
	server := newTestServerWithJobSvc(mock)

	req := httptest.NewRequest(http.MethodGet, "/reports/550e8400-e29b-41d4-a716-446655440000/events/stream", nil).WithContext(ctx)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("X-User-Id", "usr_owner")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cacheControl)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"id: evt-1",
		"event: report.event",
		`"eventType":"outline.delta"`,
		`"message":"{\"title\":\"Live outline\"}"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("SSE body missing %q in:\n%s", want, body)
		}
	}
}
