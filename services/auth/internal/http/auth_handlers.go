package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/auth/internal/service"
)

type credentialRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createAdminUserRequest struct {
	Username          string  `json:"username"`
	TemporaryPassword string  `json:"temporaryPassword"`
	Role              string  `json:"role"`
	DisplayName       *string `json:"displayName"`
	Email             *string `json:"email"`
	Phone             *string `json:"phone"`
}

type updateProfileRequest struct {
	DisplayName json.RawMessage `json:"displayName"`
	Email       json.RawMessage `json:"email"`
	Phone       json.RawMessage `json:"phone"`
}

type updateAdminUserRequest struct {
	DisplayName json.RawMessage `json:"displayName"`
	Email       json.RawMessage `json:"email"`
	Phone       json.RawMessage `json:"phone"`
	Status      json.RawMessage `json:"status"`
	Role        json.RawMessage `json:"role"`
}

type passwordChangeRequest struct {
	CurrentPassword         string `json:"currentPassword"`
	NewPassword             string `json:"newPassword"`
	NewPasswordConfirmation string `json:"newPasswordConfirmation"`
}

type adminPasswordResetRequest struct {
	TemporaryPassword string `json:"temporaryPassword"`
}

type sessionResponseData struct {
	User    userSummaryResponse    `json:"user"`
	Session sessionSummaryResponse `json:"session"`
}

type userSummaryResponse struct {
	ID                 string   `json:"id"`
	Username           string   `json:"username"`
	DisplayName        string   `json:"displayName"`
	Email              *string  `json:"email"`
	Phone              *string  `json:"phone"`
	Status             string   `json:"status"`
	MustChangePassword bool     `json:"mustChangePassword"`
	Roles              []string `json:"roles"`
	Permissions        []string `json:"permissions"`
}

type userRecordResponse struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"displayName"`
	Email              *string   `json:"email"`
	Phone              *string   `json:"phone"`
	Roles              []string  `json:"roles"`
	Permissions        []string  `json:"permissions"`
	Status             string    `json:"status"`
	MustChangePassword bool      `json:"mustChangePassword"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type adminUserResponse struct {
	userRecordResponse
	ManageableRoles []string         `json:"manageableRoles"`
	Actions         adminUserActions `json:"actions"`
}

type adminUserActions struct {
	CanDisable       bool `json:"canDisable"`
	CanEnable        bool `json:"canEnable"`
	CanResetPassword bool `json:"canResetPassword"`
	CanChangeRole    bool `json:"canChangeRole"`
}

type adminUserListResponse struct {
	Data []adminUserResponse `json:"data"`
	Page pageInfoResponse    `json:"page"`
}

type pageInfoResponse struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
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
	Status       string              `json:"status"`
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

func (s *Server) handleListAdminUsers(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	page, pageSize, ok := paginationFromQuery(w, r)
	if !ok {
		return
	}
	result, err := auth.ListManagedUsers(r.Context(), requestContextFromHeaders(r), service.ListManagedUsersInput{
		Page:     page,
		PageSize: pageSize,
		Username: r.URL.Query().Get("username"),
		Role:     r.URL.Query().Get("role"),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	list := adminUserListFromDomain(result)
	writePaginatedJSON(w, http.StatusOK, list.Data, list.Page, requestIDFromContext(r.Context()))
}

func (s *Server) handleCreateAdminUser(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload createAdminUserRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	result, err := auth.CreateAdminUser(r.Context(), requestContextFromHeaders(r), service.CreateAdminUserInput{
		Username:          payload.Username,
		TemporaryPassword: payload.TemporaryPassword,
		Role:              payload.Role,
		DisplayName:       payload.DisplayName,
		Email:             payload.Email,
		Phone:             payload.Phone,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminUserFromDomain(result), requestIDFromContext(r.Context()))
}

func (s *Server) handleUpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload updateAdminUserRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	input, err := adminPatchFromRequest(payload)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	result, err := auth.UpdateManagedUser(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"), input)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminUserFromDomain(result), requestIDFromContext(r.Context()))
}

func (s *Server) handleResetAdminUserPassword(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload adminPasswordResetRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	result, err := auth.ResetManagedUserPassword(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"), service.ResetAdminPasswordInput{
		TemporaryPassword: payload.TemporaryPassword,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, adminUserFromDomain(result), requestIDFromContext(r.Context()))
}

func (s *Server) handleUpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload updateProfileRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	input, err := profilePatchFromRequest(payload)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	user, err := auth.UpdateProfile(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"), input)
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userRecordFromDomain(user), requestIDFromContext(r.Context()))
}

func (s *Server) handleChangeUserPassword(w http.ResponseWriter, r *http.Request) {
	auth, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	var payload passwordChangeRequest
	if !decodeJSONBody(w, r, &payload) {
		return
	}
	user, err := auth.ChangeRequiredPassword(r.Context(), requestContextFromHeaders(r), r.PathValue("userId"), service.ChangePasswordInput{
		CurrentPassword:         payload.CurrentPassword,
		NewPassword:             payload.NewPassword,
		NewPasswordConfirmation: payload.NewPasswordConfirmation,
	})
	if err != nil {
		writeAppError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, userRecordFromDomain(user), requestIDFromContext(r.Context()))
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
		ActorUserID:    strings.TrimSpace(r.Header.Get("X-User-Id")),
		ActorRoles:     csvHeader(r.Header.Get("X-User-Roles")),
		ActorPerms:     csvHeader(r.Header.Get("X-User-Permissions")),
		ClientIP:       clientIPFromRequest(r),
		UserAgent:      strings.TrimSpace(r.UserAgent()),
		ForwardedFor:   strings.TrimSpace(r.Header.Get("X-Forwarded-For")),
		ForwardedProto: strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")),
	}
}

func paginationFromQuery(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page, err := intQuery(r, "page", 1)
	if err != nil || page <= 0 {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"page": "must be a positive integer"}))
		return 0, 0, false
	}
	pageSize, err := intQuery(r, "pageSize", 20)
	if err != nil || pageSize <= 0 {
		writeAppError(w, r, service.ValidationError("request validation failed", map[string]string{"pageSize": "must be a positive integer"}))
		return 0, 0, false
	}
	return page, pageSize, true
}

