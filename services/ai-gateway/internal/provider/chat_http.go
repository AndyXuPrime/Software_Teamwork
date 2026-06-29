package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

const maxProviderErrorBodyBytes = int64(4096)

type HTTPChatClient struct {
	httpClient *http.Client
}

func NewHTTPChatClient(httpClient *http.Client) *HTTPChatClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 0}
	}
	return &HTTPChatClient{httpClient: httpClient}
}

func (c *HTTPChatClient) CompleteChat(ctx context.Context, req service.ProviderChatRequest) (service.ProviderChatResult, error) {
	body, err := json.Marshal(req.Payload)
	if err != nil {
		return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider request could not be encoded", "upstream_error", "dependency_error", 0, err)
	}
	httpReq, err := c.newRequest(ctx, req, bytes.NewReader(body))
	if err != nil {
		return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider request could not be created", "upstream_error", "dependency_error", 0, err)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider request timed out", "upstream_error", "timeout", 0, err)
		}
		return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider request failed", "upstream_error", "dependency_error", 0, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxProviderErrorBodyBytes))
		return service.ProviderChatResult{}, normalizeProviderStatus(resp.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider response could not be read", "upstream_error", "dependency_error", resp.StatusCode, err)
	}
	usage, err := validateChatCompletion(responseBody)
	if err != nil {
		return service.ProviderChatResult{}, providerError(http.StatusBadGateway, "provider returned a non-contract response", "upstream_error", "dependency_error", resp.StatusCode, err)
	}
	return service.ProviderChatResult{
		Body:               responseBody,
		Usage:              usage,
		ProviderStatusCode: resp.StatusCode,
	}, nil
}

func (c *HTTPChatClient) StreamChat(ctx context.Context, req service.ProviderChatRequest) (service.ProviderChatStream, error) {
	body, err := json.Marshal(req.Payload)
	if err != nil {
		return service.ProviderChatStream{}, providerError(http.StatusBadGateway, "provider request could not be encoded", "upstream_error", "dependency_error", 0, err)
	}
	httpReq, err := c.newRequest(ctx, req, bytes.NewReader(body))
	if err != nil {
		return service.ProviderChatStream{}, providerError(http.StatusBadGateway, "provider request could not be created", "upstream_error", "dependency_error", 0, err)
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return service.ProviderChatStream{}, providerError(http.StatusBadGateway, "provider request timed out", "upstream_error", "timeout", 0, err)
		}
		return service.ProviderChatStream{}, providerError(http.StatusBadGateway, "provider request failed", "upstream_error", "dependency_error", 0, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxProviderErrorBodyBytes))
		return service.ProviderChatStream{}, normalizeProviderStatus(resp.StatusCode)
	}
	return service.ProviderChatStream{
		Body:               resp.Body,
		ProviderStatusCode: resp.StatusCode,
	}, nil
}

func (c *HTTPChatClient) newRequest(ctx context.Context, req service.ProviderChatRequest, body io.Reader) (*http.Request, error) {
	endpoint, err := chatCompletionURL(req.Profile.BaseURL)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	if strings.TrimSpace(req.RequestID) != "" {
		httpReq.Header.Set("X-Request-Id", strings.TrimSpace(req.RequestID))
	}
	return httpReq, nil
}

func chatCompletionURL(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid provider base URL")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	return parsed.String(), nil
}

func validateChatCompletion(body []byte) (*service.TokenUsage, error) {
	var response struct {
		Object  string `json:"object"`
		Choices []any  `json:"choices"`
		Usage   *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if response.Object != "chat.completion" || len(response.Choices) == 0 {
		return nil, fmt.Errorf("missing chat completion fields")
	}
	if response.Usage == nil {
		return nil, nil
	}
	return &service.TokenUsage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}, nil
}

func normalizeProviderStatus(status int) error {
	switch status {
	case http.StatusUnauthorized:
		return providerError(http.StatusBadGateway, "provider authentication failed", "authentication_error", "dependency_error", status, nil)
	case http.StatusForbidden:
		return providerError(http.StatusForbidden, "provider permission denied", "permission_error", "forbidden", status, nil)
	case http.StatusTooManyRequests:
		return providerError(http.StatusTooManyRequests, "provider rate limit exceeded", "rate_limit_error", "rate_limited", status, nil)
	default:
		if status >= 500 {
			return providerError(http.StatusBadGateway, "provider is unavailable", "upstream_error", "dependency_error", status, nil)
		}
		return providerError(http.StatusBadGateway, "provider request failed", "upstream_error", "dependency_error", status, nil)
	}
}

func providerError(httpStatus int, message, errorType, code string, providerStatus int, err error) *service.OpenAIError {
	openErr := &service.OpenAIError{
		HTTPStatus: httpStatus,
		Message:    message,
		Type:       errorType,
		Code:       code,
		Err:        err,
	}
	if providerStatus > 0 {
		openErr.ProviderStatusCode = &providerStatus
	}
	return openErr
}

var _ service.ChatProvider = (*HTTPChatClient)(nil)
