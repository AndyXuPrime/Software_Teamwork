package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const authGatewayRedisFullSmokeGate = "AUTH_GATEWAY_REDIS_FULL_SMOKE"

func TestAuthGatewayRedisFullSmoke(t *testing.T) {
	if os.Getenv(authGatewayRedisFullSmokeGate) != "1" {
		t.Skip("set AUTH_GATEWAY_REDIS_FULL_SMOKE=1 to run the full Auth/Gateway/Redis smoke")
	}

	cfg := loadAuthGatewayRedisFullSmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	assertTCPReachable(t, ctx, "postgres", databaseHostPort(cfg.databaseURL))
	assertTCPReachable(t, ctx, "redis", cfg.redisAddr)
	applyAuthMigrations(t, ctx, cfg.databaseURL)

	fakeOwner := newFakeOwnerService(t)
	authURL := startAuthServer(t, ctx, cfg)
	gatewayURL := startGatewayServer(t, ctx, cfg, authURL, fakeOwner.URL)

	requestID := "req_auth_gateway_redis_full_smoke_" + shortID(newSmokeRunID())
	client := smokeHTTPClient()

	assertCurrentUserRejectsMissingAuth(t, ctx, client, authGatewayRedisSmokeConfig{
		gatewayBaseURL: gatewayURL,
		redisAddr:      cfg.redisAddr,
		redisPassword:  cfg.redisPassword,
		redisDB:        cfg.redisDB,
	}, requestID+"_missing")
	assertGatewayIgnoresSpoofedUserHeaders(t, ctx, client, authGatewayRedisSmokeConfig{
		gatewayBaseURL: gatewayURL,
		redisAddr:      cfg.redisAddr,
		redisPassword:  cfg.redisPassword,
		redisDB:        cfg.redisDB,
	}, requestID+"_spoof")

	cacheCfg := authGatewayRedisSmokeConfig{
		gatewayBaseURL: gatewayURL,
		redisAddr:      cfg.redisAddr,
		redisPassword:  cfg.redisPassword,
		redisDB:        cfg.redisDB,
	}

	createdUser := createSmokeUser(t, ctx, client, gatewayURL, requestID+"_create")
	createCacheKey := assertRedisSessionCacheSafe(t, ctx, cacheCfg, requestID+"_create", createdUser.Session.AccessToken)
	assertLogoutInvalidatesSession(t, ctx, client, cacheCfg, createdUser.Session, requestID+"_create_logout")
	assertRedisSessionDeleted(t, ctx, cacheCfg, createCacheKey)

	session := createSmokeSession(t, ctx, client, gatewayURL, createdUser.Username, createdUser.Password, requestID+"_login")
	if session.UserID != createdUser.Session.UserID {
		t.Fatalf("login returned a different user id: created=%q login=%q", createdUser.Session.UserID, session.UserID)
	}
	cacheKey := assertRedisSessionCacheSafe(t, ctx, cacheCfg, requestID+"_login", session.AccessToken)
	assertCurrentUserViaGateway(t, ctx, client, cacheCfg, session, requestID+"_me")
	assertFakeOwnerReceivesGatewayContext(t, ctx, client, gatewayURL, session, fakeOwner, cfg.serviceToken, requestID+"_owner")
	assertLogoutInvalidatesSession(t, ctx, client, cacheCfg, session, requestID+"_logout")
	assertRedisSessionDeleted(t, ctx, cacheCfg, cacheKey)

	t.Logf("AUTH_GATEWAY_REDIS_FULL_SMOKE_RESULT pass gateway=%s auth=%s fakeOwner=%s redis=%s", gatewayURL, authURL, fakeOwner.URL, cfg.redisAddr)
}

type authGatewayRedisFullSmokeConfig struct {
	databaseURL       string
	redisAddr         string
	redisPassword     string
	redisDB           int
	serviceToken      string
	adminServiceToken string
	tokenHashSecret   string
}

