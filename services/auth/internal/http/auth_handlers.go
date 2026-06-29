package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/auth/internal/service"
)

type credentialRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionResponseData struct {
	User    userSummaryResponse    `json:"user"`
	Session sessionSummaryResponse `json:"session"`
}

type userSummaryResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type userRecordResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type sessionSummaryResponse struct {
	SessionID   string    `json:"sessionId"`
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type sessionIdentityResponse struct {
	SessionID    string              `json:"sessionId"`
	User         userSummaryResponse `json:"user"`
	TokenType    string              `json:"tokenType"`
	ExpiresAt    time.Time           `json:"expiresAt"`
	IssuedAt     time.Time           `json:"issuedAt"`
	RevokedAt    *time.Time          `json:"revokedAt,omitempty"`
	RevokeReason *string             `json:"revokeReason,omitempty"`
}

type userPermissionsResponse struct {
	UserID      string    `json:"userId"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload credentialRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	result, err := auth.CreateUser(r.Context(), requestContextFromHeaders(r), service.CreateUserInput{
		Username: payload.Username,
		Password: payload.Password,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, sessionResponseFromDomain(result), requestIDFromContext(r.Context()))
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload credentialRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	result, err := auth.CreateSession(r.Context(), requestContextFromHeaders(r), service.CreateSessionInput{
		Username: payload.Username,
		Password: payload.Password,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionResponseFromDomain(result), requestIDFromContext(r.Context()))
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	user, err := auth.GetUser(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userRecordFromDomain(user), requestIDFromContext(r.Context()))
}

func (s *Server) handleGetUserPermissions(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	permissions, err := auth.GetUserPermissions(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userPermissionsFromDomain(permissions), requestIDFromContext(r.Context()))
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	identity, err := auth.GetSession(r.Context(), requestContextFromHeaders(r), r.PathValue("sessionId"))
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionIdentityFromDomain(identity), requestIDFromContext(r.Context()))
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if err := auth.RevokeSession(r.Context(), requestContextFromHeaders(r), r.PathValue("sessionId"), r.URL.Query().Get("reason")); err != nil {
		writeAppError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (AuthService, bool) {
	if s.auth == nil {
		writeAppError(w, r, service.DependencyError("auth repository is not configured", nil))
		return nil, false
	}
	return s.auth, true
}

func requestContextFromHeaders(r *http.Request) service.RequestContext {
	return service.RequestContext{
		RequestID:      requestIDFromContext(r.Context()),
		CallerService:  strings.TrimSpace(r.Header.Get("X-Caller-Service")),
		ClientIP:       clientIPFromRequest(r),
		UserAgent:      strings.TrimSpace(r.UserAgent()),
		ForwardedFor:   strings.TrimSpace(r.Header.Get("X-Forwarded-For")),
		ForwardedProto: strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")),
	}
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

func clientIPFromRequest(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func sessionResponseFromDomain(result service.SessionResponse) sessionResponseData {
	return sessionResponseData{
		User:    userSummaryFromDomain(result.User),
		Session: sessionSummaryFromDomain(result.Session),
	}
}

func userSummaryFromDomain(user service.UserSummary) userSummaryResponse {
	return userSummaryResponse{
		ID:          user.ID,
		Username:    user.Username,
		Roles:       safeStrings(user.Roles),
		Permissions: safeStrings(user.Permissions),
	}
}

func userRecordFromDomain(user service.UserRecord) userRecordResponse {
	return userRecordResponse{
		ID:          user.ID,
		Username:    user.Username,
		Roles:       safeStrings(user.Roles),
		Permissions: safeStrings(user.Permissions),
		Status:      user.Status,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func sessionSummaryFromDomain(session service.SessionSummary) sessionSummaryResponse {
	return sessionSummaryResponse{
		SessionID:   session.SessionID,
		AccessToken: session.AccessToken,
		TokenType:   session.TokenType,
		ExpiresAt:   session.ExpiresAt,
	}
}

func sessionIdentityFromDomain(identity service.SessionIdentity) sessionIdentityResponse {
	return sessionIdentityResponse{
		SessionID:    identity.Session.ID,
		User:         userSummaryFromDomain(identity.User),
		TokenType:    identity.Session.TokenType,
		ExpiresAt:    identity.Session.ExpiresAt,
		IssuedAt:     identity.Session.IssuedAt,
		RevokedAt:    identity.Session.RevokedAt,
		RevokeReason: identity.Session.RevokeReason,
	}
}

func userPermissionsFromDomain(permissions service.UserPermissions) userPermissionsResponse {
	return userPermissionsResponse{
		UserID:      permissions.UserID,
		Roles:       safeStrings(permissions.Roles),
		Permissions: safeStrings(permissions.Permissions),
		UpdatedAt:   permissions.UpdatedAt,
	}
}

func safeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
