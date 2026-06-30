package service

import "context"

type EmbeddingRequest struct {
	Texts     []string
	UserID    string
	RequestID string
}

type EmbeddingResult struct {
	Vectors   [][]float32
	Provider  string
	Model     string
	Dimension int
}

type Embedder interface {
	Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResult, error)
}

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

type VectorPoint struct {
	ID      string
	Vector  []float32
	Payload map[string]any
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

type VectorIndex interface {
	Upsert(ctx context.Context, points []VectorPoint) error
	DeleteByDocumentIngestionAttempt(ctx context.Context, documentID string, ingestionAttempt string) error
	DeleteStaleDocumentPoints(ctx context.Context, documentID string, activeIngestionAttempt string) error
	Search(ctx context.Context, request VectorSearchRequest) ([]VectorSearchHit, error)
}
