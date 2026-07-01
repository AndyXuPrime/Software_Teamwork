package aigatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

const (
	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 2 << 20 // 2 MiB
	callerService    = "document"
	aiGatewayPort    = "8086"
)

// trustedBaseURL is a validated, normalised AI Gateway base URL that is safe
// to use as an HTTP request target. The type prevents unvalidated strings from
// reaching the HTTP client.
type trustedBaseURL string

func (b trustedBaseURL) join(elem ...string) (string, error) {
	return url.JoinPath(string(b), elem...)
}

// Client is a lightweight HTTP client for the AI Gateway chat completions
// endpoint. It enforces the same trusted-host URL policy as the rest of the
// Document service and embeds a default profile/model for each call.
type Client struct {
	baseURL          trustedBaseURL
	defaultProfileID string
	defaultModel     string
	httpClient       *http.Client
}

// New validates baseURL against the Document service AI Gateway URL policy and
// returns a ready Client. profileID is required; model falls back to profileID
// when empty. Pass a non-nil httpClient to override the default (e.g., in tests).
func New(baseURL, profileID, model string, httpClient *http.Client) (*Client, error) {
	normalized, err := validateBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, errors.New("aigatewayclient: profileID must not be empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = profileID
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		baseURL:          normalized,
		defaultProfileID: profileID,
		defaultModel:     model,
		httpClient:       httpClient,
	}, nil
}

// ChatCompletion sends messages to the AI Gateway and returns the first
// choice's content string.
//
// requestID is forwarded as X-Request-Id when non-empty.
// serviceToken is forwarded as X-Service-Token when non-empty.
// Non-2xx responses, invalid JSON, empty choices and whitespace-only content
// all map to service.CodeDependency; downstream body is never included in the
// returned error message.
func (c *Client) ChatCompletion(ctx context.Context, requestID, serviceToken string, messages []service.ChatMessage) (string, error) {
	reqMsgs := make([]chatMessage, len(messages))
	for i, m := range messages {
		reqMsgs[i] = chatMessage{Role: m.Role, Content: m.Content}
	}
	raw, err := json.Marshal(chatRequest{
		Model:     c.defaultModel,
		ProfileID: c.defaultProfileID,
		Messages:  reqMsgs,
	})
	if err != nil {
		return "", service.NewError(service.CodeInternal, "encode chat request", err)
	}
	endpoint, err := c.baseURL.join("internal/v1/chat/completions")
	if err != nil {
		return "", service.NewError(service.CodeDependency, "build chat endpoint", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", service.NewError(service.CodeDependency, "build chat request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Caller-Service", callerService)
	if id := strings.TrimSpace(requestID); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	if token := strings.TrimSpace(serviceToken); token != "" {
		req.Header.Set("X-Service-Token", token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", service.NewError(service.CodeDependency, "chat request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", service.NewError(service.CodeDependency, "chat request failed", fmt.Errorf("status %d", resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", service.NewError(service.CodeDependency, "read chat response", err)
	}
	if len(data) > maxResponseBytes {
		return "", service.NewError(service.CodeDependency, "chat response too large", nil)
	}
	var decoded chatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", service.NewError(service.CodeDependency, "decode chat response", err)
	}
	if len(decoded.Choices) == 0 {
		return "", service.NewError(service.CodeDependency, "chat response has no choices", nil)
	}
	content := decoded.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", service.NewError(service.CodeDependency, "chat response content is empty", nil)
	}
	return content, nil
}

// ── internal request / response types ────────────────────────────────────────

type chatRequest struct {
	Model     string        `json:"model"`
	ProfileID string        `json:"profile_id"`
	Messages  []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// ── URL validation ────────────────────────────────────────────────────────────

// validateBaseURL enforces the same AI Gateway URL policy used by all Document
// service platform clients: absolute http(s) URL, trusted internal host only,
// standard port (8086 or omitted), no credentials, no query string or fragment,
// clean base path (empty or /internal/v1).
func validateBaseURL(raw string) (trustedBaseURL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("aigatewayclient: baseURL must not be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("aigatewayclient: baseURL must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return "", errors.New("aigatewayclient: baseURL must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("aigatewayclient: baseURL must not contain query or fragment")
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path != "" && path != "/internal/v1" {
		return "", errors.New("aigatewayclient: baseURL must be an AI Gateway service base URL")
	}
	if !trustedHost(parsed.Hostname()) {
		return "", errors.New("aigatewayclient: baseURL host is not trusted")
	}
	if port := parsed.Port(); port != "" && port != aiGatewayPort {
		return "", errors.New("aigatewayclient: baseURL port is not trusted")
	}
	base, ok := canonicalBase(parsed.Scheme, parsed.Hostname())
	if !ok {
		return "", errors.New("aigatewayclient: baseURL host is not trusted")
	}
	return base, nil
}

func trustedHost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "" {
		return false
	}
	switch host {
	case "localhost", "ai-gateway":
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func canonicalBase(scheme, host string) (trustedBaseURL, bool) {
	host = strings.Trim(strings.ToLower(host), "[]")
	prefix := scheme + "://"
	switch host {
	case "localhost":
		return trustedBaseURL(prefix + "localhost:" + aiGatewayPort), true
	case "ai-gateway":
		return trustedBaseURL(prefix + "ai-gateway:" + aiGatewayPort), true
	case "127.0.0.1":
		return trustedBaseURL(prefix + "127.0.0.1:" + aiGatewayPort), true
	case "::1":
		return trustedBaseURL(prefix + "[::1]:" + aiGatewayPort), true
	}
	return "", false
}
