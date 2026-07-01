package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	AttachmentStatusUploaded = "uploaded"
	AttachmentStatusParsing  = "parsing"
	AttachmentStatusReady    = "ready"
	AttachmentStatusFailed   = "failed"
)

type SessionAttachment struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"sessionId"`
	OwnerUserID  string     `json:"-"`
	FileRef      string     `json:"-"`
	Filename     string     `json:"filename"`
	ContentType  string     `json:"contentType"`
	SizeBytes    int64      `json:"sizeBytes"`
	Status       string     `json:"status"`
	ErrorSummary string     `json:"errorSummary,omitempty"`
	PageCount    int        `json:"pageCount"`
	ChunkCount   int        `json:"chunkCount"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type SessionAttachmentChunk struct {
	ID             string `json:"id"`
	AttachmentID   string `json:"attachmentId"`
	SessionID      string `json:"sessionId"`
	ChunkIndex     int    `json:"chunkIndex"`
	PageNumber     int    `json:"pageNumber,omitempty"`
	SectionPath    string `json:"sectionPath,omitempty"`
	Content        string `json:"-"`
	ContentPreview string `json:"contentPreview"`
	TokenCount     int    `json:"tokenCount"`
	Filename       string `json:"filename,omitempty"`
}

type CreateAttachmentInput struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	Body        io.Reader
}

type AttachmentUploadResult struct {
	Attachment SessionAttachment `json:"attachment"`
}

type AttachmentListOptions struct {
	Page     int
	PageSize int
}

type ParsedAttachmentChunk struct {
	PageNumber  int
	SectionPath string
	Content     string
}

type ParsedAttachment struct {
	PageCount int
	Chunks    []ParsedAttachmentChunk
}

type AttachmentFileClient interface {
	Upload(ctx context.Context, name, contentType string, size int64, body io.Reader) (string, error)
	Read(ctx context.Context, fileRef string) ([]byte, error)
	Delete(ctx context.Context, fileRef string) error
}

type AttachmentParserClient interface {
	Parse(ctx context.Context, filename, contentType string, data []byte) (ParsedAttachment, error)
}

type SessionAttachmentSearcher interface {
	SearchSessionAttachments(context.Context, string, string, []string, string, int) ([]SessionAttachmentChunk, error)
}

type AttachmentRepository interface {
	GetConversation(context.Context, string, string) (Conversation, error)
	CreateAttachment(context.Context, SessionAttachment) (SessionAttachment, error)
	ListAttachments(context.Context, string, string, AttachmentListOptions) (Page[SessionAttachment], error)
	GetAttachment(context.Context, string, string, string) (SessionAttachment, error)
	SoftDeleteAttachment(context.Context, string, string, string, time.Time) error
	MarkAttachmentParsing(context.Context, string, string, string, time.Time) error
	MarkAttachmentFailed(context.Context, string, string, string, string, time.Time) error
	ReplaceAttachmentChunks(context.Context, string, string, string, []SessionAttachmentChunk, int, time.Time) error
	ValidateReadyAttachments(context.Context, string, string, []string) ([]SessionAttachment, error)
	BindMessageAttachments(context.Context, string, string, string, []string, time.Time) error
	SearchSessionAttachmentChunks(context.Context, string, string, []string, string, int) ([]SessionAttachmentChunk, error)
	CleanupExpiredAttachments(context.Context, time.Time, int) ([]SessionAttachment, error)
}

type AttachmentService struct {
	repository     AttachmentRepository
	fileClient     AttachmentFileClient
	parserClient   AttachmentParserClient
	ttl            time.Duration
	maxBytes       int64
	maxPerSession  int
	processTimeout time.Duration
	now            func() time.Time
}

type AttachmentServiceConfig struct {
	TTL            time.Duration
	MaxBytes       int64
	MaxPerSession  int
	ProcessTimeout time.Duration
}

func NewAttachmentService(repository AttachmentRepository, fileClient AttachmentFileClient, parserClient AttachmentParserClient, cfg AttachmentServiceConfig) (*AttachmentService, error) {
	if repository == nil || fileClient == nil || parserClient == nil {
		return nil, errors.New("attachment repository, file client and parser client are required")
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 20 << 20
	}
	if cfg.MaxPerSession <= 0 {
		cfg.MaxPerSession = 10
	}
	if cfg.ProcessTimeout <= 0 {
		cfg.ProcessTimeout = 60 * time.Second
	}
	return &AttachmentService{repository: repository, fileClient: fileClient, parserClient: parserClient, ttl: cfg.TTL, maxBytes: cfg.MaxBytes, maxPerSession: cfg.MaxPerSession, processTimeout: cfg.ProcessTimeout, now: time.Now}, nil
}

