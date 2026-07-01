package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestQdrantClientDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "redirect-target.test" {
			redirected = true
			if r.Header.Get("api-key") == "qdrant_secret" {
				t.Fatal("redirect target received qdrant api key")
			}
			return textResponse(http.StatusOK, "{}"), nil
		}
		return &http.Response{
			StatusCode: http.StatusPermanentRedirect,
			Header: http.Header{
				"Location": []string{"http://redirect-target.test/collections/kb_chunks/points?wait=true"},
			},
			Body: io.NopCloser(strings.NewReader("redirect")),
		}, nil
	})
	client, err := NewQdrantClient(QdrantConfig{
		BaseURL:    "http://qdrant.test",
		APIKey:     "qdrant_secret",
		Collection: "kb_chunks",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatalf("NewQdrantClient() error = %v", err)
	}

	err = client.Upsert(context.Background(), []service.VectorPoint{{
		ID:     "point_1",
		Vector: []float32{0.1, 0.2},
		Payload: map[string]any{
			"knowledge_base_id": "kb_1",
			"document_id":       "doc_1",
			"chunk_id":          "chunk_1",
		},
	}})
	if err == nil {
		t.Fatal("Upsert() error = nil, want redirect response error")
	}
	var appErr *service.AppError
	if !errors.As(err, &appErr) || appErr.Code != service.CodeDependency {
		t.Fatalf("Upsert() error = %v, want dependency AppError", err)
	}
	if redirected {
		t.Fatal("qdrant client followed redirect and risked forwarding api key or vector payload")
	}
}

func TestQdrantClientSearchSendsRequestAndMapsHits(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost || r.URL.Path != "/collections/knowledge_chunks/points/search" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("api-key") != "qdrant-token" {
			t.Fatalf("headers = %+v", r.Header)
		}
		var body qdrantSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Limit != 3 || body.ScoreThreshold != 0.35 || len(body.Vector) != 2 || !body.WithPayload {
			t.Fatalf("body = %+v", body)
		}
		if body.Filter == nil || len(body.Filter.Must) != 3 {
			t.Fatalf("filter = %+v", body.Filter)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"id":"point_1","score":0.91,"payload":{"chunk_id":"chunk_1","document_id":"doc_1"}}],"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(QdrantConfig{
		BaseURL:    server.URL,
		APIKey:     "qdrant-token",
		Collection: "knowledge_chunks",
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := client.Search(context.Background(), service.VectorSearchRequest{
		Vector:           []float32{0.1, 0.2},
		KnowledgeBaseIDs: []string{"kb_1"},
		Tags:             []string{"manual"},
		MetadataFilter:   map[string]string{"page": "3"},
		Limit:            3,
		ScoreThreshold:   0.35,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !called {
		t.Fatal("expected Qdrant call")
	}
	if len(hits) != 1 || hits[0].ID != "point_1" || hits[0].Score != 0.91 || hits[0].Payload["chunk_id"] != "chunk_1" {
		t.Fatalf("hits = %+v", hits)
	}
}

func TestQdrantClientUpsertIncludesVectorAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/collections/knowledge_chunks/points" || r.URL.Query().Get("wait") != "true" {
			t.Fatalf("request = %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		var body qdrantUpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Points) != 1 || body.Points[0].ID != "point_1" || len(body.Points[0].Vector) != 2 {
			t.Fatalf("points = %+v", body.Points)
		}
		payload := body.Points[0].Payload
		if payload["chunk_id"] != "chunk_1" || payload["document_id"] != "doc_1" || payload["knowledge_base_id"] != "kb_1" {
			t.Fatalf("payload = %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"operation_id":1,"status":"completed"},"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(QdrantConfig{BaseURL: server.URL, Collection: "knowledge_chunks"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Upsert(context.Background(), []service.VectorPoint{{
		ID:     "point_1",
		Vector: []float32{0.1, 0.2},
		Payload: map[string]any{
			"chunk_id":          "chunk_1",
			"document_id":       "doc_1",
			"knowledge_base_id": "kb_1",
			"tags":              []string{"manual"},
			"metadata":          map[string]any{"page": 3},
		},
	}}); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
}

func TestQdrantClientDeleteByDocumentIngestionAttemptUsesDocumentFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/knowledge_chunks/points/delete" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var body qdrantDeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Filter.Must) != 2 ||
			body.Filter.Must[0].Key != service.VectorPayloadDocumentID ||
			body.Filter.Must[0].Match.Value != "doc_1" ||
			body.Filter.Must[1].Key != service.VectorPayloadIngestionAttempt ||
			body.Filter.Must[1].Match.Value != "job_1:1" {
			t.Fatalf("filter = %+v", body.Filter)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"operation_id":1,"status":"completed"},"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(QdrantConfig{BaseURL: server.URL, Collection: "knowledge_chunks"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteByDocumentIngestionAttempt(context.Background(), "doc_1", "job_1:1"); err != nil {
		t.Fatalf("DeleteByDocumentIngestionAttempt() error = %v", err)
	}
}

func TestQdrantClientDeleteByDocumentUsesDocumentOnlyFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collections/knowledge_chunks/points/delete" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var body qdrantDeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Filter.Must) != 1 ||
			body.Filter.Must[0].Key != service.VectorPayloadDocumentID ||
			body.Filter.Must[0].Match.Value != "doc_1" ||
			len(body.Filter.MustNot) != 0 {
			t.Fatalf("filter = %+v", body.Filter)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"operation_id":1,"status":"completed"},"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewQdrantClient(QdrantConfig{BaseURL: server.URL, Collection: "knowledge_chunks"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteByDocument(context.Background(), "doc_1"); err != nil {
		t.Fatalf("DeleteByDocument() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func textResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
