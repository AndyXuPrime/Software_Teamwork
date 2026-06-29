package authclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateSessionSendsGatewayForwardingContext(t *testing.T) {
	var forwardedFor string
	var forwardedProto string
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/sessions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		forwardedFor = r.Header.Get("X-Forwarded-For")
		forwardedProto = r.Header.Get("X-Forwarded-Proto")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"user":{"id":"usr_1","username":"alice","roles":[],"permissions":[]},"session":{"sessionId":"sess_1","accessToken":"tok_1","tokenType":"Bearer","expiresAt":"2026-06-29T10:00:00Z"}},"requestId":"req_1"}`))
	}))
	defer auth.Close()

	client, err := New(auth.URL, "svc-token", time.Second)
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
}
