package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
)

const (
	ToolSearchSessionAttachments     = "search_session_attachments"
	maxAttachmentResultSize          = 8192
	maxAttachmentContentExcerptRunes = 1500
	maxAttachmentReportSourceBytes   = 20_000
	maxAttachmentReportSourceChunks  = 200
)

type SessionAttachmentHit struct {
	AttachmentID   string
	ChunkID        string
	Filename       string
	SectionPath    string
	Content        string
	ContentPreview string
	PageNumber     int
	ChunkIndex     int
}

type SessionAttachmentSearcher interface {
	SearchSessionAttachments(context.Context, string, string, []string, string, int) ([]SessionAttachmentHit, error)
	ListSessionAttachmentReportSource(context.Context, string, string, []string, int) ([]SessionAttachmentHit, error)
}

type AttachmentToolClient struct {
	searcher SessionAttachmentSearcher
	timeout  time.Duration
}

type AttachmentToolConfig struct {
	Searcher SessionAttachmentSearcher
	Timeout  time.Duration
}

func NewAttachmentToolClient(cfg AttachmentToolConfig) (*AttachmentToolClient, error) {
	if cfg.Searcher == nil {
		return nil, errors.New("session attachment searcher is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &AttachmentToolClient{searcher: cfg.Searcher, timeout: cfg.Timeout}, nil
}

func (c *AttachmentToolClient) ListTools(_ context.Context) ([]agent.ToolDefinition, error) {
	return []agent.ToolDefinition{{
		Type: "function",
		Function: agent.FunctionTool{
			Name:        ToolSearchSessionAttachments,
			Description: "Search parsed content from attachments linked to the current QA session message.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Keywords to match within session attachment chunks.",
					},
					"attachment_ids": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Optional attachment IDs to restrict the search scope.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"minimum":     1,
						"maximum":     20,
						"description": "Maximum number of chunks to return.",
					},
					"include_report_source": map[string]any{
						"type":        "boolean",
						"description": "When true, include a bounded report_source_excerpt aggregated from bound ready attachment chunks for document report generation.",
					},
				},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
	}}, nil
}

