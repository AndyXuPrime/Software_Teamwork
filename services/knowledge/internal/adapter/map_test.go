package adapter

import (
	"encoding/json"
	"testing"
)

func TestBuildRetrievalBodyForwardsSearchParams(t *testing.T) {
	rerankTopN := 5
	body, err := buildRetrievalBody(knowledgeQueryRequest{
		Query:            "maintenance",
		KnowledgeBaseIDs: []string{"kb_1"},
		DocumentIDs:      []string{"doc_1", "doc_2"},
		TopK:             8,
		ScoreThreshold:   ptrFloat64(0.4),
		Tags:             []string{"锅炉"},
		MetadataFilter:   map[string]string{"专业": "锅炉"},
		Rerank:           true,
		RerankTopN:       &rerankTopN,
	}, retrievalBuildOptions{VendorRerankID: "BAAI/bge-reranker-v2-m3"})
	if err != nil {
		t.Fatalf("buildRetrievalBody: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["question"] != "maintenance" {
		t.Fatalf("question=%v", payload["question"])
	}
	if got, _ := payload["dataset_ids"].([]any); len(got) != 1 || got[0] != "kb_1" {
		t.Fatalf("dataset_ids=%v", payload["dataset_ids"])
	}
	if got, _ := payload["doc_ids"].([]any); len(got) != 2 {
		t.Fatalf("doc_ids=%v", payload["doc_ids"])
	}
	if payload["top_k"].(float64) != 8 {
		t.Fatalf("top_k=%v", payload["top_k"])
	}
	if payload["similarity_threshold"].(float64) != 0.4 {
		t.Fatalf("similarity_threshold=%v", payload["similarity_threshold"])
	}
	if payload["rerank_id"] != "BAAI/bge-reranker-v2-m3" {
		t.Fatalf("rerank_id=%v", payload["rerank_id"])
	}
	if payload["size"].(float64) != 5 {
		t.Fatalf("size=%v", payload["size"])
	}

	filter, ok := payload["meta_data_filter"].(map[string]any)
	if !ok {
		t.Fatalf("meta_data_filter=%v", payload["meta_data_filter"])
	}
	if filter["method"] != "manual" {
		t.Fatalf("method=%v", filter["method"])
	}
	manual, ok := filter["manual"].([]any)
	if !ok || len(manual) != 2 {
		t.Fatalf("manual=%v", filter["manual"])
	}
}

func TestBuildRetrievalBodyOmitsRerankWithoutVendorModel(t *testing.T) {
	body, err := buildRetrievalBody(knowledgeQueryRequest{
		Query:            "q",
		KnowledgeBaseIDs: []string{"kb_1"},
		Rerank:           true,
	}, retrievalBuildOptions{})
	if err != nil {
		t.Fatalf("buildRetrievalBody: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["rerank_id"]; ok {
		t.Fatalf("rerank_id should be omitted without vendor rerank id: %v", payload)
	}
}

func ptrFloat64(v float64) *float64 {
	return &v
}
