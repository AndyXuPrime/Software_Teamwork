package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ReportGenerationRepository interface {
	WithinGenerationTx(ctx context.Context, fn func(ReportGenerationRepository) error) error
	GetReportByID(ctx context.Context, id string) (Report, error)
	FindReportJobByID(ctx context.Context, id string) (ReportJob, error)
	GetReportTemplateStructure(ctx context.Context, id string) (ReportTemplateStructure, error)
	GetReportSettings(ctx context.Context) (ReportSettings, error)
	CreateReportOutline(ctx context.Context, value ReportOutline) (ReportOutline, error)
	ListReportOutlines(ctx context.Context, reportID string) ([]ReportOutline, error)
	CreateReportSection(ctx context.Context, value ReportSection) (ReportSection, error)
	ListReportSections(ctx context.Context, reportID string) ([]ReportSection, error)
	GetReportSectionByIDForUpdate(ctx context.Context, id string) (ReportSection, error)
	UpdateReportSection(ctx context.Context, value ReportSection) (ReportSection, error)
	MarkReportSectionGenerationRunning(ctx context.Context, sectionID, jobID string, updatedAt time.Time) (ReportSection, error)
	MarkReportSectionGenerationFailed(ctx context.Context, sectionID, jobID string, updatedAt time.Time) (ReportSection, error)
	CreateReportSectionVersion(ctx context.Context, value ReportSectionVersion) (ReportSectionVersion, error)
	ListReportSectionVersions(ctx context.Context, sectionID string) ([]ReportSectionVersion, error)
	CreateReportEvent(ctx context.Context, value ReportEvent) (ReportEvent, error)
	UpdateReportJobProgress(ctx context.Context, jobID string, completed, total int) error
}

type ReportGenerationChatClient interface {
	CreateChatCompletion(ctx context.Context, reqCtx RequestContext, input ChatCompletionRequest) (ChatCompletionResponse, error)
}

type ReportGenerationKnowledgeRetriever interface {
	RetrieveReportContext(ctx context.Context, reqCtx RequestContext, input ReportKnowledgeRetrievalInput) ([]ReportKnowledgeSnippet, error)
}

type ReportGenerationService struct {
	repo      ReportGenerationRepository
	chat      ReportGenerationChatClient
	retriever ReportGenerationKnowledgeRetriever
	clock     func() time.Time
}

