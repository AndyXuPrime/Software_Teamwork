package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestReportGenerationServicePersistsAIOutlineAndSectionSkeletons(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeOutlineGeneration, ReportID: "report-1"}
	repo.templateStructures["template-1"] = ReportTemplateStructure{OutlineSchema: []byte(`{"sections":["overview"]}`)}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{
			Content: `{"sections":[{"title":"Overview"},{"title":"Risk inspection","children":[{"title":"Equipment load"}]}]}`,
		}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-outline",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	if len(repo.outlines) != 1 {
		t.Fatalf("outline count = %d, want 1", len(repo.outlines))
	}
	var outline ReportOutline
	for _, item := range repo.outlines {
		outline = item
	}
	if outline.ReportID != "report-1" || outline.Source != OutlineSourceAI || !outline.IsCurrent || outline.SourceJobID != "job-1" {
		t.Fatalf("outline = %+v", outline)
	}
	if len(outline.Sections) != 2 || outline.Sections[1].Children[0].Numbering != "2.1" {
		t.Fatalf("outline sections not normalized: %+v", outline.Sections)
	}
	if len(repo.sections) != 3 {
		t.Fatalf("section skeleton count = %d, want 3", len(repo.sections))
	}
	for _, section := range repo.sections {
		if section.OutlineID != outline.ID || section.LastJobID != "job-1" || section.GenerationStatus != JobStatusPending {
			t.Fatalf("section skeleton = %+v", section)
		}
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat request count = %d, want 1", len(chat.requests))
	}
	if strings.Contains(chat.requests[0].Messages[0].Content, "sk-secret") {
		t.Fatalf("prompt unexpectedly contains secret marker: %+v", chat.requests[0].Messages)
	}
}

func TestReportGenerationServiceSkipsExecutionWhenJobAlreadyCanceled(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:       "job-1",
		JobType:  JobTypeOutlineGeneration,
		ReportID: "report-1",
		Status:   JobStatusCanceled,
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"sections":[{"title":"Overview"}]}`}},
	}
	svc := NewReportGenerationService(repo, chat)

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-canceled",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusCanceled {
		t.Fatalf("result status = %q, want canceled", result.Status)
	}
	if len(chat.requests) != 0 {
		t.Fatalf("chat request count = %d, want 0 for canceled job", len(chat.requests))
	}
	if len(repo.outlines) != 0 || len(repo.sections) != 0 {
		t.Fatalf("canceled job should not write outline/sections, outlines=%+v sections=%+v", repo.outlines, repo.sections)
	}
}

func TestReportGenerationServiceSupportsCoalInventoryAuditAIJobs(t *testing.T) {
	t.Run("outline generation", func(t *testing.T) {
		repo := newFakeReportGenerationRepository()
		repo.reports["report-1"] = Report{
			ID:         "report-1",
			Name:       "煤库存审计报告",
			ReportType: "coal_inventory_audit",
			Topic:      "煤场库存核查",
			CreatorID:  "user-1",
			Status:     ReportStatusDraft,
		}
		repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeOutlineGeneration, ReportID: "report-1"}
		chat := &fakeGenerationChatClient{
			responses: []ChatCompletionResponse{{
				Content: `{"sections":[{"title":"审计范围与依据"},{"title":"库存账实核查"}]}`,
			}},
		}
		svc := NewReportGenerationService(repo, chat)

		result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
			RequestID: "req-outline",
			JobType:   JobTypeOutlineGeneration,
			JobID:     "job-1",
			UserID:    "user-1",
		})
		if err != nil {
			t.Fatalf("ExecuteReportGeneration() error = %v", err)
		}
		if result.Status != JobStatusSucceeded {
			t.Fatalf("result status = %q, want succeeded", result.Status)
		}
		if len(chat.requests) != 1 {
			t.Fatalf("chat request count = %d, want 1", len(chat.requests))
		}
		systemPrompt := chat.requests[0].Messages[0].Content
		if !strings.Contains(systemPrompt, "煤库存审计报告") {
			t.Fatalf("system prompt = %q, want coal inventory report label", systemPrompt)
		}
		if strings.Contains(systemPrompt, "迎峰度夏检查报告") {
			t.Fatalf("system prompt still uses summer inspection label: %q", systemPrompt)
		}
	})

	t.Run("content generation", func(t *testing.T) {
		repo := newFakeReportGenerationRepository()
		repo.reports["report-1"] = Report{
			ID:         "report-1",
			Name:       "煤库存审计报告",
			ReportType: "coal_inventory_audit",
			Topic:      "煤场库存核查",
			CreatorID:  "user-1",
			Status:     ReportStatusOutlineGenerated,
		}
		repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
		repo.sections["section-1"] = ReportSection{
			ID:               "section-1",
			ReportID:         "report-1",
			Title:            "库存账实核查",
			SortOrder:        0,
			Version:          1,
			GenerationStatus: JobStatusPending,
		}
		chat := &fakeGenerationChatClient{
			responses: []ChatCompletionResponse{{Content: `{"content":"煤场账实相符率估算为98.2%。","tables":[]}`}},
		}
		svc := NewReportGenerationService(repo, chat)

		result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
			RequestID: "req-content",
			JobType:   JobTypeContentGeneration,
			JobID:     "job-1",
			UserID:    "user-1",
		})
		if err != nil {
			t.Fatalf("ExecuteReportGeneration() error = %v", err)
		}
		if result.Status != JobStatusSucceeded {
			t.Fatalf("result status = %q, want succeeded", result.Status)
		}
		if len(chat.requests) != 1 {
			t.Fatalf("chat request count = %d, want 1", len(chat.requests))
		}
		systemPrompt := chat.requests[0].Messages[0].Content
		if !strings.Contains(systemPrompt, "煤库存审计报告") {
			t.Fatalf("system prompt = %q, want coal inventory report label", systemPrompt)
		}
		if strings.Contains(systemPrompt, "迎峰度夏检查报告") {
			t.Fatalf("system prompt still uses summer inspection label: %q", systemPrompt)
		}
	})
}

func TestReportGenerationServiceRollsBackOutlineAndSkeletonsWhenSkeletonCreationFails(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeOutlineGeneration, ReportID: "report-1"}
	repo.templateStructures["template-1"] = ReportTemplateStructure{OutlineSchema: []byte(`{"sections":["overview"]}`)}
	repo.outlines["outline-old"] = ReportOutline{
		ID:        "outline-old",
		ReportID:  "report-1",
		Version:   1,
		IsCurrent: true,
	}
	repo.createSectionErrAfter = 1
	repo.createSectionErr = errors.New("insert section skeleton failed")
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{
			Content: `{"sections":[{"title":"Overview"},{"title":"Risk inspection"}]}`,
		}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-outline",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err == nil {
		t.Fatal("ExecuteReportGeneration() error = nil, want skeleton creation failure")
	}
	if len(repo.outlines) != 1 {
		t.Fatalf("outline count = %d, want previous outline only after rollback", len(repo.outlines))
	}
	if !repo.outlines["outline-old"].IsCurrent {
		t.Fatalf("previous outline current flag = false, want restored current outline")
	}
	if len(repo.sections) != 0 {
		t.Fatalf("section skeleton count = %d, want rollback of partial skeletons", len(repo.sections))
	}
}

func TestReportGenerationServiceKeepsGeneratedSectionsOnPartialFailure(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
	repo.sections["section-1"] = ReportSection{ID: "section-1", ReportID: "report-1", Title: "Overview", SortOrder: 0, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-2"] = ReportSection{ID: "section-2", ReportID: "report-1", Title: "Risk inspection", SortOrder: 1, Version: 1, GenerationStatus: JobStatusPending}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"first section body","tables":[{"name":"stats table"}]}`}},
		errs:      []error{nil, errors.New("provider raw error sk-secret https://provider.internal")},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusPartialSucceeded {
		t.Fatalf("result status = %q, want partial_succeeded", result.Status)
	}
	first := repo.sections["section-1"]
	if first.Content != "first section body" || first.Version != 2 || first.GenerationStatus != JobStatusSucceeded || first.ContentSource != ContentSourceAI {
		t.Fatalf("first section not persisted as generated: %+v", first)
	}
	if len(repo.sectionVersions["section-1"]) != 1 || repo.sectionVersions["section-1"][0].Content != "first section body" {
		t.Fatalf("section versions = %+v", repo.sectionVersions["section-1"])
	}
	second := repo.sections["section-2"]
	if second.GenerationStatus != JobStatusFailed || second.Content != "" {
		t.Fatalf("second section should be failed without content overwrite: %+v", second)
	}
	for _, event := range repo.events {
		if strings.Contains(event.Message, "sk-secret") || strings.Contains(event.Message, "provider.internal") {
			t.Fatalf("event leaked provider details: %+v", event)
		}
	}
}

