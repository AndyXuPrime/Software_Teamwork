package repository

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func (r *Postgres) enrichMessages(ctx context.Context, userID, conversationID string, messages []service.Message, options service.MessageListOptions) error {
	if len(messages) == 0 {
		return nil
	}
	ids := messageIDs(messages)
	if err := r.enrichMessageAttachments(ctx, ids, messages); err != nil {
		return err
	}
	if err := r.enrichMessageRunRecoverables(ctx, userID, conversationID, ids, messages); err != nil {
		return err
	}
	if options.IncludeThinking {
		steps, err := r.listThinkingForMessages(ctx, userID, conversationID, ids)
		if err != nil {
			return err
		}
		for index := range messages {
			messages[index].Thinking = steps[messages[index].ID]
		}
	}
	if options.IncludeCitations {
		citations, err := r.listCitationsForMessages(ctx, userID, conversationID, ids)
		if err != nil {
			return err
		}
		for index := range messages {
			messages[index].Citations = citations[messages[index].ID]
		}
	}
	return nil
}

func (r *Postgres) enrichMessageRunRecoverables(ctx context.Context, userID, conversationID string, ids []string, messages []service.Message) error {
	runIDs, err := r.listResponseRunsForMessages(ctx, userID, conversationID, ids)
	if err != nil {
		return err
	}
	if len(runIDs) == 0 {
		return nil
	}
	reasoning, err := r.listReasoningContentForMessages(ctx, userID, conversationID, ids)
	if err != nil {
		return err
	}
	artifacts, err := r.listReportArtifactsForMessages(ctx, userID, conversationID, ids)
	if err != nil {
		return err
	}
	for index := range messages {
		messageID := messages[index].ID
		messages[index].ResponseRunID = runIDs[messageID]
		messages[index].ReasoningContent = reasoning[messageID]
		messages[index].Artifacts = artifacts[messageID]
	}
	return nil
}

func (r *Postgres) enrichMessageAttachments(ctx context.Context, ids []string, messages []service.Message) error {
	rows, err := r.pool.Query(ctx, `SELECT message_id::text, attachment_id::text FROM message_attachments WHERE message_id::text = ANY($1) ORDER BY message_id, created_at`, ids)
	if err != nil {
		return fmt.Errorf("list message attachment IDs: %w", err)
	}
	defer rows.Close()
	attachments := map[string][]string{}
	for rows.Next() {
		var messageID, attachmentID string
		if err := rows.Scan(&messageID, &attachmentID); err != nil {
			return fmt.Errorf("scan message attachment ID: %w", err)
		}
		attachments[messageID] = append(attachments[messageID], attachmentID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration for message attachment IDs: %w", err)
	}
	for index := range messages {
		messages[index].AttachmentIDs = attachments[messages[index].ID]
	}
	return nil
}

func messageIDs(messages []service.Message) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}
	return ids
}

func (r *Postgres) listResponseRunsForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `
SELECT rr.assistant_message_id::text, rr.id::text
FROM response_runs rr
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id::text = ANY($1::text[])
    AND rr.conversation_id::text = $2
    AND c.external_user_id = $3
    AND c.deleted_at IS NULL
ORDER BY rr.assistant_message_id::text`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message response runs: %w", err)
	}
	defer rows.Close()

	items := map[string]string{}
	for rows.Next() {
		var messageID, runID string
		if err := rows.Scan(&messageID, &runID); err != nil {
			return nil, fmt.Errorf("scan message response run: %w", err)
		}
		items[messageID] = runID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration for message response runs: %w", err)
	}
	return items, nil
}

func (r *Postgres) listReasoningContentForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `
SELECT rr.assistant_message_id::text, COALESCE(e.payload->>'text', '')
FROM response_stream_events e
JOIN response_runs rr ON rr.id = e.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id::text = ANY($1::text[])
    AND rr.conversation_id::text = $2
    AND c.external_user_id = $3
    AND c.deleted_at IS NULL
    AND e.event_type = 'reasoning.delta'
    AND e.payload->>'messageId' = rr.assistant_message_id::text
ORDER BY rr.assistant_message_id::text, e.event_seq`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message reasoning content: %w", err)
	}
	defer rows.Close()

	builders := map[string]*strings.Builder{}
	for rows.Next() {
		var messageID, text string
		if err := rows.Scan(&messageID, &text); err != nil {
			return nil, fmt.Errorf("scan message reasoning content: %w", err)
		}
		if text == "" || service.ContainsUnsafeReasoningContent(text) {
			continue
		}
		builder := builders[messageID]
		if builder == nil {
			builder = &strings.Builder{}
			builders[messageID] = builder
		}
		builder.WriteString(text)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration for message reasoning content: %w", err)
	}

	items := map[string]string{}
	for messageID, builder := range builders {
		content := builder.String()
		if content == "" || service.ContainsUnsafeReasoningContent(content) {
			continue
		}
		items[messageID] = content
	}
	return items, nil
}