func (c *AttachmentToolClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (agent.ToolResult, error) {
	if name != ToolSearchSessionAttachments {
		return agent.ToolResult{}, fmt.Errorf("attachment tool %q is not registered", name)
	}
	return c.searchSessionAttachments(ctx, arguments)
}

func (c *AttachmentToolClient) searchSessionAttachments(ctx context.Context, arguments json.RawMessage) (agent.ToolResult, error) {
	var input struct {
		Query               string   `json:"query"`
		AttachmentIDs       []string `json:"attachment_ids"`
		Limit               *int     `json:"limit"`
		IncludeReportSource bool     `json:"include_report_source"`
	}
	if err := decodeToolArguments(arguments, &input); err != nil {
		return toolFailure("invalid_arguments", err.Error()), nil
	}
	if strings.TrimSpace(input.Query) == "" {
		return toolFailure("invalid_arguments", "query must not be empty"), nil
	}
	userID := contextutil.UserIDFromContext(ctx)
	if strings.TrimSpace(userID) == "" {
		return toolFailure("invalid_arguments", "user ID is required"), nil
	}
	sessionID := contextutil.SessionIDFromContext(ctx)
	if strings.TrimSpace(sessionID) == "" {
		return toolFailure("invalid_arguments", "session ID is required"), nil
	}
	limit := 5
	if input.Limit != nil && *input.Limit > 0 {
		limit = *input.Limit
		if limit > 20 {
			limit = 20
		}
	}
	allowed := contextutil.MessageAttachmentIDsFromContext(ctx)
	if len(allowed) == 0 {
		return toolFailure("no_bound_attachments", "no attachments are bound to the current message"), nil
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	targetIDs := input.AttachmentIDs
	if len(targetIDs) == 0 {
		targetIDs = allowed
	} else {
		for _, id := range targetIDs {
			if _, ok := allowedSet[id]; !ok {
				return toolFailure("unauthorized_attachments", "one or more requested attachments are not accessible"), nil
			}
		}
	}
	toolCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	results, err := c.searcher.SearchSessionAttachments(toolCtx, userID, sessionID, targetIDs, input.Query, limit)
	if err != nil {
		return toolFailure("search_failed", "session attachment search failed"), nil
	}
	startCitationNo := contextutil.CitationNoFromContext(ctx)
	if startCitationNo <= 0 {
		startCitationNo = 1
	}
	content := generateAttachmentSearchSummary(results, startCitationNo)
	if input.IncludeReportSource {
		sourceChunks, err := c.searcher.ListSessionAttachmentReportSource(toolCtx, userID, sessionID, targetIDs, maxAttachmentReportSourceChunks)
		if err != nil {
			return toolFailure("search_failed", "session attachment report source failed"), nil
		}
		source := buildAttachmentReportSource(sourceChunks, maxAttachmentReportSourceBytes)
		content = addAttachmentReportSource(content, source)
	}
	return agent.ToolResult{Content: content}, nil
}

func generateAttachmentSearchSummary(results []SessionAttachmentHit, startCitationNo int) string {
	if len(results) == 0 {
		return `{"hit_count":0,"message":"No relevant attachment chunks found"}`
	}
	for _, budget := range attachmentResultBudgets {
		payload := marshalAttachmentSearchSummary(results, len(results), startCitationNo, budget.previewRunes, budget.contextRunes, budget.excerptRunes, false)
		if len(payload) <= maxAttachmentResultSize {
			return string(payload)
		}
	}
	for returned := len(results) - 1; returned >= 1; returned-- {
		budget := attachmentResultBudgets[len(attachmentResultBudgets)-1]
		payload := marshalAttachmentSearchSummary(results[:returned], len(results), startCitationNo, budget.previewRunes, budget.contextRunes, budget.excerptRunes, true)
		if len(payload) <= maxAttachmentResultSize {
			return string(payload)
		}
	}
	budget := attachmentResultBudgets[len(attachmentResultBudgets)-1]
	payload := marshalAttachmentSearchSummary(results[:1], len(results), startCitationNo, 60, 20, budget.excerptRunes, true)
	if len(payload) <= maxAttachmentResultSize {
		return string(payload)
	}
	truncated, _ := json.Marshal(map[string]any{
		"hit_count": len(results),
		"returned":  0,
		"truncated": true,
		"message":   "Results truncated due to size limit",
	})
	return string(truncated)
}

func marshalAttachmentSearchSummary(results []SessionAttachmentHit, totalHits, startCitationNo, previewRunes, contextRunes, excerptRunes int, truncated bool) []byte {
	summary := map[string]any{
		"hit_count": totalHits,
		"returned":  len(results),
		"results":   make([]map[string]any, 0, len(results)),
	}
	if truncated {
		summary["truncated"] = true
	}
	for i, result := range results {
		content := strings.TrimSpace(result.Content)
		if content == "" {
			content = result.ContentPreview
		}
		item := map[string]any{
			"citation_no":     startCitationNo + i,
			"attachment_id":   truncateString(result.AttachmentID, 64),
			"chunk_id":        truncateString(result.ChunkID, 64),
			"filename":        truncateString(result.Filename, 100),
			"section_path":    truncateString(result.SectionPath, 100),
			"preview":         truncateString(result.ContentPreview, previewRunes),
			"context":         truncateString(content, contextRunes),
			"content_excerpt": truncateString(content, excerptRunes),
		}
		if result.PageNumber > 0 {
			item["page_number"] = result.PageNumber
		}
		summary["results"] = append(summary["results"].([]map[string]any), item)
	}
	payload, _ := json.Marshal(summary)
	return payload
}

type attachmentReportSource struct {
	Excerpt            string
	AttachmentCount    int
	ChunkCount         int
	OriginalBytes      int
	ExcerptBytes       int
	ContentBudgetBytes int
	Truncated          bool
}

func buildAttachmentReportSource(chunks []SessionAttachmentHit, maxBytes int) attachmentReportSource {
	source := attachmentReportSource{ContentBudgetBytes: maxBytes}
	if maxBytes <= 0 || len(chunks) == 0 {
		return source
	}
	attachments := map[string]struct{}{}
	var b strings.Builder
	for _, chunk := range chunks {
		content := strings.TrimSpace(chunk.Content)
		if content == "" {
			content = strings.TrimSpace(chunk.ContentPreview)
		}
		if content == "" {
			continue
		}
		attachments[chunk.AttachmentID] = struct{}{}
		source.ChunkCount++
		part := formatAttachmentReportSourceChunk(chunk, content)
		if b.Len() > 0 {
			part = "\n\n" + part
		}
		source.OriginalBytes += len([]byte(part))
		if source.Truncated {
			continue
		}
		remaining := maxBytes - b.Len()
		if remaining <= 0 {
			source.Truncated = true
			continue
		}
		if len([]byte(part)) > remaining {
			truncated, _ := truncateUTF8ByBytes(part, remaining)
			b.WriteString(truncated)
			source.Truncated = true
			continue
		}
		b.WriteString(part)
	}
	source.AttachmentCount = len(attachments)
	source.Excerpt = strings.TrimSpace(b.String())
	source.ExcerptBytes = len([]byte(source.Excerpt))
	return source
}

func formatAttachmentReportSourceChunk(chunk SessionAttachmentHit, content string) string {
	labels := []string{}
	if filename := truncateString(chunk.Filename, 100); filename != "" {
		labels = append(labels, "文件："+filename)
	}
	if chunk.PageNumber > 0 {
		labels = append(labels, fmt.Sprintf("页码：%d", chunk.PageNumber))
	}
	if section := truncateString(chunk.SectionPath, 100); section != "" {
		labels = append(labels, "章节："+section)
	}
	if chunk.ChunkIndex > 0 {
		labels = append(labels, fmt.Sprintf("片段：%d", chunk.ChunkIndex))
	}
	if len(labels) == 0 {
		return content
	}
	return "【附件摘录：" + strings.Join(labels, "，") + "】\n" + content
}

func addAttachmentReportSource(content string, source attachmentReportSource) string {
	var summary map[string]any
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		summary = map[string]any{"hit_count": 0, "message": "Attachment search summary unavailable"}
	}
	summary["report_source"] = map[string]any{
		"attachment_count":     source.AttachmentCount,
		"chunk_count":          source.ChunkCount,
		"original_bytes":       source.OriginalBytes,
		"excerpt_bytes":        source.ExcerptBytes,
		"content_budget_bytes": source.ContentBudgetBytes,
		"truncated":            source.Truncated,
	}
	if source.Excerpt != "" {
		summary["report_source_excerpt"] = source.Excerpt
	}
	payload, _ := json.Marshal(summary)
	return string(payload)
}

func truncateUTF8ByBytes(text string, limit int) (string, bool) {
	if limit <= 0 {
		return "", len(text) > 0
	}
	if len([]byte(text)) <= limit {
		return text, false
	}
	var builder strings.Builder
	builder.Grow(limit)
	used := 0
	for _, r := range text {
		size := len(string(r))
		if used+size > limit {
			break
		}
		builder.WriteRune(r)
		used += size
	}
	return builder.String(), true
}

type attachmentResultBudget struct {
	previewRunes int
	contextRunes int
	excerptRunes int
}

var attachmentResultBudgets = []attachmentResultBudget{
	{previewRunes: 200, contextRunes: 500, excerptRunes: maxAttachmentContentExcerptRunes},
	{previewRunes: 160, contextRunes: 300, excerptRunes: 900},
	{previewRunes: 140, contextRunes: 220, excerptRunes: 600},
	{previewRunes: 120, contextRunes: 160, excerptRunes: 360},
	{previewRunes: 100, contextRunes: 120, excerptRunes: 240},
	{previewRunes: 80, contextRunes: 80, excerptRunes: 160},
}
