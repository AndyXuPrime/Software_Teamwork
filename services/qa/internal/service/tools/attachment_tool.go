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
	ToolSearchSessionAttachments = "search_session_attachments"
	maxAttachmentResultSize      = 8192
)

type SessionAttachmentHit struct {
	AttachmentID   string
	ChunkID        string
	Filename       string
	SectionPath    string
	ContentPreview string
	PageNumber     int
	ChunkIndex     int
}

type SessionAttachmentSearcher interface {
	SearchSessionAttachments(context.Context, string, string, []string, string, int) ([]SessionAttachmentHit, error)
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
		Query         string   `json:"query"`
		AttachmentIDs []string `json:"attachment_ids"`
		Limit         *int     `json:"limit"`
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
	targetIDs := input.AttachmentIDs
	allowed := contextutil.MessageAttachmentIDsFromContext(ctx)
	if len(allowed) > 0 {
		allowedSet := make(map[string]struct{}, len(allowed))
		for _, id := range allowed {
			allowedSet[id] = struct{}{}
		}
		if len(targetIDs) == 0 {
			targetIDs = allowed
		} else {
			for _, id := range targetIDs {
				if _, ok := allowedSet[id]; !ok {
					return toolFailure("unauthorized_attachments", "one or more requested attachments are not accessible"), nil
				}
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
	return agent.ToolResult{Content: generateAttachmentSearchSummary(results, startCitationNo)}, nil
}

func generateAttachmentSearchSummary(results []SessionAttachmentHit, startCitationNo int) string {
	if len(results) == 0 {
		return `{"hit_count":0,"message":"No relevant attachment chunks found"}`
	}
	summary := map[string]any{
		"hit_count": len(results),
		"returned":  len(results),
		"results":   make([]map[string]any, 0, len(results)),
	}
	for i, result := range results {
		item := map[string]any{
			"citation_no":   startCitationNo + i,
			"attachment_id": truncateString(result.AttachmentID, 64),
			"chunk_id":      truncateString(result.ChunkID, 64),
			"filename":      truncateString(result.Filename, 100),
			"section_path":  truncateString(result.SectionPath, 100),
			"preview":       truncateString(result.ContentPreview, 200),
			"context":       truncateString(result.ContentPreview, 500),
		}
		if result.PageNumber > 0 {
			item["page_number"] = result.PageNumber
		}
		summary["results"] = append(summary["results"].([]map[string]any), item)
	}
	payload, _ := json.Marshal(summary)
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