func (r *Postgres) listReportArtifactsForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string][]service.ReportArtifact, error) {
	rows, err := r.pool.Query(ctx, `
SELECT rr.assistant_message_id::text, tc.result_summary->'reportArtifact'
FROM agent_tool_calls tc
JOIN response_runs rr ON rr.id = tc.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id::text = ANY($1::text[])
    AND rr.conversation_id::text = $2
    AND c.external_user_id = $3
    AND c.deleted_at IS NULL
    AND tc.result_summary ? 'reportArtifact'
ORDER BY rr.assistant_message_id::text, tc.started_at, tc.id`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message report artifacts: %w", err)
	}
	defer rows.Close()

	items := map[string][]service.ReportArtifact{}
	for rows.Next() {
		var messageID string
		var raw []byte
		if err := rows.Scan(&messageID, &raw); err != nil {
			return nil, fmt.Errorf("scan message report artifact: %w", err)
		}
		artifact, ok := reportArtifactFromJSON(raw)
		if !ok {
			continue
		}
		items[messageID] = mergeReportArtifact(items[messageID], artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration for message report artifacts: %w", err)
	}
	return items, nil
}

func reportArtifactFromJSON(raw []byte) (service.ReportArtifact, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false
	}
	var decoded map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if decoded["artifactType"] != "report_generation" || containsUnsafeReportArtifactValue(decoded) {
		return nil, false
	}
	projected := projectReportArtifact(decoded)
	if len(projected) == 0 {
		return nil, false
	}
	return projected, true
}

func projectReportArtifact(decoded map[string]any) service.ReportArtifact {
	artifact := service.ReportArtifact{"artifactType": "report_generation"}
	for _, key := range []string{
		"reportId", "reportName", "reportType", "jobId", "reportStatus",
		"reportFileId", "filename",
	} {
		if value, ok := decoded[key].(string); ok && strings.TrimSpace(value) != "" {
			artifact[key] = strings.TrimSpace(value)
		}
	}
	if jobType := normalizedReportArtifactJobType(decoded["jobType"]); jobType != "" {
		artifact["jobType"] = jobType
	}
	if value, ok := decoded["jobStatus"].(string); ok {
		if status := normalizedReportArtifactStatus(value); status != "" {
			artifact["jobStatus"] = status
		}
	}
	if value, ok := decoded["fileStatus"].(string); ok {
		if status := normalizedReportArtifactStatus(value); status != "" {
			artifact["fileStatus"] = status
		}
	}
	if value, ok := decoded["format"].(string); ok && strings.EqualFold(strings.TrimSpace(value), "docx") {
		artifact["format"] = "docx"
	}
	if value, ok := decoded["downloadPath"].(string); ok && isPublicReportFileDownloadPath(value) {
		artifact["downloadPath"] = strings.TrimSpace(value)
	}
	if value, ok := decoded["detailPath"].(string); ok && isPublicReportDetailPath(value) {
		artifact["detailPath"] = strings.TrimSpace(value)
	}
	if value, ok := normalizedReportArtifactFileSize(decoded["fileSize"]); ok {
		artifact["fileSize"] = value
	}
	if preview, ok := projectReportArtifactPreview(decoded["preview"]); ok {
		artifact["preview"] = preview
	}
	return artifact
}

func projectReportArtifactPreview(value any) (map[string]any, bool) {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	preview := map[string]any{}
	for _, key := range []string{"title", "summary", "statusText"} {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			preview[key] = strings.TrimSpace(value)
		}
	}
	for _, key := range []string{"outlineTitles", "sectionTitles"} {
		values := stringsFromArtifactValue(raw[key])
		if len(values) > 0 {
			preview[key] = values
		}
	}
	if value, ok := normalizedReportArtifactProgressPercent(raw["progressPercent"]); ok {
		preview["progressPercent"] = value
	}
	return preview, len(preview) > 0
}

func stringsFromArtifactValue(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			values = append(values, text)
		}
	}
	return values
}

func normalizedReportArtifactStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "accepted", "pending", "running", "succeeded", "failed", "canceled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizedReportArtifactJobType(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "outline_generation", "outline_regeneration", "content_generation", "content_regeneration", "section_regeneration", "report_file_creation":
		return strings.ToLower(strings.TrimSpace(text))
	default:
		return ""
	}
}

func normalizedReportArtifactFileSize(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		size, err := typed.Int64()
		if err != nil || size < 0 {
			return 0, false
		}
		return size, true
	case float64:
		const maxExactFloatInteger = 1<<53 - 1
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed < 0 || typed > maxExactFloatInteger || math.Trunc(typed) != typed {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func normalizedReportArtifactProgressPercent(value any) (float64, bool) {
	var percent float64
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		percent = number
	case float64:
		percent = typed
	default:
		return 0, false
	}
	if math.IsNaN(percent) || math.IsInf(percent, 0) || percent < 0 || percent > 100 {
		return 0, false
	}
	return percent, true
}

func isPublicReportFileDownloadPath(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "/api/v1/report-files/") &&
		strings.HasSuffix(value, "/content") &&
		safePublicPathSegment(strings.TrimSuffix(strings.TrimPrefix(value, "/api/v1/report-files/"), "/content"))
}

