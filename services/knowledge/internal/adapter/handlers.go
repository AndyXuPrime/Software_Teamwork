package adapter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func (s *Server) handleListKnowledgeBases(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	items, total, err := s.vendor.ListDatasets(r.Context(), reqCtx.UserID, page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writePageJSON(w, http.StatusOK, knowledgeBasesFromVendor(items), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleCreateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body createKnowledgeBaseRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"name": "is required"}))
		return
	}
	payload, err := buildCreateDatasetBody(body)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	created, err := s.vendor.CreateDataset(r.Context(), reqCtx.UserID, payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusCreated, knowledgeBaseFromVendor(created), reqCtx.RequestID)
}

func (s *Server) handleGetKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	dataset, err := s.vendor.GetDataset(r.Context(), reqCtx.UserID, r.PathValue("knowledgeBaseId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, knowledgeBaseFromVendor(dataset), reqCtx.RequestID)
}

func (s *Server) handleUpdateKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body updateKnowledgeBaseRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	payload, err := buildUpdateDatasetBody(body)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	updated, err := s.vendor.UpdateDataset(r.Context(), reqCtx.UserID, r.PathValue("knowledgeBaseId"), payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, knowledgeBaseFromVendor(updated), reqCtx.RequestID)
}

func (s *Server) handleDeleteKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.vendor.DeleteDataset(r.Context(), reqCtx.UserID, r.PathValue("knowledgeBaseId")); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	items, total, err := s.vendor.ListDocuments(r.Context(), reqCtx.UserID, r.PathValue("knowledgeBaseId"), page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writePageJSON(w, http.StatusOK, documentsFromVendor(items), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	file, header, ok := parseDocumentUpload(w, r, s.maxUploadBytes)
	if !ok {
		return
	}
	defer file.Close()

	contentType := ""
	if header != nil {
		contentType = strings.TrimSpace(header.Header.Get("Content-Type"))
	}
	uploaded, err := s.vendor.UploadDocument(r.Context(), reqCtx.UserID, r.PathValue("knowledgeBaseId"), header.Filename, contentType, file)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	kbID := r.PathValue("knowledgeBaseId")
	docID := stringField(uploaded, "id")
	if s.cfg.AutoStartIngestion && docID != "" {
		if err := s.vendor.StartDocumentParse(r.Context(), reqCtx.UserID, kbID, []string{docID}); err != nil {
			writeAppError(w, r, mapVendorError(err))
			return
		}
		uploaded["run"] = "RUNNING"
	}
	writeJSON(w, http.StatusCreated, documentFromVendor(uploaded), reqCtx.RequestID)
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	doc, err := s.vendor.GetDocument(r.Context(), reqCtx.UserID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, documentFromVendor(doc), reqCtx.RequestID)
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body updateDocumentRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Tags == nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must include at least one supported field"}))
		return
	}
	doc, err := s.vendor.GetDocument(r.Context(), reqCtx.UserID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	kbID := stringField(doc, "kb_id", "dataset_id")
	payload, err := buildUpdateDocumentBody(*body.Tags)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	updated, err := s.vendor.UpdateDocument(r.Context(), reqCtx.UserID, kbID, r.PathValue("documentId"), payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writeJSON(w, http.StatusOK, documentFromVendor(updated), reqCtx.RequestID)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := mutationScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	if err := s.vendor.DeleteDocument(r.Context(), reqCtx.UserID, r.PathValue("documentId")); err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDocumentChunks(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	page, err := parsePageQuery(r)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	doc, err := s.vendor.GetDocument(r.Context(), reqCtx.UserID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	kbID := stringField(doc, "kb_id", "dataset_id")
	chunks, total, err := s.vendor.ListChunks(r.Context(), reqCtx.UserID, kbID, r.PathValue("documentId"), page.Page, page.PageSize)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	writePageJSON(w, http.StatusOK, documentChunksFromVendor(chunks, kbID, r.PathValue("documentId")), service.Page{
		Page:     page.Page,
		PageSize: page.PageSize,
		Total:    total,
	}, reqCtx.RequestID)
}

func (s *Server) handleGetDocumentContent(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	doc, err := s.vendor.GetDocument(r.Context(), reqCtx.UserID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	kbID := stringField(doc, "kb_id", "dataset_id")
	contentType, body, err := s.vendor.DownloadDocument(r.Context(), reqCtx.UserID, kbID, r.PathValue("documentId"))
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	w.Header().Set("Content-Type", contentType)
	if len(body) > 0 {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) handleCreateKnowledgeQuery(w http.ResponseWriter, r *http.Request) {
	reqCtx, ok := s.gatewayContext(w, r)
	if !ok {
		return
	}
	if _, err := readScope(reqCtx); err != nil {
		writeAppError(w, r, err)
		return
	}
	var body knowledgeQueryRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	payload, err := buildRetrievalBody(body)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	data, err := s.vendor.RetrievalSearch(r.Context(), reqCtx.UserID, payload)
	if err != nil {
		writeAppError(w, r, mapVendorError(err))
		return
	}
	topK := body.TopK
	if topK <= 0 {
		topK = 10
	}
	scoreThreshold := 0.0
	if body.ScoreThreshold != nil {
		scoreThreshold = *body.ScoreThreshold
	}
	writeJSON(w, http.StatusCreated, knowledgeQueryFromVendor(newQueryID(), strings.TrimSpace(body.Query), data, topK, scoreThreshold, body.Rerank, body.RerankTopN), reqCtx.RequestID)
}

func parsePageQuery(r *http.Request) (service.PageInput, error) {
	page := parsePositiveIntParam(r, "page")
	pageSize := parsePositiveIntParam(r, "pageSize")
	return normalizePage(page, pageSize)
}

func parsePositiveIntParam(r *http.Request, name string) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must be a valid JSON object"}))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"body": "must contain only one JSON object"}))
		return false
	}
	return true
}

func parseDocumentUpload(w http.ResponseWriter, r *http.Request, maxUploadBytes int64) (multipart.File, *multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		fieldMessage := "multipart form is invalid"
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			fieldMessage = "exceeds maximum upload size"
		}
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": fieldMessage}))
		return nil, nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": "is required"}))
		return nil, nil, false
	}
	if header == nil || header.Size == 0 {
		_ = file.Close()
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"file": "must not be empty"}))
		return nil, nil, false
	}
	return file, header, true
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "req_fallback"
	}
	return hex.EncodeToString(buf[:])
}

func (s *Server) logRequestFailure(ctx context.Context, requestID, method, path string, status int, durationMs int64) {
	s.logger.ErrorContext(ctx, "http request failed",
		"service", "knowledge-adapter",
		"request_id", requestID,
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", durationMs,
	)
}
