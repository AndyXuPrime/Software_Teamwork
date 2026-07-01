package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

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
	maxBytes := s.attachmentMaxBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
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
	writeData(w, r, http.StatusCreated, result.Attachment)
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
	result, err := s.attachments.List(r.Context(), userID, r.PathValue("sessionId"), service.AttachmentListOptions{Page: page, PageSize: pageSize})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writePage(w, r, http.StatusOK, result)
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
	writeData(w, r, http.StatusOK, result)
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