func TestReportGenerationServiceContinuesContentGenerationAfterSectionFailure(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
	repo.sections["section-1"] = ReportSection{ID: "section-1", ReportID: "report-1", Title: "Overview", SortOrder: 0, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-2"] = ReportSection{ID: "section-2", ReportID: "report-1", Title: "Risk inspection", SortOrder: 1, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-3"] = ReportSection{ID: "section-3", ReportID: "report-1", Title: "Action plan", SortOrder: 2, Version: 1, GenerationStatus: JobStatusPending}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{
			{Content: `{"content":"first section body","tables":[]}`},
			{},
			{Content: `{"content":"third section body","tables":[]}`},
		},
		errs: []error{nil, errors.New("provider raw error sk-secret https://provider.internal"), nil},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusPartialSucceeded {
		t.Fatalf("result status = %q, want partial_succeeded", result.Status)
	}
	if len(chat.requests) != 3 {
		t.Fatalf("chat request count = %d, want 3", len(chat.requests))
	}
	first := repo.sections["section-1"]
	if first.Content != "first section body" || first.GenerationStatus != JobStatusSucceeded {
		t.Fatalf("first section not generated: %+v", first)
	}
	second := repo.sections["section-2"]
	if second.GenerationStatus != JobStatusFailed || second.Content != "" {
		t.Fatalf("second section should be failed without content overwrite: %+v", second)
	}
	third := repo.sections["section-3"]
	if third.Content != "third section body" || third.GenerationStatus != JobStatusSucceeded {
		t.Fatalf("third section should still be generated after second failed: %+v", third)
	}
	lastProgress := repo.progressUpdates[len(repo.progressUpdates)-1]
	if lastProgress["completed"] != 3 || lastProgress["total"] != 3 {
		t.Fatalf("last progress = %+v, want 3/3 attempted sections", lastProgress)
	}
	if !hasReportEvent(repo.events, "content.partial_succeeded") {
		t.Fatalf("expected content.partial_succeeded event, got %+v", repo.events)
	}
}

func TestReportGenerationServiceStopsContentGenerationWhenJobIsCanceledBetweenSections(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1", Status: JobStatusRunning}
	repo.cancelJobAfterProgressUpdates = 1
	repo.sections["section-1"] = ReportSection{ID: "section-1", ReportID: "report-1", Title: "Overview", SortOrder: 0, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-2"] = ReportSection{ID: "section-2", ReportID: "report-1", Title: "Risk inspection", SortOrder: 1, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-3"] = ReportSection{ID: "section-3", ReportID: "report-1", Title: "Action plan", SortOrder: 2, Version: 1, GenerationStatus: JobStatusPending}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{
			{Content: `{"content":"first section body","tables":[]}`},
			{Content: `{"content":"second section body","tables":[]}`},
			{Content: `{"content":"third section body","tables":[]}`},
		},
	}
	svc := NewReportGenerationService(repo, chat)

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusCanceled {
		t.Fatalf("result status = %q, want canceled", result.Status)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat request count = %d, want only first section before cancellation", len(chat.requests))
	}
	if got := repo.sections["section-1"]; got.Content != "first section body" || got.GenerationStatus != JobStatusSucceeded {
		t.Fatalf("first section should be generated before cancellation: %+v", got)
	}
	if got := repo.sections["section-2"]; got.Content != "" || got.GenerationStatus != JobStatusPending {
		t.Fatalf("second section should remain pending after cancellation: %+v", got)
	}
	if got := repo.sections["section-3"]; got.Content != "" || got.GenerationStatus != JobStatusPending {
		t.Fatalf("third section should remain pending after cancellation: %+v", got)
	}
	if !hasReportEvent(repo.events, "content.canceled") {
		t.Fatalf("expected content.canceled event, got %+v", repo.events)
	}
}

