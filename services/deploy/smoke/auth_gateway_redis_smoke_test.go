package smoke_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const authGatewayRedisSmokeGate = "AUTH_GATEWAY_REDIS_SMOKE"

func TestAuthGatewayRedisSmoke(t *testing.T) {
	if os.Getenv(authGatewayRedisSmokeGate) != "1" {
		t.Skip("set AUTH_GATEWAY_REDIS_SMOKE=1 to run the Auth/Gateway/Redis smoke")
	}

	cfg := loadAuthGatewayRedisSmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	assertHTTPReady(t, ctx, "gateway", cfg.gatewayBaseURL)

	requestID := "req_auth_gateway_redis_smoke_" + shortID(newSmokeRunID())
	client := smokeHTTPClient()

	assertCurrentUserRejectsMissingAuth(t, ctx, client, cfg, requestID+"_missing")
	assertGatewayIgnoresSpoofedUserHeaders(t, ctx, client, cfg, requestID+"_spoof")

	session := createSmokeSession(t, ctx, client, cfg.gatewayBaseURL, cfg.username, cfg.password, requestID+"_login")
	cacheKey := assertRedisSessionCacheSafe(t, ctx, cfg, requestID+"_login", session.AccessToken)
	assertCurrentUserViaGateway(t, ctx, client, cfg, session, requestID+"_me")
	assertLogoutInvalidatesSession(t, ctx, client, cfg, session, requestID+"_logout")
	assertRedisSessionDeleted(t, ctx, cfg, cacheKey)
}

type authGatewayRedisSmokeConfig struct {
	gatewayBaseURL string
	username       string
	password       string
	redisAddr      string
	redisPassword  string
	redisDB        int
}

func loadAuthGatewayRedisSmokeConfig(t *testing.T) authGatewayRedisSmokeConfig {
	t.Helper()
	gatewayBaseURL := strings.TrimSpace(os.Getenv("GATEWAY_BASE_URL"))
	username := firstNonEmptyEnv("GATEWAY_SMOKE_USERNAME", "LOCAL_ADMIN_USERNAME")
	password := firstNonEmptyEnv("GATEWAY_SMOKE_PASSWORD", "LOCAL_ADMIN_PASSWORD")
	missing := []string{}
	if gatewayBaseURL == "" {
		missing = append(missing, "GATEWAY_BASE_URL")
	}
	if username == "" {
		missing = append(missing, "GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME")
	}
	if password == "" {
		missing = append(missing, "GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD")
	}
	if len(missing) > 0 {
		t.Fatalf("missing required environment variables:\n - %s", strings.Join(missing, "\n - "))
	}
	redisDB := 0
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_REDIS_DB")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			t.Fatalf("GATEWAY_REDIS_DB must be a non-negative integer")
		}
		redisDB = value
	}
	return authGatewayRedisSmokeConfig{
		gatewayBaseURL: trimBaseURL(t, "GATEWAY_BASE_URL", gatewayBaseURL),
		username:       strings.TrimSpace(username),
		password:       strings.TrimSpace(password),
		redisAddr:      firstNonEmptyEnvOr("127.0.0.1:6379", "GATEWAY_REDIS_ADDR", "REDIS_ADDR", "AUTH_GATEWAY_REDIS_ADDR"),
		redisPassword:  os.Getenv("GATEWAY_REDIS_PASSWORD"),
		redisDB:        redisDB,
	}
}

func firstNonEmptyEnvOr(fallback string, keys ...string) string {
	if value := firstNonEmptyEnv(keys...); value != "" {
		return value
	}
	return fallback
}

func assertCurrentUserRejectsMissingAuth(t *testing.T, ctx context.Context, client *http.Client, cfg authGatewayRedisSmokeConfig, requestID string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.gatewayBaseURL+"/api/v1/users/me", nil)
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("missing-auth current user request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing-auth current user returned %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertErrorEnvelope(t, body, requestID, "unauthorized")
	assertNoLeakedInternals(t, body)
}

func assertGatewayIgnoresSpoofedUserHeaders(t *testing.T, ctx context.Context, client *http.Client, cfg authGatewayRedisSmokeConfig, requestID string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, cfg.gatewayBaseURL+"/api/v1/users/me", nil)
	req.Header.Set("X-User-Id", "spoofed-user-must-not-authenticate")
	req.Header.Set("X-User-Roles", "admin")
	req.Header.Set("X-User-Permissions", "system:admin,knowledge:write")
	req.Header.Set("X-Request-Id", requestID)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("spoofed current user request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("spoofed current user returned %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertErrorEnvelope(t, body, requestID, "unauthorized")
	assertNoLeakedInternals(t, body)
}

