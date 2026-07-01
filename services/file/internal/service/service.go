package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

const (
	checksumSHA256HexLength = 64
	storageBackendMemory    = "memory"
)

type FileRepository interface {
	CreateFile(ctx context.Context, file FileObject) (FileObject, error)
	FindFileByID(ctx context.Context, id string) (FileObject, error)
	MarkFileDeleteRequested(ctx context.Context, id string, deletedAt time.Time) (FileObject, error)
	MarkFilePurging(ctx context.Context, id string, purgingAt time.Time) (FileObject, error)
	MarkFilePurged(ctx context.Context, id string, purgedAt time.Time) (FileObject, error)
	MarkFilePurgeFailed(ctx context.Context, id string, code string, message string, failedAt time.Time) (FileObject, error)
}

type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string, sizeBytes int64) error
	Get(ctx context.Context, key string) (StoredObject, error)
	Delete(ctx context.Context, key string) error
}

type Service struct {
	repo           FileRepository
	store          ObjectStore
	storageBackend string
	now            func() time.Time
	newID          func(prefix string) (string, error)
}

type Option func(*Service)

func New(repo FileRepository, store ObjectStore, opts ...Option) *Service {
	s := &Service{
		repo:           repo,
		store:          store,
		storageBackend: storageBackendMemory,
		now:            func() time.Time { return time.Now().UTC() },
		newID:          newPublicID,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

func WithStorageBackend(backend string) Option {
	return func(s *Service) {
		trimmed := strings.TrimSpace(backend)
		if trimmed != "" {
			s.storageBackend = trimmed
		}
	}
}

func WithIDGenerator(newID func(prefix string) (string, error)) Option {
	return func(s *Service) {
		if newID != nil {
			s.newID = newID
		}
	}
}

func (s *Service) CreateFile(ctx context.Context, reqCtx RequestContext, input CreateFileInput) (FileObject, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return FileObject{}, err
	}

	fields := map[string]string{}
	name, err := normalizeFileName(input.FileName)
	if err != nil {
		fields["file"] = err.Error()
	}
	if input.Content == nil {
		fields["file"] = "is required"
	} else if input.SizeBytes == 0 {
		fields["file"] = "must not be empty"
	}
	checksum, err := normalizeSHA256(input.ChecksumSHA256)
	if err != nil {
		fields["checksumSha256"] = err.Error()
	}
	if len(fields) > 0 {
		return FileObject{}, ValidationError("request validation failed", fields)
	}

	data, err := io.ReadAll(input.Content)
	if err != nil {
		return FileObject{}, DependencyError("file content read failed", err)
	}
	if len(data) == 0 {
		return FileObject{}, ValidationError("request validation failed", map[string]string{"file": "must not be empty"})
	}
	if input.SizeBytes > 0 && int64(len(data)) != input.SizeBytes {
		return FileObject{}, ValidationError("request validation failed", map[string]string{"file": "size does not match multipart metadata"})
	}

	computed := sha256.Sum256(data)
	computedChecksum := hex.EncodeToString(computed[:])
	if checksum != "" && checksum != computedChecksum {
		return FileObject{}, ValidationError("request validation failed", map[string]string{"checksumSha256": "does not match file content"})
	}
	if checksum == "" {
		checksum = computedChecksum
	}

	fileID, err := s.newID("file")
	if err != nil {
		return FileObject{}, DependencyError("file id generation failed", err)
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectKey := "files/" + fileID
	if err := s.store.Put(ctx, objectKey, bytes.NewReader(data), contentType, int64(len(data))); err != nil {
		return FileObject{}, DependencyError("object storage write failed", err)
	}

	now := s.now()
	file := FileObject{
		ID:               fileID,
		Filename:         name,
		ContentType:      contentType,
		SizeBytes:        int64(len(data)),
		ChecksumSHA256:   checksum,
		StorageBackend:   s.storageBackend,
		StorageObjectKey: objectKey,
		Status:           FileStatusAvailable,
		CreatedByService: callerService(reqCtx),
		RequestID:        strings.TrimSpace(reqCtx.RequestID),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := s.repo.CreateFile(ctx, file)
	if err != nil {
		_ = s.store.Delete(ctx, objectKey)
		if errors.Is(err, ErrConflict) {
			return FileObject{}, ConflictError("file already exists", err)
		}
		return FileObject{}, DependencyError("file metadata write failed", err)
	}
	return created, nil
}

func (s *Service) GetFile(ctx context.Context, reqCtx RequestContext, fileID string) (FileObject, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return FileObject{}, err
	}
	id := strings.TrimSpace(fileID)
	if id == "" {
		return FileObject{}, ValidationError("request validation failed", map[string]string{"fileId": "is required"})
	}
	file, err := s.repo.FindFileByID(ctx, id)
	if err != nil {
		return FileObject{}, mapFileRepositoryError(err, "file not found")
	}
	return file, nil
}

func (s *Service) DeleteFile(ctx context.Context, reqCtx RequestContext, fileID string) error {
	if err := validateInternalCaller(reqCtx); err != nil {
		return err
	}
	id := strings.TrimSpace(fileID)
	if id == "" {
		return ValidationError("request validation failed", map[string]string{"fileId": "is required"})
	}

	file, err := s.repo.MarkFileDeleteRequested(ctx, id, s.now())
	if err != nil {
		return mapFileRepositoryError(err, "file not found")
	}
	if file.Status == FileStatusPurged {
		return nil
	}

	file, err = s.repo.MarkFilePurging(ctx, id, s.now())
	if err != nil {
		return DependencyError("file cleanup metadata update failed", err)
	}
	if strings.TrimSpace(file.StorageObjectKey) == "" {
		_, _ = s.repo.MarkFilePurgeFailed(ctx, id, string(CodeDependency), "object storage reference is missing", s.now())
		return DependencyError("object storage reference is missing", errors.New("missing storage object key"))
	}
	if err := s.store.Delete(ctx, file.StorageObjectKey); err != nil {
		if errors.Is(err, ErrNotFound) {
			_, _ = s.repo.MarkFilePurged(ctx, id, s.now())
			return nil
		}
		_, _ = s.repo.MarkFilePurgeFailed(ctx, id, string(CodeDependency), "object storage delete failed", s.now())
		return DependencyError("object storage delete failed", err)
	}
	if _, err := s.repo.MarkFilePurged(ctx, id, s.now()); err != nil {
		return DependencyError("file cleanup metadata update failed", err)
	}
	return nil
}

func (s *Service) GetFileContent(ctx context.Context, reqCtx RequestContext, fileID string) (FileContent, error) {
	if err := validateInternalCaller(reqCtx); err != nil {
		return FileContent{}, err
	}
	id := strings.TrimSpace(fileID)
	if id == "" {
		return FileContent{}, ValidationError("request validation failed", map[string]string{"fileId": "is required"})
	}

	file, err := s.repo.FindFileByID(ctx, id)
	if err != nil {
		return FileContent{}, mapFileRepositoryError(err, "file not found")
	}
	object, err := s.store.Get(ctx, file.StorageObjectKey)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return FileContent{}, NotFoundError("file content not found")
		}
		return FileContent{}, DependencyError("object storage read failed", err)
	}
	contentType := object.ContentType
	if contentType == "" {
		contentType = file.ContentType
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	sizeBytes := object.SizeBytes
	if sizeBytes < 0 {
		sizeBytes = file.SizeBytes
	}
	return FileContent{File: file, Body: object.Body, ContentType: contentType, SizeBytes: sizeBytes}, nil
}

func validateInternalCaller(reqCtx RequestContext) error {
	if strings.TrimSpace(reqCtx.CallerService) == "" && strings.TrimSpace(reqCtx.UserID) == "" {
		return UnauthorizedError()
	}
	return nil
}

func normalizeFileName(name string) (string, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	trimmed = path.Base(trimmed)
	if trimmed == "." || trimmed == "/" || trimmed == "" {
		return "", fmt.Errorf("filename is required")
	}
	if strings.ContainsAny(trimmed, "\x00\r\n") {
		return "", fmt.Errorf("filename is invalid")
	}
	return trimmed, nil
}

func normalizeSHA256(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", nil
	}
	if len(trimmed) != checksumSHA256HexLength {
		return "", fmt.Errorf("must be a 64-character hexadecimal SHA-256 value")
	}
	if _, err := hex.DecodeString(trimmed); err != nil {
		return "", fmt.Errorf("must be a 64-character hexadecimal SHA-256 value")
	}
	return trimmed, nil
}

func callerService(reqCtx RequestContext) string {
	caller := strings.TrimSpace(reqCtx.CallerService)
	if caller != "" {
		return caller
	}
	return "gateway"
}

func mapFileRepositoryError(err error, notFoundMessage string) error {
	if errors.Is(err, ErrNotFound) {
		return NotFoundError(notFoundMessage)
	}
	if errors.Is(err, ErrConflict) {
		return ConflictError("file state conflict", err)
	}
	return DependencyError("file metadata access failed", err)
}

func newPublicID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
