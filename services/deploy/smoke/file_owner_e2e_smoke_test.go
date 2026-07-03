package smoke_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

const fileOwnerE2ESmokeGate = "FILE_OWNER_E2E_SMOKE"

func TestFileOwnerE2ESmoke(t *testing.T) {
	if os.Getenv(fileOwnerE2ESmokeGate) != "1" {
		t.Skip("set FILE_OWNER_E2E_SMOKE=1 to run the File owner-service E2E smoke")
	}

	cfg := loadFileOwnerSmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	assertHTTPReady(t, ctx, "file", cfg.fileBaseURL)
	assertHTTPReady(t, ctx, "document", cfg.documentBaseURL)
	assertHTTPReady(t, ctx, "gateway", cfg.gatewayBaseURL)

	requestID := "req_file_owner_e2e_smoke_" + shortID(newSmokeRunID())
	client := smokeHTTPClient()

	t.Run("spoofed_auth_rejected", func(t *testing.T) {
		assertGatewayRejectsSpoofedAuth(t, ctx, client, cfg, requestID+"_spoof")
	})
	t.Run("file_internal_requires_service_token", func(t *testing.T) {
		assertFileInternalRequiresServiceToken(t, ctx, client, cfg, requestID+"_file_token")
	})
	t.Run("document_read_via_gateway", func(t *testing.T) {
		session := createSmokeSession(t, ctx, client, cfg.gatewayBaseURL, cfg.username, cfg.password, requestID)
		assertDocumentReportReadViaGateway(t, ctx, client, cfg, session, requestID)
	})
}

type fileOwnerSmokeConfig struct {
	gatewayBaseURL  string
	documentBaseURL string
	fileBaseURL     string
	username        string
	password        string
}

func loadFileOwnerSmokeConfig(t *testing.T) fileOwnerSmokeConfig {
	t.Helper()
	required := map[string]string{
		"GATEWAY_BASE_URL":                               os.Getenv("GATEWAY_BASE_URL"),
		"DOCUMENT_SERVICE_BASE_URL":                      os.Getenv("DOCUMENT_SERVICE_BASE_URL"),
		"FILE_SERVICE_BASE_URL":                          os.Getenv("FILE_SERVICE_BASE_URL"),
		"GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME": firstNonEmptyEnv("GATEWAY_SMOKE_USERNAME", "LOCAL_ADMIN_USERNAME"),
		"GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD": firstNonEmptyEnv("GATEWAY_SMOKE_PASSWORD", "LOCAL_ADMIN_PASSWORD"),
	}
	missing := []string{}
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing required environment variables:\n - %s", strings.Join(missing, "\n - "))
	}
	return fileOwnerSmokeConfig{
		gatewayBaseURL:  trimBaseURL(t, "GATEWAY_BASE_URL", required["GATEWAY_BASE_URL"]),
		documentBaseURL: trimBaseURL(t, "DOCUMENT_SERVICE_BASE_URL", required["DOCUMENT_SERVICE_BASE_URL"]),
		fileBaseURL:     trimBaseURL(t, "FILE_SERVICE_BASE_URL", required["FILE_SERVICE_BASE_URL"]),
		username:        strings.TrimSpace(required["GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME"]),
		password:        strings.TrimSpace(required["GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD"]),
	}
}

// ---- shared helpers (kept locally per smoke convention) ----

func assertHTTPReady(t *testing.T, ctx context.Context, name, baseURL string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/readyz", nil)
	resp, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("%s not reachable at %s: %v", name, baseURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("%s /readyz returned %d", name, resp.StatusCode)
	}
}

func smokeHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func trimBaseURL(t *testing.T, key, raw string) string {
	t.Helper()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("%s is not a valid URL: %v", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		t.Fatalf("%s must be absolute http(s): %s", key, raw)
	}
	if parsed.User != nil {
		t.Fatalf("%s must not contain credentials", key)
	}
	return strings.TrimRight(raw, "/")
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func newSmokeRunID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102_150405")
	}
	return time.Now().Format("20060102_150405") + "_" + hex.EncodeToString(b)
}

func shortID(runID string) string {
	if len(runID) > 20 {
		return runID[len(runID)-12:]
	}
	return runID
}

// ---- auth helpers ----

type smokeSession struct{ AccessToken, UserID string }

