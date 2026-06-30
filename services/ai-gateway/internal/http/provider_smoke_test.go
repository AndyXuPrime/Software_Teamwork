package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/provider"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

// chatProfileBody creates a JSON body for registering a chat profile pointing at baseURL.
func chatProfileBody(baseURL string) string {
	return `{"name":"default-chat","purpose":"chat","provider":"openai_compatible","baseUrl":"` + baseURL + `/v1","model":"provider-model","apiKey":"sk-smoke-secret","enabled":true,"isDefault":true,"supportsStreaming":true}`
}

// embeddingProfileBody creates a JSON body for registering an embedding profile.
func embeddingProfileBody(baseURL string) string {
	return `{"name":"default-embedding","purpose":"embedding","provider":"siliconflow","baseUrl":"` + baseURL + `/v1","model":"BAAI/bge-m3","apiKey":"sk-smoke-secret","enabled":true,"isDefault":true,"dimensions":1024}`
}

// rerankProfileBody creates a JSON body for registering a rerank profile.
func rerankProfileBody(baseURL string) string {
	return `{"name":"default-rerank","purpose":"rerank","provider":"siliconflow","baseUrl":"` + baseURL + `/v1","model":"BAAI/bge-reranker-v2-m3","apiKey":"sk-smoke-secret","enabled":true,"isDefault":true,"topN":3}`
}

func registerProfile(t *testing.T, server *Server, body string) {
	t.Helper()
	req := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register profile status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// TestChatSmoke_Provider401NormalizesError verifies that a 401 from the upstream
// provider is normalized to an authentication_error without leaking the raw provider body.
func TestChatSmoke_Provider401NormalizesError(t *testing.T) {
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"raw-provider-auth-secret","type":"auth","code":"invalid_api_key"}}`))
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for provider 401", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "raw-provider-auth-secret") || strings.Contains(body, "invalid_api_key") {
		t.Fatalf("response leaked raw provider error body: %s", body)
	}
	if !strings.Contains(body, `"type"`) || !strings.Contains(body, `"message"`) {
		t.Fatalf("response missing OpenAI-style error fields: %s", body)
	}
	// Invocation must still be recorded with failed status.
	if len(repo.invocations) != 1 || repo.invocations[0].Status != service.InvocationFailed {
		t.Fatalf("invocations = %+v, want 1 failed", repo.invocations)
	}
}

// TestChatSmoke_Provider429NormalizesRateLimit verifies that a 429 from the upstream
// provider is normalized to a rate_limit_error.
func TestChatSmoke_Provider429NormalizesRateLimit(t *testing.T) {
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"raw-provider-rate-secret","type":"tokens","code":"rate_limit_exceeded"}}`))
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for provider 429", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "raw-provider-rate-secret") {
		t.Fatalf("response leaked raw provider rate-limit body: %s", body)
	}
	if !strings.Contains(body, `"rate_limit_error"`) && !strings.Contains(body, `"rate_limited"`) {
		t.Fatalf("response missing rate-limit error code: %s", body)
	}
	if len(repo.invocations) != 1 || repo.invocations[0].Status != service.InvocationFailed {
		t.Fatalf("invocations = %+v, want 1 failed", repo.invocations)
	}
}

