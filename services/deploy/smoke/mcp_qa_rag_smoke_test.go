package smoke_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

const qaMCPRAGSmokeGate = "QA_MCP_RAG_SMOKE"

func TestQAMCPRAGSmoke(t *testing.T) {
	if os.Getenv(qaMCPRAGSmokeGate) != "1" {
		t.Skip("set QA_MCP_RAG_SMOKE=1 to run the QA MCP RAG smoke")
	}

	cfg := loadQASmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	assertHTTPReady(t, ctx, "gateway", cfg.gatewayBaseURL)
	assertHTTPReady(t, ctx, "qa", cfg.qaBaseURL)
	assertHTTPReady(t, ctx, "knowledge", cfg.knowledgeBaseURL)

	requestID := "req_qa_mcp_rag_smoke_" + shortID(newSmokeRunID())
	client := smokeHTTPClient()
	session := createSmokeSession(t, ctx, client, cfg.gatewayBaseURL, cfg.username, cfg.password, requestID)

	// Basic Gateway→QA contract checks always run.
	t.Run("qa_response_envelope", func(t *testing.T) {
		assertQAResponseEnvelope(t, ctx, client, cfg, session, requestID)
	})
	t.Run("no_sensitive_leaks", func(t *testing.T) {
		assertQANoSensitiveLeaks(t, ctx, client, cfg, session, requestID)
	})

	// RAG sub-test requires AI Gateway with real provider profile.
	// Local seed uses placeholder profiles; skip if not configured.
	t.Run("qa_rag_knowledge_response", func(t *testing.T) {
		if cfg.aiGatewayBaseURL == "" {
			t.Skip("AI_GATEWAY_BASE_URL not set; RAG requires a real AI Gateway profile")
		}
		assertHTTPReady(t, ctx, "ai-gateway", cfg.aiGatewayBaseURL)
		kbID := assertKnowledgeToolAvailable(t, ctx, client, cfg, session, requestID)
		assertQAKnowledgeRAGResponse(t, ctx, client, cfg, session, kbID, requestID)
	})
}

type qaSmokeConfig struct {
	gatewayBaseURL   string
	qaBaseURL        string
	knowledgeBaseURL string
	aiGatewayBaseURL string
	username         string
	password         string
}

func loadQASmokeConfig(t *testing.T) qaSmokeConfig {
	t.Helper()
	required := map[string]string{
		"GATEWAY_BASE_URL":                               os.Getenv("GATEWAY_BASE_URL"),
		"QA_SERVICE_BASE_URL":                            os.Getenv("QA_SERVICE_BASE_URL"),
		"KNOWLEDGE_SERVICE_BASE_URL":                     os.Getenv("KNOWLEDGE_SERVICE_BASE_URL"),
		"GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME": firstNonEmptyEnv("GATEWAY_SMOKE_USERNAME", "LOCAL_ADMIN_USERNAME"),
		"GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD": firstNonEmptyEnv("GATEWAY_SMOKE_PASSWORD", "LOCAL_ADMIN_PASSWORD"),
	}
	missing := []string{}
	for k, v := range required {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing required env vars:\n - %s", strings.Join(missing, "\n - "))
	}
	return qaSmokeConfig{
		gatewayBaseURL:   trimBaseURL(t, "GATEWAY_BASE_URL", required["GATEWAY_BASE_URL"]),
		qaBaseURL:        trimBaseURL(t, "QA_SERVICE_BASE_URL", required["QA_SERVICE_BASE_URL"]),
		knowledgeBaseURL: trimBaseURL(t, "KNOWLEDGE_SERVICE_BASE_URL", required["KNOWLEDGE_SERVICE_BASE_URL"]),
		username:         strings.TrimSpace(required["GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME"]),
		password:         strings.TrimSpace(required["GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD"]),
		aiGatewayBaseURL: strings.TrimSpace(os.Getenv("AI_GATEWAY_BASE_URL")),
	}
}

