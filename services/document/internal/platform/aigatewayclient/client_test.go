package aigatewayclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// rewriteTransport redirects all requests to target, preserving the original
// path. This allows tests to pass a validated base URL (e.g. http://localhost:8086)
// to New() while still hitting the ephemeral httptest server port.
type rewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host
	return t.base.RoundTrip(cloned)
}

// newTestHTTPClient returns an *http.Client whose transport redirects every
// request to rawURL (the httptest server), keeping the original path intact.
func newTestHTTPClient(t *testing.T, rawURL string) *http.Client {
	t.Helper()
	target, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	return &http.Client{Transport: rewriteTransport{target: target, base: http.DefaultTransport}}
}

// newTestClient constructs a Client that talks to server through the rewrite
// transport, using a trusted base URL that passes strict validation.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c, err := New("http://localhost:8086", "profile-test", "model-test", newTestHTTPClient(t, server.URL))
	if err != nil {
		t.Fatalf("newTestClient: New() error = %v", err)
	}
	return c
}

// validChatResponse is a minimal AI Gateway response that satisfies all success checks.
func validChatResponse(content string) map[string]any {
	return map[string]any{
		"choices": []map[string]any{{
			"message": map[string]any{"role": "assistant", "content": content},
		}},
	}
}

// ── constructor tests ─────────────────────────────────────────────────────────

func TestNewClientRejectsInvalidURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		profileID string
		wantErr   bool
	}{
		// --- invalid base URLs ---
		{name: "empty string", baseURL: "", profileID: "p", wantErr: true},
		{name: "relative path", baseURL: "/api", profileID: "p", wantErr: true},
		{name: "ftp scheme", baseURL: "ftp://localhost:8086", profileID: "p", wantErr: true},
		{name: "empty host", baseURL: "http://", profileID: "p", wantErr: true},
		{name: "untrusted host", baseURL: "http://example.com", profileID: "p", wantErr: true},
		{name: "wrong port", baseURL: "http://localhost:9999", profileID: "p", wantErr: true},
		{name: "with credentials", baseURL: "http://user:pass@localhost:8086", profileID: "p", wantErr: true},
		{name: "with query string", baseURL: "http://localhost:8086?q=1", profileID: "p", wantErr: true},
		// --- invalid profileID ---
		{name: "empty profileID", baseURL: "http://localhost:8086", profileID: "", wantErr: true},
		{name: "non-loopback IP", baseURL: "http://192.168.1.1:8086", profileID: "p", wantErr: true},
		{name: "dirty path", baseURL: "http://localhost:8086/other/path", profileID: "p", wantErr: true},
		// --- valid inputs ---
		{name: "valid localhost with port", baseURL: "http://localhost:8086", profileID: "p", wantErr: false},
		{name: "valid ai-gateway with port", baseURL: "http://ai-gateway:8086", profileID: "p", wantErr: false},
		{name: "valid 127.0.0.1 with port", baseURL: "http://127.0.0.1:8086", profileID: "p", wantErr: false},
		{name: "valid ::1 with port", baseURL: "http://[::1]:8086", profileID: "p", wantErr: false},
		{name: "valid localhost no port", baseURL: "http://localhost", profileID: "p", wantErr: false},
		{name: "valid with /internal/v1 path", baseURL: "http://localhost:8086/internal/v1", profileID: "p", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.baseURL, tt.profileID, "", nil)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("New(%q, %q) error = nil, want error", tt.baseURL, tt.profileID)
				}
				if client != nil {
					t.Fatalf("New(%q, %q) client = non-nil, want nil on error", tt.baseURL, tt.profileID)
				}
			} else {
				if err != nil {
					t.Fatalf("New(%q, %q) error = %v, want nil", tt.baseURL, tt.profileID, err)
				}
				if client == nil {
					t.Fatal("New() client = nil, want non-nil")
				}
			}
		})
	}
}

// ── ChatCompletion tests ──────────────────────────────────────────────────────