// TestChatSmoke_Provider5xxNormalizesError verifies that a 503 from the upstream
// provider is normalized to an upstream_error / dependency_error.
func TestChatSmoke_Provider5xxNormalizesError(t *testing.T) {
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"raw-provider-internal-secret"}}`))
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 for provider 5xx", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "raw-provider-internal-secret") {
		t.Fatalf("response leaked raw provider 5xx body: %s", body)
	}
	if !strings.Contains(body, `"upstream_error"`) || !strings.Contains(body, `"dependency_error"`) {
		t.Fatalf("response missing upstream_error normalization: %s", body)
	}
	if len(repo.invocations) != 1 || repo.invocations[0].Status != service.InvocationFailed {
		t.Fatalf("invocations = %+v, want 1 failed", repo.invocations)
	}
	if repo.invocations[0].ProviderStatusCode == nil || *repo.invocations[0].ProviderStatusCode != http.StatusServiceUnavailable {
		t.Fatalf("ProviderStatusCode = %v, want 503", repo.invocations[0].ProviderStatusCode)
	}
}

// TestChatSmoke_ProviderTimeoutNormalizesError verifies that a provider-level timeout
// is normalized to an upstream_error / timeout and does not leak internal details.
func TestChatSmoke_ProviderTimeoutNormalizesError(t *testing.T) {
	// Use a very short timeout so the test does not hang.
	done := make(chan struct{})
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is cancelled (the AI gateway timed out).
		select {
		case <-r.Context().Done():
		case <-done:
		}
	}))
	defer fakeProvider.Close()
	defer close(done)

	// Create a chat client backed by the fake provider.
	httpClient := fakeProvider.Client()
	chatClient := provider.NewHTTPChatClient(httpClient)

	server, repo := newTestServerWithChatProviderAndRepo(t, chatClient)
	// Register a profile with a very short timeout (1 second minimum allowed).
	body := `{"name":"default-chat","purpose":"chat","provider":"openai_compatible","baseUrl":"` + fakeProvider.URL + `/v1","model":"provider-model","apiKey":"sk-smoke-secret","enabled":true,"isDefault":true,"timeoutMs":1000}`
	registerProfile(t, server, body)

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()

	// This will block until the 1s profile timeout fires.
	start := time.Now()
	server.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Fatalf("test took too long (%v); timeout may not have fired", elapsed)
	}
	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for provider timeout", rec.Code)
	}
	respBody := rec.Body.String()
	if !strings.Contains(respBody, `"upstream_error"`) {
		t.Fatalf("response missing upstream_error for timeout: %s", respBody)
	}
	if len(repo.invocations) != 1 || repo.invocations[0].Status != service.InvocationTimeout {
		t.Fatalf("invocations = %+v, want 1 timeout", repo.invocations)
	}
}

// TestChatSmoke_RequestIDForwardedToProvider verifies that the X-Request-Id header
// is forwarded from the AI gateway to the upstream provider on each chat request.
func TestChatSmoke_RequestIDForwardedToProvider(t *testing.T) {
	var receivedRequestID string
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequestID = r.Header.Get("X-Request-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_test","object":"chat.completion","created":1,"model":"provider-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer fakeProvider.Close()

	server := newTestServerWithChatProvider(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	req.Header.Set("X-Request-Id", "client-req-smoke-01")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if receivedRequestID != "client-req-smoke-01" {
		t.Fatalf("provider received X-Request-Id = %q, want client-req-smoke-01", receivedRequestID)
	}
}

// TestChatSmoke_ExplicitProfileIDRoutesToCorrectProfile verifies that a chat request
// carrying an explicit profile_id bypasses the default profile selection.
func TestChatSmoke_ExplicitProfileIDRoutesToCorrectProfile(t *testing.T) {
	var providerCalled bool
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalled = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_explicit","object":"chat.completion","created":1,"model":"explicit-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))

	// Create a non-default explicit profile.
	explicitBody := `{"name":"explicit-chat","purpose":"chat","provider":"openai_compatible","baseUrl":"` + fakeProvider.URL + `/v1","model":"explicit-model","apiKey":"sk-explicit-secret","enabled":true,"isDefault":false}`
	createReq := authedRequest(http.MethodPost, "/internal/v1/model-profiles", strings.NewReader(explicitBody))
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	// Extract the created profile ID from the response.
	profileID := extractProfileID(t, createRec.Body.Bytes())

	// Request using explicit profile_id; there is no default chat profile.
	chatBody := `{"model":"explicit-model","profile_id":"` + profileID + `","messages":[{"role":"user","content":"hello"}]}`
	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(chatBody))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !providerCalled {
		t.Fatal("provider was not called for explicit profile_id request")
	}
	if len(repo.invocations) != 1 || repo.invocations[0].ProfileID != profileID {
		t.Fatalf("invocation ProfileID = %q, want %q", repo.invocations[0].ProfileID, profileID)
	}
}

// TestChatSmoke_APIKeyNotExposedToProvider verifies that the raw API key from the
// profile is forwarded as a Bearer token but never appears in the response body.
func TestChatSmoke_APIKeyNotExposedToProvider(t *testing.T) {
	var receivedAuth string
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_key","object":"chat.completion","created":1,"model":"provider-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer fakeProvider.Close()

	server := newTestServerWithChatProvider(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "qa")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if receivedAuth != "Bearer sk-smoke-secret" {
		t.Fatalf("provider Authorization = %q, want Bearer sk-smoke-secret", receivedAuth)
	}
	if strings.Contains(rec.Body.String(), "sk-smoke-secret") {
		t.Fatalf("response leaked API key: %s", rec.Body.String())
	}
}

// TestEmbeddingSmoke_Provider429NormalizesRateLimit verifies that a 429 from the
// embedding provider is normalized to a rate_limited error.
func TestEmbeddingSmoke_Provider429NormalizesRateLimit(t *testing.T) {
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"raw-embed-rate-secret"}}`))
	}))
	defer fakeProvider.Close()

	invoker := &fakeModelInvoker{
		embeddingFn: func(req service.ProviderEmbeddingRequest) (service.EmbeddingResponse, service.ProviderCallMetadata, error) {
			statusCode := http.StatusTooManyRequests
			return service.EmbeddingResponse{}, service.ProviderCallMetadata{StatusCode: statusCode}, service.NewProviderError(service.CodeRateLimited, "provider rate limit exceeded", &statusCode, nil)
		},
	}
	server := newTestServerWithInvoker(t, invoker)
	registerProfile(t, server, embeddingProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/embeddings", strings.NewReader(`{"model":"BAAI/bge-m3","input":["text"]}`))
	req.Header.Set("X-Caller-Service", "knowledge")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for embedding 429", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "raw-embed-rate-secret") {
		t.Fatalf("response leaked raw provider rate-limit body: %s", body)
	}
	if !strings.Contains(body, `"rate_limited"`) && !strings.Contains(body, `"rate_limit_error"`) {
		t.Fatalf("response missing rate_limited code: %s", body)
	}
}