func TestReportGenerationServiceContinuesAfterSectionVersionPersistenceFailure(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
	repo.sections["section-1"] = ReportSection{ID: "section-1", ReportID: "report-1", Title: "Overview", SortOrder: 0, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-2"] = ReportSection{ID: "section-2", ReportID: "report-1", Title: "Risk inspection", SortOrder: 1, Version: 1, GenerationStatus: JobStatusPending}
	repo.sections["section-3"] = ReportSection{ID: "section-3", ReportID: "report-1", Title: "Action plan", SortOrder: 2, Version: 1, GenerationStatus: JobStatusPending}
	repo.createSectionVersionErrBySection = map[string]error{
		"section-2": errors.New("insert section version failed"),
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{
			{Content: `{"content":"first section body","tables":[]}`},
			{Content: `{"content":"second section body","tables":[]}`},
			{Content: `{"content":"third section body","tables":[]}`},
		},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusPartialSucceeded {
		t.Fatalf("result status = %q, want partial_succeeded", result.Status)
	}
	if len(chat.requests) != 3 {
		t.Fatalf("chat request count = %d, want 3", len(chat.requests))
	}
	if got := repo.sections["section-1"]; got.Content != "first section body" || got.GenerationStatus != JobStatusSucceeded {
		t.Fatalf("first section not generated: %+v", got)
	}
	if got := repo.sections["section-2"]; got.Content != "" || got.GenerationStatus != JobStatusFailed || got.Version != 1 {
		t.Fatalf("second section should roll back content and be marked failed: %+v", got)
	}
	if got := repo.sections["section-3"]; got.Content != "third section body" || got.GenerationStatus != JobStatusSucceeded {
		t.Fatalf("third section should still be generated after version write failure: %+v", got)
	}
	lastProgress := repo.progressUpdates[len(repo.progressUpdates)-1]
	if lastProgress["completed"] != 3 || lastProgress["total"] != 3 {
		t.Fatalf("last progress = %+v, want 3/3 attempted sections", lastProgress)
	}
	if !hasReportEvent(repo.events, "content.partial_succeeded") {
		t.Fatalf("expected content.partial_succeeded event, got %+v", repo.events)
	}
}

func TestReportGenerationServiceFailsWhenSectionRunningMarkerCannotPersist(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Pending section",
		SortOrder:        0,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	repo.markSectionRunningErr = errors.New("update running marker failed")
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)

	_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if code := errorCode(t, err); code != CodeDependency {
		t.Fatalf("error code = %q, want %q", code, CodeDependency)
	}
	if len(chat.requests) != 0 {
		t.Fatalf("chat request count = %d, want 0 when running marker cannot persist", len(chat.requests))
	}
	got := repo.sections["section-1"]
	if got.GenerationStatus != JobStatusPending || got.LastJobID != "" || got.Content != "" {
		t.Fatalf("section changed after failed running marker: %+v", got)
	}
	if !hasReportEvent(repo.events, "section.failed") {
		t.Fatalf("expected section.failed event, got %+v", repo.events)
	}
	if hasReportEvent(repo.events, "section.skipped") || hasReportEvent(repo.events, "content.succeeded") {
		t.Fatalf("unexpected success/skip event after running marker failure: %+v", repo.events)
	}
	if len(repo.progressUpdates) != 1 || repo.progressUpdates[0]["completed"] != 0 || repo.progressUpdates[0]["total"] != 1 {
		t.Fatalf("progress updates = %+v, want one 0/1 update", repo.progressUpdates)
	}
}

func TestReportGenerationServiceContentGenerationUsesCurrentOutlineSections(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
	repo.outlines["outline-old"] = ReportOutline{
		ID:        "outline-old",
		ReportID:  "report-1",
		Version:   1,
		IsCurrent: false,
	}
	repo.outlines["outline-current"] = ReportOutline{
		ID:        "outline-current",
		ReportID:  "report-1",
		Version:   2,
		IsCurrent: true,
		Sections: []ReportOutlineNode{
			{ID: "node-current-1", Title: "Edited overview", Level: 1, Numbering: "1"},
			{ID: "node-current-2", Title: "Edited risk", Level: 1, Numbering: "2"},
		},
	}
	repo.sections["section-old"] = ReportSection{
		ID:               "section-old",
		ReportID:         "report-1",
		OutlineID:        "outline-old",
		Title:            "Legacy outline section",
		SortOrder:        0,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	repo.sections["section-current-1"] = ReportSection{
		ID:               "section-current-1",
		ReportID:         "report-1",
		OutlineID:        "outline-current",
		OutlineNodeID:    "node-current-1",
		Title:            "Original overview",
		SortOrder:        1,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	repo.sections["section-current-2"] = ReportSection{
		ID:               "section-current-2",
		ReportID:         "report-1",
		OutlineID:        "outline-current",
		OutlineNodeID:    "node-current-2",
		Title:            "Original risk",
		SortOrder:        2,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	repo.sections["section-removed-current-outline"] = ReportSection{
		ID:               "section-removed-current-outline",
		ReportID:         "report-1",
		OutlineID:        "outline-current",
		OutlineNodeID:    "node-removed",
		Title:            "Removed stale node",
		SortOrder:        3,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{
			{Content: `{"content":"current overview body","tables":[]}`},
			{Content: `{"content":"current risk body","tables":[]}`},
			{Content: `{"content":"stale body","tables":[]}`},
		},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	if len(chat.requests) != 2 {
		t.Fatalf("chat request count = %d, want 2 current outline sections", len(chat.requests))
	}
	firstPrompt := chat.requests[0].Messages[1].Content
	if !strings.Contains(firstPrompt, "Edited overview") || strings.Contains(firstPrompt, "Original overview") {
		t.Fatalf("first section prompt did not use current outline title: %s", firstPrompt)
	}
	secondPrompt := chat.requests[1].Messages[1].Content
	if !strings.Contains(secondPrompt, "Edited risk") || strings.Contains(secondPrompt, "Original risk") {
		t.Fatalf("second section prompt did not use current outline title: %s", secondPrompt)
	}
	if got := repo.sections["section-old"]; got.Content != "" || got.GenerationStatus != JobStatusPending {
		t.Fatalf("old outline section was generated: %+v", got)
	}
	if got := repo.sections["section-removed-current-outline"]; got.Content != "" || got.GenerationStatus != JobStatusPending {
		t.Fatalf("removed current-outline node was generated: %+v", got)
	}
	if got := repo.sections["section-current-1"]; got.Content != "current overview body" || got.Version != 2 {
		t.Fatalf("current section 1 = %+v", got)
	}
	if got := repo.sections["section-current-2"]; got.Content != "current risk body" || got.Version != 2 {
		t.Fatalf("current section 2 = %+v", got)
	}
	lastProgress := repo.progressUpdates[len(repo.progressUpdates)-1]
	if lastProgress["completed"] != 2 || lastProgress["total"] != 2 {
		t.Fatalf("last progress = %+v, want 2/2 current outline sections", lastProgress)
	}
}

func TestReportGenerationServiceSectionRegenerationRejectsStaleOutlineSection(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-old",
		TargetType: "section",
	}
	repo.outlines["outline-old"] = ReportOutline{
		ID:        "outline-old",
		ReportID:  "report-1",
		Version:   1,
		IsCurrent: false,
		Sections:  []ReportOutlineNode{{ID: "node-old", Title: "Old"}},
	}
	repo.outlines["outline-current"] = ReportOutline{
		ID:        "outline-current",
		ReportID:  "report-1",
		Version:   2,
		IsCurrent: true,
		Sections:  []ReportOutlineNode{{ID: "node-current", Title: "Current"}},
	}
	repo.sections["section-old"] = ReportSection{
		ID:               "section-old",
		ReportID:         "report-1",
		OutlineID:        "outline-old",
		OutlineNodeID:    "node-old",
		Title:            "Old section",
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	repo.sections["section-current"] = ReportSection{
		ID:               "section-current",
		ReportID:         "report-1",
		OutlineID:        "outline-current",
		OutlineNodeID:    "node-current",
		Title:            "Current section",
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"stale body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)

	_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-section",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if code := errorCode(t, err); code != CodeValidation {
		t.Fatalf("error code = %q, want %q", code, CodeValidation)
	}
	if len(chat.requests) != 0 {
		t.Fatalf("chat request count = %d, want 0 for stale section regeneration", len(chat.requests))
	}
	if got := repo.sections["section-old"]; got.Content != "" || got.GenerationStatus != JobStatusPending || got.LastJobID != "" {
		t.Fatalf("stale section changed: %+v", got)
	}
}

func TestReportGenerationServiceRejectsUnknownReportTypeForContentJobs(t *testing.T) {
	tests := []struct {
		name       string
		jobType    JobType
		targetType string
		targetID   string
	}{
		{name: "content generation", jobType: JobTypeContentGeneration, targetType: "report", targetID: "report-1"},
		{name: "content regeneration", jobType: JobTypeContentRegeneration, targetType: "report", targetID: "report-1"},
		{name: "section regeneration", jobType: JobTypeSectionRegeneration, targetType: "section", targetID: "section-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeReportGenerationRepository()
			repo.reports["report-1"] = Report{
				ID:         "report-1",
				Name:       "Custom unsupported report",
				ReportType: "custom_report",
				Topic:      "custom topic",
				CreatorID:  "user-1",
				Status:     ReportStatusOutlineGenerated,
			}
			repo.jobs["job-1"] = ReportJob{
				ID:         "job-1",
				JobType:    tt.jobType,
				ReportID:   "report-1",
				TargetType: tt.targetType,
				TargetID:   tt.targetID,
			}
			repo.sections["section-1"] = ReportSection{
				ID:               "section-1",
				ReportID:         "report-1",
				Title:            "Unsupported content",
				SortOrder:        0,
				Version:          1,
				GenerationStatus: JobStatusPending,
			}
			chat := &fakeGenerationChatClient{}
			svc := NewReportGenerationService(repo, chat)

			_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
				RequestID: "req-content",
				JobType:   tt.jobType,
				JobID:     "job-1",
				UserID:    "user-1",
			})
			if code := errorCode(t, err); code != CodeValidation {
				t.Fatalf("error code = %q, want %q", code, CodeValidation)
			}
			if len(chat.requests) != 0 {
				t.Fatalf("chat request count = %d, want 0", len(chat.requests))
			}
		})
	}
}

func TestReportGenerationServicePreservesManualEditedSectionsByDefault(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{ID: "job-1", JobType: JobTypeContentGeneration, ReportID: "report-1"}
	manualSection := ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Manual section",
		SortOrder:        0,
		Content:          "manual body",
		GenerationStatus: JobStatusSucceeded,
		ContentSource:    ContentSourceManual,
		ManualEdited:     true,
		Version:          2,
		LastJobID:        "manual-job",
	}
	repo.sections["section-1"] = manualSection
	repo.sections["section-2"] = ReportSection{
		ID:               "section-2",
		ReportID:         "report-1",
		Title:            "AI section",
		SortOrder:        1,
		Version:          1,
		GenerationStatus: JobStatusPending,
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeContentGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat request count = %d, want 1", len(chat.requests))
	}
	if got := repo.sections["section-1"]; got.Content != manualSection.Content || got.Version != manualSection.Version || !got.ManualEdited || got.LastJobID != manualSection.LastJobID {
		t.Fatalf("manual section was modified: %+v", got)
	}
	if len(repo.sectionVersions["section-1"]) != 0 {
		t.Fatalf("manual section versions = %+v, want none", repo.sectionVersions["section-1"])
	}
	generated := repo.sections["section-2"]
	if generated.Content != "generated body" || generated.Version != 2 || generated.GenerationStatus != JobStatusSucceeded || generated.ContentSource != ContentSourceAI {
		t.Fatalf("generated section = %+v", generated)
	}
	if len(repo.progressUpdates) != 2 {
		t.Fatalf("progress update count = %d, want 2", len(repo.progressUpdates))
	}
	lastProgress := repo.progressUpdates[len(repo.progressUpdates)-1]
	if lastProgress["completed"] != 2 || lastProgress["total"] != 2 {
		t.Fatalf("last progress = %+v, want 2/2", lastProgress)
	}
	foundSkippedEvent := false
	for _, event := range repo.events {
		if event.EventType == "section.skipped" {
			foundSkippedEvent = true
			break
		}
	}
	if !foundSkippedEvent {
		t.Fatalf("expected section.skipped event, got %+v", repo.events)
	}
}

func TestReportGenerationServiceCanOverwriteManualEditedSectionWhenExplicitlyAllowed(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
		RequestPayload: map[string]any{
			"options": map[string]any{"preserveManualEdits": false},
		},
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Manual section",
		SortOrder:        0,
		Content:          "manual body",
		GenerationStatus: JobStatusSucceeded,
		ContentSource:    ContentSourceManual,
		ManualEdited:     true,
		Version:          2,
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"AI replacement","tables":[{"name":"replacement"}]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat request count = %d, want 1", len(chat.requests))
	}
	updated := repo.sections["section-1"]
	if updated.Content != "AI replacement" || updated.Version != 3 || updated.ManualEdited || updated.ContentSource != ContentSourceAI {
		t.Fatalf("manual section was not overwritten by explicit opt-out: %+v", updated)
	}
	if len(repo.sectionVersions["section-1"]) != 1 || repo.sectionVersions["section-1"][0].Content != "AI replacement" {
		t.Fatalf("section versions = %+v", repo.sectionVersions["section-1"])
	}
}