func loadAuthGatewayRedisFullSmokeConfig(t *testing.T) authGatewayRedisFullSmokeConfig {
	t.Helper()
	databaseURL := firstNonEmptyEnv(
		"AUTH_GATEWAY_REDIS_DATABASE_URL",
		"AUTH_DATABASE_URL",
		"DATABASE_URL",
	)
	if strings.TrimSpace(databaseURL) == "" {
		t.Fatalf("blocked: AUTH_GATEWAY_REDIS_FULL_SMOKE requires AUTH_GATEWAY_REDIS_DATABASE_URL, AUTH_DATABASE_URL, or DATABASE_URL")
	}
	redisDB := 0
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_REDIS_DB")); raw != "" {
		value, err := strconvAtoiNonNegative(raw)
		if err != nil {
			t.Fatalf("GATEWAY_REDIS_DB must be a non-negative integer")
		}
		redisDB = value
	}
	return authGatewayRedisFullSmokeConfig{
		databaseURL:       strings.TrimSpace(databaseURL),
		redisAddr:         firstNonEmptyEnvOr("127.0.0.1:6379", "AUTH_GATEWAY_REDIS_ADDR", "GATEWAY_REDIS_ADDR", "REDIS_ADDR"),
		redisPassword:     os.Getenv("GATEWAY_REDIS_PASSWORD"),
		redisDB:           redisDB,
		serviceToken:      firstNonEmptyEnvOr("local-dev-internal-service-token-change-me", "AUTH_GATEWAY_REDIS_SERVICE_TOKEN", "INTERNAL_SERVICE_TOKEN"),
		adminServiceToken: firstNonEmptyEnvOr("local-dev-gateway-admin-token-change-me", "AUTH_GATEWAY_REDIS_ADMIN_SERVICE_TOKEN", "AUTH_GATEWAY_ADMIN_SERVICE_TOKEN"),
		tokenHashSecret:   firstNonEmptyEnvOr("local-demo-token-hash-secret-change-me", "AUTH_GATEWAY_REDIS_TOKEN_HASH_SECRET", "TOKEN_HASH_SECRET"),
	}
}

func strconvAtoiNonNegative(raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return value, nil
}

type fakeOwnerService struct {
	URL      string
	close    func()
	mu       sync.Mutex
	requests []fakeOwnerRequest
}

type fakeOwnerRequest struct {
	Path   string
	Header http.Header
}

type smokeCreatedUser struct {
	Username string
	Password string
	Session  smokeSession
}

func newFakeOwnerService(t *testing.T) *fakeOwnerService {
	t.Helper()
	fake := &fakeOwnerService{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz", "/readyz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"status":"ready","service":"fake-owner"},"requestId":"req_fake_owner_ready"}`))
			return
		case "/internal/v1/knowledge-bases":
			fake.mu.Lock()
			fake.requests = append(fake.requests, fakeOwnerRequest{
				Path:   r.URL.Path,
				Header: r.Header.Clone(),
			})
			fake.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[],"page":{"page":1,"pageSize":20,"total":0},"requestId":"req_fake_owner"}`))
			return
		default:
			http.NotFound(w, r)
		}
	}))
	fake.URL = server.URL
	fake.close = server.Close
	t.Cleanup(fake.close)
	return fake
}

func (f *fakeOwnerService) lastRequest() (fakeOwnerRequest, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return fakeOwnerRequest{}, false
	}
	return f.requests[len(f.requests)-1], true
}

func applyAuthMigrations(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()
	authDir := repoPath(t, "services", "auth")
	cmd := exec.CommandContext(ctx, "go", "run", "github.com/pressly/goose/v3/cmd/goose@v3.27.0", "-dir", "migrations", "postgres", databaseURL, "up")
	cmd.Dir = authDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("blocked: auth migration apply failed: %v\n%s", err, output)
	}
	t.Logf("auth migrations applied")
}

func startAuthServer(t *testing.T, ctx context.Context, cfg authGatewayRedisFullSmokeConfig) string {
	t.Helper()
	addr := freeLocalAddr(t)
	binary := buildServiceServer(t, ctx, "auth")
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = repoPath(t, "services", "auth")
	cmd.Env = append(os.Environ(),
		"AUTH_HTTP_ADDR="+addr,
		"AUTH_DATABASE_URL="+cfg.databaseURL,
		"AUTH_INTERNAL_SERVICE_TOKEN="+cfg.serviceToken,
		"AUTH_GATEWAY_ADMIN_SERVICE_TOKEN="+cfg.adminServiceToken,
		"AUTH_TOKEN_HASH_SECRET="+cfg.tokenHashSecret,
		"AUTH_ENV=smoke",
	)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("blocked: start auth service: %v", err)
	}
	t.Cleanup(func() {
		stopProcess(cmd)
		if t.Failed() {
			t.Logf("auth logs:\n%s", logs.String())
		}
	})
	baseURL := "http://" + addr
	waitForHTTPStatus(t, ctx, "auth", baseURL+"/readyz", http.StatusOK)
	return baseURL
}