// ---- QA-specific helpers ----

// assertKnowledgeToolAvailable verifies that the knowledge base list endpoint
// returns data and returns the first KB ID for use in RAG requests.
func assertKnowledgeToolAvailable(t *testing.T, ctx context.Context, client *http.Client, cfg qaSmokeConfig, session smokeSession, requestID string) string {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/knowledge-bases?page=1&pageSize=5", session.AccessToken, requestID+"_tools", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list knowledge bases: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing knowledge bases, got %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode knowledge bases: %v", err)
	}
	if len(envelope.Data) == 0 {
		t.Skip("no knowledge bases available; seed-local required for RAG smoke")
	}
	return envelope.Data[0].ID
}

// assertQAKnowledgeRAGResponse sends a QA message in knowledge_qa mode and
// waits for the full SSE stream or non-streaming response.
func assertQAKnowledgeRAGResponse(t *testing.T, ctx context.Context, client *http.Client, cfg qaSmokeConfig, session smokeSession, kbID, requestID string) {
	t.Helper()

	// Create a QA session and use its ID for subsequent requests
	sessionBody, _ := json.Marshal(map[string]string{"title": "smoke test rag"})
	createReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions", bytes.NewReader(sessionBody))
	createReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Request-Id", requestID+"_create")
	sessionResp, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create qa session: %v", err)
	}
	defer sessionResp.Body.Close()
	if sessionResp.StatusCode != http.StatusCreated {
		t.Fatalf("create qa session returned %d", sessionResp.StatusCode)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(sessionResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created qa session: %v", err)
	}
	createdSessionID := created.Data.ID
	if createdSessionID == "" {
		t.Fatal("created qa session has empty id")
	}

	// Use a client without body timeout for SSE streaming.
	sseClient := &http.Client{
		Timeout:       0,
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	// Send a message to the newly created session — accept SSE stream
	msgBody, _ := json.Marshal(map[string]any{
		"message":          "根据规程，锅炉巡检时油温正常范围是多少？",
		"mode":             "knowledge_qa",
		"knowledgeBaseIds": []string{kbID},
	})
	sendReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions/"+createdSessionID+"/messages", bytes.NewReader(msgBody))
	sendReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	sendReq.Header.Set("Content-Type", "application/json")
	sendReq.Header.Set("Accept", "text/event-stream")
	sendReq.Header.Set("X-Request-Id", requestID+"_ask")
	r, err := sseClient.Do(sendReq)
	if err != nil {
		t.Fatalf("send qa message: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		t.Fatalf("send qa message returned %d: %s", r.StatusCode, string(data))
	}

	// Parse SSE stream — verify expected event types appear.
	seen := map[string]bool{}
	var responseRunID string
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			seen[eventType] = true
		}
		if strings.HasPrefix(line, "data: ") && responseRunID == "" {
			dataStr := strings.TrimPrefix(line, "data: ")
			var payload struct {
				ResponseRunID string `json:"responseRunId"`
			}
			if err := json.Unmarshal([]byte(dataStr), &payload); err == nil && payload.ResponseRunID != "" {
				responseRunID = payload.ResponseRunID
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("SSE stream scanner error: %v", err)
	}
	required := []string{"message.created", "answer.completed"}
	for _, event := range required {
		if !seen[event] {
			t.Errorf("SSE stream is missing expected event %q, seen=%v", event, seen)
		}
	}
	if responseRunID == "" {
		t.Fatal("SSE stream did not include a responseRunId; cannot verify tool-calls")
	}
	assertToolCallsRecorded(t, ctx, client, cfg, session, responseRunID, requestID)
}

func assertToolCallsRecorded(t *testing.T, ctx context.Context, client *http.Client, cfg qaSmokeConfig, session smokeSession, responseRunID, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet,
		cfg.gatewayBaseURL+"/api/v1/response-runs/"+responseRunID+"/tool-calls",
		session.AccessToken, requestID+"_tc", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("tool-calls request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tool-calls returned %d: %s", resp.StatusCode, string(body))
	}
	assertNoLeakedInternals(t, body)
	// Verify at least one tool call is recorded.
	var tcEnvelope struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &tcEnvelope); err != nil {
		t.Fatalf("decode tool-calls: %v", err)
	}
	if len(tcEnvelope.Data) == 0 {
		t.Fatal("tool-calls response is empty; expected at least one knowledge search tool call")
	}
	t.Logf("tool-calls recorded for response run %s (count=%d)", responseRunID, len(tcEnvelope.Data))
}

