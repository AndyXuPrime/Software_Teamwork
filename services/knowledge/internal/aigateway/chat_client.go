package aigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	callerService        = "knowledge"
	defaultTimeout       = 60 * time.Second
	maxChatResponseBytes = 2 << 20
)

// Message is a chat completion message for AI Gateway.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest carries parameters for a non-streaming chat completion.
type ChatRequest struct {
	Model       string
	ProfileID   string
	Messages    []Message
	MaxTokens   int
	Temperature *float64
}

// ChatResponse is the sanitized completion result from AI Gateway.
type ChatResponse struct {
	Content      string
	FinishReason string
}

// RequestContext carries tracing and caller identity headers.
type RequestContext struct {
	RequestID   string
	UserID      string
	Roles       []string
	Permissions []string
}

// ChatClient calls AI Gateway internal chat/completions.
type ChatClient struct {
	baseURL      string
	serviceToken string
	httpClient   *http.Client
}

// NewChatClient creates a client for the given AI Gateway base URL.
func NewChatClient(baseURL, serviceToken string, httpClient *http.Client) (*ChatClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("KNOWLEDGE_AI_GATEWAY_URL must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return nil, errors.New("KNOWLEDGE_AI_GATEWAY_URL must not contain credentials")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &ChatClient{
		baseURL:      strings.TrimRight(parsed.String(), "/"),
		serviceToken: strings.TrimSpace(serviceToken),
		httpClient:   httpClient,
	}, nil
}

// NewChatClientFromEnv loads configuration from KNOWLEDGE_AI_GATEWAY_URL and
// KNOWLEDGE_AI_GATEWAY_SERVICE_TOKEN (falling back to INTERNAL_SERVICE_TOKEN).
// Returns nil, nil when the gateway URL is not configured.
func NewChatClientFromEnv() (*ChatClient, error) {
	baseURL := strings.TrimSpace(os.Getenv("KNOWLEDGE_AI_GATEWAY_URL"))
	if baseURL == "" {
		return nil, nil
	}
	token := firstEnv("KNOWLEDGE_AI_GATEWAY_SERVICE_TOKEN", "INTERNAL_SERVICE_TOKEN")
	return NewChatClient(baseURL, token, nil)
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

// CreateChatCompletion posts to AI Gateway /internal/v1/chat/completions.
func (c *ChatClient) CreateChatCompletion(ctx context.Context, reqCtx RequestContext, input ChatRequest) (ChatResponse, error) {
	if c == nil {
		return ChatResponse{}, errors.New("ai gateway client is not configured")
	}
	if len(input.Messages) == 0 {
		return ChatResponse{}, errors.New("messages must not be empty")
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if profileID == "" {
		return ChatResponse{}, errors.New("modelProfileId is required")
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = profileID
	}

	body := chatCompletionRequest{
		Model:       model,
		ProfileID:   profileID,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		MaxTokens:   input.MaxTokens,
		Stream:      false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("encode ai gateway request: %w", err)
	}

	endpoint, err := url.JoinPath(c.baseURL, "internal/v1/chat/completions")
	if err != nil {
		return ChatResponse{}, fmt.Errorf("build ai gateway chat request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("build ai gateway chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
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
		return ChatResponse{}, fmt.Errorf("ai gateway chat request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return ChatResponse{}, fmt.Errorf("ai gateway chat request failed with status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxChatResponseBytes+1))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read ai gateway chat response: %w", err)
	}
	if len(data) > maxChatResponseBytes {
		return ChatResponse{}, errors.New("ai gateway chat response too large")
	}
	var decoded chatCompletionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return ChatResponse{}, fmt.Errorf("decode ai gateway chat response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return ChatResponse{}, errors.New("ai gateway chat response has no choices")
	}
	choice := decoded.Choices[0]
	return ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
	}, nil
}

type chatCompletionRequest struct {
	Model       string    `json:"model"`
	ProfileID   string    `json:"profile_id"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}
