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

func TestParserHTTPClientParsePagesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"pages":[{"pageNumber":1,"content":"page one"},{"pageNumber":2,"content":"page two"}]}}`))
	}))
	defer server.Close()

	client, err := NewParserHTTPClient(ParserHTTPConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Parse(context.Background(), "doc.pdf", "application/pdf", []byte("dummy"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.PageCount != 2 {
		t.Fatalf("PageCount = %d, want 2", result.PageCount)
	}
	if len(result.Chunks) != 2 {
		t.Fatalf("len(Chunks) = %d, want 2", len(result.Chunks))
	}
	if result.Chunks[0].PageNumber != 1 || result.Chunks[0].Content != "page one" {
		t.Fatalf("Chunks[0] = %+v", result.Chunks[0])
	}
}

func TestParserHTTPClientParseContentFallback(t *testing.T) {
	// Simulates text/docx backends that return data.content without pages.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"content":"plain text content from docx"}}`))
	}))
	defer server.Close()

	client, err := NewParserHTTPClient(ParserHTTPConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Parse(context.Background(), "notes.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", []byte("dummy"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.PageCount != 1 {
		t.Fatalf("PageCount = %d, want 1", result.PageCount)
	}
	if len(result.Chunks) != 1 {
		t.Fatalf("len(Chunks) = %d, want 1", len(result.Chunks))
	}
	if result.Chunks[0].PageNumber != 1 || result.Chunks[0].Content != "plain text content from docx" {
		t.Fatalf("Chunks[0] = %+v", result.Chunks[0])
	}
}

func TestParserHTTPClientParseContentFallbackTrimsWhitespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"content":"  \t\n  meaningful  \n  "}}`))
	}))
	defer server.Close()

	client, err := NewParserHTTPClient(ParserHTTPConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Parse(context.Background(), "notes.txt", "text/plain", []byte("dummy"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Chunks[0].Content != "meaningful" {
		t.Fatalf("Chunks[0].Content = %q, want %q", result.Chunks[0].Content, "meaningful")
	}
}

func TestParserHTTPClientParseEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	client, err := NewParserHTTPClient(ParserHTTPConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Parse(context.Background(), "empty.txt", "text/plain", []byte("dummy"))
	if err == nil {
		t.Fatal("Parse() should return an error when both pages and content are empty")
	}
}

func TestParserHTTPClientParseContentOnlyWhitespaceFallbackIsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"content":"   \n\t  "}}`))
	}))
	defer server.Close()

	client, err := NewParserHTTPClient(ParserHTTPConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Parse(context.Background(), "blank.txt", "text/plain", []byte("dummy"))
	if err == nil {
		t.Fatal("Parse() should return an error when content is only whitespace")
	}
}