func TestReportGenerationServicePreserveUserEditsFalseOverwritesOnlyTargetSection(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	originalReport := repo.reports["report-1"]
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
		RequestPayload: map[string]any{
			"options": map[string]any{"preserveUserEdits": false},
		},
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Manual section",
		SortOrder:        0,
		Content:          "manual body",
		GenerationStatus: JobStatusSucceeded,
		ContentSource:    ContentSourceManual,
		ManualEdited:     true,
		Version:          2,
	}
	untouched := ReportSection{
		ID:               "section-2",
		ReportID:         "report-1",
		Title:            "Untouched section",
		SortOrder:        1,
		Content:          "untouched body",
		GenerationStatus: JobStatusSucceeded,
		ContentSource:    ContentSourceManual,
		ManualEdited:     true,
		Version:          5,
	}
	repo.sections["section-2"] = untouched
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"AI replacement","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat request count = %d, want 1", len(chat.requests))
	}
	updated := repo.sections["section-1"]
	if updated.Content != "AI replacement" || updated.Version != 3 || updated.ManualEdited || updated.ContentSource != ContentSourceAI {
		t.Fatalf("target section was not overwritten by preserveUserEdits:false: %+v", updated)
	}
	if got := repo.sections["section-2"]; got.Content != untouched.Content || got.Version != untouched.Version || got.ManualEdited != untouched.ManualEdited || got.ContentSource != untouched.ContentSource {
		t.Fatalf("unrelated section was modified: %+v", got)
	}
	if got := repo.reports["report-1"]; got != originalReport {
		t.Fatalf("report base data was modified: %+v", got)
	}
}

