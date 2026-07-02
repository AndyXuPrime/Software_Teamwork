package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestAttachmentCreateQuotaFilterAndPurgeIntegration(t *testing.T) {
	databaseURL := os.Getenv("QA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("QA_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	var defaultTools string
	if err := repo.pool.QueryRow(ctx, `SELECT enabled_tool_names::text FROM qa_config_versions WHERE version_no=1 AND created_by_user_id='system'`).Scan(&defaultTools); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(defaultTools, `"search_session_attachments"`) {
		t.Fatalf("system default enabled_tool_names=%s", defaultTools)
	}
	if !strings.Contains(defaultTools, `"document__generate_report_from_content"`) {
		t.Fatalf("system default enabled_tool_names=%s, want document content report tool", defaultTools)
	}

	now := time.Now().UTC()
	suffix := uint64(now.UnixNano()) & 0xffffffffffff
	conversationID := integrationUUID(suffix)
	userID := "attachment-integration-user"
	conversation := service.Conversation{
		ID: conversationID, OwnerUserID: userID, Title: "attachment quota",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if _, err := repo.CreateConversation(ctx, conversation); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			<-start
			_, createErr := repo.CreateAttachment(ctx, service.SessionAttachment{
				ID: integrationUUID(suffix + uint64(i) + 1), SessionID: conversationID,
				OwnerUserID: userID, FileRef: "file-ref", Filename: "quota.txt",
				ContentType: "text/plain", SizeBytes: 60, Status: service.AttachmentStatusUploaded,
				ExpiresAt: now.Add(-time.Minute), CreatedAt: now, UpdatedAt: now,
			}, 10, 100)
			results <- createErr
		}()
	}
	close(start)

	var successes, conflicts int
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			successes++
			continue
		}
		if appErr, ok := service.Classify(err); ok && appErr.Code == service.CodeConflict {
			conflicts++
			continue
		}
		t.Fatalf("CreateAttachment() unexpected error = %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want one of each", successes, conflicts)
	}

	uploaded, err := repo.ListAttachments(ctx, userID, conversationID, service.AttachmentListOptions{Page: 1, PageSize: 10, Status: service.AttachmentStatusUploaded})
	if err != nil || uploaded.Total != 1 {
		t.Fatalf("uploaded page=%+v err=%v", uploaded, err)
	}
	ready, err := repo.ListAttachments(ctx, userID, conversationID, service.AttachmentListOptions{Page: 1, PageSize: 10, Status: service.AttachmentStatusReady})
	if err != nil || ready.Total != 0 {
		t.Fatalf("ready page=%+v err=%v", ready, err)
	}

	expired, err := repo.ListExpiredAttachments(ctx, now, 1000)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	var expiredID string
	for _, attachment := range expired {
		if attachment.SessionID == conversationID {
			found = true
			expiredID = attachment.ID
			break
		}
	}
	if !found {
		t.Fatalf("expired=%+v, want expired attachment for session %s", expired, conversationID)
	}
	if err := repo.PurgeAttachments(ctx, []string{expiredID}, now); err != nil {
		t.Fatal(err)
	}
}

func TestAttachmentReportSourceChunkLimitIntegration(t *testing.T) {
	databaseURL := os.Getenv("QA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("QA_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	repo, err := NewPostgres(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	now := time.Now().UTC()
	suffix := uint64(now.UnixNano()) & 0xffffffffffff
	conversationID := integrationUUID(suffix)
	attachmentID := integrationUUID(suffix + 1)
	userID := "attachment-report-source-user"
	if _, err := repo.CreateConversation(ctx, service.Conversation{
		ID: conversationID, OwnerUserID: userID, Title: "report source limit",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAttachment(ctx, service.SessionAttachment{
		ID: attachmentID, SessionID: conversationID, OwnerUserID: userID, FileRef: "file-ref", Filename: "inspection.txt",
		ContentType: "text/plain", SizeBytes: 1024, Status: service.AttachmentStatusUploaded,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	}, 10, 1<<20); err != nil {
		t.Fatal(err)
	}
	chunks := make([]service.SessionAttachmentChunk, 0, 6)
	for i := 1; i <= 6; i++ {
		chunks = append(chunks, service.SessionAttachmentChunk{
			ID:             integrationUUID(suffix + uint64(10+i)),
			AttachmentID:   attachmentID,
			SessionID:      conversationID,
			ChunkIndex:     i,
			PageNumber:     i,
			Content:        fmt.Sprintf("report source chunk %02d", i),
			ContentPreview: fmt.Sprintf("chunk %02d", i),
			TokenCount:     4,
			Filename:       "inspection.txt",
		})
	}
	if err := repo.ReplaceAttachmentChunks(ctx, userID, conversationID, attachmentID, chunks, 6, now); err != nil {
		t.Fatal(err)
	}
	listed, err := repo.ListSessionAttachmentChunks(ctx, userID, conversationID, []string{attachmentID}, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 6 || listed[5].ChunkIndex != 6 {
		t.Fatalf("ListSessionAttachmentChunks() returned %+v, want all 6 chunks", listed)
	}
}
