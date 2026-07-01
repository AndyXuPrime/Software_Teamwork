package service_test

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/platform/storage"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

func TestCreateFileComputesChecksumAndStoresContent(t *testing.T) {
	files := newTestService(t, storage.NewMemoryStore())
	created, err := files.CreateFile(context.Background(), internalContext(), service.CreateFileInput{
		FileName:    "policy.pdf",
		ContentType: "application/pdf",
		SizeBytes:   int64(len("content")),
		Content:     strings.NewReader("content"),
	})
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	if created.ID != "file_1" {
		t.Fatalf("file id = %q", created.ID)
	}
	if created.Filename != "policy.pdf" || created.ContentType != "application/pdf" || created.SizeBytes != int64(len("content")) {
		t.Fatalf("file metadata = %+v", created)
	}
	if created.ChecksumSHA256 != "ed7002b439e9ac845f22357d822bac1444730fbdb6016d3ec9432297b9ec9f73" {
		t.Fatalf("checksum = %q", created.ChecksumSHA256)
	}
	if created.StorageObjectKey == "" {
		t.Fatal("storage object key is empty")
	}

	content, err := files.GetFileContent(context.Background(), internalContext(), created.ID)
	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}
	defer content.Body.Close()
	body, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "content" {
		t.Fatalf("content = %q", string(body))
	}
}

func TestCreateFileRejectsChecksumMismatch(t *testing.T) {
	files := newTestService(t, storage.NewMemoryStore())
	_, err := files.CreateFile(context.Background(), internalContext(), service.CreateFileInput{
		FileName:       "policy.pdf",
		ContentType:    "application/pdf",
		SizeBytes:      int64(len("content")),
		ChecksumSHA256: strings.Repeat("0", 64),
		Content:        strings.NewReader("content"),
	})
	if !hasCode(err, service.CodeValidation) {
		t.Fatalf("CreateFile() error = %v, want validation_error", err)
	}
}

func TestDeleteFileHidesMetadataAndContent(t *testing.T) {
	files := newTestService(t, storage.NewMemoryStore())
	created := createTestFile(t, files)

	if err := files.DeleteFile(context.Background(), internalContext(), created.ID); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}
	if _, err := files.GetFile(context.Background(), internalContext(), created.ID); !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetFile() error = %v, want not_found", err)
	}
	if _, err := files.GetFileContent(context.Background(), internalContext(), created.ID); !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetFileContent() error = %v, want not_found", err)
	}
}

func TestDeleteFileIsIdempotentAfterPurge(t *testing.T) {
	files := newTestService(t, storage.NewMemoryStore())
	created := createTestFile(t, files)

	if err := files.DeleteFile(context.Background(), internalContext(), created.ID); err != nil {
		t.Fatalf("DeleteFile() first error = %v", err)
	}
	if err := files.DeleteFile(context.Background(), internalContext(), created.ID); err != nil {
		t.Fatalf("DeleteFile() second error = %v", err)
	}
}

func TestDeleteFileCanRetryAfterStorageFailure(t *testing.T) {
	store := storage.NewMemoryStore()
	files := newTestService(t, &failOnceDeleteStore{ObjectStore: store})
	created := createTestFile(t, files)

	err := files.DeleteFile(context.Background(), internalContext(), created.ID)
	if !hasCode(err, service.CodeDependency) {
		t.Fatalf("DeleteFile() first error = %v, want dependency_error", err)
	}
	if _, err := files.GetFile(context.Background(), internalContext(), created.ID); !hasCode(err, service.CodeNotFound) {
		t.Fatalf("GetFile() after failed purge = %v, want not_found", err)
	}
	if err := files.DeleteFile(context.Background(), internalContext(), created.ID); err != nil {
		t.Fatalf("DeleteFile() retry error = %v", err)
	}
}

func TestCreateFileRequiresInternalCaller(t *testing.T) {
	files := newTestService(t, storage.NewMemoryStore())
	_, err := files.CreateFile(context.Background(), service.RequestContext{}, service.CreateFileInput{
		FileName:    "policy.pdf",
		ContentType: "application/pdf",
		SizeBytes:   int64(len("content")),
		Content:     strings.NewReader("content"),
	})
	if !hasCode(err, service.CodeUnauthorized) {
		t.Fatalf("CreateFile() error = %v, want unauthorized", err)
	}
}

func newTestService(t *testing.T, store service.ObjectStore) *service.Service {
	t.Helper()
	repo := repository.NewMemoryRepository()
	counter := 0
	return service.New(repo, store, service.WithIDGenerator(func(prefix string) (string, error) {
		counter++
		return prefix + "_" + strconv.Itoa(counter), nil
	}))
}

func createTestFile(t *testing.T, files *service.Service) service.FileObject {
	t.Helper()
	created, err := files.CreateFile(context.Background(), internalContext(), service.CreateFileInput{
		FileName:    "policy.pdf",
		ContentType: "application/pdf",
		SizeBytes:   int64(len("content")),
		Content:     strings.NewReader("content"),
	})
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	return created
}

func internalContext() service.RequestContext {
	return service.RequestContext{
		RequestID:     "req_test",
		CallerService: "knowledge",
		ServiceToken:  "test-token",
	}
}

func hasCode(err error, code service.Code) bool {
	var appErr *service.AppError
	return errors.As(err, &appErr) && appErr.Code == code
}

type failOnceDeleteStore struct {
	service.ObjectStore
	failed bool
}

func (s *failOnceDeleteStore) Delete(ctx context.Context, key string) error {
	if !s.failed {
		s.failed = true
		return errors.New("storage backend unavailable with object key files/secret")
	}
	return s.ObjectStore.Delete(ctx, key)
}
