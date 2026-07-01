//go:build integration

package adapter

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapterconfig"
)

// Live vendor integration tests. Run with:
//
//	KNOWLEDGE_VENDOR_INTEGRATION_URL=http://127.0.0.1:9380 \
//	KNOWLEDGE_INTEGRATION_USER_ID=seed-user-id \
//	go test -tags=integration ./internal/adapter/... -run Integration -count=1
func TestIntegrationKnowledgeBaseAndUpload(t *testing.T) {
	vendorURL := strings.TrimSpace(os.Getenv("KNOWLEDGE_VENDOR_INTEGRATION_URL"))
	userID := strings.TrimSpace(os.Getenv("KNOWLEDGE_INTEGRATION_USER_ID"))
	if vendorURL == "" || userID == "" {
		t.Skip("set KNOWLEDGE_VENDOR_INTEGRATION_URL and KNOWLEDGE_INTEGRATION_USER_ID for live vendor integration tests")
	}

	server := NewServer(adapterconfig.Config{
		ServiceVersion:     "integration",
		VendorRuntimeURL:   vendorURL,
		AutoStartIngestion: true,
	}, nil)

	kbReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases", strings.NewReader(`{"name":"Integration KB","description":"adapter integration test"}`))
	kbReq.Header.Set("Content-Type", "application/json")
	kbReq.Header.Set("X-User-Id", userID)
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
	t.Cleanup(func() {
		delReq := httptest.NewRequest(http.MethodDelete, "/internal/v1/knowledge-bases/"+kbID, nil)
		delReq.Header.Set("X-User-Id", userID)
		delReq.Header.Set("X-User-Permissions", "knowledge:write")
		delRec := httptest.NewRecorder()
		server.Handler().ServeHTTP(delRec, delReq)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "integration.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "integration upload triggers vendor deepdoc parse"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/internal/v1/knowledge-bases/"+kbID+"/documents", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-Id", userID)
	uploadReq.Header.Set("X-User-Permissions", "knowledge:write")
	uploadRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", uploadRec.Code, uploadRec.Body.String())
	}

	var docBody struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &docBody); err != nil {
		t.Fatalf("decode upload: %v", err)
	}
	if docBody.Data["status"] != "parsing" && docBody.Data["status"] != "uploaded" {
		t.Fatalf("unexpected document status=%v", docBody.Data["status"])
	}
}
