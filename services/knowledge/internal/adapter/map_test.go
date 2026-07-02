package adapter

import (
	"encoding/json"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestBuildCreateDatasetBodyUsesDefaultParserConfigWhenChunkStrategyMissing(t *testing.T) {
	parserConfig := map[string]any{
		"layout_recognize": ragflowLayoutPaddleOCR,
		"chunk_token_num":  float64(1024),
	}
	body, err := buildCreateDatasetBody(createKnowledgeBaseRequest{Name: "Manuals"}, parserConfig)
	if err != nil {
		t.Fatalf("buildCreateDatasetBody: %v", err)
	}
	payload := decodeMap(t, body)
	cfg, ok := payload["parser_config"].(map[string]any)
	if !ok {
		t.Fatalf("parser_config=%v", payload["parser_config"])
	}
	if cfg["layout_recognize"] != ragflowLayoutPaddleOCR {
		t.Fatalf("layout_recognize=%v", cfg["layout_recognize"])
	}
}

func TestBuildCreateDatasetBodyPreservesExplicitChunkStrategy(t *testing.T) {
	explicit := json.RawMessage(`{"layout_recognize":"DeepDOC","chunk_token_num":256}`)
	body, err := buildCreateDatasetBody(createKnowledgeBaseRequest{
		Name:          "Manuals",
		ChunkStrategy: &explicit,
	}, map[string]any{"layout_recognize": ragflowLayoutPaddleOCR})
	if err != nil {
		t.Fatalf("buildCreateDatasetBody: %v", err)
	}
	payload := decodeMap(t, body)
	cfg, ok := payload["parser_config"].(map[string]any)
	if !ok {
		t.Fatalf("parser_config=%v", payload["parser_config"])
	}
	if cfg["layout_recognize"] != ragflowLayoutDeepDOC {
		t.Fatalf("layout_recognize=%v", cfg["layout_recognize"])
	}
}

func TestBuildUpdateDatasetBodyPreservesExplicitChunkStrategy(t *testing.T) {
	explicit := json.RawMessage(`{"layout_recognize":"OpenDataLoader"}`)
	body, err := buildUpdateDatasetBody(updateKnowledgeBaseRequest{ChunkStrategy: &explicit})
	if err != nil {
		t.Fatalf("buildUpdateDatasetBody: %v", err)
	}
	payload := decodeMap(t, body)
	cfg, ok := payload["parser_config"].(map[string]any)
	if !ok {
		t.Fatalf("parser_config=%v", payload["parser_config"])
	}
	if cfg["layout_recognize"] != ragflowLayoutOpenDataLoader {
		t.Fatalf("layout_recognize=%v", cfg["layout_recognize"])
	}
}

func TestRAGFlowParserConfigFromSnapshotMapsBackends(t *testing.T) {
	endpoint := "https://parser.internal/v1"
	tests := []struct {
		name              string
		snapshot          service.ParserConfigSnapshot
		wantLayout        string
		wantTokenFiltered bool
	}{
		{
			name: "builtin uses deepdoc",
			snapshot: service.ParserConfigSnapshot{
				ParserConfigID:        "parser_builtin",
				Backend:               service.ParserBackendBuiltin,
				Concurrency:           4,
				SupportedContentTypes: []string{"application/pdf"},
				DefaultParameters:     json.RawMessage(`{"chunk_token_num":1024}`),
			},
			wantLayout: ragflowLayoutDeepDOC,
		},
		{
			name: "local ocr uses paddleocr",
			snapshot: service.ParserConfigSnapshot{
				ParserConfigID:    "parser_local",
				Backend:           service.ParserBackendLocalOCR,
				Concurrency:       2,
				DefaultParameters: json.RawMessage(`{"chunk_token_num":768}`),
			},
			wantLayout: ragflowLayoutPaddleOCR,
		},
		{
			name: "remote compatible respects layoutRecognize parameter",
			snapshot: service.ParserConfigSnapshot{
				ParserConfigID:    "parser_remote",
				Backend:           service.ParserBackendRemoteCompatible,
				Concurrency:       8,
				EndpointURL:       &endpoint,
				DefaultParameters: json.RawMessage(`{"layoutRecognize":"MinerU","accessToken":"secret","chunk_token_num":2048}`),
			},
			wantLayout:        ragflowLayoutMinerU,
			wantTokenFiltered: true,
		},
		{
			name: "remote compatible defaults to paddleocr",
			snapshot: service.ParserConfigSnapshot{
				ParserConfigID:    "parser_remote_default",
				Backend:           service.ParserBackendRemoteCompatible,
				Concurrency:       4,
				DefaultParameters: json.RawMessage(`{"delimiter":"\n"}`),
			},
			wantLayout: ragflowLayoutPaddleOCR,
		},
		{
			name: "tika uses plain text",
			snapshot: service.ParserConfigSnapshot{
				ParserConfigID:    "parser_tika",
				Backend:           service.ParserBackendTika,
				Concurrency:       1,
				DefaultParameters: json.RawMessage(`{}`),
			},
			wantLayout: ragflowLayoutPlainText,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ragflowParserConfigFromSnapshot(tc.snapshot)
			if cfg["layout_recognize"] != tc.wantLayout {
				t.Fatalf("layout_recognize=%v want %s", cfg["layout_recognize"], tc.wantLayout)
			}
			trace, ok := cfg[parserConfigTraceKey].(map[string]any)
			if !ok {
				t.Fatalf("%s=%v", parserConfigTraceKey, cfg[parserConfigTraceKey])
			}
			if trace["backend"] != string(tc.snapshot.Backend) {
				t.Fatalf("trace backend=%v", trace["backend"])
			}
			if tc.snapshot.ParserConfigID != "" && trace["parserConfigId"] != tc.snapshot.ParserConfigID {
				t.Fatalf("trace parserConfigId=%v", trace["parserConfigId"])
			}
			if tc.wantTokenFiltered {
				if _, ok := cfg["accessToken"]; ok {
					t.Fatalf("sensitive accessToken should be filtered: %v", cfg)
				}
				if cfg["chunk_token_num"].(float64) != 2048 {
					t.Fatalf("chunk_token_num=%v", cfg["chunk_token_num"])
				}
				if trace["endpointUrl"] != endpoint {
					t.Fatalf("trace endpointUrl=%v", trace["endpointUrl"])
				}
			}
		})
	}
}

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

func decodeMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}