func isPublicReportDetailPath(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "/api/v1/reports/") &&
		safePublicPathSegment(strings.TrimPrefix(value, "/api/v1/reports/"))
}

func safePublicPathSegment(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "/") || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func containsUnsafeReportArtifactValue(value any) bool {
	switch typed := value.(type) {
	case nil, bool, float64, json.Number:
		return false
	case string:
		return isUnsafeReportArtifactString(typed)
	case map[string]any:
		for key, entry := range typed {
			if isUnsafeReportArtifactKey(key) || containsUnsafeReportArtifactValue(entry) {
				return true
			}
		}
		return false
	case []any:
		for _, entry := range typed {
			if containsUnsafeReportArtifactValue(entry) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func isUnsafeReportArtifactKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(strings.ToLower(key))
	for _, marker := range []string{"objectkey", "bucket", "internalurl", "fileinternalid", "fileref", "prompt", "providerraw", "raw", "secret", "token", "apikey"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isUnsafeReportArtifactString(value string) bool {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "/api/v1/report-files/") || strings.HasPrefix(trimmed, "/api/v1/reports/") {
		return false
	}
	for _, marker := range []string{
		"api_key", "apikey", "api key", "authorization:", "bearer ", "token=", "sk-",
		"object key", "objectkey", "file_ref", "system prompt", "developer prompt",
		"tool arguments", "tool result", "mcp result", "raw provider", "provider raw",
		"raw error", "localhost", "127.0.0.1", "minio",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if strings.Contains(lower, "://") {
		return true
	}
	return false
}

func mergeReportArtifact(existing []service.ReportArtifact, artifact service.ReportArtifact) []service.ReportArtifact {
	key := reportArtifactKey(artifact)
	for index, item := range existing {
		if reportArtifactsMatch(item, artifact) || (key != "" && reportArtifactKey(item) == key) {
			next := append([]service.ReportArtifact(nil), existing...)
			next[index] = artifact
			return next
		}
	}
	return append(existing, artifact)
}

func reportArtifactsMatch(left, right service.ReportArtifact) bool {
	rightIDs := map[string]struct{}{}
	for _, id := range reportArtifactStableIDs(right) {
		rightIDs[id] = struct{}{}
	}
	for _, id := range reportArtifactStableIDs(left) {
		if _, ok := rightIDs[id]; ok {
			return true
		}
	}
	return false
}

func reportArtifactStableIDs(artifact service.ReportArtifact) []string {
	ids := make([]string, 0, 3)
	for _, key := range []string{"reportId", "reportFileId", "jobId"} {
		if value, ok := artifact[key].(string); ok && strings.TrimSpace(value) != "" {
			ids = append(ids, strings.TrimSpace(value))
		}
	}
	return ids
}

func reportArtifactKey(artifact service.ReportArtifact) string {
	for _, key := range []string{"reportId", "reportFileId", "jobId", "downloadPath", "detailPath", "reportName"} {
		if value, ok := artifact[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (r *Postgres) listThinkingForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string][]service.ReasoningStep, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
    rr.assistant_message_id::text,
    ps.id::text,
    ps.step_type,
    COALESCE(ps.label, ''),
    COALESCE(ps.detail, ''),
    ps.status,
    ps.created_at
FROM response_process_steps ps
JOIN response_runs rr ON rr.id = ps.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id::text = ANY($1::text[])
    AND rr.conversation_id::text = $2
    AND c.external_user_id = $3
    AND c.deleted_at IS NULL
ORDER BY rr.assistant_message_id::text, ps.step_order`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message thinking: %w", err)
	}
	defer rows.Close()

	items := map[string][]service.ReasoningStep{}
	for rows.Next() {
		var messageID string
		var step service.ReasoningStep
		var label, detail sql.NullString
		if err := rows.Scan(&messageID, &step.ID, &step.Type, &label, &detail, &step.Status, &step.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message thinking: %w", err)
		}
		step.MessageID = messageID
		if label.Valid {
			step.Title = label.String
		}
		if detail.Valid {
			step.Summary = detail.String
		}
		items[messageID] = append(items[messageID], step)
	}
	return items, rows.Err()
}

func (r *Postgres) listCitationsForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string][]service.Citation, error) {
	rows, err := r.pool.Query(ctx, r.messageCitationSelect(ctx)+` WHERE ci.message_id::text = ANY($1::text[]) AND m.conversation_id::text = $2 AND c.external_user_id = $3 AND c.deleted_at IS NULL ORDER BY ci.message_id, ci.citation_no`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message citations: %w", err)
	}
	defer rows.Close()
	citations, err := scanCitations(rows)
	if err != nil {
		return nil, err
	}
	items := map[string][]service.Citation{}
	for _, citation := range citations {
		items[citation.MessageID] = append(items[citation.MessageID], citation)
	}
	return items, nil
}
