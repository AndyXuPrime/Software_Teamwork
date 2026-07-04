package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

const reportEventStreamPollInterval = time.Second

type jobResponse struct {
	ID            string            `json:"id"`
	ReportID      string            `json:"reportId"`
	TemplateID    string            `json:"templateId,omitempty"`
	JobType       string            `json:"jobType"`
	TargetType    string            `json:"targetType,omitempty"`
	TargetID      string            `json:"targetId,omitempty"`
	Status        string            `json:"status"`
	Progress      map[string]any    `json:"progress"`
	ResultSummary string            `json:"resultSummary,omitempty"`
	Error         *jobErrorResponse `json:"error,omitempty"`
	StartedAt     *string           `json:"startedAt,omitempty"`
	FinishedAt    *string           `json:"finishedAt,omitempty"`
	CreatedAt     string            `json:"createdAt"`
}

type jobErrorResponse struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type attemptResponse struct {
	ID            string            `json:"id"`
	JobID         string            `json:"jobId"`
	AttemptNumber int               `json:"attemptNumber"`
	Status        string            `json:"status"`
	Error         *jobErrorResponse `json:"error,omitempty"`
	StartedAt     *string           `json:"startedAt,omitempty"`
	FinishedAt    *string           `json:"finishedAt,omitempty"`
	CreatedAt     string            `json:"createdAt"`
}

type eventResponse struct {
	ID        string `json:"id"`
	ReportID  string `json:"reportId"`
	JobID     string `json:"jobId,omitempty"`
	EventType string `json:"eventType"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"createdAt"`
}

func toJobResponse(j service.ReportJob) jobResponse {
	r := jobResponse{
		ID:            j.ID,
		ReportID:      j.ReportID,
		TemplateID:    j.TemplateID,
		JobType:       string(j.JobType),
		TargetType:    j.TargetType,
		TargetID:      j.TargetID,
		Status:        string(j.Status),
		Progress:      j.Progress,
		ResultSummary: reportJobResultSummary(j),
		CreatedAt:     j.CreatedAt.UTC().Format(time.RFC3339),
	}
	if r.Progress == nil {
		r.Progress = map[string]any{}
	}
	if j.ErrorCode != "" || j.ErrorMessage != "" {
		r.Error = &jobErrorResponse{Code: j.ErrorCode, Message: j.ErrorMessage}
	}
	if j.StartedAt != nil {
		s := j.StartedAt.UTC().Format(time.RFC3339)
		r.StartedAt = &s
	}
	if j.FinishedAt != nil {
		f := j.FinishedAt.UTC().Format(time.RFC3339)
		r.FinishedAt = &f
	}
	return r
}

func reportJobResultSummary(j service.ReportJob) string {
	terminal := j.Status == service.JobStatusSucceeded || j.Status == service.JobStatusPartialSucceeded
	inProgress := j.Status == service.JobStatusRunning
	if !terminal && !inProgress {
		return ""
	}

	completed, total, hasProgress := reportJobProgressCounts(j.Progress)
	meaningfulProgress := hasProgress && total > 0 && (completed > 0 || total > 1 || terminal)

	switch j.JobType {
	case service.JobTypeOutlineGeneration, service.JobTypeOutlineRegeneration:
		if meaningfulProgress {
			if terminal && completed >= total {
				return fmt.Sprintf("已生成 %d 个大纲节点", completed)
			}
			return fmt.Sprintf("已生成 %d / %d 个大纲节点", completed, total)
		}
		if terminal {
			return "已生成大纲初稿"
		}
	case service.JobTypeContentGeneration, service.JobTypeContentRegeneration, service.JobTypeSectionRegeneration:
		if meaningfulProgress {
			return fmt.Sprintf("已生成 %d / %d 个章节", completed, total)
		}
		if j.Status == service.JobStatusPartialSucceeded {
			return "已完成部分正文生成"
		}
		if terminal {
			return "已完成正文生成"
		}
	case service.JobTypeReportFileCreation:
		if terminal {
			return "已生成报告文件"
		}
		if inProgress {
			return "报告文件生成中"
		}
	}
	return ""
}

func reportJobProgressCounts(progress map[string]any) (int, int, bool) {
	completed, completedOK := reportJobProgressInt(progress, "completed", "completedSections")
	total, totalOK := reportJobProgressInt(progress, "total", "totalSections")
	if !completedOK || !totalOK {
		return 0, 0, false
	}
	if completed < 0 {
		completed = 0
	}
	if total < 0 {
		total = 0
	}
	if completed > total && total > 0 {
		completed = total
	}
	return completed, total, true
}