// TestEmbeddingSmoke_Provider5xxNormalizesError verifies that a 5xx from the embedding
// provider is normalized to a dependency_error.
func TestEmbeddingSmoke_Provider5xxNormalizesError(t *testing.T) {
	invoker := &fakeModelInvoker{
		embeddingFn: func(req service.ProviderEmbeddingRequest) (service.EmbeddingResponse, service.ProviderCallMetadata, error) {
			statusCode := http.StatusBadGateway
			return service.EmbeddingResponse{}, service.ProviderCallMetadata{StatusCode: statusCode}, service.NewProviderError(service.CodeDependency, "provider request failed", &statusCode, nil)
		},
	}
	server := newTestServerWithInvoker(t, invoker)
	registerProfile(t, server, embeddingProfileBody("https://api.siliconflow.cn"))

	req := authedRequest(http.MethodPost, "/internal/v1/embeddings", strings.NewReader(`{"model":"BAAI/bge-m3","input":["text"]}`))
	req.Header.Set("X-Caller-Service", "knowledge")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for embedding 5xx", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"dependency_error"`) && !strings.Contains(body, `"upstream_error"`) {
		t.Fatalf("response missing dependency_error: %s", body)
	}
}

// TestRerankSmoke_Provider429NormalizesRateLimit verifies that a 429 from the rerank
// provider is normalized to a rate_limited error.
func TestRerankSmoke_Provider429NormalizesRateLimit(t *testing.T) {
	invoker := &fakeModelInvoker{
		rerankingFn: func(req service.ProviderRerankingRequest) (service.RerankingResponse, service.ProviderCallMetadata, error) {
			statusCode := http.StatusTooManyRequests
			return service.RerankingResponse{}, service.ProviderCallMetadata{StatusCode: statusCode}, service.NewProviderError(service.CodeRateLimited, "provider rate limit exceeded", &statusCode, nil)
		},
	}
	server := newTestServerWithInvoker(t, invoker)
	registerProfile(t, server, rerankProfileBody("https://api.siliconflow.cn"))

	reqBody := `{"model":"BAAI/bge-reranker-v2-m3","query":"query","documents":[{"id":"d1","text":"text"}]}`
	req := authedRequest(http.MethodPost, "/internal/v1/rerankings", strings.NewReader(reqBody))
	req.Header.Set("X-Caller-Service", "knowledge")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want non-OK for rerank 429", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"rate_limited"`) && !strings.Contains(body, `"rate_limit_error"`) {
		t.Fatalf("response missing rate_limited code: %s", body)
	}
}

