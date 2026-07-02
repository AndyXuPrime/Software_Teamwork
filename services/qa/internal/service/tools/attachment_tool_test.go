package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

type attachmentSearcherStub struct {
	results []SessionAttachmentHit
}

func (s attachmentSearcherStub) SearchSessionAttachments(context.Context, string, string, []string, string, int) ([]SessionAttachmentHit, error) {
	return s.results, nil
}

func TestAttachmentToolClientReturnsCitationReadyResults(t *testing.T) {
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{
		results: []SessionAttachmentHit{{
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "guide.pdf", ContentPreview: "pressure limit", Content: "pressure limit from full parsed chunk",
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	ctx = contextutil.WithMessageAttachmentIDs(ctx, []string{"att-1"})
	ctx = contextutil.WithCitationNo(ctx, 1)
	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"pressure"}`))
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v err=%v", result, err)
	}
	if !strings.Contains(result.Content, `"attachment_id":"att-1"`) {
		t.Fatalf("result = %s", result.Content)
	}
	if !strings.Contains(result.Content, `"content_excerpt":"pressure limit from full parsed chunk"`) {
		t.Fatalf("result missing content excerpt = %s", result.Content)
	}
}

func TestAttachmentToolClientBoundsContentExcerpt(t *testing.T) {
	longChunk := strings.Repeat("summer peak inspection ", 200)
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{
		results: []SessionAttachmentHit{{
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "guide.pdf", ContentPreview: "summer peak", Content: longChunk,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	ctx = contextutil.WithMessageAttachmentIDs(ctx, []string{"att-1"})
	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"summer"}`))
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v err=%v", result, err)
	}
	var decoded struct {
		Results []struct {
			ContentExcerpt string `json:"content_excerpt"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(result.Content), &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(decoded.Results) != 1 || len([]rune(decoded.Results[0].ContentExcerpt)) > maxAttachmentContentExcerptRunes {
		t.Fatalf("content excerpt length = %d, want <= %d", len([]rune(decoded.Results[0].ContentExcerpt)), maxAttachmentContentExcerptRunes)
	}
	if strings.Contains(result.Content, longChunk) {
		t.Fatalf("result leaked full chunk content")
	}
}

func TestAttachmentToolClientKeepsUsableResultsWhenLongExcerptsHitResultBudget(t *testing.T) {
	longChunk := strings.Repeat("夏峰巡检附件正文 ", 260)
	results := make([]SessionAttachmentHit, 0, 5)
	allowed := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		attachmentID := "att-" + string(rune('1'+i))
		allowed = append(allowed, attachmentID)
		results = append(results, SessionAttachmentHit{
			AttachmentID:   attachmentID,
			ChunkID:        "chunk-" + string(rune('1'+i)),
			Filename:       "guide.pdf",
			ContentPreview: "夏峰巡检",
			Content:        longChunk,
		})
	}
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{results: results}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	ctx = contextutil.WithMessageAttachmentIDs(ctx, allowed)

	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"夏峰","limit":5}`))
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v err=%v", result, err)
	}
	if len([]byte(result.Content)) > maxAttachmentResultSize {
		t.Fatalf("result size = %d, want <= %d", len([]byte(result.Content)), maxAttachmentResultSize)
	}
	var decoded struct {
		Returned  int  `json:"returned"`
		Truncated bool `json:"truncated"`
		Results   []struct {
			ContentExcerpt string `json:"content_excerpt"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(result.Content), &decoded); err != nil {
		t.Fatalf("decode result: %v\n%s", err, result.Content)
	}
	if decoded.Returned == 0 || len(decoded.Results) == 0 {
		t.Fatalf("long excerpts should still return usable results: %s", result.Content)
	}
	if decoded.Truncated {
		t.Fatalf("result should shrink excerpts instead of dropping all hits: %s", result.Content)
	}
	for _, item := range decoded.Results {
		if strings.TrimSpace(item.ContentExcerpt) == "" {
			t.Fatalf("result contained an empty content excerpt: %s", result.Content)
		}
	}
}

func TestAttachmentToolClientRejectsUnboundAttachments(t *testing.T) {
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"pressure"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(result.Content, "no_bound_attachments") {
		t.Fatalf("CallTool() = %+v, want no_bound_attachments error", result)
	}
}