func intQuery(r *http.Request, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func profilePatchFromRequest(payload updateProfileRequest) (service.UpdateProfileInput, error) {
	displayName, err := optionalStringField("displayName", payload.DisplayName)
	if err != nil {
		return service.UpdateProfileInput{}, err
	}
	email, err := optionalStringField("email", payload.Email)
	if err != nil {
		return service.UpdateProfileInput{}, err
	}
	phone, err := optionalStringField("phone", payload.Phone)
	if err != nil {
		return service.UpdateProfileInput{}, err
	}
	return service.UpdateProfileInput{DisplayName: displayName, Email: email, Phone: phone}, nil
}

func adminPatchFromRequest(payload updateAdminUserRequest) (service.UpdateAdminUserInput, error) {
	displayName, err := optionalStringField("displayName", payload.DisplayName)
	if err != nil {
		return service.UpdateAdminUserInput{}, err
	}
	email, err := optionalStringField("email", payload.Email)
	if err != nil {
		return service.UpdateAdminUserInput{}, err
	}
	phone, err := optionalStringField("phone", payload.Phone)
	if err != nil {
		return service.UpdateAdminUserInput{}, err
	}
	status, err := optionalStringField("status", payload.Status)
	if err != nil {
		return service.UpdateAdminUserInput{}, err
	}
	role, err := optionalStringField("role", payload.Role)
	if err != nil {
		return service.UpdateAdminUserInput{}, err
	}
	return service.UpdateAdminUserInput{
		DisplayName: displayName,
		Email:       email,
		Phone:       phone,
		Status:      status,
		Role:        role,
	}, nil
}

func optionalStringField(name string, raw json.RawMessage) (service.OptionalStringField, error) {
	if len(raw) == 0 {
		return service.OptionalStringField{}, nil
	}
	if string(raw) == "null" {
		return service.OptionalStringField{Set: true}, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return service.OptionalStringField{}, service.ValidationError("request validation failed", map[string]string{name: "must be a string or null"})
	}
	return service.OptionalStringField{Set: true, Value: &value}, nil
}

func csvHeader(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
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
		ID:                 user.ID,
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              cloneStringPtr(user.Email),
		Phone:              cloneStringPtr(user.Phone),
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		Roles:              safeStrings(user.Roles),
		Permissions:        safeStrings(user.Permissions),
	}
}

func userRecordFromDomain(user service.UserRecord) userRecordResponse {
	return userRecordResponse{
		ID:                 user.ID,
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              cloneStringPtr(user.Email),
		Phone:              cloneStringPtr(user.Phone),
		Roles:              safeStrings(user.Roles),
		Permissions:        safeStrings(user.Permissions),
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}
}

func adminUserFromDomain(user service.AdminUserRecord) adminUserResponse {
	return adminUserResponse{
		userRecordResponse: userRecordFromDomain(user.UserRecord),
		ManageableRoles:    safeStrings(user.ManageableRoles),
		Actions: adminUserActions{
			CanDisable:       user.Actions.CanDisable,
			CanEnable:        user.Actions.CanEnable,
			CanResetPassword: user.Actions.CanResetPassword,
			CanChangeRole:    user.Actions.CanChangeRole,
		},
	}
}

func adminUserListFromDomain(result service.AdminUserList) adminUserListResponse {
	users := make([]adminUserResponse, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, adminUserFromDomain(user))
	}
	return adminUserListResponse{
		Data: users,
		Page: pageInfoResponse{
			Page:     result.Page.Page,
			PageSize: result.Page.PageSize,
			Total:    result.Page.Total,
		},
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
		Status:       identity.Session.Status,
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

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