func (s *AttachmentService) Upload(ctx context.Context, userID, sessionID string, input CreateAttachmentInput) (AttachmentUploadResult, error) {
	if strings.TrimSpace(userID) == "" {
		return AttachmentUploadResult{}, NewError(CodeUnauthorized, "authentication required", nil)
	}
	if strings.TrimSpace(sessionID) == "" {
		return AttachmentUploadResult{}, ValidationError(map[string]string{"sessionId": "is required"})
	}
	filename := sanitizeAttachmentName(input.Filename)
	if filename == "" {
		return AttachmentUploadResult{}, ValidationError(map[string]string{"filename": "is required"})
	}
	if utf8.RuneCountInString(filename) > 255 {
		return AttachmentUploadResult{}, ValidationError(map[string]string{"filename": "must not exceed 255 characters"})
	}
	if input.SizeBytes <= 0 || input.SizeBytes > s.maxBytes {
		return AttachmentUploadResult{}, ValidationError(map[string]string{"sizeBytes": "exceeds attachment size limit"})
	}
	if _, err := s.repository.GetConversation(ctx, userID, sessionID); err != nil {
		return AttachmentUploadResult{}, err
	}
	page, err := s.repository.ListAttachments(ctx, userID, sessionID, AttachmentListOptions{Page: 1, PageSize: s.maxPerSession + 1})
	if err != nil {
		return AttachmentUploadResult{}, err
	}
	if page.Total >= s.maxPerSession {
		return AttachmentUploadResult{}, NewError(CodeConflict, "session attachment limit reached", nil)
	}
	fileRef, err := s.fileClient.Upload(ctx, filename, strings.TrimSpace(input.ContentType), input.SizeBytes, input.Body)
	if err != nil {
		return AttachmentUploadResult{}, NewError(CodeDependency, "file upload failed", err)
	}
	now := s.now().UTC()
	attachment := SessionAttachment{ID: newID("att"), SessionID: sessionID, OwnerUserID: userID, FileRef: fileRef, Filename: filename, ContentType: strings.TrimSpace(input.ContentType), SizeBytes: input.SizeBytes, Status: AttachmentStatusUploaded, ExpiresAt: now.Add(s.ttl), CreatedAt: now, UpdatedAt: now}
	created, err := s.repository.CreateAttachment(ctx, attachment)
	if err != nil {
		return AttachmentUploadResult{}, err
	}
	return AttachmentUploadResult{Attachment: created}, nil
}

func (s *AttachmentService) List(ctx context.Context, userID, sessionID string, opts AttachmentListOptions) (Page[SessionAttachment], error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 20
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100
	}
	return s.repository.ListAttachments(ctx, userID, sessionID, opts)
}
func (s *AttachmentService) Get(ctx context.Context, userID, sessionID, attachmentID string) (SessionAttachment, error) {
	return s.repository.GetAttachment(ctx, userID, sessionID, attachmentID)
}
func (s *AttachmentService) Delete(ctx context.Context, userID, sessionID, attachmentID string) error {
	att, err := s.repository.GetAttachment(ctx, userID, sessionID, attachmentID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	if err := s.repository.SoftDeleteAttachment(ctx, userID, sessionID, attachmentID, now); err != nil {
		return err
	}
	if att.FileRef != "" {
		_ = s.fileClient.Delete(context.WithoutCancel(ctx), att.FileRef)
	}
	return nil
}
func (s *AttachmentService) Process(ctx context.Context, userID, sessionID, attachmentID string) error {
	att, err := s.repository.GetAttachment(ctx, userID, sessionID, attachmentID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	if err := s.repository.MarkAttachmentParsing(ctx, userID, sessionID, attachmentID, now); err != nil {
		return err
	}
	procCtx, cancel := context.WithTimeout(ctx, s.processTimeout)
	defer cancel()
	data, err := s.fileClient.Read(procCtx, att.FileRef)
	if err != nil {
		_ = s.repository.MarkAttachmentFailed(context.WithoutCancel(ctx), userID, sessionID, attachmentID, "file read failed", s.now().UTC())
		return NewError(CodeDependency, "file read failed", err)
	}
	parsed, err := s.parserClient.Parse(procCtx, att.Filename, att.ContentType, data)
	if err != nil {
		_ = s.repository.MarkAttachmentFailed(context.WithoutCancel(ctx), userID, sessionID, attachmentID, "parser failed", s.now().UTC())
		return NewError(CodeDependency, "parser failed", err)
	}
	chunks := make([]SessionAttachmentChunk, 0, len(parsed.Chunks))
	for i, ch := range parsed.Chunks {
		content := strings.TrimSpace(ch.Content)
		if content == "" {
			continue
		}
		chunks = append(chunks, SessionAttachmentChunk{ID: newID("ach"), AttachmentID: attachmentID, SessionID: sessionID, ChunkIndex: i + 1, PageNumber: ch.PageNumber, SectionPath: strings.TrimSpace(ch.SectionPath), Content: content, ContentPreview: previewText(content, 240), TokenCount: roughTokenCount(content), Filename: att.Filename})
	}
	return s.repository.ReplaceAttachmentChunks(ctx, userID, sessionID, attachmentID, chunks, parsed.PageCount, s.now().UTC())
}
func (s *AttachmentService) SearchSessionAttachments(ctx context.Context, userID, sessionID string, attachmentIDs []string, query string, limit int) ([]SessionAttachmentChunk, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, NewError(CodeUnauthorized, "authentication required", nil)
	}
	if _, err := s.repository.GetConversation(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	return s.repository.SearchSessionAttachmentChunks(ctx, userID, sessionID, normalizeIDList(attachmentIDs), query, limit)
}

func (s *AttachmentService) CleanupExpired(ctx context.Context, limit int) ([]SessionAttachment, error) {
	if limit <= 0 {
		limit = 100
	}
	expired, err := s.repository.CleanupExpiredAttachments(ctx, s.now().UTC(), limit)
	if err != nil {
		return nil, err
	}
	for _, att := range expired {
		if att.FileRef != "" {
			_ = s.fileClient.Delete(context.WithoutCancel(ctx), att.FileRef)
		}
	}
	return expired, nil
}

func sanitizeAttachmentName(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	return value
}
func previewText(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}
	r := []rune(value)
	return string(r[:max])
}
func roughTokenCount(value string) int {
	count := len(strings.Fields(value))
	if count == 0 && strings.TrimSpace(value) != "" {
		return 1
	}
	return count
}
