package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
	"github.com/xeipuuv/gojsonschema"
)

// Policy enforces tool whitelist, permissions, and schema validation.
type Policy struct {
	enabledTools map[string]struct{}
	schemaLoader gojsonschema.JSONLoader
}

type PolicyConfig struct {
	EnabledToolNames []string
}

func NewPolicy(cfg PolicyConfig) (*Policy, error) {
	enabled := make(map[string]struct{}, len(cfg.EnabledToolNames))
	for _, name := range cfg.EnabledToolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := enabled[name]; exists {
			return nil, fmt.Errorf("duplicate enabled tool name %q", name)
		}
		enabled[name] = struct{}{}
	}

	return &Policy{
		enabledTools: enabled,
	}, nil
}

// IsAllowed checks if a tool is in the whitelist.
// Empty whitelist means no tools are allowed (secure default).
func (p *Policy) IsAllowed(toolName string) bool {
	if len(p.enabledTools) == 0 {
		return false
	}
	_, ok := p.enabledTools[toolName]
	return ok
}

// FilterTools removes unauthorized tools from the list exposed to the model.
func (p *Policy) FilterTools(tools []agent.ToolDefinition) []agent.ToolDefinition {
	var filtered []agent.ToolDefinition
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Function.Name)
		if name == "" {
			continue
		}
		if !p.IsAllowed(name) {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}

// ValidateCall checks tool name, arguments schema, and policy before execution.
func (p *Policy) ValidateCall(toolName string, arguments json.RawMessage, toolDef agent.ToolDefinition) error {
	// Check whitelist
	if !p.IsAllowed(toolName) {
		return fmt.Errorf("tool %q is not in the enabled whitelist", toolName)
	}

	// Validate JSON schema if parameters are defined
	if toolDef.Function.Parameters != nil {
		schemaLoader := gojsonschema.NewGoLoader(toolDef.Function.Parameters)
		documentLoader := gojsonschema.NewBytesLoader(arguments)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			return fmt.Errorf("schema validation error: %w", err)
		}

		if !result.Valid() {
			var errors []string
			for _, desc := range result.Errors() {
				errors = append(errors, desc.String())
			}
			return fmt.Errorf("arguments do not match tool schema: %s", strings.Join(errors, "; "))
		}
	}

	return nil
}

// GenerateArgumentsSummary creates a sanitized summary of tool arguments.
// It must not expose complete parameters, internal URLs, object keys, or secrets.
func GenerateArgumentsSummary(toolName string, arguments json.RawMessage) map[string]any {
	var decoded map[string]any
	if err := json.Unmarshal(arguments, &decoded); err != nil {
		return map[string]any{"error": "failed to decode arguments"}
	}

	switch toolName {
	case ToolSearchKnowledge:
		return generateSearchArgumentsSummary(decoded)
	case ToolSearchSessionAttachments:
		return generateAttachmentSearchArgumentsSummary(decoded)
	case ToolGetCitationSource:
		return generateCitationArgumentsSummary(decoded)
	default:
		// Generic summary - only show field names, not values
		fields := make([]string, 0, len(decoded))
		for key := range decoded {
			fields = append(fields, key)
		}
		return map[string]any{
			"tool":      toolName,
			"fields":    fields,
			"sanitized": true,
		}
	}
}

func generateSearchArgumentsSummary(args map[string]any) map[string]any {
	summary := map[string]any{
		"tool": ToolSearchKnowledge,
	}

	if query, ok := args["query"].(string); ok {
		summary["query_length"] = len(query)
		summary["query_empty"] = len(strings.TrimSpace(query)) == 0
	}

	if kbIDs, ok := args["knowledge_base_ids"].([]any); ok {
		summary["knowledge_base_count"] = len(kbIDs)
	}

	if topK, ok := args["top_k"].(float64); ok {
		summary["top_k"] = int(topK)
	}

	if threshold, ok := args["score_threshold"].(float64); ok {
		summary["score_threshold"] = threshold
	}

	return summary
}

func generateAttachmentSearchArgumentsSummary(args map[string]any) map[string]any {
	summary := map[string]any{"tool": ToolSearchSessionAttachments}
	if query, ok := args["query"].(string); ok {
		summary["query_length"] = len(query)
	}
	if ids, ok := args["attachment_ids"].([]any); ok {
		summary["attachment_count"] = len(ids)
	}
	return summary
}

func generateAttachmentSearchResultSummary(content string) map[string]any {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return map[string]any{"error": "failed to decode result"}
	}
	summary := map[string]any{"tool": ToolSearchSessionAttachments}
	if hitCount, ok := decoded["hit_count"].(float64); ok {
		summary["hit_count"] = int(hitCount)
	}
	if results, ok := decoded["results"].([]any); ok {
		summary["citation_count"] = len(results)
	}
	return summary
}

func generateCitationArgumentsSummary(args map[string]any) map[string]any {
	summary := map[string]any{
		"tool": ToolGetCitationSource,
	}

	// Only indicate which ID type was provided, not the actual ID value
	if _, ok := args["citation_id"].(string); ok {
		summary["id_type"] = "citation"
	}
	if _, ok := args["chunk_id"].(string); ok {
		summary["id_type"] = "chunk"
	}

	return summary
}

// GenerateResultSummary creates a sanitized summary of tool results.
// It must not expose full content, internal URLs, object keys, or secrets.
func GenerateResultSummary(toolName string, resultContent string) map[string]any {
	if summary, ok := generateDocumentReportResultSummary(toolName, resultContent); ok {
		return summary
	}
	if summary, ok := toolFailureSummary(toolName, resultContent); ok {
		summary["tool"] = toolName
		return summary
	}
	switch toolName {
	case ToolSearchKnowledge:
		return generateSearchResultSummary(resultContent)
	case ToolSearchSessionAttachments:
		return generateAttachmentSearchResultSummary(resultContent)
	case ToolGetCitationSource:
		return generateCitationResultSummary(resultContent)
	default:
		return map[string]any{
			"tool":      toolName,
			"sanitized": true,
			"size_hint": len(resultContent),
		}
	}
}

func toolFailureSummary(toolName string, content string) (map[string]any, bool) {
	var decoded struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return nil, false
	}
	code := strings.TrimSpace(decoded.Error.Code)
	if code == "" {
		return nil, false
	}
	message := strings.TrimSpace(decoded.Error.Message)
	if message == "" {
		message = "tool execution failed"
	}
	if !isBuiltinTool(toolName) {
		message = "tool execution failed"
	}
	return map[string]any{
		"error":     code,
		"message":   truncateSummary(message, 512),
		"sanitized": true,
	}, true
}

func isBuiltinTool(toolName string) bool {
	return toolName == ToolSearchKnowledge || toolName == ToolGetCitationSource || toolName == ToolSearchSessionAttachments
}

func truncateSummary(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func generateSearchResultSummary(content string) map[string]any {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return map[string]any{"error": "failed to decode result"}
	}

	summary := map[string]any{
		"tool": ToolSearchKnowledge,
	}

	if hitCount, ok := decoded["hit_count"].(float64); ok {
		summary["hit_count"] = int(hitCount)
	}

	if results, ok := decoded["results"].([]any); ok {
		summary["citation_count"] = len(results)
	}

	return summary
}

func generateCitationResultSummary(content string) map[string]any {
	// Citation source lookup is pending, just indicate status
	return map[string]any{
		"tool":   ToolGetCitationSource,
		"status": "pending",
	}
}
