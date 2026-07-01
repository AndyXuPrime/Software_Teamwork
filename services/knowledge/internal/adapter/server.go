package adapter

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/vendorclient"
)

type Server struct {
	cfg            adapterconfig.Config
	logger         *slog.Logger
	vendor         *vendorclient.Client
	maxUploadBytes int64
	mux            *http.ServeMux
}

func NewServer(cfg adapterconfig.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		cfg:            cfg,
		logger:         logger,
		vendor:         vendorclient.New(cfg.VendorRuntimeURL, 60*time.Second),
		maxUploadBytes: defaultMaxUploadBytes,
		mux:            http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = newRequestID()
	}

	ctx := contextWithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)
	w.Header().Set("X-Request-Id", requestID)

	recorder := &statusRecorder{ResponseWriter: w}
	start := time.Now()
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.ErrorContext(ctx, "http panic recovered", "service", "knowledge-adapter", "request_id", requestID)
			writeAppError(recorder, r, service.NewError(service.CodeInternal, "internal server error", nil))
		}
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= http.StatusInternalServerError {
			s.logRequestFailure(ctx, requestID, r.Method, r.URL.Path, status, time.Since(start).Milliseconds())
		}
	}()

	s.mux.ServeHTTP(recorder, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("GET /internal/v1/runtime/status", s.handleRuntimeStatus)

	s.mux.HandleFunc("GET /internal/v1/knowledge-bases", s.handleListKnowledgeBases)
	s.mux.HandleFunc("POST /internal/v1/knowledge-bases", s.handleCreateKnowledgeBase)
	s.mux.HandleFunc("GET /internal/v1/knowledge-bases/{knowledgeBaseId}", s.handleGetKnowledgeBase)
	s.mux.HandleFunc("PATCH /internal/v1/knowledge-bases/{knowledgeBaseId}", s.handleUpdateKnowledgeBase)
	s.mux.HandleFunc("DELETE /internal/v1/knowledge-bases/{knowledgeBaseId}", s.handleDeleteKnowledgeBase)
	s.mux.HandleFunc("GET /internal/v1/knowledge-bases/{knowledgeBaseId}/documents", s.handleListDocuments)
	s.mux.HandleFunc("POST /internal/v1/knowledge-bases/{knowledgeBaseId}/documents", s.handleUploadDocument)
	s.mux.HandleFunc("GET /internal/v1/documents/{documentId}", s.handleGetDocument)
	s.mux.HandleFunc("PATCH /internal/v1/documents/{documentId}", s.handleUpdateDocument)
	s.mux.HandleFunc("DELETE /internal/v1/documents/{documentId}", s.handleDeleteDocument)
	s.mux.HandleFunc("GET /internal/v1/documents/{documentId}/chunks", s.handleListDocumentChunks)
	s.mux.HandleFunc("GET /internal/v1/documents/{documentId}/content", s.handleGetDocumentContent)
	s.mux.HandleFunc("POST /internal/v1/knowledge-queries", s.handleCreateKnowledgeQuery)

	s.mux.HandleFunc("GET /internal/v1/parser-configs", s.handleParserConfigNotImplemented)
	s.mux.HandleFunc("POST /internal/v1/parser-configs", s.handleParserConfigNotImplemented)
	s.mux.HandleFunc("GET /internal/v1/parser-configs/{parserConfigId}", s.handleParserConfigNotImplemented)
	s.mux.HandleFunc("PATCH /internal/v1/parser-configs/{parserConfigId}", s.handleParserConfigNotImplemented)
	s.mux.HandleFunc("DELETE /internal/v1/parser-configs/{parserConfigId}", s.handleParserConfigNotImplemented)

	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "knowledge-adapter",
		"version": s.cfg.ServiceVersion,
	}, requestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vendorOK, vendorDetail := s.checkVendorRuntime(ctx)
	statusCode := http.StatusOK
	ready := "ok"
	if !vendorOK {
		statusCode = http.StatusServiceUnavailable
		ready = "degraded"
	}
	writeJSON(w, statusCode, map[string]any{
		"status":            ready,
		"service":           "knowledge-adapter",
		"vendor_runtime":    vendorDetail,
		"vendor_runtime_ok": vendorOK,
	}, requestIDFromContext(ctx))
}

func (s *Server) handleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vendorOK, vendorDetail := s.checkVendorRuntime(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":               "adapter",
		"environment":        s.cfg.Environment,
		"vendor_runtime_url": s.cfg.VendorRuntimeURL,
		"vendor_runtime":     vendorDetail,
		"vendor_runtime_ok":  vendorOK,
		"contract_routes":    "implemented",
	}, requestIDFromContext(ctx))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, r, service.NotFoundError("route not found"))
}

func (s *Server) checkVendorRuntime(ctx context.Context) (bool, map[string]any) {
	pingURL := s.cfg.VendorRuntimeURL + "/api/v1/system/ping"
	if err := s.vendor.Ping(ctx); err != nil {
		return false, map[string]any{"url": pingURL, "error": err.Error()}
	}
	return true, map[string]any{"url": pingURL, "status_code": http.StatusOK, "body": "pong"}
}
