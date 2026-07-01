package adapter

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
)

type fakeVendorState struct {
	mu         sync.Mutex
	datasets   map[string]map[string]any
	documents  map[string]map[string]any
	parseCalls []string
	deleteCalls []string
	failParse  bool
	searchBody []byte
}

func newFakeVendorState() *fakeVendorState {
	return &fakeVendorState{
		datasets:  map[string]map[string]any{},
		documents: map[string]map[string]any{},
	}
}

func startFakeVendor(t *testing.T, state *fakeVendorState) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/system/ping":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("pong"))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/datasets":
			items := make([]map[string]any, 0, len(state.datasets))
			for _, item := range state.datasets {
				items = append(items, item)
			}
			writeVendorJSON(w, http.StatusOK, map[string]any{"code": 0, "data": items, "total_datasets": len(items)})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/datasets":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := "kb_fake_1"
			item := map[string]any{
				"id": id, "name": body["name"], "description": body["description"],
				"chunk_method": "naive", "document_count": 0, "chunk_count": 0,
				"create_time": float64(1700000000000),
			}
			state.datasets[id] = item
			writeVendorJSON(w, http.StatusOK, map[string]any{"code": 0, "data": item})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/documents/parse"):
			if state.failParse {
				writeVendorJSON(w, http.StatusBadRequest, map[string]any{"code": 100, "message": "parse failed"})
				return
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			ids := documentIDsFromParseBody(body)
			for _, id := range ids {
				state.parseCalls = append(state.parseCalls, id)
				if doc, ok := state.documents[id]; ok {
					doc["run"] = "RUNNING"
				}
			}
			writeVendorJSON(w, http.StatusOK, map[string]any{"code": 0, "data": map[string]any{"started": len(ids)}})
			return
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/documents") && strings.Contains(r.URL.RawQuery, "type=local"):
			kbID := strings.TrimPrefix(r.URL.Path, "/api/v1/datasets/")
			kbID = strings.TrimSuffix(kbID, "/documents")
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer file.Close()
			docID := "doc_fake_1"
			doc := map[string]any{
				"id": docID, "kb_id": kbID, "dataset_id": kbID, "name": header.Filename,
				"type": "txt", "size": header.Size, "run": "UNSTART", "chunk_count": 0,
			}
			state.documents[docID] = doc
			writeVendorJSON(w, http.StatusOK, map[string]any{"code": 0, "data": doc})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/datasets/search":
			raw, _ := io.ReadAll(r.Body)
			state.searchBody = append([]byte(nil), raw...)
			writeVendorJSON(w, http.StatusOK, map[string]any{
				"code": 0,
				"data": map[string]any{"total": 0, "chunks": []any{}},
			})
			return
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/v1/documents/"):
			docID := strings.TrimPrefix(r.URL.Path, "/api/v1/documents/")
			state.deleteCalls = append(state.deleteCalls, docID)
			delete(state.documents, docID)
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func writeVendorJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func documentIDsFromParseBody(body map[string]any) []string {
	if raw, ok := body["document_ids"].([]any); ok {
		return anyStrings(raw)
	}
	if raw, ok := body["documents"].([]any); ok {
		return anyStrings(raw)
	}
	return nil
}

func anyStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestAdapterDocumentUploadStartsVendorIngestion(t *testing.T) {
	state := newFakeVendorState()
	vendor := startFakeVendor(t, state)
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:     "test",
		VendorRuntimeURL:   vendor.URL,
		AutoStartIngestion: true,
	}, nil)

	kbReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases", strings.NewReader(`{"name":"Manuals"}`))
	kbReq.Header.Set("Content-Type", "application/json")
	kbReq.Header.Set("X-User-Id", "usr_test")
	kbReq.Header.Set("X-User-Permissions", "knowledge:write")
	kbRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(kbRec, kbReq)
	if kbRec.Code != http.StatusCreated {
		t.Fatalf("create kb status=%d body=%s", kbRec.Code, kbRec.Body.String())
	}

	var kbBody struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(kbRec.Body.Bytes(), &kbBody); err != nil {
		t.Fatalf("decode kb: %v", err)
	}
	kbID, _ := kbBody.Data["id"].(string)
	if kbID == "" {
		t.Fatalf("kb id missing: %v", kbBody.Data)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "hello ingestion"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases/"+kbID+"/documents", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-Id", "usr_test")
	uploadReq.Header.Set("X-User-Permissions", "knowledge:write")
	uploadRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.parseCalls) != 1 || state.parseCalls[0] != "doc_fake_1" {
		t.Fatalf("parseCalls=%v", state.parseCalls)
	}
}

func TestAdapterDocumentUploadSkipsIngestionWhenDisabled(t *testing.T) {
	state := newFakeVendorState()
	vendor := startFakeVendor(t, state)
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:     "test",
		VendorRuntimeURL:   vendor.URL,
		AutoStartIngestion: false,
	}, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "notes.txt")
	_, _ = io.WriteString(part, "hello")
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases/kb_fake_1/documents", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-Id", "usr_test")
	uploadReq.Header.Set("X-User-Permissions", "knowledge:write")
	uploadRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}
	if len(state.parseCalls) != 0 {
		t.Fatalf("parseCalls=%v want none", state.parseCalls)
	}
}

func TestAdapterKnowledgeQueryForwardsDocumentIDs(t *testing.T) {
	state := newFakeVendorState()
	vendor := startFakeVendor(t, state)
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:   "test",
		VendorRuntimeURL: vendor.URL,
		VendorRerankID:   "rerank-model",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-queries", strings.NewReader(`{"query":"maintenance","knowledgeBaseIds":["kb_fake_1"],"documentIds":["doc_1"],"tags":["锅炉"],"metadataFilter":{"专业":"锅炉"},"rerank":true,"rerankTopN":5,"topK":8,"scoreThreshold":0.4}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "usr_test")
	req.Header.Set("X-User-Permissions", "knowledge:read")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("query status=%d body=%s", rec.Code, rec.Body.String())
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.searchBody) == 0 {
		t.Fatal("vendor search body missing")
	}
	var payload map[string]any
	if err := json.Unmarshal(state.searchBody, &payload); err != nil {
		t.Fatalf("decode search body: %v", err)
	}
	docIDs, _ := payload["doc_ids"].([]any)
	if len(docIDs) != 1 || docIDs[0] != "doc_1" {
		t.Fatalf("doc_ids=%v", payload["doc_ids"])
	}
	if payload["rerank_id"] != "rerank-model" {
		t.Fatalf("rerank_id=%v", payload["rerank_id"])
	}
}

func TestAdapterDocumentUploadRollsBackWhenParseFails(t *testing.T) {
	state := newFakeVendorState()
	state.failParse = true
	vendor := startFakeVendor(t, state)
	defer vendor.Close()

	server := NewServer(adapterconfig.Config{
		ServiceVersion:     "test",
		VendorRuntimeURL:   vendor.URL,
		AutoStartIngestion: true,
	}, nil)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "notes.txt")
	_, _ = io.WriteString(part, "hello")
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases/kb_fake_1/documents", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-Id", "usr_test")
	uploadReq.Header.Set("X-User-Permissions", "knowledge:write")
	uploadRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code == http.StatusCreated {
		t.Fatalf("expected upload failure, got status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.deleteCalls) != 1 || state.deleteCalls[0] != "doc_fake_1" {
		t.Fatalf("deleteCalls=%v", state.deleteCalls)
	}
	if _, ok := state.documents["doc_fake_1"]; ok {
		t.Fatal("document should be removed after parse failure")
	}
}
