package knowledgeclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestRetrievePropagatesTrustedContextAndMapsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/knowledge-queries" {
			t.Errorf("path=%q", r.URL.Path)
		}
		for name, want := range map[string]string{"X-Service-Token": "service-token", "X-Caller-Service": "qa", "X-User-Id": "user-1", "X-Request-Id": "req-knowledge-test"} {
			if got := r.Header.Get(name); got != want {
				t.Errorf("%s=%q want %q", name, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"results":[{"score":0.9,"knowledgeBaseId":"kb-1","documentId":"doc-1","chunkId":"chunk-1","documentName":"guide","contentPreview":"preview"}]},"requestId":"req-knowledge-test"}`))
	}))
	defer server.Close()
	client, err := New(server.URL, "service-token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx := service.WithRequestID(context.Background(), "req-knowledge-test")
	results, err := client.Retrieve(ctx, "user-1", service.RetrievalTestInput{Question: "query", KnowledgeBaseIDs: []string{"kb-1"}, Retrieval: service.RetrievalSettings{TopK: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].DocumentID != "doc-1" {
		t.Fatalf("results=%+v", results)
	}
}

func TestCheckCitationSourcesPropagatesContextAndMapsVisibility(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, want := range map[string]string{"X-Service-Token": "service-token", "X-Caller-Service": "qa", "X-User-Id": "user-1", "X-Request-Id": "req-citation-source"} {
			if got := r.Header.Get(name); got != want {
				t.Errorf("%s=%q want %q", name, got, want)
			}
		}
		seen[r.URL.Path] = true
		switch r.URL.Path {
		case "/internal/v1/documents/doc-1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"doc-1"}}`))
		case "/internal/v1/documents/doc-missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"not_found"}}`))
		default:
			t.Errorf("unexpected path=%q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := New(server.URL, "service-token", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx := service.WithRequestID(context.Background(), "req-citation-source")
	availability, err := client.CheckCitationSources(ctx, "user-1", []string{"doc-1", "doc-missing", "doc-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !availability["doc-1"] || availability["doc-missing"] {
		t.Fatalf("availability=%+v", availability)
	}
	if !seen["/internal/v1/documents/doc-1"] || !seen["/internal/v1/documents/doc-missing"] {
		t.Fatalf("paths were not checked: %+v", seen)
	}
}
