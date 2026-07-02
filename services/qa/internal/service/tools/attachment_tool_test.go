package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

type attachmentSearcherStub struct {
	results         []SessionAttachmentHit
	reportResults   []SessionAttachmentHit
	reportSearchMax int
}

func (s attachmentSearcherStub) SearchSessionAttachments(_ context.Context, _ string, _ string, _ []string, query string, _ int) ([]SessionAttachmentHit, error) {
	if strings.TrimSpace(query) == "" && s.reportResults != nil {
		if s.reportSearchMax > 0 && len(s.reportResults) > s.reportSearchMax {
			return s.reportResults[:s.reportSearchMax], nil
		}
		return s.reportResults, nil
	}
	return s.results, nil
}

func (s attachmentSearcherStub) ListSessionAttachmentReportSource(_ context.Context, _ string, _ string, _ []string, _ int) ([]SessionAttachmentHit, error) {
	return s.reportResults, nil
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

func TestAttachmentToolClientIncludesReportSourceFromBoundAttachments(t *testing.T) {
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{
		results: []SessionAttachmentHit{{
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "inspection.pdf", ContentPreview: "matched overload", Content: "matched overload evidence",
		}},
		reportResults: []SessionAttachmentHit{{
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "inspection.pdf", Content: "matched overload evidence", PageNumber: 1, ChunkIndex: 1,
		}, {
			AttachmentID: "att-1", ChunkID: "chunk-2", Filename: "inspection.pdf", Content: "unmatched transformer cooling note", PageNumber: 2, ChunkIndex: 2,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	ctx = contextutil.WithMessageAttachmentIDs(ctx, []string{"att-1"})

	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"overload","include_report_source":true}`))
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v err=%v", result, err)
	}
	var decoded struct {
		ReportSourceExcerpt string `json:"report_source_excerpt"`
		ReportSource        struct {
			AttachmentCount int  `json:"attachment_count"`
			ChunkCount      int  `json:"chunk_count"`
			Truncated       bool `json:"truncated"`
		} `json:"report_source"`
	}
	if err := json.Unmarshal([]byte(result.Content), &decoded); err != nil {
		t.Fatalf("decode result: %v\n%s", err, result.Content)
	}
	if !strings.Contains(decoded.ReportSourceExcerpt, "matched overload evidence") || !strings.Contains(decoded.ReportSourceExcerpt, "unmatched transformer cooling note") {
		t.Fatalf("report source excerpt did not include bounded content across chunks: %s", decoded.ReportSourceExcerpt)
	}
	if decoded.ReportSource.AttachmentCount != 1 || decoded.ReportSource.ChunkCount != 2 || decoded.ReportSource.Truncated {
		t.Fatalf("report source metadata = %+v", decoded.ReportSource)
	}
}

func TestAttachmentToolClientReportSourceUsesDedicatedChunkListing(t *testing.T) {
	reportChunks := make([]SessionAttachmentHit, 0, 6)
	for i := 1; i <= 6; i++ {
		reportChunks = append(reportChunks, SessionAttachmentHit{
			AttachmentID: "att-1",
			ChunkID:      "chunk-" + string(rune('0'+i)),
			Filename:     "inspection.pdf",
			Content:      "report source chunk " + string(rune('0'+i)),
			PageNumber:   i,
			ChunkIndex:   i,
		})
	}
	client, err := NewAttachmentToolClient(AttachmentToolConfig{Searcher: attachmentSearcherStub{
		results: []SessionAttachmentHit{{
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "inspection.pdf", ContentPreview: "matched", Content: "matched evidence",
		}},
		reportResults:   reportChunks,
		reportSearchMax: 5,
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithSessionID(ctx, "sess-1")
	ctx = contextutil.WithMessageAttachmentIDs(ctx, []string{"att-1"})

	result, err := client.CallTool(ctx, ToolSearchSessionAttachments, json.RawMessage(`{"query":"matched","include_report_source":true}`))
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %+v err=%v", result, err)
	}
	var decoded struct {
		ReportSourceExcerpt string `json:"report_source_excerpt"`
		ReportSource        struct {
			ChunkCount int `json:"chunk_count"`
		} `json:"report_source"`
	}
	if err := json.Unmarshal([]byte(result.Content), &decoded); err != nil {
		t.Fatalf("decode result: %v\n%s", err, result.Content)
	}
	if decoded.ReportSource.ChunkCount != 6 || !strings.Contains(decoded.ReportSourceExcerpt, "report source chunk 6") {
		t.Fatalf("report source should include chunks beyond ordinary search cap: %+v excerpt=%q", decoded.ReportSource, decoded.ReportSourceExcerpt)
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
