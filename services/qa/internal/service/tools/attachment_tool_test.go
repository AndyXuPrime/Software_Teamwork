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
			AttachmentID: "att-1", ChunkID: "chunk-1", Filename: "guide.pdf", ContentPreview: "pressure limit",
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
}
