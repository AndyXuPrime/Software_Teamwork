package smoke_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

const documentMCPSmokeGate = "DOCUMENT_REST_SMOKE"

func TestDocumentRESTContractSmoke(t *testing.T) {
	if os.Getenv(documentMCPSmokeGate) != "1" {
		t.Skip("set DOCUMENT_REST_SMOKE=1 to run the Document REST contract smoke")
	}

	cfg := loadDocumentSmokeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	assertHTTPReady(t, ctx, "gateway", cfg.gatewayBaseURL)
	assertHTTPReady(t, ctx, "document", cfg.documentBaseURL)

	requestID := "req_document_rest_smoke_" + shortID(newSmokeRunID())
	client := smokeHTTPClient()
	session := createSmokeSession(t, ctx, client, cfg.gatewayBaseURL, cfg.username, cfg.password, requestID)

	t.Run("report_types_list", func(t *testing.T) {
		assertReportTypesList(t, ctx, client, cfg, session, requestID)
	})
	t.Run("seed_report_read", func(t *testing.T) {
		assertSeedReportRead(t, ctx, client, cfg, session, requestID)
	})
	t.Run("seed_report_outline_read", func(t *testing.T) {
		assertSeedReportOutlineRead(t, ctx, client, cfg, session, requestID)
	})
	t.Run("missing_report_404", func(t *testing.T) {
		assertMissingReportReturns404(t, ctx, client, cfg, session, requestID)
	})
	t.Run("document_response_no_leaks", func(t *testing.T) {
		assertDocumentResponseNoLeaks(t, ctx, client, cfg, session, requestID)
	})
}

type documentSmokeConfig struct {
	gatewayBaseURL  string
	documentBaseURL string
	username        string
	password        string
}

func loadDocumentSmokeConfig(t *testing.T) documentSmokeConfig {
	t.Helper()
	required := map[string]string{
		"GATEWAY_BASE_URL":                               os.Getenv("GATEWAY_BASE_URL"),
		"DOCUMENT_SERVICE_BASE_URL":                      os.Getenv("DOCUMENT_SERVICE_BASE_URL"),
		"GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME": firstNonEmptyEnv("GATEWAY_SMOKE_USERNAME", "LOCAL_ADMIN_USERNAME"),
		"GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD": firstNonEmptyEnv("GATEWAY_SMOKE_PASSWORD", "LOCAL_ADMIN_PASSWORD"),
	}
	missing := []string{}
	for k, v := range required {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing required environment variables:\n - %s", strings.Join(missing, "\n - "))
	}
	return documentSmokeConfig{
		gatewayBaseURL:  trimBaseURL(t, "GATEWAY_BASE_URL", required["GATEWAY_BASE_URL"]),
		documentBaseURL: trimBaseURL(t, "DOCUMENT_SERVICE_BASE_URL", required["DOCUMENT_SERVICE_BASE_URL"]),
		username:        strings.TrimSpace(required["GATEWAY_SMOKE_USERNAME or LOCAL_ADMIN_USERNAME"]),
		password:        strings.TrimSpace(required["GATEWAY_SMOKE_PASSWORD or LOCAL_ADMIN_PASSWORD"]),
	}
}

// ---- test cases ----

func assertReportTypesList(t *testing.T, ctx context.Context, client *http.Client, cfg documentSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/report-types", session.AccessToken, requestID+"_rt", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list report types: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Data      json.RawMessage `json:"data"`
		RequestID string          `json:"requestId"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode report types: %v body=%s", err, string(body))
	}
	if envelope.RequestID != requestID+"_rt" {
		t.Fatalf("requestId mismatch: want=%q got=%q", requestID+"_rt", envelope.RequestID)
	}
}

func assertSeedReportRead(t *testing.T, ctx context.Context, client *http.Client, cfg documentSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	const seedReportID = "22222222-2222-4222-8222-222222222301"
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/reports/"+seedReportID, session.AccessToken, requestID+"_r", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get seed report: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for seed report, got %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Data struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			ReportType string `json:"reportType"`
			Status     string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if envelope.Data.ID != seedReportID {
		t.Fatalf("report id mismatch: want=%q got=%q", seedReportID, envelope.Data.ID)
	}
	if envelope.Data.Status != "generated" {
		t.Fatalf("expected seed report status=generated, got %q", envelope.Data.Status)
	}
	assertNoLeakedInternals(t, body)
}

func assertSeedReportOutlineRead(t *testing.T, ctx context.Context, client *http.Client, cfg documentSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	const seedReportID = "22222222-2222-4222-8222-222222222301"
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/reports/"+seedReportID+"/outlines", session.AccessToken, requestID+"_o", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list report outlines: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	// Accept 200 or 404 (seed report may not have outlines in all environments)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 200 or 404 for outlines, got %d: %s", resp.StatusCode, string(body))
	}
}

func assertMissingReportReturns404(t *testing.T, ctx context.Context, client *http.Client, cfg documentSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/reports/00000000-0000-4000-8000-000000000000", session.AccessToken, requestID+"_nf", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get missing report: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent report, got %d: %s", resp.StatusCode, string(body))
	}
	// Error envelope should contain error code and requestId
	var errEnv struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errEnv); err != nil {
		t.Fatalf("404 response is not valid JSON error envelope: %v body=%s", err, string(body))
	}
	if errEnv.Error.Code == "" {
		t.Fatalf("404 error envelope missing code: %s", string(body))
	}
	if errEnv.Error.Message == "" {
		t.Fatalf("404 error envelope missing message: %s", string(body))
	}
	if errEnv.Error.RequestID != requestID+"_nf" {
		t.Fatalf("404 error requestId mismatch: want=%q got=%q", requestID+"_nf", errEnv.Error.RequestID)
	}
}

func assertDocumentResponseNoLeaks(t *testing.T, ctx context.Context, client *http.Client, cfg documentSmokeConfig, session smokeSession, requestID string) {
	t.Helper()
	req := gatewayAuthRequest(http.MethodGet, cfg.gatewayBaseURL+"/api/v1/report-types", session.AccessToken, requestID+"_nl", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("no-leak request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
	lower := strings.ToLower(string(body))
	for _, forbidden := range []string{
		"objectkey", "storagekey", "bucket",
		"internalurl", "service_token", "api_key",
		"rawvector", "rawpayload",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("document response leaked %q", forbidden)
		}
	}
}