// assertQAResponseEnvelope creates a new QA session, sends a non-streaming
// message, and validates the response envelope contains requestId.
func assertQAResponseEnvelope(t *testing.T, ctx context.Context, client *http.Client, cfg qaSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	// Create a fresh QA session
	sessionBody, _ := json.Marshal(map[string]string{"title": "smoke envelope test"})
	createReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions", bytes.NewReader(sessionBody))
	createReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Request-Id", requestID+"_create")
	cr, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create qa session: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create qa session returned %d", cr.StatusCode)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(cr.Body).Decode(&created); err != nil {
		t.Fatalf("decode created session: %v", err)
	}

	// Non-streaming ask
	msgBody, _ := json.Marshal(map[string]any{"message": "你好", "mode": "general_chat"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions/"+created.Data.ID+"/messages", bytes.NewReader(msgBody))
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID+"_plain")
	r, err := client.Do(req)
	if err != nil {
		t.Fatalf("non-streaming ask: %v", err)
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(r.Body, 65536))
	if r.StatusCode != http.StatusOK {
		t.Fatalf("non-streaming ask returned %d: %s", r.StatusCode, string(body))
	}
	var envelope struct {
		Data struct {
			ResponseRun struct {
				ID string `json:"id"`
			} `json:"responseRun"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode ask envelope: %v body=%s", err, string(body))
	}
	if envelope.RequestID == "" {
		t.Fatal("response missing requestId")
	}
	if envelope.Data.ResponseRun.ID == "" {
		t.Fatal("response missing responseRun.id")
	}
}

// assertQANoSensitiveLeaks sends a general_chat request and checks the
// response does not contain secrets, raw prompts, or internal URLs.
func assertQANoSensitiveLeaks(t *testing.T, ctx context.Context, client *http.Client, cfg qaSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	// Create a fresh QA session
	sessionBody, _ := json.Marshal(map[string]string{"title": "smoke leak test"})
	createReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions", bytes.NewReader(sessionBody))
	createReq.Header.Set("Authorization", "Bearer "+session.AccessToken)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("X-Request-Id", requestID+"_create")
	cr, err := client.Do(createReq)
	if err != nil {
		t.Fatalf("create qa session: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create qa session returned %d", cr.StatusCode)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(cr.Body).Decode(&created); err != nil {
		t.Fatalf("decode created session: %v", err)
	}

	msgBody, _ := json.Marshal(map[string]any{
		"message": "hello",
		"mode":    "general_chat",
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/qa-sessions/"+created.Data.ID+"/messages", bytes.NewReader(msgBody))
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID+"_noleak")
	r, err := client.Do(req)
	if err != nil {
		t.Fatalf("no-leak request: %v", err)
	}
	defer r.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(r.Body, 65536))
	if r.StatusCode != http.StatusOK {
		t.Fatalf("no-leak request returned %d: %s", r.StatusCode, string(body))
	}
	for _, forbidden := range []string{
		"prompt", "<|begin_of_text|>", "system prompt",
		"api_key", "service_token", "objectKey", "internalUrl",
		"rawVector", "parser_backend",
	} {
		if strings.Contains(strings.ToLower(string(body)), strings.ToLower(forbidden)) {
			t.Fatalf("response leaked %q", forbidden)
		}
	}
}
