package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type attachmentRepoStub struct {
	conversation Conversation
	attachments  []SessionAttachment
	chunks       []SessionAttachmentChunk
}

func (s *attachmentRepoStub) GetConversation(_ context.Context, _, sessionID string) (Conversation, error) {
	if s.conversation.ID != sessionID {
		return Conversation{}, NewError(CodeNotFound, "conversation not found", nil)
	}
	return s.conversation, nil
}

func (s *attachmentRepoStub) CreateAttachment(_ context.Context, attachment SessionAttachment) (SessionAttachment, error) {
	s.attachments = append(s.attachments, attachment)
	return attachment, nil
}

func (s *attachmentRepoStub) ListAttachments(_ context.Context, _, sessionID string, opts AttachmentListOptions) (Page[SessionAttachment], error) {
	items := make([]SessionAttachment, 0, len(s.attachments))
	for _, item := range s.attachments {
		if item.SessionID == sessionID {
			items = append(items, item)
		}
	}
	return Page[SessionAttachment]{Items: items, Total: len(items), Page: opts.Page, PageSize: opts.PageSize}, nil
}

func (s *attachmentRepoStub) GetAttachment(_ context.Context, _, sessionID, attachmentID string) (SessionAttachment, error) {
	for _, item := range s.attachments {
		if item.ID == attachmentID && item.SessionID == sessionID {
			return item, nil
		}
	}
	return SessionAttachment{}, NewError(CodeNotFound, "attachment not found", nil)
}

func (s *attachmentRepoStub) SoftDeleteAttachment(_ context.Context, _, sessionID, attachmentID string, now time.Time) error {
	for i, item := range s.attachments {
		if item.ID == attachmentID && item.SessionID == sessionID {
			item.DeletedAt = &now
			s.attachments[i] = item
			return nil
		}
	}
	return NewError(CodeNotFound, "attachment not found", nil)
}

func (s *attachmentRepoStub) MarkAttachmentParsing(_ context.Context, _, sessionID, attachmentID string, now time.Time) error {
	return s.setStatus(sessionID, attachmentID, AttachmentStatusParsing, "", now)
}

func (s *attachmentRepoStub) MarkAttachmentFailed(_ context.Context, _, sessionID, attachmentID, summary string, now time.Time) error {
	return s.setStatus(sessionID, attachmentID, AttachmentStatusFailed, summary, now)
}

func (s *attachmentRepoStub) ReplaceAttachmentChunks(_ context.Context, _, sessionID, attachmentID string, chunks []SessionAttachmentChunk, pageCount int, now time.Time) error {
	s.chunks = append([]SessionAttachmentChunk(nil), chunks...)
	return s.setReady(sessionID, attachmentID, pageCount, len(chunks), now)
}

