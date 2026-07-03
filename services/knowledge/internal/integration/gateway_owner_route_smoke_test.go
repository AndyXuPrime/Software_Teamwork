package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const gatewayKnowledgeOwnerSmokeGate = "GATEWAY_KNOWLEDGE_OWNER_SMOKE"

type gatewayKnowledgeOwnerSmokeConfig struct {
	gatewayOwnerSmokeConfig
	vendorRuntimeURL string
	timeout          time.Duration
}

func TestGatewayKnowledgeOwnerRouteSmoke(t *testing.T) {
	if os.Getenv(gatewayKnowledgeOwnerSmokeGate) != "1" {
		t.Skip("set GATEWAY_KNOWLEDGE_OWNER_SMOKE=1 to run the Gateway -> Knowledge owner route smoke")
	}

	cfg := loadGatewayKnowledgeOwnerSmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	assertGatewayKnowledgeOwnerPrechecks(t, ctx, cfg)
	requestID := "req_gateway_knowledge_owner_smoke_" + safeIdentifierSuffix(newSmokeRunID(t))
	assertGatewayRejectsSpoofedKnowledgeOwnerHeaders(t, ctx, cfg, requestID+"_spoofed")

	session := createGatewaySession(t, ctx, cfg.gatewayOwnerSmokeConfig, requestID+"_session")
	requestedKnowledgeBaseID := "kb_gateway_owner_smoke_" + safeIdentifierSuffix(newSmokeRunID(t))
	createdKnowledgeBaseID := ""
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		if strings.TrimSpace(createdKnowledgeBaseID) == "" {
			return
		}
		deleteGatewayKnowledgeBase(cleanupCtx, t, cfg.gatewayOwnerSmokeConfig, session, "req_gateway_owner_cleanup_"+createdKnowledgeBaseID, createdKnowledgeBaseID)
		if err := deleteGatewaySmokeKnowledgeBaseRows(cleanupCtx, cfg.gatewayOwnerSmokeConfig, createdKnowledgeBaseID); err != nil {
			t.Errorf("cleanup Gateway owner smoke knowledge base %q: %v", createdKnowledgeBaseID, err)
		}
	})

	created := createGatewayKnowledgeBase(t, ctx, cfg.gatewayOwnerSmokeConfig, session, requestID+"_kb", requestedKnowledgeBaseID)
	createdKnowledgeBaseID = created.ID
	assertGatewayKnowledgeBaseOwnedBySessionUser(t, created, session)
	read := getGatewayKnowledgeBase(t, ctx, cfg.gatewayOwnerSmokeConfig, session, requestID+"_kb_read", created.ID)
	assertGatewayKnowledgeBaseOwnedBySessionUser(t, read, session)
}