func startGatewayServer(t *testing.T, ctx context.Context, cfg authGatewayRedisFullSmokeConfig, authURL, ownerURL string) string {
	t.Helper()
	addr := freeLocalAddr(t)
	metricsAddr := freeLocalAddr(t)
	binary := buildServiceServer(t, ctx, "gateway")
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = repoPath(t, "services", "gateway")
	cmd.Env = append(os.Environ(),
		"GATEWAY_HTTP_ADDR="+addr,
		"GATEWAY_METRICS_ADDR="+metricsAddr,
		"GATEWAY_REDIS_ADDR="+cfg.redisAddr,
		"GATEWAY_REDIS_PASSWORD="+cfg.redisPassword,
		"GATEWAY_REDIS_DB="+fmt.Sprintf("%d", cfg.redisDB),
		"GATEWAY_AUTH_BASE_URL="+authURL,
		"GATEWAY_KNOWLEDGE_BASE_URL="+ownerURL,
		"GATEWAY_QA_BASE_URL="+ownerURL,
		"GATEWAY_DOCUMENT_BASE_URL="+ownerURL,
		"GATEWAY_AI_GATEWAY_BASE_URL="+ownerURL,
		"GATEWAY_INTERNAL_SERVICE_TOKEN="+cfg.serviceToken,
		"GATEWAY_AUTH_ADMIN_SERVICE_TOKEN="+cfg.adminServiceToken,
		"GATEWAY_TOKEN_HASH_SECRET="+cfg.tokenHashSecret,
		"GATEWAY_ENV=smoke",
	)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("blocked: start gateway service: %v", err)
	}
	t.Cleanup(func() {
		stopProcess(cmd)
		if t.Failed() {
			t.Logf("gateway logs:\n%s", logs.String())
		}
	})
	baseURL := "http://" + addr
	waitForHTTPStatus(t, ctx, "gateway", baseURL+"/readyz", http.StatusOK)
	return baseURL
}

func buildServiceServer(t *testing.T, ctx context.Context, service string) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), service+"-server")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", output, "./cmd/server")
	cmd.Dir = repoPath(t, "services", service)
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("blocked: build %s server failed: %v\n%s", service, err, data)
	}
	return output
}

