package integration_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type gatewayOwnerSmokeConfig struct {
	gatewayBaseURL          string
	fileServiceBaseURL      string
	parserServiceBaseURL    string
	knowledgeServiceBaseURL string
	knowledgeDatabaseURL    string
	redisAddr               string
	username                string
	password                string
}

type gatewaySmokeSession struct {
	AccessToken string
	UserID      string
}

func assertHTTPReady(t *testing.T, ctx context.Context, name string, baseURL string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/readyz", nil)
	if err != nil {
		t.Fatalf("build %s ready request: %v", name, err)
	}
	req.Header.Set("X-Request-Id", "req_gateway_knowledge_owner_precheck")
	res, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("%s readyz request failed: %v", name, err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("%s readyz returned HTTP %d", name, res.StatusCode)
	}
}

func assertPostgresReady(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect Knowledge PostgreSQL for owner smoke: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping Knowledge PostgreSQL for owner smoke: %v", err)
	}
}

func assertRedisReady(t *testing.T, ctx context.Context, addr string) {
	t.Helper()
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Fatalf("connect Redis for owner smoke: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.WriteString(conn, "PING\r\n"); err != nil {
		t.Fatalf("send Redis PING for owner smoke: %v", err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read Redis PING for owner smoke: %v", err)
	}
	if !strings.HasPrefix(line, "+PONG") {
		t.Fatalf("Redis PING response = %q, want PONG", strings.TrimSpace(line))
	}
}

func createGatewaySession(t *testing.T, ctx context.Context, cfg gatewayOwnerSmokeConfig, requestID string) gatewaySmokeSession {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"username": cfg.username,
		"password": cfg.password,
	})
	if err != nil {
		t.Fatalf("encode gateway session request: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/sessions", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build gateway session request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID)

	res, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("gateway session request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
		t.Fatalf("gateway session returned HTTP %d", res.StatusCode)
	}

	var decoded struct {
		Data struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
			Session struct {
				AccessToken string `json:"accessToken"`
				TokenType   string `json:"tokenType"`
			} `json:"session"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&decoded); err != nil {
		t.Fatalf("decode gateway session response: %v", err)
	}
	if strings.TrimSpace(decoded.RequestID) != requestID {
		t.Fatalf("gateway session requestId = %q, want %q", decoded.RequestID, requestID)
	}
	if strings.TrimSpace(decoded.Data.Session.AccessToken) == "" || !strings.EqualFold(decoded.Data.Session.TokenType, "Bearer") {
		t.Fatal("gateway session response did not include a bearer access token")
	}
	if strings.TrimSpace(decoded.Data.User.ID) == "" {
		t.Fatal("gateway session response did not include user id")
	}
	return gatewaySmokeSession{
		AccessToken: strings.TrimSpace(decoded.Data.Session.AccessToken),
		UserID:      strings.TrimSpace(decoded.Data.User.ID),
	}
}

type gatewayKnowledgeBaseSummary struct {
	ID        string `json:"id"`
	CreatedBy string `json:"createdBy"`
}

func createGatewayKnowledgeBase(t *testing.T, ctx context.Context, cfg gatewayOwnerSmokeConfig, session gatewaySmokeSession, requestID string, knowledgeBaseID string) gatewayKnowledgeBaseSummary {
	t.Helper()
	payload, err := json.Marshal(map[string]string{
		"id":          knowledgeBaseID,
		"name":        "Gateway owner smoke " + knowledgeBaseID,
		"description": "A-021 owner context smoke",
		"docType":     "SMOKE",
	})
	if err != nil {
		t.Fatalf("encode gateway knowledge base create request: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.gatewayBaseURL+"/api/v1/knowledge-bases", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build gateway knowledge base create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", requestID)
	req.Header.Set("X-User-Id", spoofedGatewayUserID(session.UserID))

	res, err := smokeHTTPClient().Do(req)
	if err != nil {
		t.Fatalf("gateway knowledge base create request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
		t.Fatalf("gateway knowledge base create returned HTTP %d", res.StatusCode)
	}
	return decodeGatewayKnowledgeBaseResponse(t, res.Body, requestID)
}

func decodeGatewayKnowledgeBaseResponse(t *testing.T, body io.Reader, requestID string) gatewayKnowledgeBaseSummary {
	t.Helper()
	var decoded struct {
		Data      gatewayKnowledgeBaseSummary `json:"data"`
		RequestID string                      `json:"requestId"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&decoded); err != nil {
		t.Fatalf("decode gateway knowledge base response: %v", err)
	}
	if strings.TrimSpace(decoded.RequestID) != requestID {
		t.Fatalf("gateway knowledge base requestId = %q, want %q", decoded.RequestID, requestID)
	}
	if strings.TrimSpace(decoded.Data.ID) == "" {
		t.Fatal("gateway knowledge base response id is empty")
	}
	if strings.TrimSpace(decoded.Data.CreatedBy) == "" {
		t.Fatal("gateway knowledge base response createdBy is empty")
	}
	return decoded.Data
}

func deleteGatewaySmokeKnowledgeBaseRows(ctx context.Context, cfg gatewayOwnerSmokeConfig, knowledgeBaseID string) error {
	pool, err := pgxpool.New(ctx, cfg.knowledgeDatabaseURL)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()
	statements := []string{
		"DELETE FROM document_chunks WHERE knowledge_base_id = $1",
		"DELETE FROM processing_jobs WHERE knowledge_base_id = $1",
		"DELETE FROM knowledge_documents WHERE knowledge_base_id = $1",
		"DELETE FROM knowledge_bases WHERE id = $1",
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement, knowledgeBaseID); err != nil {
			return err
		}
	}
	return nil
}

func spoofedGatewayUserID(realUserID string) string {
	const spoofed = "usr_spoofed_gateway_owner_smoke"
	if strings.TrimSpace(realUserID) == spoofed {
		return spoofed + "_other"
	}
	return spoofed
}

func trimHTTPBaseURL(t *testing.T, key string, raw string) string {
	t.Helper()
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		t.Fatalf("%s must be an absolute http(s) URL", key)
	}
	if parsed.User != nil {
		t.Fatalf("%s must not contain credentials", key)
	}
	return value
}

func normalizeRedisAddr(t *testing.T, raw string) string {
	t.Helper()
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "redis://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Host == "" {
			t.Fatalf("KNOWLEDGE_REDIS_ADDR must be host:port or redis://host:port")
		}
		return parsed.Host
	}
	if _, _, err := net.SplitHostPort(value); err != nil {
		t.Fatalf("KNOWLEDGE_REDIS_ADDR must include host and port")
	}
	return value
}

func smokeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func requireEnvSet(t *testing.T, gate string, required map[string]string) {
	t.Helper()
	var missing []string
	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("%s=1 requires %s", gate, strings.Join(missing, ", "))
	}
}

func newSmokeRunID(t *testing.T) string {
	t.Helper()
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("generate smoke run id: %v", err)
	}
	return time.Now().UTC().Format("20060102150405") + "_" + hex.EncodeToString(buf[:])
}

func safeIdentifierSuffix(value string) string {
	value = regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(strings.TrimSpace(value), "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "run"
	}
	return strings.ToLower(value)
}