func loadGatewayKnowledgeOwnerSmokeConfig(t *testing.T) gatewayKnowledgeOwnerSmokeConfig {
	t.Helper()
	required := map[string]string{
		"GATEWAY_BASE_URL":                               os.Getenv("GATEWAY_BASE_URL"),
		"VENDOR_RUNTIME_URL":                             os.Getenv("VENDOR_RUNTIME_URL"),
		"KNOWLEDGE_SERVICE_BASE_URL":                     os.Getenv("KNOWLEDGE_SERVICE_BASE_URL"),
		"KNOWLEDGE_TEST_DATABASE_URL":                    os.Getenv("KNOWLEDGE_TEST_DATABASE_URL"),
		"KNOWLEDGE_REDIS_ADDR":                           firstNonEmptyEnv("KNOWLEDGE_REDIS_ADDR", "GATEWAY_REDIS_ADDR"),
		"GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME": firstNonEmptyEnv("GATEWAY_SMOKE_USERNAME", "LOCAL_ADMIN_USERNAME"),
		"GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD": firstNonEmptyEnv("GATEWAY_SMOKE_PASSWORD", "LOCAL_ADMIN_PASSWORD"),
	}
	requireEnvSet(t, gatewayKnowledgeOwnerSmokeGate, required)
	timeout := 90 * time.Second
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_KNOWLEDGE_OWNER_SMOKE_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			t.Fatalf("GATEWAY_KNOWLEDGE_OWNER_SMOKE_TIMEOUT must be a positive duration")
		}
		timeout = value
	}
	return gatewayKnowledgeOwnerSmokeConfig{
		gatewayOwnerSmokeConfig: gatewayOwnerSmokeConfig{
			gatewayBaseURL:          trimHTTPBaseURL(t, "GATEWAY_BASE_URL", required["GATEWAY_BASE_URL"]),
			knowledgeServiceBaseURL: trimHTTPBaseURL(t, "KNOWLEDGE_SERVICE_BASE_URL", required["KNOWLEDGE_SERVICE_BASE_URL"]),
			knowledgeDatabaseURL:    strings.TrimSpace(required["KNOWLEDGE_TEST_DATABASE_URL"]),
			redisAddr:               normalizeRedisAddr(t, required["KNOWLEDGE_REDIS_ADDR"]),
			username:                strings.TrimSpace(required["GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME"]),
			password:                strings.TrimSpace(required["GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD"]),
		},
		vendorRuntimeURL: trimHTTPBaseURL(t, "VENDOR_RUNTIME_URL", required["VENDOR_RUNTIME_URL"]),
		timeout:          timeout,
	}
}

func assertGatewayKnowledgeOwnerPrechecks(t *testing.T, ctx context.Context, cfg gatewayKnowledgeOwnerSmokeConfig) {
	t.Helper()
	assertRuntimeHealthy(t, ctx, cfg.vendorRuntimeURL)
	assertPostgresReady(t, ctx, cfg.knowledgeDatabaseURL)
	assertRedisReady(t, ctx, cfg.redisAddr)
	assertHTTPReady(t, ctx, "knowledge", cfg.knowledgeServiceBaseURL)
	assertHTTPReady(t, ctx, "gateway", cfg.gatewayBaseURL)
}

func assertGatewayRejectsSpoofedKnowledgeOwnerHeaders(t *testing.T, ctx context.Context, cfg gatewayKnowledgeOwnerSmokeConfig, requestID string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.gatewayBaseURL+"/api/v1/knowledge-bases", nil)
	if err != nil {
		t.Fatalf("build spoofed unauthenticated knowledge route request: %v", err)
	}
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("X-User-Id", "usr_spoofed_gateway_owner_smoke")
	req.Header.Set("X-User-Roles", "super_admin")
	req.Header.Set("X-User-Permissions", "knowledge:write,knowledge:read")

	res, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("spoofed unauthenticated knowledge route request failed: %v", err)
	}
	defer res.Body.Close()
	discardResponse(res.Body)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("spoofed unauthenticated knowledge route returned HTTP %d, want 401", res.StatusCode)
	}
}

func getGatewayKnowledgeBase(t *testing.T, ctx context.Context, cfg gatewayOwnerSmokeConfig, session gatewaySmokeSession, requestID string, knowledgeBaseID string) gatewayKnowledgeBaseSummary {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.gatewayBaseURL+"/api/v1/knowledge-bases/"+url.PathEscape(knowledgeBaseID), nil)
	if err != nil {
		t.Fatalf("build gateway knowledge base get request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("X-User-Id", spoofedGatewayUserID(session.UserID))

	res, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("gateway knowledge base get request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		discardResponse(res.Body)
		t.Fatalf("gateway knowledge base get returned HTTP %d", res.StatusCode)
	}
	return decodeGatewayKnowledgeBaseResponse(t, res.Body, requestID)
}

func assertGatewayKnowledgeBaseOwnedBySessionUser(t *testing.T, kb gatewayKnowledgeBaseSummary, session gatewaySmokeSession) {
	t.Helper()
	if strings.TrimSpace(kb.CreatedBy) != session.UserID {
		data, _ := json.Marshal(kb)
		t.Fatalf("gateway knowledge base createdBy = %q, want real session user %q; response=%s", kb.CreatedBy, session.UserID, data)
	}
}
