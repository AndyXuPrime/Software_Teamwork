package service

import (
	"bytes"
	"context"
	"errors"
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
	createErr    error
}

func (s *attachmentRepoStub) GetConversation(_ context.Context, _, sessionID string) (Conversation, error) {
	if s.conversation.ID != sessionID {
		return Conversation{}, NewError(CodeNotFound, "conversation not found", nil)
	}
	return s.conversation, nil
}

func (s *attachmentRepoStub) CreateAttachment(_ context.Context, attachment SessionAttachment, maxPerSession int, maxSessionBytes int64) (SessionAttachment, error) {
	if s.createErr != nil {
		return SessionAttachment{}, s.createErr
	}
	var activeCount int
	var activeBytes int64
	for _, item := range s.attachments {
		if item.SessionID == attachment.SessionID && item.DeletedAt == nil {
			activeCount++
			activeBytes += item.SizeBytes
		}
	}
	if activeCount >= maxPerSession {
		return SessionAttachment{}, NewError(CodeConflict, "session attachment limit reached", nil)
	}
	if attachment.SizeBytes > maxSessionBytes-activeBytes {
		return SessionAttachment{}, NewError(CodeConflict, "session attachment size quota exceeded", nil)
	}
	s.attachments = append(s.attachments, attachment)
	return attachment, nil
}