func NewReportGenerationService(repo ReportGenerationRepository, chat ReportGenerationChatClient, retrievers ...ReportGenerationKnowledgeRetriever) *ReportGenerationService {
	var retriever ReportGenerationKnowledgeRetriever
	if len(retrievers) > 0 {
		retriever = retrievers[0]
	}
	return &ReportGenerationService{
		repo:      repo,
		chat:      chat,
		retriever: retriever,
		clock:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *ReportGenerationService) ExecuteReportGeneration(ctx context.Context, payload ReportGenerationExecutionPayload) (ReportGenerationExecutionResult, error) {
	if s.chat == nil {
		return ReportGenerationExecutionResult{}, NewError(CodeDependency, "ai gateway chat client is not configured", nil)
	}
	job, err := s.repo.FindReportJobByID(ctx, payload.JobID)
	if err != nil {
		return ReportGenerationExecutionResult{}, mapRepositoryReadError(err, "report job not found")
	}
	jobType := payload.JobType
	if jobType == "" {
		jobType = job.JobType
	}
	report, err := s.repo.GetReportByID(ctx, job.ReportID)
	if err != nil {
		return ReportGenerationExecutionResult{}, mapRepositoryReadError(err, "report not found")
	}
	if report.Status == ReportStatusDeleted || report.DeletedAt != nil {
		return ReportGenerationExecutionResult{}, NewError(CodeConflict, "report has been deleted", nil)
	}
	reqCtx := RequestContext{RequestID: payload.RequestID, UserID: payload.UserID, CallerService: "worker"}

	switch jobType {
	case JobTypeOutlineGeneration, JobTypeOutlineRegeneration:
		return s.executeOutlineGeneration(ctx, reqCtx, payload, job, report)
	case JobTypeContentGeneration, JobTypeContentRegeneration, JobTypeSectionRegeneration:
		return s.executeContentGeneration(ctx, reqCtx, payload, job, report)
	default:
		return ReportGenerationExecutionResult{}, ValidationError(map[string]string{"jobType": "unsupported report generation job type"})
	}
}

func (s *ReportGenerationService) executeOutlineGeneration(ctx context.Context, reqCtx RequestContext, payload ReportGenerationExecutionPayload, job ReportJob, report Report) (ReportGenerationExecutionResult, error) {
	reportKind, err := resolveAIReportType(report.ReportType, "outline")
	if err != nil {
		return ReportGenerationExecutionResult{}, err
	}
	settings, err := s.safeSettings(ctx)
	if err != nil {
		return ReportGenerationExecutionResult{}, err
	}
	structure := ReportTemplateStructure{}
	if strings.TrimSpace(report.TemplateID) != "" {
		structure, err = s.repo.GetReportTemplateStructure(ctx, report.TemplateID)
		if err != nil {
			return ReportGenerationExecutionResult{}, mapRepositoryReadError(err, "report template structure not found")
		}
	}
	_ = s.recordEvent(ctx, report.ID, payload.JobID, "outline.started", "outline generation started")
	generationContext, err := s.loadGenerationContext(ctx, reqCtx, report, ReportSection{}, job)
	if err != nil {
		_ = s.recordEvent(ctx, report.ID, payload.JobID, "outline.failed", "outline generation failed")
		return ReportGenerationExecutionResult{}, err
	}
	resp, err := s.chat.CreateChatCompletion(ctx, reqCtx, ChatCompletionRequest{
		Model:     settings.LLM.Model,
		ProfileID: settings.LLM.ProfileID,
		Messages: []ChatMessage{
			{Role: "system", Content: buildOutlineSystemPrompt(reportKind.DisplayName)},
			{Role: "user", Content: buildOutlinePrompt(report, structure, generationContext, reportKind)},
		},
	})
	if err != nil {
		_ = s.recordEvent(ctx, report.ID, payload.JobID, "outline.failed", "outline generation failed")
		return ReportGenerationExecutionResult{}, dependencyError("generate report outline", err)
	}
	nodes, err := parseGeneratedOutline(resp.Content)
	if err != nil {
		_ = s.recordEvent(ctx, report.ID, payload.JobID, "outline.failed", "outline generation failed")
		return ReportGenerationExecutionResult{}, err
	}
	existing, err := s.repo.ListReportOutlines(ctx, report.ID)
	if err != nil {
		return ReportGenerationExecutionResult{}, dependencyError("list report outlines", err)
	}
	nextVersion := nextOutlineVersion(existing)
	now := s.clock()
	outline := ReportOutline{
		ID:           newID(),
		ReportID:     report.ID,
		Sections:     nodes,
		Version:      nextVersion,
		Source:       OutlineSourceAI,
		SourceJobID:  payload.JobID,
		IsCurrent:    true,
		ManualEdited: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	var created ReportOutline
	if err := s.repo.WithinGenerationTx(ctx, func(txRepo ReportGenerationRepository) error {
		var err error
		created, err = txRepo.CreateReportOutline(ctx, outline)
		if err != nil {
			return dependencyError("create report outline", err)
		}
		if err := createSectionSkeletons(ctx, txRepo, report.ID, created, payload.JobID, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return ReportGenerationExecutionResult{}, err
	}
	_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, CountOutlineNodes(created.Sections), CountOutlineNodes(created.Sections))
	_ = s.recordEvent(ctx, report.ID, payload.JobID, "outline.succeeded", "outline generation succeeded")
	return ReportGenerationExecutionResult{Status: JobStatusSucceeded}, nil
}

func (s *ReportGenerationService) executeContentGeneration(ctx context.Context, reqCtx RequestContext, payload ReportGenerationExecutionPayload, job ReportJob, report Report) (ReportGenerationExecutionResult, error) {
	reportKind, err := resolveAIReportType(report.ReportType, "content")
	if err != nil {
		return ReportGenerationExecutionResult{}, err
	}
	settings, err := s.safeSettings(ctx)
	if err != nil {
		return ReportGenerationExecutionResult{}, err
	}
	sections, err := s.repo.ListReportSections(ctx, report.ID)
	if err != nil {
		return ReportGenerationExecutionResult{}, dependencyError("list report sections", err)
	}
	if job.JobType != JobTypeSectionRegeneration {
		sections, err = s.currentOutlineSections(ctx, report.ID, sections)
		if err != nil {
			return ReportGenerationExecutionResult{}, err
		}
	}
	sections = targetGenerationSections(sections, job)
	if len(sections) == 0 {
		return ReportGenerationExecutionResult{}, ValidationError(map[string]string{"sections": "no report sections available for content generation"})
	}
	sortSections(sections)
	_ = s.recordEvent(ctx, report.ID, payload.JobID, "content.started", "content generation started")
	completed := 0
	total := len(sections)
	preserveManual := preserveManualEdits(job)
	for _, section := range sections {
		if preserveManual && section.ManualEdited {
			completed++
			_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
			_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.skipped", "section generation skipped because manual edits are preserved")
			continue
		}
		section, err = s.markSectionGenerationRunning(ctx, section, payload.JobID)
		if err != nil {
			s.markSectionGenerationFailed(ctx, section.ID, payload.JobID)
			_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.failed", "section generation failed")
			_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
			return ReportGenerationExecutionResult{}, err
		}
		generationContext, err := s.loadGenerationContext(ctx, reqCtx, report, section, job)
		if err != nil {
			s.markSectionGenerationFailed(ctx, section.ID, payload.JobID)
			_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.failed", "section generation failed")
			_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
			if completed > 0 {
				_ = s.recordEvent(ctx, report.ID, payload.JobID, "content.partial_succeeded", "content generation partially succeeded")
				return ReportGenerationExecutionResult{Status: JobStatusPartialSucceeded}, nil
			}
			return ReportGenerationExecutionResult{}, err
		}
		resp, err := s.chat.CreateChatCompletion(ctx, reqCtx, ChatCompletionRequest{
			Model:     settings.LLM.Model,
			ProfileID: settings.LLM.ProfileID,
			Messages: []ChatMessage{
				{Role: "system", Content: buildSectionSystemPrompt(reportKind.DisplayName)},
				{Role: "user", Content: buildSectionPrompt(report, section, generationContext, sectionHasChildren(sections, section.ID), reportKind)},
			},
		})
		if err != nil {
			s.markSectionGenerationFailed(ctx, section.ID, payload.JobID)
			_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.failed", "section generation failed")
			_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
			if completed > 0 {
				_ = s.recordEvent(ctx, report.ID, payload.JobID, "content.partial_succeeded", "content generation partially succeeded")
				return ReportGenerationExecutionResult{Status: JobStatusPartialSucceeded}, nil
			}
			return ReportGenerationExecutionResult{}, dependencyError("generate report section", err)
		}
		generated, err := parseGeneratedSection(resp.Content)
		if err != nil {
			s.markSectionGenerationFailed(ctx, section.ID, payload.JobID)
			_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.failed", "section generation failed")
			_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
			if completed > 0 {
				_ = s.recordEvent(ctx, report.ID, payload.JobID, "content.partial_succeeded", "content generation partially succeeded")
				return ReportGenerationExecutionResult{Status: JobStatusPartialSucceeded}, nil
			}
			return ReportGenerationExecutionResult{}, err
		}
		now := s.clock()
		sectionID := section.ID
		var updated ReportSection
		if err := s.repo.WithinGenerationTx(ctx, func(txRepo ReportGenerationRepository) error {
			currentSection, err := txRepo.GetReportSectionByIDForUpdate(ctx, section.ID)
			if err != nil {
				return mapRepositoryReadError(err, "report section not found")
			}
			if currentSection.ReportID != report.ID {
				return NewError(CodeNotFound, "report section not found", nil)
			}
			if currentSection.LastJobID != payload.JobID || currentSection.GenerationStatus != JobStatusRunning {
				return NewError(CodeConflict, "section generation has been superseded", nil)
			}
			if currentSection.Version != section.Version || currentSection.ManualEdited != section.ManualEdited {
				return NewError(CodeConflict, "section changed during generation", nil)
			}
			existingVersions, err := txRepo.ListReportSectionVersions(ctx, section.ID)
			if err != nil {
				return dependencyError("list report section versions", err)
			}
			nextVersion := nextReportSectionVersion(currentSection, existingVersions)
			currentSection.Content = generated.Content
			currentSection.Tables = generated.Tables
			currentSection.GenerationStatus = JobStatusSucceeded
			currentSection.ContentSource = ContentSourceAI
			currentSection.ManualEdited = false
			currentSection.Version = nextVersion
			currentSection.LastJobID = payload.JobID
			currentSection.GeneratedAt = &now
			currentSection.UpdatedAt = now
			updated, err = txRepo.UpdateReportSection(ctx, currentSection)
			if err != nil {
				return dependencyError("update generated report section", err)
			}
			if _, err := txRepo.CreateReportSectionVersion(ctx, ReportSectionVersion{
				ID:        newID(),
				ReportID:  report.ID,
				SectionID: updated.ID,
				Version:   nextVersion,
				Source:    ContentSourceAI,
				Content:   updated.Content,
				Tables:    updated.Tables,
				JobID:     payload.JobID,
				CreatedBy: payload.UserID,
				CreatedAt: now,
			}); err != nil {
				return dependencyError("create report section version", err)
			}
			return nil
		}); err != nil {
			if appErr, ok := Classify(err); ok && appErr.Code == CodeConflict {
				completed++
				_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
				_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.skipped", "section generation skipped because current section changed during generation")
				continue
			}
			s.markSectionGenerationFailed(ctx, sectionID, payload.JobID)
			return ReportGenerationExecutionResult{}, err
		}
		completed++
		_ = s.repo.UpdateReportJobProgress(ctx, payload.JobID, completed, total)
		_ = s.recordEvent(ctx, report.ID, payload.JobID, "section.succeeded", "section generation succeeded")
	}
	_ = s.recordEvent(ctx, report.ID, payload.JobID, "content.succeeded", "content generation succeeded")
	return ReportGenerationExecutionResult{Status: JobStatusSucceeded}, nil
}

func (s *ReportGenerationService) markSectionGenerationRunning(ctx context.Context, section ReportSection, jobID string) (ReportSection, error) {
	updatedAt := s.clock()
	updated, err := s.repo.MarkReportSectionGenerationRunning(ctx, section.ID, jobID, updatedAt)
	if err != nil {
		return section, dependencyError("mark report section generation running", err)
	}
	return updated, nil
}

func (s *ReportGenerationService) markSectionGenerationFailed(ctx context.Context, sectionID, jobID string) {
	_, _ = s.repo.MarkReportSectionGenerationFailed(ctx, sectionID, jobID, s.clock())
}

type aiReportTypeMetadata struct {
	DisplayName string
}

var supportedAIReportTypes = map[string]aiReportTypeMetadata{
	"summer_peak_inspection": {DisplayName: "迎峰度夏检查报告"},
	"coal_inventory_audit":   {DisplayName: "煤库存审计报告"},
}

func resolveAIReportType(reportType, generationKind string) (aiReportTypeMetadata, error) {
	reportType = strings.TrimSpace(reportType)
	metadata, ok := supportedAIReportTypes[reportType]
	if !ok {
		return aiReportTypeMetadata{}, ValidationError(map[string]string{"reportType": fmt.Sprintf("unsupported report type for AI %s generation", generationKind)})
	}
	return metadata, nil
}

func preserveManualEdits(job ReportJob) bool {
	payload := jsonObject(job.RequestPayload)
	options, ok := payload["options"].(map[string]any)
	if !ok {
		return true
	}
	for _, key := range []string{"preserveUserEdits", "preserveManualEdits"} {
		value, ok := options[key].(bool)
		if ok && !value {
			return false
		}
	}
	return true
}

func (s *ReportGenerationService) safeSettings(ctx context.Context) (ReportSettings, error) {
	settings, err := s.repo.GetReportSettings(ctx)
	if err != nil {
		return ReportSettings{}, dependencyError("get report settings", err)
	}
	return normalizeReportSettings(settings), nil
}

type reportGenerationContext struct {
	Requirements         string
	MaterialIDs          []string
	SourceContentExcerpt string
	Snippets             []ReportKnowledgeSnippet
}

func (s *ReportGenerationService) loadGenerationContext(ctx context.Context, reqCtx RequestContext, report Report, section ReportSection, job ReportJob) (reportGenerationContext, error) {
	payload := jsonObject(job.RequestPayload)
	result := reportGenerationContext{
		Requirements:         stringValue(payload["requirements"]),
		MaterialIDs:          stringSliceValue(payload["materialIds"]),
		SourceContentExcerpt: sourceContentExcerptFromPayload(payload),
	}
	retrieval := mergedRetrievalOptions(payload)
	knowledgeBaseIDs := stringSliceValue(retrieval["knowledgeBaseIds"])
	if len(knowledgeBaseIDs) == 0 || s.retriever == nil {
		return result, nil
	}
	query := strings.TrimSpace(report.Topic)
	if strings.TrimSpace(section.Title) != "" {
		query = strings.TrimSpace(query + " " + section.Title)
	}
	if query == "" {
		query = strings.TrimSpace(report.ReportType)
	}
	snippets, err := s.retriever.RetrieveReportContext(ctx, reqCtx, ReportKnowledgeRetrievalInput{
		Query:            query,
		KnowledgeBaseIDs: knowledgeBaseIDs,
		TopK:             intValue(retrieval["topK"]),
		ScoreThreshold:   floatPtrValue(retrieval["scoreThreshold"]),
		Rerank:           boolValue(retrieval["rerank"]),
		RerankTopN:       intPtrValue(retrieval["rerankTopN"]),
	})
	if err != nil {
		return reportGenerationContext{}, dependencyError("retrieve report knowledge context", err)
	}
	result.Snippets = snippets
	return result, nil
}

func createSectionSkeletons(ctx context.Context, repo ReportGenerationRepository, reportID string, outline ReportOutline, jobID string, now time.Time) error {
	var createNodes func(nodes []ReportOutlineNode, parentID string) error
	sortOrder := 0
	createNodes = func(nodes []ReportOutlineNode, parentID string) error {
		for _, node := range nodes {
			id := newID()
			section := ReportSection{
				ID:               id,
				ReportID:         reportID,
				OutlineID:        outline.ID,
				ParentID:         parentID,
				OutlineNodeID:    node.ID,
				SectionPath:      id,
				Title:            node.Title,
				Level:            node.Level,
				SortOrder:        sortOrder,
				Numbering:        node.Numbering,
				SectionType:      SectionTypeText,
				GenerationStatus: JobStatusPending,
				ContentSource:    ContentSourceAI,
				ManualEdited:     false,
				Version:          1,
				LastJobID:        jobID,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			sortOrder++
			created, err := repo.CreateReportSection(ctx, section)
			if err != nil {
				return dependencyError("create report section skeleton", err)
			}
			if len(node.Children) > 0 {
				if err := createNodes(node.Children, created.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return createNodes(outline.Sections, "")
}

func (s *ReportGenerationService) recordEvent(ctx context.Context, reportID, jobID, eventType, message string) error {
	_, err := s.repo.CreateReportEvent(ctx, ReportEvent{
		ID:        newID(),
		ReportID:  reportID,
		JobID:     jobID,
		EventType: eventType,
		Message:   sanitizeStringValue(message),
		CreatedAt: s.clock(),
	})
	return err
}

func buildOutlineSystemPrompt(reportDisplayName string) string {
	return fmt.Sprintf(`你是一名中国电力行业报告撰写专家，正在生成%s的章节大纲。
输出要求：
1. 仅输出合法 JSON，不加任何 Markdown 代码块或额外说明。
2. 格式：{"sections":[{"title":"章节标题","children":[{"title":"子节标题","children":[]}]}]}
3. 标题使用中文，简洁专业，不含编号（编号由系统自动生成）。
4. 根据报告主题和参考资料生成真实适用的章节结构，不使用泛泛的占位标题。`, reportDisplayName)
}

func buildSectionSystemPrompt(reportDisplayName string) string {
	return fmt.Sprintf(`你是一名中国电力行业报告撰写专家，正在生成%s的某一章节内容。
输出要求：
1. 仅输出合法 JSON，不加任何 Markdown 代码块或额外说明。
2. 格式：{"content":"正文段落（段落间用\n分隔）","tables":[{"headers":["列名1","列名2"],"rows":[["值","值"]],"footnote":"注释（可选，无则省略key）"}]}
3. 使用正式中文，专业术语准确。
4. 严禁使用 XX、N/A、待定、（数字）等任何占位符；必须填写具体的、合理的技术数据或描述，若无精确数据则给出合理估算值并注明"估算"。
5. 表格数据必须与正文一致，不得与其他章节的数字相矛盾。
6. 总结/结论章节应综合参考资料中已有的数据得出实质性结论，而非重复列举 XX 项。
7. content 开头不得重复章节标题，直接进入正文内容。
8. 若提示“本节含子章节”，则 content 只需写 1-2 段简短导言，具体数据和分析留给子章节；tables 为空数组。`, reportDisplayName)
}

func buildOutlinePrompt(report Report, structure ReportTemplateStructure, generationContext reportGenerationContext, reportKind aiReportTypeMetadata) string {
	var b strings.Builder
	fmt.Fprintf(&b, "报告类型：%s\n", report.ReportType)
	fmt.Fprintf(&b, "报告名称：%s\n", reportKind.DisplayName)
	fmt.Fprintf(&b, "报告主题：%s\n", report.Topic)
	if req := compactTextForPrompt(generationContext.Requirements, 1024); req != "" {
		fmt.Fprintf(&b, "额外要求：%s\n", req)
	}
	if len(generationContext.MaterialIDs) > 0 {
		fmt.Fprintf(&b, "参考材料ID：%s\n", strings.Join(generationContext.MaterialIDs, ","))
	}
	if source := compactTextForPrompt(generationContext.SourceContentExcerpt, 12000); source != "" {
		fmt.Fprintf(&b, "附件内容摘录：\n%s\n", source)
	}
	if snippets := formatKnowledgeSnippets(generationContext.Snippets); snippets != "" {
		fmt.Fprintf(&b, "参考资料摘录：\n%s\n", snippets)
	}
	if schema := compactJSONForPrompt(structure.OutlineSchema); schema != "{}" {
		fmt.Fprintf(&b, "大纲模板（仅供参考，可据实调整）：%s\n", schema)
	}
	b.WriteString(`请输出JSON大纲，sections数组中每项含title和children（可为空数组）。`)
	return b.String()
}

func buildSectionPrompt(report Report, section ReportSection, generationContext reportGenerationContext, hasChildren bool, reportKind aiReportTypeMetadata) string {
	var b strings.Builder
	fmt.Fprintf(&b, "报告类型：%s\n", report.ReportType)
	fmt.Fprintf(&b, "报告名称：%s\n", reportKind.DisplayName)
	fmt.Fprintf(&b, "报告主题：%s\n", report.Topic)
	sectionLabel := strings.TrimSpace(section.Numbering + " " + section.Title)
	fmt.Fprintf(&b, "当前章节：%s\n", sectionLabel)
	if hasChildren {
		b.WriteString("提示：本节含子章节，content 只需写 1-2 段覆盖范围的导言，tables 为空数组。\n")
	}
	if req := compactTextForPrompt(generationContext.Requirements, 1024); req != "" {
		fmt.Fprintf(&b, "额外要求：%s\n", req)
	}
	if source := compactTextForPrompt(generationContext.SourceContentExcerpt, 12000); source != "" {
		fmt.Fprintf(&b, "附件内容摘录（请结合本节主题取用）：\n%s\n", source)
	}
	if snippets := formatKnowledgeSnippets(generationContext.Snippets); snippets != "" {
		fmt.Fprintf(&b, "参考资料（请基于以下资料生成具体内容）：\n%s\n", snippets)
	}
	b.WriteString("请输出该章节的JSON内容，content为正文，tables为表格数组（无表格则为空数组）。")
	return b.String()
}

func sectionHasChildren(sections []ReportSection, id string) bool {
	for _, s := range sections {
		if s.ParentID == id {
			return true
		}
	}
	return false
}

func formatKnowledgeSnippets(snippets []ReportKnowledgeSnippet) string {
	if len(snippets) == 0 {
		return ""
	}
	parts := make([]string, 0, len(snippets))
	for _, snippet := range snippets {
		preview := compactTextForPrompt(snippet.ContentPreview, 512)
		if preview == "" {
			continue
		}
		parts = append(parts, preview)
	}
	return strings.Join(parts, "\n")
}

func compactTextForPrompt(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]byte(text)) <= limit {
		return text
	}
	truncated, _ := truncateUTF8ByBytes(text, limit)
	return truncated
}

func compactJSONForPrompt(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "{}"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	text := string(data)
	if len(text) > 2048 {
		return text[:2048]
	}
	return text
}

type generatedOutlinePayload struct {
	Sections []generatedOutlineNode `json:"sections"`
}

type generatedOutlineNode struct {
	Title    string                 `json:"title"`
	Children []generatedOutlineNode `json:"children"`
}

func parseGeneratedOutline(content string) ([]ReportOutlineNode, error) {
	var payload generatedOutlinePayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, NewError(CodeDependency, "AI outline response was not valid JSON", nil)
	}
	nodes, err := generatedNodesToOutline(payload.Sections)
	if err != nil {
		return nil, err
	}
	return RenumberOutline(nodes), nil
}

func generatedNodesToOutline(nodes []generatedOutlineNode) ([]ReportOutlineNode, error) {
	if len(nodes) == 0 {
		return nil, NewError(CodeDependency, "AI outline response did not include sections", nil)
	}
	result := make([]ReportOutlineNode, 0, len(nodes))
	for _, node := range nodes {
		title := strings.TrimSpace(node.Title)
		if title == "" {
			return nil, NewError(CodeDependency, "AI outline response included an empty section title", nil)
		}
		children, err := generatedNodesToOutlineOptional(node.Children)
		if err != nil {
			return nil, err
		}
		result = append(result, ReportOutlineNode{
			ID:       newID(),
			Title:    title,
			Children: children,
		})
	}
	return result, nil
}

func generatedNodesToOutlineOptional(nodes []generatedOutlineNode) ([]ReportOutlineNode, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	return generatedNodesToOutline(nodes)
}

type generatedSectionPayload struct {
	Content string           `json:"content"`
	Tables  []map[string]any `json:"tables"`
}

func parseGeneratedSection(content string) (generatedSectionPayload, error) {
	var payload generatedSectionPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return generatedSectionPayload{}, NewError(CodeDependency, "AI section response was not valid JSON", nil)
	}
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Content == "" && len(payload.Tables) == 0 {
		return generatedSectionPayload{}, NewError(CodeDependency, "AI section response was empty", nil)
	}
	if payload.Tables == nil {
		payload.Tables = []map[string]any{}
	}
	return payload, nil
}

func nextOutlineVersion(existing []ReportOutline) int {
	next := 1
	for _, outline := range existing {
		if outline.Version >= next {
			next = outline.Version + 1
		}
	}
	return next
}

func (s *ReportGenerationService) currentOutlineSections(ctx context.Context, reportID string, sections []ReportSection) ([]ReportSection, error) {
	outlines, err := s.repo.ListReportOutlines(ctx, reportID)
	if err != nil {
		return nil, dependencyError("list report outlines", err)
	}
	currentOutlineID := currentReportOutlineID(outlines)
	if currentOutlineID == "" {
		return sections, nil
	}
	return sectionsForOutline(sections, currentOutlineID), nil
}

func currentReportOutlineID(outlines []ReportOutline) string {
	var current ReportOutline
	for _, outline := range outlines {
		if !outline.IsCurrent || strings.TrimSpace(outline.ID) == "" {
			continue
		}
		if current.ID == "" || outline.Version > current.Version {
			current = outline
		}
	}
	return current.ID
}

func sectionsForOutline(sections []ReportSection, outlineID string) []ReportSection {
	outlineID = strings.TrimSpace(outlineID)
	if outlineID == "" {
		return sections
	}
	filtered := make([]ReportSection, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section.OutlineID) == outlineID {
			filtered = append(filtered, section)
		}
	}
	return filtered
}

func targetGenerationSections(sections []ReportSection, job ReportJob) []ReportSection {
	if job.JobType != JobTypeSectionRegeneration || strings.TrimSpace(job.TargetID) == "" || job.TargetType != "section" {
		return sections
	}
	targetID := strings.TrimSpace(job.TargetID)
	for _, section := range sections {
		if section.ID == targetID {
			return []ReportSection{section}
		}
	}
	return nil
}

func sortSections(sections []ReportSection) {
	sort.SliceStable(sections, func(i, j int) bool {
		if sections[i].SortOrder == sections[j].SortOrder {
			return sections[i].ID < sections[j].ID
		}
		return sections[i].SortOrder < sections[j].SortOrder
	})
}

func jsonObject(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mergedRetrievalOptions(payload map[string]any) map[string]any {
	result := map[string]any{}
	for _, key := range []string{"options", "retrieval"} {
		if nested, ok := payload[key].(map[string]any); ok {
			for nestedKey, nestedValue := range nested {
				result[nestedKey] = nestedValue
			}
		}
	}
	return result
}

func sourceContentExcerptFromPayload(payload map[string]any) string {
	for _, value := range []any{payload["sourceContent"], mapValue(payload["options"], "sourceContent")} {
		source, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if excerpt := stringValue(source["excerpt"]); excerpt != "" {
			return excerpt
		}
	}
	return ""
}

func mapValue(value any, key string) any {
	mapped, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return mapped[key]
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return trimStringSlice(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func trimStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			result = append(result, text)
		}
	}
	return result
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int32:
		if typed > 0 {
			return int(typed)
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return 0
}

func intPtrValue(value any) *int {
	if parsed := intValue(value); parsed > 0 {
		return &parsed
	}
	return nil
}

func floatPtrValue(value any) *float64 {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return &typed
		}
	case float32:
		parsed := float64(typed)
		if parsed > 0 {
			return &parsed
		}
	}
	return nil
}

func boolValue(value any) bool {
	parsed, ok := value.(bool)
	return ok && parsed
}