func TestReportGenerationServiceRollsBackGeneratedSectionWhenVersionCreationFails(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
	}
	original := ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Pending section",
		SortOrder:        0,
		Content:          "previous body",
		GenerationStatus: JobStatusPending,
		ContentSource:    ContentSourceManual,
		ManualEdited:     false,
		Version:          1,
	}
	repo.sections["section-1"] = original
	repo.createSectionVersionErr = errors.New("insert section version failed")
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if code := errorCode(t, err); code != CodeDependency {
		t.Fatalf("error code = %q, want %q", code, CodeDependency)
	}
	if got := repo.sections["section-1"]; got.Content != original.Content || got.Version != original.Version || got.GenerationStatus != JobStatusFailed || got.ContentSource != original.ContentSource || got.ManualEdited != original.ManualEdited {
		t.Fatalf("generated section was not rolled back: %+v", got)
	}
	if len(repo.sectionVersions["section-1"]) != 0 {
		t.Fatalf("section versions were created despite rollback: %+v", repo.sectionVersions["section-1"])
	}
}

func TestReportGenerationServiceFailureCompensationPreservesConcurrentSectionEdit(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Pending section",
		SortOrder:        0,
		Content:          "previous body",
		GenerationStatus: JobStatusPending,
		ContentSource:    ContentSourceManual,
		Version:          1,
	}
	repo.createSectionVersionErr = errors.New("insert section version failed")
	repo.afterGenerationRollback = func(f *fakeReportGenerationRepository) {
		section := f.sections["section-1"]
		section.Content = "manual edit during generation"
		section.Tables = []map[string]any{{"name": "manual table"}}
		section.GenerationStatus = JobStatusRunning
		section.ContentSource = ContentSourceManual
		section.ManualEdited = true
		section.Version = 2
		section.LastJobID = "job-1"
		f.sections["section-1"] = section
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if code := errorCode(t, err); code != CodeDependency {
		t.Fatalf("error code = %q, want %q", code, CodeDependency)
	}
	got := repo.sections["section-1"]
	if got.Content != "manual edit during generation" || got.Version != 2 || got.GenerationStatus != JobStatusFailed || got.ContentSource != ContentSourceManual || !got.ManualEdited {
		t.Fatalf("concurrent section edit was not preserved after failure compensation: %+v", got)
	}
	if len(got.Tables) != 1 || got.Tables[0]["name"] != "manual table" {
		t.Fatalf("concurrent section tables were not preserved: %+v", got.Tables)
	}
	if len(repo.sectionVersions["section-1"]) != 0 {
		t.Fatalf("section versions were created despite rollback: %+v", repo.sectionVersions["section-1"])
	}
}

func TestReportGenerationServicePreservesConcurrentSectionEditBeforeSuccessfulWrite(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Pending section",
		SortOrder:        0,
		Content:          "previous body",
		GenerationStatus: JobStatusPending,
		ContentSource:    ContentSourceManual,
		Version:          1,
	}
	repo.beforeGenerationTx = func(f *fakeReportGenerationRepository) {
		section := f.sections["section-1"]
		section.Content = "manual edit during generation"
		section.Tables = []map[string]any{{"name": "manual table"}}
		section.GenerationStatus = JobStatusRunning
		section.ContentSource = ContentSourceManual
		section.ManualEdited = true
		section.Version = 2
		section.LastJobID = "job-1"
		f.sections["section-1"] = section
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	got := repo.sections["section-1"]
	if got.Content != "manual edit during generation" || got.Version != 2 || got.ContentSource != ContentSourceManual || !got.ManualEdited {
		t.Fatalf("concurrent section edit was overwritten: %+v", got)
	}
	if got.GenerationStatus != JobStatusRunning || got.LastJobID != "job-1" {
		t.Fatalf("stale generated response changed current generation status: %+v", got)
	}
	if len(got.Tables) != 1 || got.Tables[0]["name"] != "manual table" {
		t.Fatalf("concurrent section tables were overwritten: %+v", got.Tables)
	}
	if len(repo.sectionVersions["section-1"]) != 0 {
		t.Fatalf("section versions were created from stale generated content: %+v", repo.sectionVersions["section-1"])
	}
	if len(repo.progressUpdates) != 1 || repo.progressUpdates[0]["completed"] != 1 || repo.progressUpdates[0]["total"] != 1 {
		t.Fatalf("progress updates = %+v, want one 1/1 update", repo.progressUpdates)
	}
	if !hasReportEvent(repo.events, "section.skipped") {
		t.Fatalf("expected section.skipped event, got %+v", repo.events)
	}
}

