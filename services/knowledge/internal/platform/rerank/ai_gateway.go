package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type AIGatewayConfig struct {
	BaseURL      string
	ServiceToken string
	Model        string
	ProfileID    string
	HTTPClient   *http.Client
}

type AIGatewayReranker struct {
	baseURL      string
	serviceToken string
	model        string
	profileID    string
	client       *http.Client
}

func NewAIGatewayReranker(cfg AIGatewayConfig) (*AIGatewayReranker, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("ai gateway base URL is required")
	}
	serviceToken := strings.TrimSpace(cfg.ServiceToken)
	if serviceToken == "" {
		return nil, fmt.Errorf("ai gateway service token is required")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		return nil, fmt.Errorf("ai gateway rerank model is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &AIGatewayReranker{
		baseURL:      baseURL,
		serviceToken: serviceToken,
		model:        model,
		profileID:    strings.TrimSpace(cfg.ProfileID),
		client:       client,
	}, nil
}

func (r *AIGatewayReranker) Rerank(ctx context.Context, request service.RerankRequest) ([]service.RerankResult, error) {
	documents := make([]aiGatewayRerankDocument, 0, len(request.Documents))
	for _, doc := range request.Documents {
		documents = append(documents, aiGatewayRerankDocument{
			ID:   strings.TrimSpace(doc.ID),
			Text: doc.Text,
		})
	}
	payload := aiGatewayRerankRequest{
		ProfileID: r.profileID,
		Model:     r.model,
		Query:     request.Query,
		Documents: documents,
		TopN:      request.TopN,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ai gateway rerank request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+"/internal/v1/rerankings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-Caller-Service", "knowledge")
	httpRequest.Header.Set("X-User-Id", strings.TrimSpace(request.UserID))
	httpRequest.Header.Set("X-Request-Id", strings.TrimSpace(request.RequestID))
	httpRequest.Header.Set("X-Service-Token", r.serviceToken)

	response, err := r.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("ai gateway reranking request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ai gateway reranking returned HTTP %d", response.StatusCode)
	}

	var decoded aiGatewayRerankResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode ai gateway rerank response: %w", err)
	}
	results := make([]service.RerankResult, 0, len(decoded.Data))
	for _, item := range decoded.Data {
		results = append(results, service.RerankResult{
			DocumentID: strings.TrimSpace(item.DocumentID),
			Score:      item.Score,
		})
	}
	return results, nil
}

type aiGatewayRerankRequest struct {
	ProfileID string                    `json:"profile_id,omitempty"`
	Model     string                    `json:"model"`
	Query     string                    `json:"query"`
	Documents []aiGatewayRerankDocument `json:"documents"`
	TopN      int                       `json:"top_n,omitempty"`
}

type aiGatewayRerankDocument struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type aiGatewayRerankResponse struct {
	Data []struct {
		DocumentID string  `json:"document_id"`
		Score      float64 `json:"score"`
	} `json:"data"`
}
