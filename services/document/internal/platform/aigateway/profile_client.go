package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

const defaultTimeout = 10 * time.Second
const callerService = "document"
const aiGatewayPort = "8086"

type ProfileClient struct {
	baseURL      trustedBaseURL
	serviceToken string
	httpClient   *http.Client
}

func NewProfileClient(baseURL, serviceToken string, httpClient *http.Client) (*ProfileClient, error) {
	normalized, err := validateAIGatewayBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &ProfileClient{
		baseURL:      normalized,
		serviceToken: strings.TrimSpace(serviceToken),
		httpClient:   httpClient,
	}, nil
}

type trustedBaseURL string

func validateAIGatewayBaseURL(raw string) (trustedBaseURL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL must not contain query or fragment")
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path != "" && path != "/internal/v1" {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL must be an AI Gateway service base URL")
	}
	if !trustedInternalHost(parsed.Hostname()) {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL host is not trusted")
	}
	if port := parsed.Port(); port != "" && port != aiGatewayPort {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL port is not trusted")
	}
	base, ok := canonicalBaseURL(parsed.Scheme, parsed.Hostname())
	if !ok {
		return "", errors.New("DOCUMENT_AI_GATEWAY_URL host is not trusted")
	}
	return base, nil
}

func trustedInternalHost(host string) bool {
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

func canonicalBaseURL(schemeValue, hostValue string) (trustedBaseURL, bool) {
	hostValue = strings.Trim(strings.ToLower(hostValue), "[]")
	if schemeValue == "https" {
		switch hostValue {
		case "localhost":
			return "https://localhost:8086", true
		case "ai-gateway":
			return "https://ai-gateway:8086", true
		case "127.0.0.1":
			return "https://127.0.0.1:8086", true
		case "::1":
			return "https://[::1]:8086", true
		}
		return "", false
	}
	switch hostValue {
	case "localhost":
		return "http://localhost:8086", true
	case "ai-gateway":
		return "http://ai-gateway:8086", true
	case "127.0.0.1":
		return "http://127.0.0.1:8086", true
	case "::1":
		return "http://[::1]:8086", true
	}
	return "", false
}

func (b trustedBaseURL) Join(elem ...string) (string, error) {
	return url.JoinPath(string(b), elem...)
}

func (c *ProfileClient) GetModelProfile(ctx context.Context, reqCtx service.RequestContext, id string) (service.ModelProfileReference, error) {
	profileID := strings.TrimSpace(id)
	if profileID == "" {
		return service.ModelProfileReference{}, service.ValidationError(map[string]string{"llm.profileId": "is required"})
	}
	endpoint, err := c.baseURL.Join("internal/v1/model-profiles", profileID)
	if err != nil {
		return service.ModelProfileReference{}, service.NewError(service.CodeDependency, "build ai gateway profile request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return service.ModelProfileReference{}, service.NewError(service.CodeDependency, "build ai gateway profile request", err)
	}
	if c.serviceToken != "" {
		req.Header.Set("X-Service-Token", c.serviceToken)
	}
	req.Header.Set("X-Caller-Service", callerService)
	if strings.TrimSpace(reqCtx.RequestID) != "" {
		req.Header.Set("X-Request-Id", strings.TrimSpace(reqCtx.RequestID))
	}
	if strings.TrimSpace(reqCtx.UserID) != "" {
		req.Header.Set("X-User-Id", strings.TrimSpace(reqCtx.UserID))
	}
	if len(reqCtx.Roles) > 0 {
		req.Header.Set("X-User-Roles", strings.Join(reqCtx.Roles, ","))
	}
	if len(reqCtx.Permissions) > 0 {
		req.Header.Set("X-User-Permissions", strings.Join(reqCtx.Permissions, ","))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return service.ModelProfileReference{}, service.NewError(service.CodeDependency, "ai gateway profile lookup failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return service.ModelProfileReference{}, service.NewError(service.CodeNotFound, "model profile not found", nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return service.ModelProfileReference{}, service.NewError(service.CodeDependency, "ai gateway profile lookup failed", fmt.Errorf("status %d", resp.StatusCode))
	}

	var envelope struct {
		Data struct {
			ID        string `json:"id"`
			Purpose   string `json:"purpose"`
			Provider  string `json:"provider"`
			Model     string `json:"model"`
			Enabled   bool   `json:"enabled"`
			TimeoutMS int    `json:"timeoutMs"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return service.ModelProfileReference{}, service.NewError(service.CodeDependency, "decode ai gateway profile response", err)
	}
	return service.ModelProfileReference{
		ID:             envelope.Data.ID,
		Purpose:        envelope.Data.Purpose,
		Provider:       envelope.Data.Provider,
		Model:          envelope.Data.Model,
		Enabled:        envelope.Data.Enabled,
		TimeoutSeconds: timeoutMSToSeconds(envelope.Data.TimeoutMS),
	}, nil
}

func timeoutMSToSeconds(timeoutMS int) int {
	if timeoutMS <= 0 {
		return 0
	}
	return (timeoutMS + 999) / 1000
}