func TestChatCompletionSendsCorrectHeaders(t *testing.T) {
	type captureT struct {
		headers   http.Header
		path      string
		bodyModel string
		bodyPID   string
	}
	var captured captureT

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.headers = r.Header.Clone()
		captured.path = r.URL.Path
		var body struct {
			Model     string `json:"model"`
			ProfileID string `json:"profile_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		captured.bodyModel = body.Model
		captured.bodyPID = body.ProfileID
		_ = json.NewEncoder(w).Encode(validChatResponse("ok"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	msgs := []service.ChatMessage{{Role: "user", Content: "hello"}}

	// With requestID and serviceToken both non-empty.
	captured = captureT{}
	if _, err := client.ChatCompletion(context.Background(), "req-123", "svc-token", msgs); err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if got := captured.headers.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := captured.headers.Get("X-Caller-Service"); got != callerService {
		t.Fatalf("X-Caller-Service = %q, want %q", got, callerService)
	}
	if got := captured.headers.Get("X-Request-Id"); got != "req-123" {
		t.Fatalf("X-Request-Id = %q, want req-123", got)
	}
	if got := captured.headers.Get("X-Service-Token"); got != "svc-token" {
		t.Fatalf("X-Service-Token = %q, want svc-token", got)
	}
	if captured.path != "/internal/v1/chat/completions" {
		t.Fatalf("request path = %q, want /internal/v1/chat/completions", captured.path)
	}
	if captured.bodyModel != "model-test" {
		t.Fatalf("request body model = %q, want model-test", captured.bodyModel)
	}
	if captured.bodyPID != "profile-test" {
		t.Fatalf("request body profile_id = %q, want profile-test", captured.bodyPID)
	}

	// With both empty: X-Request-Id and X-Service-Token must be absent.
	captured = captureT{}
	if _, err := client.ChatCompletion(context.Background(), "", "", msgs); err != nil {
		t.Fatalf("ChatCompletion() (empty ids) error = %v", err)
	}
	if got := captured.headers.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := captured.headers.Get("X-Caller-Service"); got != callerService {
		t.Fatalf("X-Caller-Service = %q, want %q", got, callerService)
	}
	if got := captured.headers.Get("X-Request-Id"); got != "" {
		t.Fatalf("X-Request-Id = %q, want empty when requestID is empty", got)
	}
	if got := captured.headers.Get("X-Service-Token"); got != "" {
		t.Fatalf("X-Service-Token = %q, want empty when serviceToken is empty", got)
	}
}

func TestChatCompletionMapsNonSuccessStatusToDependencyError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "bad request", status: 400, body: `{"error":"bad request with sk-secret"}`},
		{name: "unauthorized", status: 401, body: `{"error":"unauthorized provider.internal/key"}`},
		{name: "forbidden", status: 403, body: `{"error":"forbidden"}`},
		{name: "internal server error", status: 500, body: `{"error":"server error secret-data"}`},
		{name: "service unavailable", status: 503, body: `{"error":"unavailable"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server)
			_, err := client.ChatCompletion(context.Background(), "req-1", "", []service.ChatMessage{
				{Role: "user", Content: "hello"},
			})
			if err == nil {
				t.Fatalf("ChatCompletion() status=%d: error = nil, want error", tt.status)
			}
			// Body must not appear in the error message.
			if strings.Contains(err.Error(), "sk-secret") ||
				strings.Contains(err.Error(), "provider.internal") ||
				strings.Contains(err.Error(), "secret-data") {
				t.Fatalf("error message leaked downstream body content: %v", err)
			}
			appErr, ok := service.Classify(err)
			if !ok || appErr.Code != service.CodeDependency {
				t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
			}
		})
	}
}

func TestChatCompletionMapsInvalidJSONToDependencyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for invalid JSON")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}

func TestChatCompletionMapsEmptyChoicesToDependencyError(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty choices array",
			body: `{"choices":[]}`,
		},
		{
			name: "whitespace-only content",
			body: `{"choices":[{"message":{"role":"assistant","content":"   "}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server)
			_, err := client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{
				{Role: "user", Content: "hello"},
			})
			if err == nil {
				t.Fatalf("ChatCompletion() %s: error = nil, want error", tt.name)
			}
			appErr, ok := service.Classify(err)
			if !ok || appErr.Code != service.CodeDependency {
				t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
			}
		})
	}
}

func TestChatCompletionReturnsContentOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"role": "assistant", "content": "result"},
			}},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	content, err := client.ChatCompletion(context.Background(), "req-1", "token", []service.ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if content != "result" {
		t.Fatalf("ChatCompletion() content = %q, want %q", content, "result")
	}
}

func TestChatCompletionHandlesNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler never reached — context is pre-cancelled.
	}))
	defer server.Close()

	client := newTestClient(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so httpClient.Do fails immediately

	_, err := client.ChatCompletion(ctx, "", "", []service.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for cancelled context")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}

func TestChatCompletionHandlesOversizeResponse(t *testing.T) {
	// Response exceeds maxResponseBytes (2<<20 = 2 MiB).
	oversized := strings.Repeat("x", (2<<20)+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oversized))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.ChatCompletion(context.Background(), "", "", []service.ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Fatal("ChatCompletion() error = nil, want error for oversized response")
	}
	appErr, ok := service.Classify(err)
	if !ok || appErr.Code != service.CodeDependency {
		t.Fatalf("error code = %v, want %q", err, service.CodeDependency)
	}
}