func TestReportGenerationServiceDoesNotOverwriteSupersededGenerationJob(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer peak inspection",
		ReportType: "summer_peak_inspection",
		Topic:      "summer power supply",
		CreatorID:  "user-1",
		Status:     ReportStatusOutlineGenerated,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:         "job-1",
		JobType:    JobTypeSectionRegeneration,
		ReportID:   "report-1",
		TargetID:   "section-1",
		TargetType: "section",
	}
	repo.sections["section-1"] = ReportSection{
		ID:               "section-1",
		ReportID:         "report-1",
		Title:            "Pending section",
		SortOrder:        0,
		Content:          "previous body",
		GenerationStatus: JobStatusPending,
		ContentSource:    ContentSourceManual,
		Version:          1,
	}
	repo.beforeGenerationTx = func(f *fakeReportGenerationRepository) {
		section := f.sections["section-1"]
		section.GenerationStatus = JobStatusRunning
		section.LastJobID = "job-2"
		section.Content = "newer job owns this section"
		f.sections["section-1"] = section
	}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"content":"generated body","tables":[]}`}},
	}
	svc := NewReportGenerationService(repo, chat)
	svc.clock = func() time.Time { return time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) }

	result, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-content",
		JobType:   JobTypeSectionRegeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	})
	if err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if result.Status != JobStatusSucceeded {
		t.Fatalf("result status = %q, want succeeded", result.Status)
	}
	got := repo.sections["section-1"]
	if got.LastJobID != "job-2" || got.GenerationStatus != JobStatusRunning || got.Content != "newer job owns this section" {
		t.Fatalf("superseded generation job was overwritten: %+v", got)
	}
	if len(repo.sectionVersions["section-1"]) != 0 {
		t.Fatalf("section versions were created from superseded job: %+v", repo.sectionVersions["section-1"])
	}
	if len(repo.progressUpdates) != 1 || repo.progressUpdates[0]["completed"] != 1 || repo.progressUpdates[0]["total"] != 1 {
		t.Fatalf("progress updates = %+v, want one 1/1 update", repo.progressUpdates)
	}
	if !hasReportEvent(repo.events, "section.skipped") {
		t.Fatalf("expected section.skipped event, got %+v", repo.events)
	}
}

func hasReportEvent(events []ReportEvent, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func TestReportGenerationServiceRetrievesKnowledgeContextForOutline(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Summer report",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "summer peak inspection",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:       "job-1",
		JobType:  JobTypeOutlineGeneration,
		ReportID: "report-1",
		RequestPayload: map[string]any{
			"requirements": "focus on overload risk",
			"materialIds":  []any{"material-1"},
			"options": map[string]any{
				"knowledgeBaseIds": []any{"kb-1"},
				"topK":             float64(2),
				"rerank":           true,
			},
		},
	}
	repo.templateStructures["template-1"] = ReportTemplateStructure{OutlineSchema: []byte(`{"sections":["overview"]}`)}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"sections":[{"title":"Overview"}]}`}},
	}
	retriever := &fakeReportKnowledgeRetriever{
		snippets: []ReportKnowledgeSnippet{{
			KnowledgeBaseID: "kb-1",
			DocumentID:      "doc-1",
			ChunkID:         "chunk-1",
			DocumentName:    "guide",
			ContentPreview:  "safe breaker context",
		}},
	}
	svc := NewReportGenerationService(repo, chat, retriever)

	if _, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-outline",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	}); err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	if retriever.input.Query != "summer peak inspection" || len(retriever.input.KnowledgeBaseIDs) != 1 || retriever.input.KnowledgeBaseIDs[0] != "kb-1" || retriever.input.TopK != 2 || !retriever.input.Rerank {
		t.Fatalf("retrieval input = %+v", retriever.input)
	}
	prompt := chat.requests[0].Messages[1].Content
	if !strings.Contains(prompt, "safe breaker context") || !strings.Contains(prompt, "focus on overload risk") || !strings.Contains(prompt, "material-1") {
		t.Fatalf("prompt did not include generation context: %s", prompt)
	}
	if strings.Contains(prompt, "chunk-1") || strings.Contains(prompt, "doc-1") {
		t.Fatalf("prompt leaked internal knowledge IDs: %s", prompt)
	}
}

