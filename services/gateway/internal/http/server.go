package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

type Config struct {
	Logger               *slog.Logger
	ServiceVersion       string
	Environment          string
	RequestTimeout       time.Duration
	MaxBodyBytes         int64
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSAllowCredentials bool
}

type Server struct {
	logger         *slog.Logger
	serviceVersion string
	environment    string
	mux            *http.ServeMux
	handler        http.Handler
}

func NewServer(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	s := &Server{
		logger:         cfg.Logger,
		serviceVersion: cfg.ServiceVersion,
		environment:    cfg.Environment,
		mux:            http.NewServeMux(),
	}
	s.routes()
	s.handler = middleware.Chain(
		s.mux,
		middleware.RequestID(),
		middleware.Recover(cfg.Logger),
		middleware.Timeout(cfg.RequestTimeout),
		middleware.CORS(middleware.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowedMethods:   cfg.CORSAllowedMethods,
			AllowedHeaders:   cfg.CORSAllowedHeaders,
			AllowCredentials: cfg.CORSAllowCredentials,
		}),
		middleware.BodyLimit(cfg.MaxBodyBytes),
	)
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{
		Status:      "ok",
		Service:     "gateway",
		Version:     s.serviceVersion,
		Environment: s.environment,
	}, middleware.RequestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, healthResponse{
		Status:      "ready",
		Service:     "gateway",
		Version:     s.serviceVersion,
		Environment: s.environment,
	}, middleware.RequestIDFromContext(r.Context()))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	response.WriteError(w, http.StatusNotFound, response.ErrorDetail{
		Code:      response.CodeNotFound,
		Message:   "route not found",
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}

type healthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Version     string `json:"version,omitempty"`
	Environment string `json:"environment,omitempty"`
}
