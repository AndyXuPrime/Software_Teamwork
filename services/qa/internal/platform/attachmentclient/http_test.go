package attachmentclient

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFileHTTPClientReadHonorsConfiguredLimit(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		maxBytes  int64
		chunked   bool
		wantError bool
	}{
		{name: "exact limit", payload: []byte("1234"), maxBytes: 4},
		{name: "over limit without content length", payload: []byte("12345"), maxBytes: 4, chunked: true, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/internal/v1/files/file-1/content" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				if tt.chunked {
					w.(http.Flusher).Flush()
				}
				_, _ = w.Write(tt.payload)
			}))
			defer server.Close()

			client, err := NewFileHTTPClient(FileHTTPConfig{BaseURL: server.URL, MaxReadBytes: tt.maxBytes})
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.Read(context.Background(), "file-1")
			if tt.wantError {
				if err == nil {
					t.Fatalf("Read() data = %q, want over-limit error", got)
				}
				if got != nil {
					t.Fatalf("Read() returned partial data = %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Read() error = %v", err)
			}
			if !bytes.Equal(got, tt.payload) {
				t.Fatalf("Read() data = %q, want %q", got, tt.payload)
			}
		})
	}
}

func TestNewFileHTTPClientRequiresMaxReadBytes(t *testing.T) {
	if _, err := NewFileHTTPClient(FileHTTPConfig{BaseURL: "http://file:8082"}); err == nil {
		t.Fatal("expected missing max read bytes to fail")
	}
}
