package adapter

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
)

type Server struct {
	cfg        adapterconfig.Config
	logger     *slog.Logger
	httpClient *http.Client
	mux        *http.ServeMux
}

func NewServer(cfg adapterconfig.Config, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		cfg:    cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("GET /internal/v1/runtime/status", s.handleRuntimeStatus)
	s.mux.HandleFunc("/", s.handleNotImplemented)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "knowledge-adapter",
		"version": s.cfg.ServiceVersion,
	})
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
	})
}

func (s *Server) handleRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vendorOK, vendorDetail := s.checkVendorRuntime(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":              "adapter",
		"environment":       s.cfg.Environment,
		"vendor_runtime_url": s.cfg.VendorRuntimeURL,
		"vendor_runtime":    vendorDetail,
		"vendor_runtime_ok": vendorOK,
		"contract_routes":   "pending",
	})
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"code":    "not_implemented",
		"message": "knowledge contract adapter routes are not implemented yet",
		"path":    r.URL.Path,
	})
}

func (s *Server) checkVendorRuntime(ctx context.Context) (bool, map[string]any) {
	pingURL := s.cfg.VendorRuntimeURL + "/api/v1/system/ping"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
	if err != nil {
		return false, map[string]any{"url": pingURL, "error": err.Error()}
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return false, map[string]any{"url": pingURL, "error": err.Error()}
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	detail := map[string]any{
		"url":         pingURL,
		"status_code": res.StatusCode,
		"body":        strings.TrimSpace(string(body)),
	}
	return res.StatusCode == http.StatusOK && strings.TrimSpace(string(body)) == "pong", detail
}

func writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
