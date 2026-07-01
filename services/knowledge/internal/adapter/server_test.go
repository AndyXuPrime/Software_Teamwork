package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
)

func TestHealthz(t *testing.T) {
	vendor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: vendor.URL,
	}, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data=%v", payload["data"])
	}
	if data["service"] != "knowledge-adapter" {
		t.Fatalf("service=%v", data["service"])
	}
}

func TestReadyzVendorUnavailable(t *testing.T) {
	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: "http://127.0.0.1:1",
	}, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListKnowledgeBasesRequiresAuth(t *testing.T) {
	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: "http://127.0.0.1:1",
	}, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/v1/knowledge-bases", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListKnowledgeBasesMapsVendorResponse(t *testing.T) {
	vendor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/datasets" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("X-User-Id"); got != "usr_test" {
			t.Fatalf("X-User-Id=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":[{"id":"kb_1","name":"Docs","description":"demo","chunk_method":"naive","document_count":2,"chunk_count":10,"create_time":1700000000000}],"total_datasets":1}`))
	}))
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: vendor.URL,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/knowledge-bases", nil)
	req.Header.Set("X-User-Id", "usr_test")
	req.Header.Set("X-User-Permissions", "knowledge:read")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data []map[string]any `json:"data"`
		Page struct {
			Total int64 `json:"total"`
		} `json:"page"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0]["id"] != "kb_1" {
		t.Fatalf("data=%v", payload.Data)
	}
	if payload.Page.Total != 1 {
		t.Fatalf("total=%d", payload.Page.Total)
	}
}

func TestCreateKnowledgeQueryMapsRetrieval(t *testing.T) {
	vendor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/datasets/search" || r.Method != http.MethodPost {
			t.Fatalf("method=%s path=%s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"total":1,"chunks":[{"id":"chunk_1","doc_id":"doc_1","kb_id":"kb_1","similarity":0.91,"docnm_kwd":"readme.md","content_with_weight":"hello world"}]}}`))
	}))
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: vendor.URL,
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-queries", strings.NewReader(`{"query":"hello","knowledgeBaseIds":["kb_1"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "usr_test")
	req.Header.Set("X-User-Permissions", "knowledge:read")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Data struct {
			Results []map[string]any `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Data.Results) != 1 {
		t.Fatalf("results=%v", payload.Data.Results)
	}
	if payload.Data.Results[0]["chunkId"] != "chunk_1" {
		t.Fatalf("chunk=%v", payload.Data.Results[0])
	}
}

func TestNotFoundRoute(t *testing.T) {
	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: "http://127.0.0.1:1",
	}, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/v1/unknown", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListParserConfigsRequiresDatabase(t *testing.T) {
	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: "http://127.0.0.1:1",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/internal/v1/parser-configs", nil)
	req.Header.Set("X-User-Id", "usr_admin")
	req.Header.Set("X-User-Permissions", "knowledge:admin")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
