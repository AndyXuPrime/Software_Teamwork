package authclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAcceptsConfiguredAuthBaseURLs(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantNil bool
	}{
		{name: "empty", baseURL: "", wantNil: true},
		{name: "http service name", baseURL: "http://auth:8001"},
		{name: "http localhost", baseURL: "http://localhost:8001"},
		{name: "https internal dns", baseURL: "https://auth.example.internal"},
		{name: "trimmed", baseURL: "  http://auth:8001  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.baseURL, "svc-token", "gateway-token", time.Second)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if tt.wantNil {
				if client != nil {
					t.Fatal("New() client is non-nil")
				}
				return
			}
			if client == nil {
				t.Fatal("New() client is nil")
			}
		})
	}
}

func TestNewRejectsUnsafeAuthBaseURLs(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "missing scheme", baseURL: "auth:8001"},
		{name: "relative path", baseURL: "/internal"},
		{name: "unsupported scheme", baseURL: "ftp://auth:8001"},
		{name: "credentials", baseURL: "http://user:pass@auth:8001"},
		{name: "query", baseURL: "http://auth:8001?token=secret"},
		{name: "fragment", baseURL: "http://auth:8001#readyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if client, err := New(tt.baseURL, "svc-token", "gateway-token", time.Second); err == nil {
				t.Fatalf("New() error = nil, client = %#v", client)
			}
		})
	}
}

func TestCreateSessionSendsGatewayForwardingContext(t *testing.T) {
	var forwardedFor string
	var forwardedProto string
	var serviceToken string
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/sessions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		forwardedFor = r.Header.Get("X-Forwarded-For")
		forwardedProto = r.Header.Get("X-Forwarded-Proto")
		serviceToken = r.Header.Get("X-Service-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"user":{"id":"usr_1","username":"alice","roles":[],"permissions":[]},"session":{"sessionId":"sess_1","accessToken":"tok_1","tokenType":"Bearer","expiresAt":"2026-06-29T10:00:00Z"}},"requestId":"req_1"}`))
	}))
	defer auth.Close()

	client, err := New(auth.URL, "svc-token", "gateway-token", time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.CreateSession(context.Background(), "req_1", []byte(`{"username":"alice","password":"secret"}`), ForwardingContext{
		ForwardedFor:   "198.51.100.10",
		ForwardedProto: "https",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if forwardedFor != "198.51.100.10" || forwardedProto != "https" {
		t.Fatalf("forwarding headers = for:%q proto:%q", forwardedFor, forwardedProto)
	}
	if serviceToken != "svc-token" {
		t.Fatalf("X-Service-Token = %q", serviceToken)
	}
}

func TestSelfWriteRoutesUseGatewayServiceToken(t *testing.T) {
	captured := map[string]string{}
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured[r.Method+" "+r.URL.Path] = r.Header.Get("X-Service-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"usr_1","username":"alice","status":"active","roles":[],"permissions":[]},"requestId":"req_1"}`))
	}))
	defer auth.Close()

	client, err := New(auth.URL, "svc-token", "gateway-token", time.Second)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.UpdateUserProfile(context.Background(), "req_1", "usr_1", []byte(`{"displayName":"Alice"}`), ForwardingContext{UserID: "usr_1"}); err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if _, err := client.ChangeUserPassword(context.Background(), "req_2", "usr_1", []byte(`{"currentPassword":"temporary","newPassword":"new-password","newPasswordConfirmation":"new-password"}`), ForwardingContext{UserID: "usr_1"}); err != nil {
		t.Fatalf("ChangeUserPassword() error = %v", err)
	}

	if got := captured["PATCH /internal/v1/users/usr_1/profile"]; got != "gateway-token" {
		t.Fatalf("profile X-Service-Token = %q", got)
	}
	if got := captured["POST /internal/v1/users/usr_1/password-changes"]; got != "gateway-token" {
		t.Fatalf("password-change X-Service-Token = %q", got)
	}
}
