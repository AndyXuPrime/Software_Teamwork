package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	knowledgehttp "github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/http"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestHealthReturnsEnvelope(t *testing.T) {
	server := newHTTPTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-Id", "req_health")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if got := res.Header().Get("X-Request-Id"); got != "req_health" {
		t.Fatalf("X-Request-Id = %q", got)
	}
	var body healthResponseBody
	decodeJSON(t, res.Body, &body)
	if body.RequestID != "req_health" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Service != "knowledge" || body.Data.Status != "ok" {
		t.Fatalf("data = %+v", body.Data)
	}
}

func TestReadyReturnsConfigurationSummary(t *testing.T) {
	server := newHTTPTestServer()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Header.Set("X-Request-Id", "req_ready")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body readyResponseBody
	decodeJSON(t, res.Body, &body)
	if body.RequestID != "req_ready" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Service != "knowledge" || body.Data.Status != "ready" {
		t.Fatalf("data = %+v", body.Data)
	}
	if body.Data.EmbeddingProvider != "local_hashing" || body.Data.EmbeddingDimension != 384 {
		t.Fatalf("embedding data = %+v", body.Data)
	}
	if body.Data.QdrantCollection != "knowledge_chunks" {
		t.Fatalf("qdrant collection = %q", body.Data.QdrantCollection)
	}
}

func TestReadyReportsInvalidConfiguration(t *testing.T) {
	status := service.NewStatusService(service.StatusConfig{
		Version:            "test",
		Environment:        "test",
		StorageBackend:     "memory",
		EmbeddingProvider:  "",
		EmbeddingModel:     "local_hashing",
		EmbeddingDimension: 0,
		QdrantCollection:   "",
	})
	server := knowledgehttp.NewServer(status, knowledgehttp.Config{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Header.Set("X-Request-Id", "req_bad_ready")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorResponseBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != "validation_error" || body.Error.RequestID != "req_bad_ready" {
		t.Fatalf("error = %+v", body.Error)
	}
	if body.Error.Fields["embeddingDimension"] == "" || body.Error.Fields["qdrantCollection"] == "" {
		t.Fatalf("fields = %+v", body.Error.Fields)
	}
}

func TestUnknownRouteReturnsErrorEnvelope(t *testing.T) {
	server := newHTTPTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge-bases", nil)
	req.Header.Set("X-Request-Id", "req_missing")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d", res.Code)
	}
	var body errorResponseBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != "not_found" || body.Error.RequestID != "req_missing" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func TestGeneratesRequestIDWhenMissing(t *testing.T) {
	server := newHTTPTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	requestID := res.Header().Get("X-Request-Id")
	if !strings.HasPrefix(requestID, "req_") {
		t.Fatalf("generated request id = %q", requestID)
	}
	var body healthResponseBody
	decodeJSON(t, res.Body, &body)
	if body.RequestID != requestID {
		t.Fatalf("body requestId = %q, header = %q", body.RequestID, requestID)
	}
}

func newHTTPTestServer() http.Handler {
	status := service.NewStatusService(service.StatusConfig{
		Version:            "test",
		Environment:        "test",
		StorageBackend:     "memory",
		EmbeddingProvider:  "local_hashing",
		EmbeddingModel:     "local_hashing",
		EmbeddingDimension: 384,
		QdrantCollection:   "knowledge_chunks",
	})
	return knowledgehttp.NewServer(status, knowledgehttp.Config{})
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

type healthResponseBody struct {
	Data struct {
		Service string `json:"service"`
		Status  string `json:"status"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type readyResponseBody struct {
	Data struct {
		Service            string `json:"service"`
		Status             string `json:"status"`
		Version            string `json:"version"`
		Environment        string `json:"environment"`
		StorageBackend     string `json:"storageBackend"`
		EmbeddingProvider  string `json:"embeddingProvider"`
		EmbeddingModel     string `json:"embeddingModel"`
		EmbeddingDimension int    `json:"embeddingDimension"`
		QdrantCollection   string `json:"qdrantCollection"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}

type errorResponseBody struct {
	Error struct {
		Code      string            `json:"code"`
		Message   string            `json:"message"`
		RequestID string            `json:"requestId"`
		Fields    map[string]string `json:"fields"`
	} `json:"error"`
}