func (s *attachmentRepoStub) ListAttachments(_ context.Context, _, sessionID string, opts AttachmentListOptions) (Page[SessionAttachment], error) {
	items := make([]SessionAttachment, 0, len(s.attachments))
	for _, item := range s.attachments {
		if item.SessionID == sessionID && (opts.Status == "" || item.Status == opts.Status) {
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
			s.chunks = removeAttachmentChunks(s.chunks, attachmentID)
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

func (s *attachmentRepoStub) ListExpiredAttachments(_ context.Context, now time.Time, limit int) ([]SessionAttachment, error) {
	out := make([]SessionAttachment, 0, limit)
	for _, item := range s.attachments {
		if item.DeletedAt != nil || item.ExpiresAt.After(now) {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *attachmentRepoStub) PurgeAttachments(_ context.Context, ids []string, now time.Time) error {
	for _, id := range ids {
		for i, item := range s.attachments {
			if item.ID == id && item.DeletedAt == nil {
				item.DeletedAt = &now
				item.Status = AttachmentStatusPurged
				s.attachments[i] = item
				break
			}
		}
	}
	return nil
}

func (s *attachmentRepoStub) CheckAttachmentQuota(_ context.Context, sessionID, _ string, sizeBytes int64, maxPerSession int, maxSessionBytes int64) error {
	var activeCount int
	var activeBytes int64
	for _, item := range s.attachments {
		if item.SessionID == sessionID && item.DeletedAt == nil {
			activeCount++
			activeBytes += item.SizeBytes
		}
	}
	if activeCount >= maxPerSession {
		return NewError(CodeConflict, "session attachment limit reached", nil)
	}
	if sizeBytes > maxSessionBytes-activeBytes {
		return NewError(CodeConflict, "session attachment size quota exceeded", nil)
	}
	return nil
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

func removeAttachmentChunks(chunks []SessionAttachmentChunk, attachmentID string) []SessionAttachmentChunk {
	out := chunks[:0]
	for _, chunk := range chunks {
		if chunk.AttachmentID != attachmentID {
			out = append(out, chunk)
		}
	}
	return out
}

type testFileClient struct {
	data      map[string][]byte
	next      int
	deleteErr error
	deleted   []string
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
	c.deleted = append(c.deleted, fileRef)
	if c.deleteErr != nil {
		return c.deleteErr
	}
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

func TestAttachmentServiceUploadRejectsUnsupportedContentType(t *testing.T) {
	repo := &attachmentRepoStub{conversation: Conversation{ID: "sess-1"}}
	svc, err := NewAttachmentService(repo, &testFileClient{data: map[string][]byte{}}, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Upload(context.Background(), "user-1", "sess-1", CreateAttachmentInput{
		Filename: "notes.bin", ContentType: "application/octet-stream", SizeBytes: 4,
		Body: bytes.NewReader([]byte("test")),
	})
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeUnsupportedMedia {
		t.Fatalf("Upload() error = %v, want unsupported media", err)
	}
}

func TestAttachmentServiceUploadRejectsSessionSizeQuota(t *testing.T) {
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		attachments: []SessionAttachment{{
			ID: "att-existing", SessionID: "sess-1", SizeBytes: maxSessionAttachmentBytes - 1,
		}},
	}
	svc, err := NewAttachmentService(repo, &testFileClient{data: map[string][]byte{}}, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Upload(context.Background(), "user-1", "sess-1", CreateAttachmentInput{
		Filename: "notes.txt", ContentType: "text/plain", SizeBytes: 2,
		Body: bytes.NewReader([]byte("ok")),
	})
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeConflict {
		t.Fatalf("Upload() error = %v, want conflict", err)
	}
}

func TestAttachmentServiceUploadDeletesFileWhenMetadataCreateFails(t *testing.T) {
	createErr := errors.New("metadata unavailable")
	repo := &attachmentRepoStub{conversation: Conversation{ID: "sess-1"}, createErr: createErr}
	files := &testFileClient{data: map[string][]byte{}}
	svc, err := NewAttachmentService(repo, files, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Upload(context.Background(), "user-1", "sess-1", CreateAttachmentInput{
		Filename: "notes.txt", ContentType: "text/plain", SizeBytes: 4,
		Body: bytes.NewReader([]byte("test")),
	})
	if !errors.Is(err, createErr) {
		t.Fatalf("Upload() error = %v, want metadata error", err)
	}
	if len(files.deleted) != 1 || len(files.data) != 0 {
		t.Fatalf("deleted=%v data=%v, want uploaded object removed", files.deleted, files.data)
	}
}

func TestAttachmentServiceUploadPreservesMetadataAndCleanupFailures(t *testing.T) {
	createErr := errors.New("metadata unavailable")
	deleteErr := errors.New("file cleanup unavailable")
	repo := &attachmentRepoStub{conversation: Conversation{ID: "sess-1"}, createErr: createErr}
	files := &testFileClient{data: map[string][]byte{}, deleteErr: deleteErr}
	svc, err := NewAttachmentService(repo, files, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Upload(context.Background(), "user-1", "sess-1", CreateAttachmentInput{
		Filename: "notes.txt", ContentType: "text/plain", SizeBytes: 4,
		Body: bytes.NewReader([]byte("test")),
	})
	if !errors.Is(err, createErr) || !errors.Is(err, deleteErr) {
		t.Fatalf("Upload() error = %v, want both metadata and cleanup failures", err)
	}
}

func TestAttachmentServiceListValidatesAndForwardsStatus(t *testing.T) {
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		attachments: []SessionAttachment{
			{ID: "ready", SessionID: "sess-1", Status: AttachmentStatusReady},
			{ID: "failed", SessionID: "sess-1", Status: AttachmentStatusFailed},
		},
	}
	svc, err := NewAttachmentService(repo, &testFileClient{data: map[string][]byte{}}, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}

	page, err := svc.List(context.Background(), "user-1", "sess-1", AttachmentListOptions{Status: AttachmentStatusReady})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "ready" {
		t.Fatalf("List() items = %+v, want ready attachment", page.Items)
	}
	_, err = svc.List(context.Background(), "user-1", "sess-1", AttachmentListOptions{Status: "unknown"})
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeValidation {
		t.Fatalf("List() invalid status error = %v, want validation error", err)
	}
}

func TestAttachmentServiceDeleteClearsTemporaryChunks(t *testing.T) {
	now := time.Now().UTC()
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		attachments: []SessionAttachment{{
			ID: "att-1", SessionID: "sess-1", FileRef: "file-1", Status: AttachmentStatusReady, ExpiresAt: now.Add(time.Hour),
		}},
		chunks: []SessionAttachmentChunk{{ID: "chunk-1", AttachmentID: "att-1", SessionID: "sess-1"}},
	}
	files := &testFileClient{data: map[string][]byte{"file-1": []byte("data")}}
	svc, err := NewAttachmentService(repo, files, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "user-1", "sess-1", "att-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(repo.chunks) != 0 {
		t.Fatalf("chunks after delete = %+v, want none", repo.chunks)
	}
	if _, ok := files.data["file-1"]; ok {
		t.Fatal("file object was not deleted")
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

func TestAttachmentServiceCleanupExpiredFileDeleteFailure(t *testing.T) {
	now := time.Now().UTC()
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		attachments: []SessionAttachment{{
			ID: "att-1", SessionID: "sess-1", FileRef: "file-1", Status: AttachmentStatusReady,
			ExpiresAt: now.Add(-time.Hour),
		}, {
			ID: "att-2", SessionID: "sess-1", FileRef: "file-2", Status: AttachmentStatusReady,
			ExpiresAt: now.Add(-time.Hour),
		}},
	}
	deleteErr := errors.New("file service unavailable")
	files := &testFileClient{data: map[string][]byte{"file-1": []byte("a"), "file-2": []byte("b")}, deleteErr: deleteErr}
	svc, err := NewAttachmentService(repo, files, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}

	expired, err := svc.CleanupExpired(context.Background(), 10)
	if err == nil {
		t.Fatal("CleanupExpired() should return an error when file deletion fails")
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("CleanupExpired() error = %v, want %v", err, deleteErr)
	}
	if len(expired) != 2 {
		t.Fatalf("CleanupExpired() returned %d attachments, want 2 (still returned even on file delete failure)", len(expired))
	}
	// File data should remain since deletion failed.
	if len(files.data) != 2 {
		t.Fatalf("file data after failed cleanup = %d entries, want 2 (files leaked)", len(files.data))
	}
}

func TestAttachmentServiceDeleteFileDeleteFailure(t *testing.T) {
	now := time.Now().UTC()
	repo := &attachmentRepoStub{
		conversation: Conversation{ID: "sess-1"},
		attachments: []SessionAttachment{{
			ID: "att-1", SessionID: "sess-1", FileRef: "file-1", Status: AttachmentStatusReady,
			ExpiresAt: now.Add(time.Hour),
		}},
	}
	deleteErr := errors.New("file service unavailable")
	files := &testFileClient{data: map[string][]byte{"file-1": []byte("data")}, deleteErr: deleteErr}
	svc, err := NewAttachmentService(repo, files, testParserClient{}, AttachmentServiceConfig{})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(context.Background(), "user-1", "sess-1", "att-1")
	if err == nil {
		t.Fatal("Delete() should return an error when file deletion fails")
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("Delete() error = %v, want %v", err, deleteErr)
	}
	// File data should remain since deletion failed.
	if _, ok := files.data["file-1"]; !ok {
		t.Fatal("file data was unexpectedly deleted despite deletion error")
	}
}