func reportJobProgressInt(progress map[string]any, keys ...string) (int, bool) {
	if progress == nil {
		return 0, false
	}
	for _, key := range keys {
		value, ok := progress[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v, true
		case int8:
			return int(v), true
		case int16:
			return int(v), true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case uint:
			return int(v), true
		case uint8:
			return int(v), true
		case uint16:
			return int(v), true
		case uint32:
			return int(v), true
		case uint64:
			return int(v), true
		case float32:
			return int(v), true
		case float64:
			return int(v), true
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return int(i), true
			}
			if f, err := v.Float64(); err == nil {
				return int(f), true
			}
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

func toAttemptResponse(a service.ReportJobAttempt) attemptResponse {
	r := attemptResponse{
		ID:            a.ID,
		JobID:         a.JobID,
		AttemptNumber: a.AttemptNumber,
		Status:        string(a.Status),
		CreatedAt:     a.CreatedAt.UTC().Format(time.RFC3339),
	}
	if a.ErrorCode != "" || a.ErrorMessage != "" {
		r.Error = &jobErrorResponse{Code: a.ErrorCode, Message: a.ErrorMessage}
	}
	if a.StartedAt != nil {
		s := a.StartedAt.UTC().Format(time.RFC3339)
		r.StartedAt = &s
	}
	if a.FinishedAt != nil {
		f := a.FinishedAt.UTC().Format(time.RFC3339)
		r.FinishedAt = &f
	}
	return r
}

func toEventResponse(e service.ReportEvent) eventResponse {
	return eventResponse{
		ID:        e.ID,
		ReportID:  e.ReportID,
		JobID:     e.JobID,
		EventType: e.EventType,
		Message:   e.Message,
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type createJobRequest struct {
	JobType      string          `json:"jobType"`
	Target       createJobTarget `json:"target"`
	Requirements string          `json:"requirements"`
	MaterialIDs  []string        `json:"materialIds"`
	Options      map[string]any  `json:"options"`
	Retrieval    map[string]any  `json:"retrieval"`
}

type createJobTarget struct {
	Scope     string `json:"scope"`
	SectionID string `json:"sectionId"`
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	reportID := r.PathValue("reportId")
	var req createJobRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.JobType == "" {
		writeError(w, r, service.ValidationError(map[string]string{"jobType": "required"}))
		return
	}
	input := service.CreateJobInput{
		RequestID:    requestIDFromContext(r.Context()),
		UserID:       rctx.UserID,
		ReportID:     reportID,
		JobType:      service.JobType(req.JobType),
		TargetScope:  req.Target.Scope,
		SectionID:    req.Target.SectionID,
		Requirements: req.Requirements,
		MaterialIDs:  req.MaterialIDs,
		Options:      req.Options,
		Retrieval:    req.Retrieval,
	}
	job, err := s.jobSvc.CreateJob(r.Context(), rctx, input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusAccepted, toJobResponse(job))
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	reportID := r.PathValue("reportId")
	jobs, err := s.jobSvc.ListJobs(r.Context(), rctx, reportID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := make([]jobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = toJobResponse(j)
	}
	writeData(w, r, http.StatusOK, resp)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	jobID := r.PathValue("jobId")
	job, err := s.jobSvc.GetJob(r.Context(), rctx, jobID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, toJobResponse(job))
}

type retryJobRequest struct {
	Reason string `json:"reason"`
}

type updateJobRequest struct {
	Status string `json:"status"`
}

func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	jobID := r.PathValue("jobId")
	var req updateJobRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status != string(service.JobStatusCanceled) {
		writeError(w, r, service.ValidationError(map[string]string{"status": "only canceled is supported"}))
		return
	}
	job, err := s.jobSvc.CancelJob(r.Context(), rctx, jobID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, toJobResponse(job))
}

func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	jobID := r.PathValue("jobId")
	var req retryJobRequest
	if r.ContentLength != 0 {
		if !decodeJSON(w, r, &req) {
			return
		}
	}
	attempt, err := s.jobSvc.RetryJob(r.Context(), rctx, jobID, req.Reason)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusAccepted, toAttemptResponse(attempt))
}

func (s *Server) handleListAttempts(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	jobID := r.PathValue("jobId")
	attempts, err := s.jobSvc.ListAttempts(r.Context(), rctx, jobID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := make([]attemptResponse, len(attempts))
	for i, a := range attempts {
		resp[i] = toAttemptResponse(a)
	}
	writeData(w, r, http.StatusOK, resp)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	rctx := s.requestContext(r)
	reportID := r.PathValue("reportId")
	events, err := s.jobSvc.ListEvents(r.Context(), rctx, reportID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := make([]eventResponse, len(events))
	for i, e := range events {
		resp[i] = toEventResponse(e)
	}
	writeData(w, r, http.StatusOK, resp)
}

func (s *Server) handleStreamEvents(w http.ResponseWriter, r *http.Request) {
	if s.jobSvc == nil {
		writeError(w, r, service.NewError(service.CodeDependency, "job service not configured", nil))
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, service.NewError(service.CodeDependency, "event stream is not supported", nil))
		return
	}

	ctx := r.Context()
	rctx := s.requestContext(r)
	reportID := r.PathValue("reportId")

	header := w.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	seen := map[string]struct{}{}
	ticker := time.NewTicker(reportEventStreamPollInterval)
	defer ticker.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		events, err := s.jobSvc.ListEvents(ctx, rctx, reportID)
		if err != nil {
			writeSSEError(w, flusher, err)
			return
		}
		sortReportEventsChronologically(events)
		for _, event := range events {
			if _, exists := seen[event.ID]; exists {
				continue
			}
			if err := writeReportEventSSE(w, event); err != nil {
				return
			}
			seen[event.ID] = struct{}{}
			flusher.Flush()
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func sortReportEventsChronologically(events []service.ReportEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
}

func writeReportEventSSE(w http.ResponseWriter, event service.ReportEvent) error {
	data, err := json.Marshal(toEventResponse(event))
	if err != nil {
		return err
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", sanitizeSSELine(event.ID)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, "event: report.event\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	return nil
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, err error) {
	appErr, ok := service.Classify(err)
	if !ok {
		appErr = service.NewError(service.CodeDependency, "report event stream failed", err)
	}
	payload := struct {
		Code    service.Code `json:"code"`
		Message string       `json:"message"`
	}{
		Code:    appErr.Code,
		Message: appErr.Message,
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		data = []byte(`{"code":"dependency_error","message":"report event stream failed"}`)
	}
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}

func sanitizeSSELine(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
