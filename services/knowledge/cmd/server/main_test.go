package main

import (
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/config"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestBuildRuntimeRetrievalDependencies(t *testing.T) {
	cfg := config.Config{
		EmbeddingProvider:  "ai_gateway",
		EmbeddingModel:     "embedding-model",
		EmbeddingDimension: 3,
		AIGatewayBaseURL:   "http://ai-gateway:8086",
		AIGatewayToken:     "svc-token",
		AIGatewayProfileID: "profile_embedding",
		QdrantURL:          "http://qdrant:6333",
		QdrantAPIKey:       "qdrant-token",
		QdrantCollection:   "knowledge_chunks_v1",
	}

	embedder, err := buildEmbedder(cfg)
	if err != nil {
		t.Fatalf("buildEmbedder() error = %v", err)
	}
	vectorIndex, err := buildVectorIndex(cfg)
	if err != nil {
		t.Fatalf("buildVectorIndex() error = %v", err)
	}
	if embedder == nil || vectorIndex == nil {
		t.Fatalf("retrieval dependencies were not wired: embedder=%T vectorIndex=%T", embedder, vectorIndex)
	}

	svc := service.NewKnowledgeService(nil, service.WithVectorIndex(embedder, vectorIndex, cfg.QdrantCollection))
	if svc == nil {
		t.Fatal("service was not created")
	}
}

func TestBuildVectorIndexUsesLocalFallbackWhenQdrantUnset(t *testing.T) {
	vectorIndex, err := buildVectorIndex(config.Config{QdrantCollection: config.DefaultQdrantCollection})
	if err != nil {
		t.Fatalf("buildVectorIndex() error = %v", err)
	}
	if vectorIndex == nil {
		t.Fatal("vector index = nil")
	}
}
