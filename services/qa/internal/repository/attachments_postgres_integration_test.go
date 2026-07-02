package repository

import (
	"context"
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
