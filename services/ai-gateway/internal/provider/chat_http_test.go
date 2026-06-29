package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

func TestHTTPChatClientCompleteChatSuccess(t *testing.T) {
	var providerBody map[string]any
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&providerBody); err != nil {
			t.Fatalf("decode provider body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_1","object":"chat.completion","created":1,"model":"provider-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer fake.Close()

	client := NewHTTPChatClient(fake.Client())
	result, err := client.CompleteChat(context.Background(), providerRequest(fake.URL))
	if err != nil {
		t.Fatalf("CompleteChat() error = %v", err)
	}
	if result.ProviderStatusCode != http.StatusOK {
		t.Fatalf("ProviderStatusCode = %d", result.ProviderStatusCode)
	}
	if result.Usage == nil || result.Usage.TotalTokens != 3 {
		t.Fatalf("Usage = %+v", result.Usage)
	}
	if providerBody["model"] != "provider-model" {
		t.Fatalf("provider model = %v", providerBody["model"])
	}
}

func TestHTTPChatClientProviderErrorsAreNormalizedAndSecretSafe(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		wantHTTP int
		wantType string
		wantCode string
		rawBody  string
	}{
		{"unauthorized", http.StatusUnauthorized, http.StatusBadGateway, "authentication_error", "dependency_error", "raw provider body sk-secret"},
		{"forbidden", http.StatusForbidden, http.StatusForbidden, "permission_error", "forbidden", "permission raw provider body"},
		{"rate-limit", http.StatusTooManyRequests, http.StatusTooManyRequests, "rate_limit_error", "rate_limited", "rate raw provider body"},
		{"server-error", http.StatusInternalServerError, http.StatusBadGateway, "upstream_error", "dependency_error", "stack trace raw provider body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tc.rawBody, tc.status)
			}))
			defer fake.Close()

			_, err := NewHTTPChatClient(fake.Client()).CompleteChat(context.Background(), providerRequest(fake.URL))
			openErr := assertOpenAIError(t, err)
			if openErr.HTTPStatus != tc.wantHTTP || openErr.Type != tc.wantType || openErr.Code != tc.wantCode {
				t.Fatalf("error = %+v", openErr)
			}
			if strings.Contains(openErr.Message, "raw provider body") || strings.Contains(openErr.Message, "sk-secret") {
				t.Fatalf("provider raw body leaked in error: %q", openErr.Message)
			}
			if openErr.ProviderStatusCode == nil || *openErr.ProviderStatusCode != tc.status {
				t.Fatalf("ProviderStatusCode = %v, want %d", openErr.ProviderStatusCode, tc.status)
			}
		})
	}
}

func TestHTTPChatClientTimeout(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer fake.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := NewHTTPChatClient(fake.Client()).CompleteChat(ctx, providerRequest(fake.URL))
	openErr := assertOpenAIError(t, err)
	if openErr.Code != "timeout" {
		t.Fatalf("Code = %q, want timeout", openErr.Code)
	}
}

func TestHTTPChatClientNonContractResponse(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":"not_chat","raw":"provider body"}`))
	}))
	defer fake.Close()

	_, err := NewHTTPChatClient(fake.Client()).CompleteChat(context.Background(), providerRequest(fake.URL))
	openErr := assertOpenAIError(t, err)
	if openErr.Type != "upstream_error" || openErr.Code != "dependency_error" {
		t.Fatalf("error = %+v", openErr)
	}
	if strings.Contains(openErr.Message, "provider body") {
		t.Fatalf("raw provider response leaked: %q", openErr.Message)
	}
}

func TestHTTPChatClientStreamCancel(t *testing.T) {
	cancelled := make(chan struct{})
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter does not flush")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chunk\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"provider-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()
		<-r.Context().Done()
		close(cancelled)
	}))
	defer fake.Close()
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := NewHTTPChatClient(fake.Client()).StreamChat(ctx, providerRequest(fake.URL))
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	reader := bufio.NewReader(stream.Body)
	if line, err := reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "data: ") {
		t.Fatalf("first stream line = %q, err = %v", line, err)
	}
	cancel()
	_ = stream.Body.Close()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("provider request was not cancelled")
	}
}

func providerRequest(baseURL string) service.ProviderChatRequest {
	return service.ProviderChatRequest{
		Profile: service.ModelProfile{
			ID:                "mp_test",
			Purpose:           service.PurposeChat,
			Provider:          service.ProviderOpenAICompatible,
			BaseURL:           baseURL + "/v1",
			Model:             "provider-model",
			TimeoutMS:         1000,
			Enabled:           true,
			APIKeyConfigured:  true,
			SupportsStreaming: true,
		},
		APIKey:    "sk-test-key",
		RequestID: "req_test",
		Payload: map[string]json.RawMessage{
			"model":    json.RawMessage(`"provider-model"`),
			"messages": json.RawMessage(`[{"role":"user","content":"hello"}]`),
		},
	}
}

func assertOpenAIError(t *testing.T, err error) *service.OpenAIError {
	t.Helper()
	var openErr *service.OpenAIError
	if !errors.As(err, &openErr) {
		t.Fatalf("error = %T %[1]v, want OpenAIError", err)
	}
	return openErr
}
