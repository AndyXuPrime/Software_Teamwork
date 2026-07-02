package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authhttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/auth/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/auth/internal/service"
)

func TestHealthReturnsEnvelope(t *testing.T) {
	server := authhttp.NewServer(authhttp.Config{ServiceVersion: "0.1.0"})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "req_health")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var body successBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.RequestID != "req_health" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data["service"] != "auth" || body.Data["status"] != "ok" {
		t.Fatalf("data = %+v", body.Data)
	}
}

func TestReadyWithoutDatabaseIsUnavailable(t *testing.T) {
	server := authhttp.NewServer(authhttp.Config{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Header.Set("X-Request-Id", "req_ready")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body readinessBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.RequestID != "req_ready" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Status != "not_ready" {
		t.Fatalf("status = %q", body.Data.Status)
	}
	if len(body.Data.Dependencies) != 1 || body.Data.Dependencies[0].Status != "not_configured" {
		t.Fatalf("dependencies = %+v", body.Data.Dependencies)
	}
}

func TestReadyWithHealthyDatabase(t *testing.T) {
	server := authhttp.NewServer(authhttp.Config{ReadinessChecker: fakeChecker{}})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body readinessBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Data.Status != "ready" || body.Data.Dependencies[0].Status != "ready" {
		t.Fatalf("body = %+v", body)
	}
}

func TestReadyWithFailedDatabase(t *testing.T) {
	server := authhttp.NewServer(authhttp.Config{ReadinessChecker: fakeChecker{err: errors.New("down")}})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", res.Code)
	}
	var body readinessBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Data.Dependencies[0].Status != "unavailable" {
		t.Fatalf("dependencies = %+v", body.Data.Dependencies)
	}
}

func TestNotFoundReturnsErrorEnvelope(t *testing.T) {
	server := authhttp.NewServer(authhttp.Config{})
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("X-Request-Id", "req_missing")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d", res.Code)
	}
	var body errorBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Error.Code != "not_found" || body.Error.RequestID != "req_missing" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func TestCreateSessionReturnsSessionEnvelope(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{Auth: auth, ServiceToken: "test-service-token"})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/sessions", bytes.NewBufferString(`{"username":"alice","password":"secret"}`))
	req.Header.Set("X-Request-Id", "req_session")
	req.Header.Set("X-Service-Token", "test-service-token")
	req.Header.Set("X-Caller-Service", "gateway")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body sessionBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.RequestID != "req_session" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.User.ID != "usr_123" || body.Data.Session.AccessToken != "atk_v1_response" {
		t.Fatalf("body = %+v", body)
	}
	if body.Data.Session.AccessTokenHash != nil {
		t.Fatalf("accessTokenHash leaked: %+v", body.Data.Session.AccessTokenHash)
	}
}

func TestCreateSessionMissingCallerReturnsUnauthorized(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{Auth: auth, ServiceToken: "test-service-token"})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/sessions", bytes.NewBufferString(`{"username":"alice","password":"secret"}`))
	req.Header.Set("X-Request-Id", "req_session")
	req.Header.Set("X-Service-Token", "test-service-token")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Error.Code != "unauthorized" || body.Error.RequestID != "req_session" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func TestCreateSessionInvalidServiceTokenReturnsUnauthorized(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{Auth: auth, ServiceToken: "test-service-token"})
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/sessions", bytes.NewBufferString(`{"username":"alice","password":"secret"}`))
	req.Header.Set("X-Request-Id", "req_session")
	req.Header.Set("X-Caller-Service", "gateway")
	req.Header.Set("X-Service-Token", "wrong-token")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Error.Code != "unauthorized" || body.Error.RequestID != "req_session" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func TestAdminUsersRequiresGatewayAdminServiceToken(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{
		Auth:              auth,
		ServiceToken:      "test-service-token",
		GatewayAdminToken: "gateway-admin-token",
	})

	sharedReq := httptest.NewRequest(http.MethodGet, "/internal/v1/admin/users?page=1", nil)
	sharedReq.Header.Set("X-Request-Id", "req_admin_shared")
	sharedReq.Header.Set("X-Service-Token", "test-service-token")
	sharedReq.Header.Set("X-Caller-Service", "gateway")
	sharedReq.Header.Set("X-User-Id", "usr_admin")
	sharedReq.Header.Set("X-User-Roles", "admin")
	sharedRes := httptest.NewRecorder()
	server.ServeHTTP(sharedRes, sharedReq)
	if sharedRes.Code != http.StatusUnauthorized {
		t.Fatalf("shared token status = %d, body = %s", sharedRes.Code, sharedRes.Body.String())
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/internal/v1/admin/users?page=1", nil)
	adminReq.Header.Set("X-Request-Id", "req_admin")
	adminReq.Header.Set("X-Service-Token", "gateway-admin-token")
	adminReq.Header.Set("X-Caller-Service", "gateway")
	adminReq.Header.Set("X-User-Id", "usr_admin")
	adminReq.Header.Set("X-User-Roles", "admin")
	adminRes := httptest.NewRecorder()
	server.ServeHTTP(adminRes, adminReq)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("admin token status = %d, body = %s", adminRes.Code, adminRes.Body.String())
	}
}

