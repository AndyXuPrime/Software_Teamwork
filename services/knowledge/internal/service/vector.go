package service

import "context"

type RerankDocument struct {
	ID   string
	Text string
}

type RerankRequest struct {
	Query     string
	Documents []RerankDocument
	TopN      int
	UserID    string
	RequestID string
}

type RerankResult struct {
	DocumentID string
	Score      float64
}

// Reranker is the provider-neutral boundary for reranking. Tests can inject a
// deterministic fake; production wiring can use an AI Gateway HTTP adapter.
type Reranker interface {
	Rerank(ctx context.Context, request RerankRequest) ([]RerankResult, error)
}

type VectorSearchRequest struct {
	Vector           []float32
	KnowledgeBaseIDs []string
	Tags             []string
	MetadataFilter   map[string]string
	Limit            int
	ScoreThreshold   float64
}

type VectorSearchHit struct {
	ID      string
	Score   float64
	Payload map[string]any
}