func (s *attachmentRepoStub) ValidateReadyAttachments(_ context.Context, _, sessionID string, ids []string) ([]SessionAttachment, error) {
	out := make([]SessionAttachment, 0, len(ids))
	for _, id := range ids {
		item, err := s.GetAttachment(context.Background(), "", sessionID, id)
		if err != nil {
			return nil, err
		}
		if item.Status != AttachmentStatusReady {
			return nil, ValidationError(map[string]string{"attachmentIds": "attachments must be ready"})
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *attachmentRepoStub) BindMessageAttachments(context.Context, string, string, string, []string, time.Time) error {
	return nil
}

func (s *attachmentRepoStub) SearchSessionAttachmentChunks(_ context.Context, _, sessionID string, attachmentIDs []string, query string, limit int) ([]SessionAttachmentChunk, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]SessionAttachmentChunk, 0, limit)
	for _, chunk := range s.chunks {
		if chunk.SessionID != sessionID {
			continue
		}
		if len(attachmentIDs) > 0 && !containsString(attachmentIDs, chunk.AttachmentID) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(chunk.Content), query) {
			continue
		}
		out = append(out, chunk)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *attachmentRepoStub) CleanupExpiredAttachments(_ context.Context, now time.Time, limit int) ([]SessionAttachment, error) {
	out := make([]SessionAttachment, 0, limit)
	for _, item := range s.attachments {
		if item.DeletedAt != nil || item.ExpiresAt.After(now) {
			continue
		}
		deletedAt := now
		item.DeletedAt = &deletedAt
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *attachmentRepoStub) setStatus(sessionID, attachmentID, status, summary string, now time.Time) error {
	for i, item := range s.attachments {
		if item.ID == attachmentID && item.SessionID == sessionID {
			item.Status = status
			item.ErrorSummary = summary
			item.UpdatedAt = now
			s.attachments[i] = item
			return nil
		}
	}
	return NewError(CodeNotFound, "attachment not found", nil)
}

func (s *attachmentRepoStub) setReady(sessionID, attachmentID string, pageCount, chunkCount int, now time.Time) error {
	for i, item := range s.attachments {
		if item.ID == attachmentID && item.SessionID == sessionID {
			item.Status = AttachmentStatusReady
			item.PageCount = pageCount
			item.ChunkCount = chunkCount
			item.UpdatedAt = now
			s.attachments[i] = item
			return nil
		}
	}
	return NewError(CodeNotFound, "attachment not found", nil)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type testFileClient struct {
	data map[string][]byte
	next int
}

func (c *testFileClient) Upload(_ context.Context, _ string, _ string, _ int64, body io.Reader) (string, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	c.next++
	ref := fmt.Sprintf("file-%d", c.next)
	c.data[ref] = raw
	return ref, nil
}

func (c *testFileClient) Read(_ context.Context, fileRef string) ([]byte, error) {
	return c.data[fileRef], nil
}

func (c *testFileClient) Delete(_ context.Context, fileRef string) error {
	delete(c.data, fileRef)
	return nil
}

type testParserClient struct{}

func (testParserClient) Parse(_ context.Context, _, _ string, data []byte) (ParsedAttachment, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return ParsedAttachment{}, fmt.Errorf("document is empty")
	}
	return ParsedAttachment{PageCount: 1, Chunks: []ParsedAttachmentChunk{{PageNumber: 1, Content: text}}}, nil
}

func TestAttachmentServiceUploadAndProcess(t *testing.T) {
	repo := &attachmentRepoStub{conversation: Conversation{ID: "sess-1"}}
	svc, err := NewAttachmentService(repo, &testFileClient{data: map[string][]byte{}}, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.Upload(context.Background(), "user-1", "sess-1", CreateAttachmentInput{
		Filename: "notes.txt", ContentType: "text/plain", SizeBytes: 11,
		Body: bytes.NewReader([]byte("hello world")),
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if result.Attachment.Status != AttachmentStatusUploaded {
		t.Fatalf("status = %q, want uploaded", result.Attachment.Status)
	}
	if err := svc.Process(context.Background(), "user-1", "sess-1", result.Attachment.ID); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	ready, err := svc.Get(context.Background(), "user-1", "sess-1", result.Attachment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Status != AttachmentStatusReady || ready.ChunkCount == 0 {
		t.Fatalf("ready attachment = %+v", ready)
	}
}

func TestAttachmentServiceSearchSessionAttachments(t *testing.T) {
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		chunks: []SessionAttachmentChunk{{
			ID: "chunk-1", AttachmentID: "att-1", SessionID: "sess-1", Content: "boiler pressure limit", ContentPreview: "boiler pressure limit",
		}},
	}
	svc, err := NewAttachmentService(repo, &testFileClient{data: map[string][]byte{}}, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	results, err := svc.SearchSessionAttachments(context.Background(), "user-1", "sess-1", []string{"att-1"}, "boiler", 5)
	if err != nil {
		t.Fatalf("SearchSessionAttachments() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "chunk-1" {
		t.Fatalf("results = %+v", results)
	}
}