func createSmokeUser(t *testing.T, ctx context.Context, client *http.Client, gatewayBaseURL, requestID string) smokeCreatedUser {
	t.Helper()
	runID := strings.ToLower(strings.ReplaceAll(shortID(newSmokeRunID()), "_", ""))
	username := "smoke_user_" + runID
	password := "SmokePass#" + runID + "123"
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL+"/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		t.Fatalf("create user returned %d: %s", resp.StatusCode, responseBodySummary(data))
	}
	var envelope struct {
		Data struct {
			Session struct {
				AccessToken string `json:"accessToken"`
			} `json:"session"`
			User struct {
				ID          string   `json:"id"`
				Username    string   `json:"username"`
				Roles       []string `json:"roles"`
				Permissions []string `json:"permissions"`
			} `json:"user"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode create user response: %v", err)
	}
	if envelope.RequestID != requestID {
		t.Fatalf("create user requestId mismatch: want=%q got=%q", requestID, envelope.RequestID)
	}
	if envelope.Data.User.Username != username || envelope.Data.User.ID == "" || envelope.Data.Session.AccessToken == "" {
		t.Fatalf("create user response missing identity: %+v", envelope.Data)
	}
	if stringSliceContains(envelope.Data.User.Roles, "admin") || stringSliceContains(envelope.Data.User.Permissions, "system:admin") {
		t.Fatalf("new smoke user unexpectedly has admin authority: roles=%v permissions=%v", envelope.Data.User.Roles, envelope.Data.User.Permissions)
	}
	return smokeCreatedUser{
		Username: username,
		Password: password,
		Session:  smokeSession{AccessToken: envelope.Data.Session.AccessToken, UserID: envelope.Data.User.ID},
	}
}

func assertFakeOwnerReceivesGatewayContext(t *testing.T, ctx context.Context, client *http.Client, gatewayBaseURL string, session smokeSession, fakeOwner *fakeOwnerService, serviceToken, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, gatewayBaseURL+"/api/v1/knowledge-bases?page=1&pageSize=20", session.AccessToken, requestID, nil)
	req.Header.Set("X-User-Id", "spoofed-user")
	req.Header.Set("X-User-Roles", "admin")
	req.Header.Set("X-User-Permissions", "system:admin")
	req.Header.Set("X-Service-Token", "spoofed-service-token")
	req.Header.Set("X-Caller-Service", "spoofed-caller")
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("fake owner gateway request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fake owner gateway request returned %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	captured, ok := fakeOwner.lastRequest()
	if !ok {
		t.Fatal("fake owner did not capture a proxied request")
	}
	if captured.Path != "/internal/v1/knowledge-bases" {
		t.Fatalf("fake owner path mismatch: %q", captured.Path)
	}
	if captured.Header.Get("X-Caller-Service") != "gateway" {
		t.Fatalf("X-Caller-Service was not injected by gateway: %#v", captured.Header)
	}
	if captured.Header.Get("X-Service-Token") != serviceToken {
		t.Fatalf("X-Service-Token mismatch: got %q", captured.Header.Get("X-Service-Token"))
	}
	if captured.Header.Get("X-User-Id") != session.UserID {
		t.Fatalf("X-User-Id mismatch: want=%q got=%q", session.UserID, captured.Header.Get("X-User-Id"))
	}
	if strings.Contains(captured.Header.Get("X-User-Roles"), "admin") {
		t.Fatalf("spoofed role leaked to owner: %q", captured.Header.Get("X-User-Roles"))
	}
	if strings.Contains(captured.Header.Get("X-User-Permissions"), "system:admin") {
		t.Fatalf("spoofed permission leaked to owner: %q", captured.Header.Get("X-User-Permissions"))
	}
	if captured.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization leaked to owner: %q", captured.Header.Get("Authorization"))
	}
}

func assertTCPReachable(t *testing.T, ctx context.Context, name, addr string) {
	t.Helper()
	addr = strings.TrimSpace(addr)
	if addr == "" {
		t.Fatalf("blocked: %s address is not configured", name)
	}
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Fatalf("blocked: %s is not reachable at %s: %v", name, addr, err)
	}
	_ = conn.Close()
}

func waitForHTTPStatus(t *testing.T, ctx context.Context, name, url string, want int) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := smokeHTTPClient().Do(req)
		if err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			if resp.StatusCode == want {
				return
			}
			last = fmt.Sprintf("status=%d body=%s", resp.StatusCode, responseBodySummary(body))
		} else {
			last = err.Error()
		}
		select {
		case <-ctx.Done():
			t.Fatalf("blocked: %s did not become ready before context deadline: %s", name, last)
		case <-time.After(500 * time.Millisecond):
		}
	}
	t.Fatalf("blocked: %s did not become ready at %s: %s", name, url, last)
}

func freeLocalAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free local port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		<-done
	}
}

func databaseHostPort(databaseURL string) string {
	trimmed := strings.TrimSpace(databaseURL)
	if trimmed == "" {
		return ""
	}
	withoutScheme := trimmed
	if idx := strings.Index(withoutScheme, "://"); idx >= 0 {
		withoutScheme = withoutScheme[idx+3:]
	}
	if at := strings.LastIndex(withoutScheme, "@"); at >= 0 {
		withoutScheme = withoutScheme[at+1:]
	}
	hostPort := withoutScheme
	if slash := strings.IndexAny(hostPort, "/?"); slash >= 0 {
		hostPort = hostPort[:slash]
	}
	if !strings.Contains(hostPort, ":") {
		hostPort += ":5432"
	}
	return hostPort
}

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()
	segments := append([]string{"..", "..", ".."}, parts...)
	path, err := filepath.Abs(filepath.Join(segments...))
	if err != nil {
		t.Fatalf("resolve repo path: %v", err)
	}
	return path
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