func TestReportGenerationServiceUsesSourceContentExcerptForOutline(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Attachment report",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "attachment summary",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:       "job-1",
		JobType:  JobTypeOutlineGeneration,
		ReportID: "report-1",
		RequestPayload: map[string]any{
			"options": map[string]any{
				"sourceContent": map[string]any{
					"excerpt":        "附件显示主变压器在夏峰负荷下出现温升异常。",
					"originalLength": float64(50000),
					"excerptLength":  float64(72),
					"truncated":      true,
				},
			},
		},
	}
	repo.templateStructures["template-1"] = ReportTemplateStructure{OutlineSchema: []byte(`{"sections":["overview"]}`)}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"sections":[{"title":"Overview"}]}`}},
	}
	svc := NewReportGenerationService(repo, chat)

	if _, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-outline",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	}); err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	prompt := chat.requests[0].Messages[1].Content
	if !strings.Contains(prompt, "附件显示主变压器在夏峰负荷下出现温升异常") {
		t.Fatalf("prompt did not include source content excerpt: %s", prompt)
	}
	if strings.Contains(prompt, "50000") || strings.Contains(prompt, "originalLength") {
		t.Fatalf("prompt leaked source content metadata: %s", prompt)
	}
}

func TestReportGenerationServiceKeepsSourceContentPromptUTF8Valid(t *testing.T) {
	repo := newFakeReportGenerationRepository()
	repo.reports["report-1"] = Report{
		ID:         "report-1",
		Name:       "Attachment report",
		ReportType: "summer_peak_inspection",
		TemplateID: "template-1",
		Topic:      "attachment summary",
		CreatorID:  "user-1",
		Status:     ReportStatusDraft,
	}
	repo.jobs["job-1"] = ReportJob{
		ID:       "job-1",
		JobType:  JobTypeOutlineGeneration,
		ReportID: "report-1",
		RequestPayload: map[string]any{
			"options": map[string]any{
				"sourceContent": map[string]any{
					"excerpt": strings.Repeat("负A荷", 2000),
				},
			},
		},
	}
	repo.templateStructures["template-1"] = ReportTemplateStructure{OutlineSchema: []byte(`{"sections":["overview"]}`)}
	chat := &fakeGenerationChatClient{
		responses: []ChatCompletionResponse{{Content: `{"sections":[{"title":"Overview"}]}`}},
	}
	svc := NewReportGenerationService(repo, chat)

	if _, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
		RequestID: "req-outline",
		JobType:   JobTypeOutlineGeneration,
		JobID:     "job-1",
		UserID:    "user-1",
	}); err != nil {
		t.Fatalf("ExecuteReportGeneration() error = %v", err)
	}
	prompt := chat.requests[0].Messages[1].Content
	if !utf8.ValidString(prompt) {
		t.Fatalf("prompt contains invalid UTF-8 after source excerpt truncation")
	}
}

func TestReportGenerationServiceHandlesAIMalformedResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "empty content string", content: ""},
		{name: "plain text non-JSON", content: "抱歉，我无法生成大纲"},
		{name: "truncated JSON", content: `{"sections":[{`},
		{name: "empty sections array", content: `{"sections":[]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeReportGenerationRepository()
			repo.reports["report-1"] = Report{
				ID:         "report-1",
				Name:       "Summer peak inspection",
				ReportType: "summer_peak_inspection",
				TemplateID: "template-1",
				Topic:      "summer power supply",
				CreatorID:  "user-1",
				Status:     ReportStatusDraft,
			}
			repo.jobs["job-1"] = ReportJob{
				ID:       "job-1",
				JobType:  JobTypeOutlineGeneration,
				ReportID: "report-1",
			}
			repo.templateStructures["template-1"] = ReportTemplateStructure{
				OutlineSchema: []byte(`{"sections":["overview"]}`),
			}
			chat := &fakeGenerationChatClient{
				responses: []ChatCompletionResponse{{Content: tc.content}},
			}
			svc := NewReportGenerationService(repo, chat)

			_, err := svc.ExecuteReportGeneration(context.Background(), ReportGenerationExecutionPayload{
				RequestID: "req-malformed",
				JobType:   JobTypeOutlineGeneration,
				JobID:     "job-1",
				UserID:    "user-1",
			})

			if err == nil {
				t.Fatalf("expected error for malformed AI content %q, got nil", tc.content)
			}
			if len(repo.outlines) != 0 {
				t.Fatalf("outlines written despite malformed content: %v", repo.outlines)
			}
			if len(repo.sections) != 0 {
				t.Fatalf("sections written despite malformed content: %v", repo.sections)
			}
			if !hasReportEvent(repo.events, "outline.failed") {
				t.Fatalf("no outline.failed event recorded; events = %v", repo.events)
			}
		})
	}
}

type fakeGenerationChatClient struct {
	requests  []ChatCompletionRequest
	responses []ChatCompletionResponse
	errs      []error
}

type fakeReportKnowledgeRetriever struct {
	input    ReportKnowledgeRetrievalInput
	snippets []ReportKnowledgeSnippet
	err      error
}

func (f *fakeReportKnowledgeRetriever) RetrieveReportContext(_ context.Context, _ RequestContext, input ReportKnowledgeRetrievalInput) ([]ReportKnowledgeSnippet, error) {
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	return f.snippets, nil
}

func (f *fakeGenerationChatClient) CreateChatCompletion(_ context.Context, _ RequestContext, input ChatCompletionRequest) (ChatCompletionResponse, error) {
	f.requests = append(f.requests, input)
	index := len(f.requests) - 1
	if index < len(f.errs) && f.errs[index] != nil {
		return ChatCompletionResponse{}, f.errs[index]
	}
	if index < len(f.responses) {
		return f.responses[index], nil
	}
	return ChatCompletionResponse{}, errors.New("missing fake chat response")
}

type fakeReportGenerationRepository struct {
	reports                          map[string]Report
	jobs                             map[string]ReportJob
	templateStructures               map[string]ReportTemplateStructure
	settings                         ReportSettings
	outlines                         map[string]ReportOutline
	sections                         map[string]ReportSection
	sectionVersions                  map[string][]ReportSectionVersion
	events                           []ReportEvent
	progressUpdates                  []map[string]any
	createSectionErr                 error
	createSectionErrAfter            int
	createdSectionCount              int
	createSectionVersionErr          error
	createSectionVersionErrBySection map[string]error
	markSectionRunningErr            error
	cancelJobAfterProgressUpdates    int
	beforeGenerationTx               func(*fakeReportGenerationRepository)
	afterGenerationRollback          func(*fakeReportGenerationRepository)
}

func newFakeReportGenerationRepository() *fakeReportGenerationRepository {
	return &fakeReportGenerationRepository{
		reports:            map[string]Report{},
		jobs:               map[string]ReportJob{},
		templateStructures: map[string]ReportTemplateStructure{},
		settings: ReportSettings{
			LLM: ReportSettingsModelConfig{Provider: DefaultReportSettingsProvider, ProfileID: "profile-default", Model: "model-default"},
		},
		outlines:        map[string]ReportOutline{},
		sections:        map[string]ReportSection{},
		sectionVersions: map[string][]ReportSectionVersion{},
	}
}

func (f *fakeReportGenerationRepository) WithinGenerationTx(ctx context.Context, fn func(ReportGenerationRepository) error) error {
	if f.beforeGenerationTx != nil {
		beforeTx := f.beforeGenerationTx
		f.beforeGenerationTx = nil
		beforeTx(f)
	}
	snapshot := *f
	snapshot.reports = make(map[string]Report, len(f.reports))
	for id, report := range f.reports {
		snapshot.reports[id] = report
	}
	snapshot.jobs = make(map[string]ReportJob, len(f.jobs))
	for id, job := range f.jobs {
		snapshot.jobs[id] = job
	}
	snapshot.templateStructures = make(map[string]ReportTemplateStructure, len(f.templateStructures))
	for id, structure := range f.templateStructures {
		structure.OutlineSchema = append([]byte(nil), structure.OutlineSchema...)
		snapshot.templateStructures[id] = structure
	}
	snapshot.outlines = make(map[string]ReportOutline, len(f.outlines))
	for id, outline := range f.outlines {
		snapshot.outlines[id] = outline
	}
	snapshot.sections = make(map[string]ReportSection, len(f.sections))
	for id, section := range f.sections {
		snapshot.sections[id] = section
	}
	snapshot.sectionVersions = make(map[string][]ReportSectionVersion, len(f.sectionVersions))
	for id, versions := range f.sectionVersions {
		snapshot.sectionVersions[id] = append([]ReportSectionVersion(nil), versions...)
	}
	snapshot.events = append([]ReportEvent(nil), f.events...)
	snapshot.progressUpdates = make([]map[string]any, len(f.progressUpdates))
	for i, update := range f.progressUpdates {
		snapshot.progressUpdates[i] = cloneJSONLikeMap(update)
	}

	if err := fn(f); err != nil {
		*f = snapshot
		if f.afterGenerationRollback != nil {
			f.afterGenerationRollback(f)
		}
		return err
	}
	return nil
}

