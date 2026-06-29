package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/auth/internal/service"
)

const (
	defaultServiceVersion   = "0.1.0"
	defaultEnvironment      = "local"
	defaultReadinessTimeout = 2 * time.Second
)

type ReadinessChecker interface {
	Check(ctx context.Context) error
}

type AuthService interface {
	CreateUser(ctx context.Context, reqCtx service.RequestContext, input service.CreateUserInput) (service.SessionResponse, error)
	CreateSession(ctx context.Context, reqCtx service.RequestContext, input service.CreateSessionInput) (service.SessionResponse, error)
	GetUser(ctx context.Context, reqCtx service.RequestContext, userID string) (service.UserRecord, error)
	GetUserPermissions(ctx context.Context, reqCtx service.RequestContext, userID string) (service.UserPermissions, error)
	GetSession(ctx context.Context, reqCtx service.RequestContext, sessionID string) (service.SessionIdentity, error)
	RevokeSession(ctx context.Context, reqCtx service.RequestContext, sessionID string, reason string) error
}

type Config struct {
	ServiceVersion   string
	Environment      string
	ReadinessTimeout time.Duration
	ReadinessChecker ReadinessChecker
	Auth             AuthService
	ServiceToken     string
	Logger           *slog.Logger
}

type Server struct {
	auth             AuthService
	serviceToken     string
	serviceVersion   string
	environment      string
	readinessTimeout time.Duration
	readinessChecker ReadinessChecker
	logger           *slog.Logger
	mux              *http.ServeMux
}

func NewServer(cfg Config) *Server {
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = defaultServiceVersion
	}
	if cfg.Environment == "" {
		cfg.Environment = defaultEnvironment
	}
	if cfg.ReadinessTimeout <= 0 {
		cfg.ReadinessTimeout = defaultReadinessTimeout
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	s := &Server{
		auth:             cfg.Auth,
		serviceToken:     strings.TrimSpace(cfg.ServiceToken),
		serviceVersion:   cfg.ServiceVersion,
		environment:      cfg.Environment,
		readinessTimeout: cfg.ReadinessTimeout,
		readinessChecker: cfg.ReadinessChecker,
		logger:           cfg.Logger,
		mux:              http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)
	s.mux.HandleFunc("POST /internal/v1/users", s.handleCreateUser)
	s.mux.HandleFunc("GET /internal/v1/users/{userId}", s.handleGetUser)
	s.mux.HandleFunc("GET /internal/v1/users/{userId}/permissions", s.handleGetUserPermissions)
	s.mux.HandleFunc("POST /internal/v1/sessions", s.handleCreateSession)
	s.mux.HandleFunc("GET /internal/v1/sessions/{sessionId}", s.handleGetSession)
	s.mux.HandleFunc("DELETE /internal/v1/sessions/{sessionId}", s.handleDeleteSession)
	s.mux.HandleFunc("/", s.handleNotFound)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = newRequestID()
	}

	ctx := contextWithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)
	w.Header().Set("X-Request-Id", requestID)

	if strings.HasPrefix(r.URL.Path, "/internal/v1/") && s.serviceToken != "" && !secureTokenEqual(r.Header.Get("X-Service-Token"), s.serviceToken) {
		writeAppError(w, r, service.NewError(service.CodeUnauthorized, "service authentication required", nil))
		return
	}

	recorder := &statusRecorder{ResponseWriter: w}
	start := time.Now()
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.ErrorContext(ctx, "http panic recovered", "service", "auth", "request_id", requestID, "operation", "http_request")
			writeAppError(recorder, r, service.NewError(service.CodeInternal, "internal server error", nil))
		}
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		if status >= http.StatusInternalServerError {
			s.logger.ErrorContext(ctx, "http request failed", "service", "auth", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(start).Milliseconds())
		}
	}()

	s.mux.ServeHTTP(recorder, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Service: "auth",
		Status:  "ok",
		Version: s.serviceVersion,
	}, requestIDFromContext(r.Context()))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.readinessTimeout)
	defer cancel()

	dependency := dependencyStatus{Name: "postgres", Status: "not_configured"}
	status := http.StatusServiceUnavailable
	overall := "not_ready"
	if s.readinessChecker != nil {
		if err := s.readinessChecker.Check(ctx); err != nil {
			dependency.Status = "unavailable"
			dependency.Message = "postgres is unavailable"
		} else {
			dependency.Status = "ready"
			status = http.StatusOK
			overall = "ready"
		}
	}

	writeJSON(w, status, readinessResponse{
		Service:      "auth",
		Status:       overall,
		Version:      s.serviceVersion,
		Environment:  s.environment,
		Dependencies: []dependencyStatus{dependency},
	}, requestIDFromContext(r.Context()))
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, r, service.NotFoundError("route not found"))
}

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "req_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "req_" + hex.EncodeToString(bytes)
}

func secureTokenEqual(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

type readinessResponse struct {
	Service      string             `json:"service"`
	Status       string             `json:"status"`
	Version      string             `json:"version"`
	Environment  string             `json:"environment"`
	Dependencies []dependencyStatus `json:"dependencies"`
}

type dependencyStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
