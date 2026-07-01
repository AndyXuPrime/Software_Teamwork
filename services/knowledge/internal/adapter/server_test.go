package adapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if payload["service"] != "knowledge-adapter" {
		t.Fatalf("service=%v", payload["service"])
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

func TestContractRouteNotImplemented(t *testing.T) {
	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: "http://127.0.0.1:1",
	}, nil)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/internal/v1/knowledge-bases", nil))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