func (f *fakeReportGenerationRepository) GetReportByID(_ context.Context, id string) (Report, error) {
	report, ok := f.reports[id]
	if !ok {
		return Report{}, NewError(CodeNotFound, "report not found", nil)
	}
	return report, nil
}

func (f *fakeReportGenerationRepository) FindReportJobByID(_ context.Context, id string) (ReportJob, error) {
	job, ok := f.jobs[id]
	if !ok {
		return ReportJob{}, NewError(CodeNotFound, "report job not found", nil)
	}
	if f.cancelJobAfterProgressUpdates > 0 && len(f.progressUpdates) >= f.cancelJobAfterProgressUpdates {
		job.Status = JobStatusCanceled
		f.jobs[id] = job
	}
	return job, nil
}

func (f *fakeReportGenerationRepository) GetReportTemplateStructure(_ context.Context, id string) (ReportTemplateStructure, error) {
	structure, ok := f.templateStructures[id]
	if !ok {
		return ReportTemplateStructure{}, NewError(CodeNotFound, "report template not found", nil)
	}
	return structure, nil
}

func (f *fakeReportGenerationRepository) GetReportSettings(context.Context) (ReportSettings, error) {
	return f.settings, nil
}

func (f *fakeReportGenerationRepository) CreateReportOutline(_ context.Context, value ReportOutline) (ReportOutline, error) {
	if value.IsCurrent {
		for id, outline := range f.outlines {
			if outline.ReportID == value.ReportID {
				outline.IsCurrent = false
				f.outlines[id] = outline
			}
		}
	}
	f.outlines[value.ID] = value
	return value, nil
}

func (f *fakeReportGenerationRepository) ListReportOutlines(_ context.Context, reportID string) ([]ReportOutline, error) {
	var result []ReportOutline
	for _, outline := range f.outlines {
		if outline.ReportID == reportID {
			result = append(result, outline)
		}
	}
	return result, nil
}

func (f *fakeReportGenerationRepository) CreateReportSection(_ context.Context, value ReportSection) (ReportSection, error) {
	if f.createSectionErr != nil && f.createdSectionCount >= f.createSectionErrAfter {
		return ReportSection{}, f.createSectionErr
	}
	f.createdSectionCount++
	f.sections[value.ID] = value
	return value, nil
}

func (f *fakeReportGenerationRepository) ListReportSections(_ context.Context, reportID string) ([]ReportSection, error) {
	var result []ReportSection
	for _, section := range f.sections {
		if section.ReportID == reportID {
			result = append(result, section)
		}
	}
	return result, nil
}

func (f *fakeReportGenerationRepository) GetReportSectionByIDForUpdate(_ context.Context, id string) (ReportSection, error) {
	section, ok := f.sections[id]
	if !ok {
		return ReportSection{}, NewError(CodeNotFound, "report section not found", nil)
	}
	return section, nil
}

func (f *fakeReportGenerationRepository) UpdateReportSection(_ context.Context, value ReportSection) (ReportSection, error) {
	f.sections[value.ID] = value
	return value, nil
}

func (f *fakeReportGenerationRepository) MarkReportSectionGenerationRunning(_ context.Context, sectionID, jobID string, updatedAt time.Time) (ReportSection, error) {
	if f.markSectionRunningErr != nil {
		return ReportSection{}, f.markSectionRunningErr
	}
	return f.updateReportSectionGenerationState(sectionID, jobID, JobStatusRunning, updatedAt, false)
}

func (f *fakeReportGenerationRepository) MarkReportSectionGenerationFailed(_ context.Context, sectionID, jobID string, updatedAt time.Time) (ReportSection, error) {
	return f.updateReportSectionGenerationState(sectionID, jobID, JobStatusFailed, updatedAt, true)
}

func (f *fakeReportGenerationRepository) updateReportSectionGenerationState(sectionID, jobID string, status JobStatus, updatedAt time.Time, requireLastJobMatch bool) (ReportSection, error) {
	section, ok := f.sections[sectionID]
	if !ok {
		return ReportSection{}, NewError(CodeNotFound, "report section not found", nil)
	}
	if requireLastJobMatch && section.LastJobID != jobID {
		return ReportSection{}, NewError(CodeNotFound, "report section not found", nil)
	}
	section.GenerationStatus = status
	section.LastJobID = jobID
	section.UpdatedAt = updatedAt
	f.sections[sectionID] = section
	return section, nil
}

func (f *fakeReportGenerationRepository) CreateReportSectionVersion(_ context.Context, value ReportSectionVersion) (ReportSectionVersion, error) {
	if err := f.createSectionVersionErrBySection[value.SectionID]; err != nil {
		return ReportSectionVersion{}, err
	}
	if f.createSectionVersionErr != nil {
		return ReportSectionVersion{}, f.createSectionVersionErr
	}
	f.sectionVersions[value.SectionID] = append(f.sectionVersions[value.SectionID], value)
	return value, nil
}

func (f *fakeReportGenerationRepository) ListReportSectionVersions(_ context.Context, sectionID string) ([]ReportSectionVersion, error) {
	return f.sectionVersions[sectionID], nil
}

func (f *fakeReportGenerationRepository) CreateReportEvent(_ context.Context, value ReportEvent) (ReportEvent, error) {
	f.events = append(f.events, value)
	return value, nil
}

func (f *fakeReportGenerationRepository) UpdateReportJobProgress(_ context.Context, jobID string, completed, total int) error {
	f.progressUpdates = append(f.progressUpdates, map[string]any{"jobId": jobID, "completed": completed, "total": total})
	return nil
}
