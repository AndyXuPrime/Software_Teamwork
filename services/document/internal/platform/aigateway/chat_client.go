package aigateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

const maxChatResponseBytes = 2 << 20

type ChatClient struct {
	baseURL          trustedBaseURL
	serviceToken     string
	defaultProfileID string
	defaultModel     string
	httpClient       *http.Client
}

func NewChatClient(baseURL, serviceToken, defaultProfileID, defaultModel string, httpClient *http.Client) (*ChatClient, error) {
	normalized, err := validateAIGatewayBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(defaultProfileID) == "" {
		return nil, errors.New("DOCUMENT_AI_GATEWAY_PROFILE_ID is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultChatTimeout}
	}
	return &ChatClient{
		baseURL:          normalized,
		serviceToken:     strings.TrimSpace(serviceToken),
		defaultProfileID: strings.TrimSpace(defaultProfileID),
		defaultModel:     strings.TrimSpace(defaultModel),
		httpClient:       httpClient,
	}, nil
}

func (c *ChatClient) CreateChatCompletion(ctx context.Context, reqCtx service.RequestContext, input service.ChatCompletionRequest) (service.ChatCompletionResponse, error) {
	if len(input.Messages) == 0 {
		return service.ChatCompletionResponse{}, service.ValidationError(map[string]string{"messages": "must not be empty"})
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = c.defaultModel
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if profileID == "" {
		profileID = c.defaultProfileID
	}
	body := chatCompletionRequest{
		Model:       model,
		ProfileID:   profileID,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		TopP:        input.TopP,
		MaxTokens:   input.MaxTokens,
		Stream:      false,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeInternal, "encode ai gateway request", err)
	}
	endpoint, err := c.baseURL.Join("internal/v1/chat/completions")
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "build ai gateway chat request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "build ai gateway chat request", err)
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
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		if resp.StatusCode == http.StatusBadRequest {
			return service.ChatCompletionResponse{}, service.NewError(service.CodeValidation, "ai gateway rejected chat request", fmt.Errorf("status %d", resp.StatusCode))
		}
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat request failed", fmt.Errorf("status %d", resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxChatResponseBytes+1))
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "read ai gateway chat response", err)
	}
	if len(data) > maxChatResponseBytes {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat response too large", nil)
	}
	var decoded chatCompletionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "decode ai gateway chat response", err)
	}
	if len(decoded.Choices) == 0 {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat response has no choices", nil)
	}
	choice := decoded.Choices[0]
	if strings.TrimSpace(choice.Message.Content) == "" {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat response content is empty", nil)
	}
	return service.ChatCompletionResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: service.ChatTokenUsage{
			PromptTokens:     decoded.Usage.PromptTokens,
			CompletionTokens: decoded.Usage.CompletionTokens,
			TotalTokens:      decoded.Usage.TotalTokens,
		},
	}, nil
}

func (c *ChatClient) StreamChatCompletion(ctx context.Context, reqCtx service.RequestContext, input service.ChatCompletionRequest, onDelta func(string)) (service.ChatCompletionResponse, error) {
	if len(input.Messages) == 0 {
		return service.ChatCompletionResponse{}, service.ValidationError(map[string]string{"messages": "must not be empty"})
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = c.defaultModel
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if profileID == "" {
		profileID = c.defaultProfileID
	}
	body := chatCompletionRequest{
		Model:       model,
		ProfileID:   profileID,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		TopP:        input.TopP,
		MaxTokens:   input.MaxTokens,
		Stream:      true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeInternal, "encode ai gateway request", err)
	}
	endpoint, err := c.baseURL.Join("internal/v1/chat/completions")
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "build ai gateway chat request", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "build ai gateway chat request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	c.setContextHeaders(req, reqCtx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat stream request failed", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		if resp.StatusCode == http.StatusBadRequest {
			return service.ChatCompletionResponse{}, service.NewError(service.CodeValidation, "ai gateway rejected chat request", fmt.Errorf("status %d", resp.StatusCode))
		}
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat stream request failed", fmt.Errorf("status %d", resp.StatusCode))
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat stream response is not event-stream", nil)
	}

	var builder strings.Builder
	var finishReason string
	var usage service.ChatTokenUsage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxChatResponseBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk chatCompletionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "decode ai gateway chat stream chunk", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				builder.WriteString(choice.Delta.Content)
				if builder.Len() > maxChatResponseBytes {
					return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat stream response too large", nil)
				}
				if onDelta != nil {
					onDelta(choice.Delta.Content)
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
		if chunk.Usage.TotalTokens != 0 || chunk.Usage.PromptTokens != 0 || chunk.Usage.CompletionTokens != 0 {
			usage = service.ChatTokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "read ai gateway chat stream response", err)
	}
	content := builder.String()
	if strings.TrimSpace(content) == "" {
		return service.ChatCompletionResponse{}, service.NewError(service.CodeDependency, "ai gateway chat response content is empty", nil)
	}
	return service.ChatCompletionResponse{
		Content:      content,
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func (c *ChatClient) setContextHeaders(req *http.Request, reqCtx service.RequestContext) {
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
}

type chatCompletionRequest struct {
	Model       string                `json:"model,omitempty"`
	ProfileID   string                `json:"profile_id"`
	Messages    []service.ChatMessage `json:"messages"`
	Temperature *float64              `json:"temperature,omitempty"`
	TopP        *float64              `json:"top_p,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Stream      bool                  `json:"stream"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message      service.ChatMessage `json:"message"`
		FinishReason string              `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type chatCompletionStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
