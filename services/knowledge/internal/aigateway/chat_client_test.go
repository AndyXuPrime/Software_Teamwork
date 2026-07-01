package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatClientCreatesCompletionWithInternalHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/internal/v1/chat/completions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Service-Token") != "service-token" {
			t.Fatalf("X-Service-Token = %q", r.Header.Get("X-Service-Token"))
		}
		if r.Header.Get("X-Caller-Service") != "knowledge" {
			t.Fatalf("X-Caller-Service = %q", r.Header.Get("X-Caller-Service"))
		}
		if r.Header.Get("X-Request-Id") != "req-chat" {
			t.Fatalf("X-Request-Id = %q", r.Header.Get("X-Request-Id"))
		}
		if r.Header.Get("X-User-Id") != "user-chat" {
			t.Fatalf("X-User-Id = %q", r.Header.Get("X-User-Id"))
		}
		var body struct {
			Model     string    `json:"model"`
			ProfileID string    `json:"profile_id"`
			Messages  []Message `json:"messages"`
			Stream    bool      `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Model != "profile-test" || body.ProfileID != "profile-test" || body.Stream {
			t.Fatalf("request body = %+v", body)
		}
		if len(body.Messages) != 2 || body.Messages[1].Role != "user" {
			t.Fatalf("messages = %+v", body.Messages)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": "The answer is 42.",
				},
				"finish_reason": "stop",
			}},
		})
	}))
	defer server.Close()

	client, err := NewChatClient(server.URL, "service-token", server.Client())
	if err != nil {
		t.Fatalf("NewChatClient() error = %v", err)
	}
	resp, err := client.CreateChatCompletion(context.Background(), RequestContext{
		RequestID: "req-chat",
		UserID:    "user-chat",
	}, ChatRequest{
		ProfileID: "profile-test",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "What is the answer?"},
		},
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion() error = %v", err)
	}
	if resp.Content != "The answer is 42." || resp.FinishReason != "stop" {
		t.Fatalf("completion response = %+v", resp)
	}
}

func TestChatClientSanitizesDownstreamError(t *testing.T) {
	rawBody := `{"error":{"message":"provider failed with sk-secret","type":"upstream_error"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(rawBody))
	}))
	defer server.Close()

	client, err := NewChatClient(server.URL, "service-token", server.Client())
	if err != nil {
		t.Fatalf("NewChatClient() error = %v", err)
	}
	_, err = client.CreateChatCompletion(context.Background(), RequestContext{}, ChatRequest{
		ProfileID: "profile-test",
		Messages:  []Message{{Role: "user", Content: "prompt must stay local"}},
	})
	if err == nil {
		t.Fatal("CreateChatCompletion() error = nil, want error")
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "prompt must stay local") {
		t.Fatalf("error leaked sensitive data: %v", err)
	}
}

func TestNewChatClientFromEnvReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("KNOWLEDGE_AI_GATEWAY_URL", "")
	client, err := NewChatClientFromEnv()
	if err != nil {
		t.Fatalf("NewChatClientFromEnv() error = %v", err)
	}
	if client != nil {
		t.Fatal("expected nil client when URL unset")
	}
}