func TestSelfWriteRoutesRequireGatewayServiceToken(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{
		Auth:              auth,
		ServiceToken:      "test-service-token",
		GatewayAdminToken: "gateway-admin-token",
	})

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "profile",
			method: http.MethodPatch,
			path:   "/internal/v1/users/usr_123/profile",
			body:   `{"displayName":"Alice"}`,
		},
		{
			name:   "password change",
			method: http.MethodPost,
			path:   "/internal/v1/users/usr_123/password-changes",
			body:   `{"currentPassword":"temporary","newPassword":"new-password","newPasswordConfirmation":"new-password"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sharedReq := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			sharedReq.Header.Set("X-Request-Id", "req_self_shared")
			sharedReq.Header.Set("X-Service-Token", "test-service-token")
			sharedReq.Header.Set("X-Caller-Service", "gateway")
			sharedReq.Header.Set("X-User-Id", "usr_123")
			sharedRes := httptest.NewRecorder()
			server.ServeHTTP(sharedRes, sharedReq)
			if sharedRes.Code != http.StatusUnauthorized {
				t.Fatalf("shared token status = %d, body = %s", sharedRes.Code, sharedRes.Body.String())
			}

			gatewayReq := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			gatewayReq.Header.Set("X-Request-Id", "req_self_gateway")
			gatewayReq.Header.Set("X-Service-Token", "gateway-admin-token")
			gatewayReq.Header.Set("X-Caller-Service", "gateway")
			gatewayReq.Header.Set("X-User-Id", "usr_123")
			gatewayRes := httptest.NewRecorder()
			server.ServeHTTP(gatewayRes, gatewayReq)
			if gatewayRes.Code != http.StatusOK {
				t.Fatalf("gateway token status = %d, body = %s", gatewayRes.Code, gatewayRes.Body.String())
			}
		})
	}
}

func TestGetSessionDoesNotReturnTokenHash(t *testing.T) {
	auth := fakeAuthService{now: time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)}
	server := authhttp.NewServer(authhttp.Config{Auth: auth, ServiceToken: "test-service-token"})
	req := httptest.NewRequest(http.MethodGet, "/internal/v1/sessions/sess_123", nil)
	req.Header.Set("X-Service-Token", "test-service-token")
	req.Header.Set("X-Caller-Service", "gateway")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body sessionIdentityBody
	decodeJSON(t, res.Body.Bytes(), &body)
	if body.Data.SessionID != "sess_123" || body.Data.AccessTokenHash != nil {
		t.Fatalf("body = %+v", body)
	}
}

type fakeChecker struct {
	err error
}

func (c fakeChecker) Check(context.Context) error {
	return c.err
}

type successBody struct {
	Data      map[string]string `json:"data"`
	RequestID string            `json:"requestId"`
}

type readinessBody struct {
	Data struct {
		Status       string `json:"status"`
		Dependencies []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"dependencies"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type errorBody struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

type sessionBody struct {
	Data struct {
		User struct {
			ID          string   `json:"id"`
			Username    string   `json:"username"`
			Roles       []string `json:"roles"`
			Permissions []string `json:"permissions"`
		} `json:"user"`
		Session struct {
			SessionID       string  `json:"sessionId"`
			AccessToken     string  `json:"accessToken"`
			TokenType       string  `json:"tokenType"`
			ExpiresAt       string  `json:"expiresAt"`
			AccessTokenHash *string `json:"accessTokenHash"`
		} `json:"session"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type sessionIdentityBody struct {
	Data struct {
		SessionID       string  `json:"sessionId"`
		AccessTokenHash *string `json:"accessTokenHash"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type fakeAuthService struct {
	now time.Time
}

func (s fakeAuthService) CreateUser(_ context.Context, reqCtx service.RequestContext, _ service.CreateUserInput) (service.SessionResponse, error) {
	if reqCtx.CallerService == "" {
		return service.SessionResponse{}, service.UnauthorizedError()
	}
	return s.sessionResponse(), nil
}

func (s fakeAuthService) CreateSession(_ context.Context, reqCtx service.RequestContext, _ service.CreateSessionInput) (service.SessionResponse, error) {
	if reqCtx.CallerService == "" {
		return service.SessionResponse{}, service.UnauthorizedError()
	}
	return s.sessionResponse(), nil
}

func (s fakeAuthService) GetUser(_ context.Context, reqCtx service.RequestContext, _ string) (service.UserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.UserRecord{}, service.UnauthorizedError()
	}
	return service.UserRecord{
		User: service.User{
			ID:        "usr_123",
			Username:  "alice",
			Status:    service.UserStatusActive,
			CreatedAt: s.now,
			UpdatedAt: s.now,
		},
		Roles:       []string{"standard"},
		Permissions: []string{"knowledge:read"},
	}, nil
}

func (s fakeAuthService) GetUserPermissions(_ context.Context, reqCtx service.RequestContext, _ string) (service.UserPermissions, error) {
	if reqCtx.CallerService == "" {
		return service.UserPermissions{}, service.UnauthorizedError()
	}
	return service.UserPermissions{UserID: "usr_123", Roles: []string{"standard"}, Permissions: []string{"knowledge:read"}, UpdatedAt: s.now}, nil
}

func (s fakeAuthService) ListManagedUsers(_ context.Context, reqCtx service.RequestContext, _ service.ListManagedUsersInput) (service.AdminUserList, error) {
	if reqCtx.CallerService == "" {
		return service.AdminUserList{}, service.UnauthorizedError()
	}
	user := s.adminUser()
	return service.AdminUserList{
		Users: []service.AdminUserRecord{user},
		Page:  service.PageInfo{Page: 1, PageSize: 20, Total: 1},
	}, nil
}

func (s fakeAuthService) CreateAdminUser(_ context.Context, reqCtx service.RequestContext, _ service.CreateAdminUserInput) (service.AdminUserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.AdminUserRecord{}, service.UnauthorizedError()
	}
	return s.adminUser(), nil
}

func (s fakeAuthService) UpdateManagedUser(_ context.Context, reqCtx service.RequestContext, _ string, _ service.UpdateAdminUserInput) (service.AdminUserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.AdminUserRecord{}, service.UnauthorizedError()
	}
	return s.adminUser(), nil
}

func (s fakeAuthService) ResetManagedUserPassword(_ context.Context, reqCtx service.RequestContext, _ string, _ service.ResetAdminPasswordInput) (service.AdminUserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.AdminUserRecord{}, service.UnauthorizedError()
	}
	return s.adminUser(), nil
}

func (s fakeAuthService) UpdateProfile(_ context.Context, reqCtx service.RequestContext, _ string, _ service.UpdateProfileInput) (service.UserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.UserRecord{}, service.UnauthorizedError()
	}
	return s.userRecord(), nil
}

func (s fakeAuthService) ChangeRequiredPassword(_ context.Context, reqCtx service.RequestContext, _ string, _ service.ChangePasswordInput) (service.UserRecord, error) {
	if reqCtx.CallerService == "" {
		return service.UserRecord{}, service.UnauthorizedError()
	}
	return s.userRecord(), nil
}

func (s fakeAuthService) GetSession(_ context.Context, reqCtx service.RequestContext, _ string) (service.SessionIdentity, error) {
	if reqCtx.CallerService == "" {
		return service.SessionIdentity{}, service.UnauthorizedError()
	}
	return service.SessionIdentity{
		Session: service.Session{
			ID:              "sess_123",
			UserID:          "usr_123",
			AccessTokenHash: "hmac-sha256:v1:secret",
			TokenType:       service.TokenTypeBearer,
			Status:          service.SessionStatusActive,
			IssuedAt:        s.now,
			ExpiresAt:       s.now.Add(time.Hour),
		},
		User: service.UserSummary{ID: "usr_123", Username: "alice", Roles: []string{"standard"}, Permissions: []string{"knowledge:read"}},
	}, nil
}

func (s fakeAuthService) RevokeSession(_ context.Context, reqCtx service.RequestContext, _ string, _ string) error {
	if reqCtx.CallerService == "" {
		return service.UnauthorizedError()
	}
	return nil
}

func (s fakeAuthService) userRecord() service.UserRecord {
	return service.UserRecord{
		User: service.User{
			ID:          "usr_123",
			Username:    "alice",
			DisplayName: "Alice",
			Status:      service.UserStatusActive,
			CreatedAt:   s.now,
			UpdatedAt:   s.now,
		},
		Roles:       []string{"standard"},
		Permissions: []string{"knowledge:read"},
	}
}

func (s fakeAuthService) adminUser() service.AdminUserRecord {
	return service.AdminUserRecord{
		UserRecord:      s.userRecord(),
		ManageableRoles: []string{"standard"},
		Actions: service.AdminUserActions{
			CanDisable:       true,
			CanResetPassword: true,
			CanChangeRole:    true,
		},
	}
}

func (s fakeAuthService) sessionResponse() service.SessionResponse {
	return service.SessionResponse{
		User: service.UserSummary{
			ID:          "usr_123",
			Username:    "alice",
			Roles:       []string{"standard"},
			Permissions: []string{"knowledge:read"},
		},
		Session: service.SessionSummary{
			SessionID:   "sess_123",
			AccessToken: "atk_v1_response",
			TokenType:   service.TokenTypeBearer,
			ExpiresAt:   s.now.Add(time.Hour),
		},
	}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, body = %s", err, string(body))
	}
}
