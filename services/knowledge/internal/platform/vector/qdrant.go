package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type QdrantConfig struct {
	BaseURL    string
	APIKey     string
	Collection string
	Dimension  int
	HTTPClient *http.Client
}

type QdrantClient struct {
	baseURL    string
	apiKey     string
	collection string
	client     *http.Client
}

func NewQdrantClient(cfg QdrantConfig) (*QdrantClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("qdrant URL must be an absolute http(s) URL")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("qdrant URL must not contain credentials")
	}
	collection := strings.TrimSpace(cfg.Collection)
	if collection == "" {
		return nil, fmt.Errorf("qdrant collection is required")
	}
	return &QdrantClient{
		baseURL:    strings.TrimRight(parsed.String(), "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		collection: collection,
		client:     noRedirectHTTPClient(cfg.HTTPClient),
	}, nil
}

func noRedirectHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	} else {
		copied := *client
		client = &copied
	}
	// Qdrant requests may include api-key headers, vectors, and metadata. Do
	// not replay them to a redirect target; surface the 3xx as dependency error.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

func (c *QdrantClient) Upsert(ctx context.Context, points []service.VectorPoint) error {
	payload := qdrantUpsertRequest{Points: make([]qdrantPoint, 0, len(points))}
	for _, point := range points {
		payload.Points = append(payload.Points, qdrantPoint{
			ID:      point.ID,
			Vector:  append([]float32(nil), point.Vector...),
			Payload: cloneMap(point.Payload),
		})
	}
	return c.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(c.collection)+"/points?wait=true", payload, nil)
}

func (c *QdrantClient) DeleteByDocumentIngestionAttempt(ctx context.Context, documentID string, ingestionAttempt string) error {
	return c.deleteByFilter(ctx, qdrantFilter{Must: []qdrantCondition{
		matchValueCondition(service.VectorPayloadDocumentID, documentID),
		matchValueCondition(service.VectorPayloadIngestionAttempt, ingestionAttempt),
	}})
}

func (c *QdrantClient) DeleteStaleDocumentPoints(ctx context.Context, documentID string, activeIngestionAttempt string) error {
	return c.deleteByFilter(ctx, qdrantFilter{
		Must:    []qdrantCondition{matchValueCondition(service.VectorPayloadDocumentID, documentID)},
		MustNot: []qdrantCondition{matchValueCondition(service.VectorPayloadIngestionAttempt, activeIngestionAttempt)},
	})
}

func (c *QdrantClient) Search(ctx context.Context, request service.VectorSearchRequest) ([]service.VectorSearchHit, error) {
	payload := qdrantSearchRequest{
		Vector:         append([]float32(nil), request.Vector...),
		Limit:          request.Limit,
		ScoreThreshold: request.ScoreThreshold,
		WithPayload:    true,
		Filter:         qdrantSearchFilter(request),
	}
	var decoded qdrantSearchResponse
	if err := c.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(c.collection)+"/points/search", payload, &decoded); err != nil {
		return nil, err
	}
	hits := make([]service.VectorSearchHit, 0, len(decoded.Result))
	for _, item := range decoded.Result {
		hits = append(hits, service.VectorSearchHit{
			ID:      fmt.Sprint(item.ID),
			Score:   item.Score,
			Payload: cloneMap(item.Payload),
		})
	}
	return hits, nil
}

func (c *QdrantClient) deleteByFilter(ctx context.Context, filter qdrantFilter) error {
	payload := qdrantDeleteRequest{Filter: filter}
	return c.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(c.collection)+"/points/delete?wait=true", payload, nil)
}

func (c *QdrantClient) doJSON(ctx context.Context, method string, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return service.NewError(service.CodeDependency, "qdrant request could not be encoded", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return service.NewError(service.CodeDependency, "qdrant request failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return service.NewError(service.CodeDependency, "qdrant unavailable", err)
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
		return service.NewError(service.CodeDependency, "qdrant request failed", nil)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(out); err != nil {
		return service.NewError(service.CodeDependency, "qdrant response could not be decoded", err)
	}
	return nil
}

func qdrantSearchFilter(request service.VectorSearchRequest) *qdrantFilter {
	var must []qdrantCondition
	if len(request.KnowledgeBaseIDs) == 1 {
		must = append(must, matchValueCondition("knowledge_base_id", request.KnowledgeBaseIDs[0]))
	} else if len(request.KnowledgeBaseIDs) > 1 {
		must = append(must, matchAnyCondition("knowledge_base_id", request.KnowledgeBaseIDs))
	}
	for _, tag := range request.Tags {
		must = append(must, matchValueCondition("tags", tag))
	}
	for key, value := range request.MetadataFilter {
		must = append(must, matchValueCondition("metadata."+strings.TrimSpace(key), value))
	}
	if len(must) == 0 {
		return nil
	}
	return &qdrantFilter{Must: must}
}

func matchValueCondition(key string, value string) qdrantCondition {
	return qdrantCondition{
		Key:   strings.TrimSpace(key),
		Match: qdrantMatch{Value: strings.TrimSpace(value)},
	}
}

func matchAnyCondition(key string, values []string) qdrantCondition {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return qdrantCondition{
		Key:   strings.TrimSpace(key),
		Match: qdrantMatch{Any: normalized},
	}
}

type qdrantUpsertRequest struct {
	Points []qdrantPoint `json:"points"`
}

type qdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type qdrantDeleteRequest struct {
	Filter qdrantFilter `json:"filter"`
}

type qdrantSearchRequest struct {
	Vector         []float32     `json:"vector"`
	Limit          int           `json:"limit"`
	ScoreThreshold float64       `json:"score_threshold,omitempty"`
	WithPayload    bool          `json:"with_payload"`
	Filter         *qdrantFilter `json:"filter,omitempty"`
}

type qdrantFilter struct {
	Must    []qdrantCondition `json:"must,omitempty"`
	MustNot []qdrantCondition `json:"must_not,omitempty"`
}

type qdrantCondition struct {
	Key   string      `json:"key"`
	Match qdrantMatch `json:"match"`
}

type qdrantMatch struct {
	Value string   `json:"value,omitempty"`
	Any   []string `json:"any,omitempty"`
}

type qdrantSearchResponse struct {
	Result []struct {
		ID      any            `json:"id"`
		Score   float64        `json:"score"`
		Payload map[string]any `json:"payload"`
	} `json:"result"`
}

func cloneMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