func createSmokeSession(t *testing.T, ctx context.Context, client *http.Client, gatewayBaseURL, username, password, requestID string) smokeSession {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL+"/api/v1/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		t.Fatalf("create session returned %d: %s", resp.StatusCode, responseBodySummary(data))
	}
	var envelope struct {
		Data struct {
			Session struct {
				AccessToken string `json:"accessToken"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if envelope.RequestID != requestID {
		t.Fatalf("expected requestId=%q got %q", requestID, envelope.RequestID)
	}
	return smokeSession{AccessToken: envelope.Data.Session.AccessToken, UserID: envelope.Data.User.ID}
}

func gatewayAuthRequest(method, url, accessToken, requestID string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID)
	return req
}

func assertResponseEnvelope(t *testing.T, body []byte, requestID string) {
	t.Helper()
	var envelope struct {
		Data      json.RawMessage `json:"data"`
		RequestID string          `json:"requestId"`
		Error     *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode response envelope: %v %s", err, responseBodySummary(body))
	}
	if envelope.RequestID != requestID {
		t.Fatalf("expected requestId=%q got %q", requestID, envelope.RequestID)
	}
}

func responseBodySummary(body []byte) string {
	var envelope struct {
		RequestID string `json:"requestId"`
		Error     struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Error.Code != "" || envelope.Error.RequestID != "" {
			return fmt.Sprintf("body_bytes=%d error.code=%q error.requestId=%q", len(body), envelope.Error.Code, envelope.Error.RequestID)
		}
		if envelope.RequestID != "" {
			return fmt.Sprintf("body_bytes=%d requestId=%q", len(body), envelope.RequestID)
		}
	}
	return fmt.Sprintf("body_bytes=%d", len(body))
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertNoLeakedInternals(t *testing.T, body []byte) {
	t.Helper()
	lower := strings.ToLower(string(body))
	for _, forbidden := range []string{
		"objectkey", "storagekey", "bucket", "minio",
		"internalurl", "file_ref", "fileinternalid",
		"service_token", "api_key", "apikey",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("response leaked %q", forbidden)
		}
	}
}

// ---- test cases ----

func assertGatewayRejectsSpoofedAuth(t *testing.T, ctx context.Context, client *http.Client, cfg fileOwnerSmokeConfig, requestID string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.gatewayBaseURL+"/api/v1/knowledge-bases?page=1&pageSize=1", nil)
	req.Header.Set("X-User-Id", "spoofed-user-must-not-authenticate")
	req.Header.Set("X-User-Roles", "admin")
	req.Header.Set("X-User-Permissions", "knowledge:read,knowledge:write")
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("spoofed request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for spoofed request, got %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertNoLeakedInternals(t, body)
}

func assertFileInternalRequiresServiceToken(t *testing.T, ctx context.Context, client *http.Client, cfg fileOwnerSmokeConfig, requestID string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.fileBaseURL+"/internal/v1/files/smoke-missing-token/content", nil)
	req.Header.Set("X-Caller-Service", "knowledge")
	req.Header.Set("X-User-Id", "smoke-user")
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("file internal missing-token request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 from File internal API without service token, got %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertErrorEnvelope(t, body, requestID, "unauthorized")
	assertNoLeakedInternals(t, body)
}

func assertDocumentReportReadViaGateway(t *testing.T, ctx context.Context, client *http.Client, cfg fileOwnerSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	// List report types through Gateway -> Document
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/report-types", session.AccessToken, requestID, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list report types: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list report types returned %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertNoLeakedInternals(t, body)
	assertResponseEnvelope(t, body, requestID)

	// Read a known seed report through Gateway
	reportReq := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/reports/22222222-2222-4222-8222-222222222301", session.AccessToken, requestID+"_report", nil)
	reportResp, err := client.Do(reportReq)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	defer reportResp.Body.Close()
	reportBody, _ := io.ReadAll(io.LimitReader(reportResp.Body, 65536))
	if reportResp.StatusCode != http.StatusOK {
		t.Fatalf("get report returned %d: %s", reportResp.StatusCode, responseBodySummary(reportBody))
	}
	assertNoLeakedInternals(t, reportBody)
	assertResponseEnvelope(t, reportBody, requestID+"_report")
}
