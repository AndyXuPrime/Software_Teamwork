package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/metrics"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
)

func TestMetricsMiddlewarePassesThrough(t *testing.T) {
	reg := metrics.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Chain(mux, middleware.Metrics(reg, mux))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestMetricsMiddlewareCapturesNonOKStatus(t *testing.T) {
	reg := metrics.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /fail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	handler := middleware.Chain(mux, middleware.Metrics(reg, mux))

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestMetricsMiddlewareNilRegistryIsNoOp(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Chain(mux, middleware.Metrics(nil, mux))

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestMetricsMiddlewareFlusherPassThrough(t *testing.T) {
	reg := metrics.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		// SSE / streaming handlers assert http.Flusher; this must not fail.
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("responseWriter does not implement http.Flusher")
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("data: hello\n\n"))
		f.Flush()
	})

	handler := middleware.Chain(mux, middleware.Metrics(reg, mux))

	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestMetricsMiddlewareImplicit200OnWrite(t *testing.T) {
	reg := metrics.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /body", func(w http.ResponseWriter, r *http.Request) {
		// Write body without calling WriteHeader — net/http implies 200.
		_, _ = w.Write([]byte("ok"))
	})

	handler := middleware.Chain(mux, middleware.Metrics(reg, mux))

	req := httptest.NewRequest(http.MethodGet, "/body", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}
