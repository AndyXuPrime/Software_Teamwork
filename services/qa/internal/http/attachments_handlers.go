package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

const multipartUploadEnvelopeBytes = int64(1 << 20)

const attachmentParseFailedCode = "attachment_parse_failed"

type sessionAttachmentSummary struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"sessionId"`
	Filename     string     `json:"filename"`
	ContentType  string     `json:"contentType"`
	SizeBytes    int64      `json:"sizeBytes"`
	Status       string     `json:"status"`
	ErrorCode    *string    `json:"errorCode"`
	ErrorMessage *string    `json:"errorMessage"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt"`
}

type AttachmentService interface {
	Upload(ctx context.Context, userID, sessionID string, input service.CreateAttachmentInput) (service.AttachmentUploadResult, error)
	List(ctx context.Context, userID string, sessionID string, options service.AttachmentListOptions) (service.Page[service.SessionAttachment], error)
	Get(ctx context.Context, userID string, sessionID string, attachmentID string) (service.SessionAttachment, error)
	Delete(ctx context.Context, userID string, sessionID string, attachmentID string) error
	Process(ctx context.Context, userID string, sessionID string, attachmentID string) error
}

func (s *Server) handleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	if s.attachments == nil {
		writeError(w, r, service.NewError(service.CodeInternal, "attachments are unavailable", nil))
		return
	}
	maxFileBytes := s.attachmentMaxBytes
	if maxFileBytes <= 0 {
		maxFileBytes = 20 << 20
	}
	requestBodyLimit := maxFileBytes + multipartUploadEnvelopeBytes
	r.Body = http.MaxBytesReader(w, r.Body, requestBodyLimit)
	if err := r.ParseMultipartForm(requestBodyLimit); err != nil {
		writeError(w, r, service.ValidationError(map[string]string{"file": "multipart form is invalid"}))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, r, service.ValidationError(map[string]string{"file": "is required"}))
		return
	}
	defer file.Close()
	var body bytes.Buffer
	size, err := io.Copy(&body, file)
	if err != nil {
		writeError(w, r, service.ValidationError(map[string]string{"file": "could not be read"}))
		return
	}
	if size > maxFileBytes {
		writeError(w, r, service.NewError(service.CodeTooLarge, "file exceeds maximum upload size", nil))
		return
	}
	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	result, err := s.attachments.Upload(r.Context(), userID, r.PathValue("sessionId"), service.CreateAttachmentInput{
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   size,
		Body:        bytes.NewReader(body.Bytes()),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	go func() {
		_ = s.attachments.Process(context.WithoutCancel(r.Context()), userID, r.PathValue("sessionId"), result.Attachment.ID)
	}()
	writeData(w, r, http.StatusCreated, publicSessionAttachment(result.Attachment))
}

func (s *Server) handleListAttachments(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	page, pageSize, err := pagination(r, 20)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.attachments.List(r.Context(), userID, r.PathValue("sessionId"), service.AttachmentListOptions{Page: page, PageSize: pageSize, Status: r.URL.Query().Get("status")})
	if err != nil {
		writeError(w, r, err)
		return
	}
	publicResult := service.Page[sessionAttachmentSummary]{
		Items:    make([]sessionAttachmentSummary, 0, len(result.Items)),
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	}
	for _, attachment := range result.Items {
		publicResult.Items = append(publicResult.Items, publicSessionAttachment(attachment))
	}
	writePage(w, r, http.StatusOK, publicResult)
}

func (s *Server) handleGetAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	result, err := s.attachments.Get(r.Context(), userID, r.PathValue("sessionId"), r.PathValue("attachmentId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, r, http.StatusOK, publicSessionAttachment(result))
}

func publicSessionAttachment(attachment service.SessionAttachment) sessionAttachmentSummary {
	expiresAt := attachment.ExpiresAt
	result := sessionAttachmentSummary{
		ID:          attachment.ID,
		SessionID:   attachment.SessionID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		SizeBytes:   attachment.SizeBytes,
		Status:      attachment.Status,
		CreatedAt:   attachment.CreatedAt,
		ExpiresAt:   &expiresAt,
	}
	if attachment.Status == service.AttachmentStatusFailed {
		code := attachmentParseFailedCode
		result.ErrorCode = &code
		if summary := strings.TrimSpace(attachment.ErrorSummary); summary != "" {
			result.ErrorMessage = &summary
		}
	}
	return result
}

func (s *Server) handleDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}
	if err := s.attachments.Delete(r.Context(), userID, r.PathValue("sessionId"), r.PathValue("attachmentId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func attachmentIDsFromQuery(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			ids = append(ids, v)
		}
	}
	return ids
}

func intQueryDefault(r *http.Request, name string, fallback int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
