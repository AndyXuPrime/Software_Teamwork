package vendorclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientUsesKnowledgeRuntimeContractPaths(t *testing.T) {
	type call struct {
		method string
		path   string
		query  string
		tenant string
	}

	var calls []call
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			tenant: r.Header.Get("X-Tenant-Id"),
		})

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/system/ping":
			_, _ = w.Write([]byte("pong"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/datasets":
			writeTestVendorJSON(w, `{"code":0,"data":[],"total_datasets":0}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/datasets/kb_1/documents" && r.URL.Query().Get("type") == "local":
			if err := r.ParseMultipartForm(1024); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			writeTestVendorJSON(w, `{"code":0,"data":{"id":"doc_1","name":"notes.txt"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/datasets/kb_1/documents":
			writeTestVendorJSON(w, `{"code":0,"data":{"total":1,"docs":[{"id":"doc_1","name":"notes.txt","kb_id":"kb_1"}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/datasets/kb_1/documents/parse":
			writeTestVendorJSON(w, `{"code":0,"data":{}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/datasets/kb_1/documents/doc_1/chunks":
			writeTestVendorJSON(w, `{"code":0,"data":{"total":0,"chunks":[]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/datasets/search":
			writeTestVendorJSON(w, `{"code":0,"data":{"total":0,"chunks":[]}}`)
		default:
			t.Fatalf("unexpected vendor request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	ctx := context.Background()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if _, _, err := client.ListDatasets(ctx, "tenant_1", 2, 10); err != nil {
		t.Fatalf("ListDatasets: %v", err)
	}
	if _, err := client.UploadDocument(ctx, "tenant_1", "kb_1", "notes.txt", "text/plain", strings.NewReader("hello")); err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if err := client.StartDocumentParse(ctx, "tenant_1", "kb_1", []string{"doc_1"}); err != nil {
		t.Fatalf("StartDocumentParse: %v", err)
	}
	if _, err := client.GetDatasetDocument(ctx, "tenant_1", "kb_1", "doc_1"); err != nil {
		t.Fatalf("GetDatasetDocument: %v", err)
	}
	if _, _, err := client.ListChunks(ctx, "tenant_1", "kb_1", "doc_1", 1, 20); err != nil {
		t.Fatalf("ListChunks: %v", err)
	}
	if _, err := client.RetrievalSearch(ctx, "tenant_1", []byte(`{"question":"hello","dataset_ids":["kb_1"]}`)); err != nil {
		t.Fatalf("RetrievalSearch: %v", err)
	}

	expected := []call{
		{method: http.MethodGet, path: "/api/v1/system/ping"},
		{method: http.MethodGet, path: "/api/v1/datasets", query: "page=2&page_size=10", tenant: "tenant_1"},
		{method: http.MethodPost, path: "/api/v1/datasets/kb_1/documents", query: "type=local", tenant: "tenant_1"},
		{method: http.MethodPost, path: "/api/v1/datasets/kb_1/documents/parse", tenant: "tenant_1"},
		{method: http.MethodGet, path: "/api/v1/datasets/kb_1/documents", query: "page=1&page_size=100", tenant: "tenant_1"},
		{method: http.MethodGet, path: "/api/v1/datasets/kb_1/documents/doc_1/chunks", query: "page=1&page_size=20", tenant: "tenant_1"},
		{method: http.MethodPost, path: "/api/v1/datasets/search", tenant: "tenant_1"},
	}

	if len(calls) != len(expected) {
		t.Fatalf("call count = %d, want %d: %#v", len(calls), len(expected), calls)
	}
	for i := range expected {
		if calls[i] != expected[i] {
			t.Fatalf("call[%d] = %#v, want %#v", i, calls[i], expected[i])
		}
	}
}

func writeTestVendorJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}