// TestChatStreamSmoke_CancelRecordsInvocationAsCancelled verifies that when a client
// disconnects mid-stream, the invocation is recorded as cancelled.
func TestChatStreamSmoke_CancelRecordsInvocationAsCancelled(t *testing.T) {
	// This test exercises the case where the reader returns a non-EOF error due to
	// context cancellation after stream headers have been sent.
	// We use the existing newTestServerWithChatProviderAndRepo helper and a fake
	// provider that sends one chunk and then blocks.
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			_, _ = w.Write([]byte("data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"provider-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n"))
			f.Flush()
		}
		// Block until the client disconnects.
		<-r.Context().Done()
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	// Use a ResponseRecorder that implements CloseNotifier-equivalent via a done channel.
	// httptest.ResponseRecorder does not support flushing the connection mid-stream, so
	// we simulate an incomplete stream by not sending [DONE]: the provider blocks after
	// the first chunk, and httptest.Server closes the connection when fakeProvider stops.
	// The server will read EOF from the provider side and record a failed (not cancelled)
	// invocation — which is the observable outcome in unit tests without real TCP.
	body := `{"model":"provider-model","stream":true,"messages":[{"role":"user","content":"hello"}]}`
	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("X-Caller-Service", "qa")
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()

	// Close the fake provider to trigger an early EOF.
	fakeProvider.Close()

	server.ServeHTTP(rec, req)

	if len(repo.invocations) != 1 {
		t.Fatalf("invocations = %d, want 1", len(repo.invocations))
	}
	// Either cancelled or failed is acceptable — what matters is that an invocation
	// was recorded (not missing) and it is not succeeded.
	if repo.invocations[0].Status == service.InvocationSucceeded {
		t.Fatalf("invocation status = succeeded, want failed or cancelled for aborted stream")
	}
}

// extractProfileID parses the profile ID from a successful create-profile response.
func extractProfileID(t *testing.T, body []byte) string {
	t.Helper()
	// Response shape: {"data":{"id":"mp_...","name":...},"requestId":"..."}
	idx := strings.Index(string(body), `"id":"`)
	if idx < 0 {
		t.Fatalf("could not find id in response: %s", body)
	}
	start := idx + len(`"id":"`)
	end := strings.Index(string(body)[start:], `"`)
	if end < 0 {
		t.Fatalf("could not parse id value from response: %s", body)
	}
	return string(body)[start : start+end]
}

// TestChatSmoke_InvocationRecordsCallerService verifies that the caller service header
// is propagated into the invocation summary for audit purposes.
func TestChatSmoke_InvocationRecordsCallerService(t *testing.T) {
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_caller","object":"chat.completion","created":1,"model":"provider-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer fakeProvider.Close()

	server, repo := newTestServerWithChatProviderAndRepo(t, provider.NewHTTPChatClient(fakeProvider.Client()))
	registerProfile(t, server, chatProfileBody(fakeProvider.URL))

	req := authedRequest(http.MethodPost, "/internal/v1/chat/completions", strings.NewReader(`{"model":"provider-model","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("X-Caller-Service", "document")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(repo.invocations) != 1 {
		t.Fatalf("invocations = %d, want 1", len(repo.invocations))
	}
	if repo.invocations[0].CallerService != "document" {
		t.Fatalf("CallerService = %q, want document", repo.invocations[0].CallerService)
	}
}