func assertCurrentUserViaGateway(t *testing.T, ctx context.Context, client *http.Client, cfg authGatewayRedisSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/users/me", session.AccessToken, requestID, nil)
	req.Header.Set("X-User-Id", "spoofed-user-must-be-ignored")
	req.Header.Set("X-User-Roles", "admin")
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("current user request failed for requestId=%s: %v", requestID, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("current user returned %d for requestId=%s: %s", resp.StatusCode, requestID, responseBodySummary(body))
	}
	var envelope struct {
		Data struct {
			ID          string   `json:"id"`
			Username    string   `json:"username"`
			Roles       []string `json:"roles"`
			Permissions []string `json:"permissions"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode current user for requestId=%s: %v %s", requestID, err, responseBodySummary(body))
	}
	if envelope.RequestID != requestID {
		t.Fatalf("current user requestId mismatch: want=%q got=%q", requestID, envelope.RequestID)
	}
	if envelope.Data.ID != session.UserID {
		t.Fatalf("current user id mismatch: session=%q current=%q", session.UserID, envelope.Data.ID)
	}
	if envelope.Data.Username == "spoofed-user-must-be-ignored" || envelope.Data.Username == "" {
		t.Fatalf("current user username was not authoritative: %q", envelope.Data.Username)
	}
	if len(envelope.Data.Roles) == 0 || len(envelope.Data.Permissions) == 0 {
		t.Fatalf("current user roles/permissions missing: roles=%v permissions=%v", envelope.Data.Roles, envelope.Data.Permissions)
	}
	assertNoLeakedInternals(t, body)
}

func assertLogoutInvalidatesSession(t *testing.T, ctx context.Context, client *http.Client, cfg authGatewayRedisSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodDelete, cfg.gatewayBaseURL+"/api/v1/sessions/current", session.AccessToken, requestID, nil)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("logout request failed for requestId=%s: %v", requestID, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout returned %d for requestId=%s", resp.StatusCode, requestID)
	}

	req = gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/users/me", session.AccessToken, requestID+"_old_token", nil)
	resp, err = client.Do(req.WithContext(ctx))
	if err != nil {
		t.Fatalf("old-token current user request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old token returned %d: %s", resp.StatusCode, responseBodySummary(body))
	}
	assertErrorEnvelope(t, body, requestID+"_old_token", "unauthorized")
}

func assertErrorEnvelope(t *testing.T, body []byte, requestID string, code string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error envelope: %v %s", err, responseBodySummary(body))
	}
	if envelope.Error.Code != code {
		t.Fatalf("error code mismatch: want=%q got=%q %s", code, envelope.Error.Code, responseBodySummary(body))
	}
	if envelope.Error.RequestID != requestID {
		t.Fatalf("error requestId mismatch: want=%q got=%q", requestID, envelope.Error.RequestID)
	}
}

func assertRedisSessionCacheSafe(t *testing.T, ctx context.Context, cfg authGatewayRedisSmokeConfig, requestID string, accessToken string) string {
	t.Helper()
	client := openRedisSmokeClient(t, ctx, cfg)
	defer client.Close()

	keysValue, err := client.Do(ctx, "KEYS", "gateway:session:*")
	if err != nil {
		t.Fatalf("redis KEYS gateway:session:* failed for requestId=%s: %v", requestID, err)
	}
	keys := redisStringArray(keysValue)
	for _, key := range keys {
		value, err := client.Do(ctx, "GET", key)
		if err != nil {
			t.Fatalf("redis GET session cache key failed for requestId=%s: %v", requestID, err)
		}
		payload, ok := value.(string)
		if !ok || payload == "" {
			continue
		}
		var entry struct {
			Username        string `json:"username"`
			AccessTokenHash string `json:"accessTokenHash"`
			AccessToken     string `json:"accessToken"`
			RequestID       string `json:"requestId"`
		}
		if err := json.Unmarshal([]byte(payload), &entry); err != nil {
			continue
		}
		if entry.RequestID != requestID {
			continue
		}
		if entry.Username == "" || entry.AccessTokenHash == "" {
			t.Fatalf("redis session cache missing username or accessTokenHash for requestId=%s", requestID)
		}
		if entry.AccessToken != "" || strings.Contains(payload, accessToken) || strings.Contains(strings.ToLower(payload), `"accesstoken":`) {
			t.Fatalf("redis session cache leaked raw access token for requestId=%s", requestID)
		}
		if !strings.HasPrefix(key, "gateway:session:hmac-sha256:") {
			t.Fatalf("redis session key prefix mismatch: %q", key)
		}
		ttlValue, err := client.Do(ctx, "TTL", key)
		if err != nil {
			t.Fatalf("redis TTL session cache key failed for requestId=%s: %v", requestID, err)
		}
		if ttl, ok := ttlValue.(int64); !ok || ttl <= 0 {
			t.Fatalf("redis session TTL must be positive for requestId=%s, got %#v", requestID, ttlValue)
		}
		return key
	}
	t.Fatalf("redis session cache entry not found for requestId=%s", requestID)
	return ""
}

func assertRedisSessionDeleted(t *testing.T, ctx context.Context, cfg authGatewayRedisSmokeConfig, key string) {
	t.Helper()
	if strings.TrimSpace(key) == "" {
		return
	}
	client := openRedisSmokeClient(t, ctx, cfg)
	defer client.Close()
	value, err := client.Do(ctx, "GET", key)
	if err != nil {
		t.Fatalf("redis GET after logout failed: %v", err)
	}
	if value != nil {
		t.Fatalf("redis session key still exists after logout: %s", key)
	}
}

type redisSmokeClient struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

func openRedisSmokeClient(t *testing.T, ctx context.Context, cfg authGatewayRedisSmokeConfig) *redisSmokeClient {
	t.Helper()
	addr := strings.TrimSpace(cfg.redisAddr)
	if addr == "" {
		t.Fatalf("redis address is required for AUTH_GATEWAY_REDIS_SMOKE")
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Fatalf("redis is not reachable at %s: %v", addr, err)
	}
	client := &redisSmokeClient{conn: conn, rw: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))}
	if cfg.redisPassword != "" {
		if _, err := client.Do(ctx, "AUTH", cfg.redisPassword); err != nil {
			client.Close()
			t.Fatalf("redis AUTH failed")
		}
	}
	if cfg.redisDB > 0 {
		if _, err := client.Do(ctx, "SELECT", strconv.Itoa(cfg.redisDB)); err != nil {
			client.Close()
			t.Fatalf("redis SELECT failed: %v", err)
		}
	}
	if _, err := client.Do(ctx, "PING"); err != nil {
		client.Close()
		t.Fatalf("redis PING failed: %v", err)
	}
	return client
}

func (c *redisSmokeClient) Close() {
	if c == nil || c.conn == nil {
		return
	}
	_ = c.conn.Close()
}

func (c *redisSmokeClient) Do(ctx context.Context, args ...string) (any, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("redis client is closed")
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
	} else {
		_ = c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	if err := writeRedisCommand(c.rw, args...); err != nil {
		return nil, err
	}
	return readRedisValue(c.rw.Reader)
}

func writeRedisCommand(rw *bufio.ReadWriter, args ...string) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, arg := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(arg), arg)
	}
	if _, err := rw.Write(b.Bytes()); err != nil {
		return err
	}
	return rw.Flush()
}

func readRedisValue(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		return readRedisLine(reader)
	case '-':
		line, _ := readRedisLine(reader)
		return nil, fmt.Errorf("redis error: %s", line)
	case ':':
		line, err := readRedisLine(reader)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(line, 10, 64)
	case '$':
		line, err := readRedisLine(reader)
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, nil
		}
		data := make([]byte, size+2)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}
		return string(data[:size]), nil
	case '*':
		line, err := readRedisLine(reader)
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, nil
		}
		values := make([]any, 0, size)
		for i := 0; i < size; i++ {
			value, err := readRedisValue(reader)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}

func readRedisLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func redisStringArray(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func TestAuthGatewayRedisSmokeConfigDefaults(t *testing.T) {
	t.Setenv(authGatewayRedisSmokeGate, "1")
	t.Setenv("GATEWAY_BASE_URL", "http://127.0.0.1:8080/")
	t.Setenv("LOCAL_ADMIN_USERNAME", "admin")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "password")
	t.Setenv("GATEWAY_REDIS_ADDR", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("AUTH_GATEWAY_REDIS_ADDR", "")
	t.Setenv("GATEWAY_REDIS_PASSWORD", "")
	t.Setenv("GATEWAY_REDIS_DB", "")
	cfg := loadAuthGatewayRedisSmokeConfig(t)
	if cfg.gatewayBaseURL != "http://127.0.0.1:8080" || cfg.redisAddr != "127.0.0.1:6379" {
		t.Fatalf("config=%+v", cfg)
	}
}
