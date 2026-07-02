package mcpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	qaconfig "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/config"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
	qatools "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/tools"
)

const (
	documentMCPSmokeReportID   = "22222222-2222-4222-8222-222222222301"
	documentMCPSmokeMaterialID = "22222222-2222-4222-8222-222222222201"
)

func TestDocumentMCPReportToolsSmoke(t *testing.T) {
	if strings.TrimSpace(os.Getenv("QA_DOCUMENT_MCP_SMOKE")) != "1" {
		t.Skip("set QA_DOCUMENT_MCP_SMOKE=1 to run the QA -> Document MCP smoke")
	}

	cfg, err := qaconfig.Load()
	if err != nil {
		t.Fatalf("load QA Document MCP smoke configuration: %v", err)
	}
	if cfg.MCPTransport != qaconfig.TransportStreamableHTTP || cfg.MCPServerAlias != "document" {
		t.Fatalf("QA_DOCUMENT_MCP_SMOKE requires MCP_TRANSPORT=streamable_http and MCP_SERVER_ALIAS=document")
	}
	if strings.TrimSpace(cfg.MCPServerToken) == "" {
		t.Fatalf("QA_DOCUMENT_MCP_SMOKE requires MCP_SERVER_TOKEN matching the Document MCP service token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client, err := Connect(ctx, Config{
		Transport:   cfg.MCPTransport,
		Endpoint:    cfg.MCPServerURL,
		Token:       cfg.MCPServerToken,
		TokenHeader: cfg.MCPServerTokenHeader,
	})
	if err != nil {
		t.Fatalf("connect Document MCP endpoint failed; verify endpoint readiness and token/header configuration")
	}
	defer client.Close()

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Document MCP tools/list failed; verify the Document MCP endpoint and service token")
	}
	assertToolNames(t, tools, []string{
		"generate_report_outline",
		"generate_report_text",
		"get_generation_status",
		"export_report_docx",
		"get_report_result",
	})

	prefixed, err := NewPrefixed("document", client, cfg.MCPToolTimeout)
	if err != nil {
		t.Fatalf("prefix Document MCP tools: %v", err)
	}
	prefixedTools, err := prefixed.ListTools(ctx)
	if err != nil {
		t.Fatalf("Document MCP prefixed tools/list failed")
	}
	assertToolNames(t, prefixedTools, qatools.DefaultDocumentReportToolNames)

	reportID := envOr("QA_DOCUMENT_MCP_SMOKE_REPORT_ID", documentMCPSmokeReportID)
	materialID := envOr("QA_DOCUMENT_MCP_SMOKE_MATERIAL_ID", documentMCPSmokeMaterialID)
	adminCtx := documentMCPContext(ctx, "qa-document-mcp-smoke-admin", "admin", "qa:report:write,qa:report:read")

	outline := callDocumentTool(t, adminCtx, prefixed, "document__generate_report_outline", map[string]any{
		"reportId":     reportID,
		"materialIds":  []string{materialID},
		"requirements": "QA Document MCP smoke: create a minimal outline job.",
	})
	if outline.IsError {
		t.Fatalf("document__generate_report_outline returned an error result")
	}
	assertNoSensitiveSmokeLeak(t, outline.Content, cfg.MCPServerToken)
	outlineArtifact := reportArtifact(t, "document__generate_report_outline", outline.Content)
	jobID := stringField(t, outlineArtifact, "jobId")
	if _, ok := outlineArtifact["downloadPath"]; ok {
		t.Fatalf("accepted/running job artifact must not include downloadPath: %#v", outlineArtifact)
	}

	status := callDocumentTool(t, adminCtx, prefixed, "document__get_generation_status", map[string]any{"jobId": jobID})
	if status.IsError {
		t.Fatalf("document__get_generation_status returned an error result")
	}
	assertNoSensitiveSmokeLeak(t, status.Content, cfg.MCPServerToken)
	_ = reportArtifact(t, "document__get_generation_status", status.Content)

	export := callDocumentTool(t, adminCtx, prefixed, "document__export_report_docx", map[string]any{
		"reportId": reportID,
		"format":   "docx",
	})
	if export.IsError {
		t.Fatalf("document__export_report_docx returned an error result")
	}
	assertNoSensitiveSmokeLeak(t, export.Content, cfg.MCPServerToken)
	exportArtifact := reportArtifact(t, "document__export_report_docx", export.Content)
	if exportArtifact["fileStatus"] != "succeeded" {
		if _, ok := exportArtifact["downloadPath"]; ok {
			t.Fatalf("non-succeeded export artifact must not include downloadPath: %#v", exportArtifact)
		}
	}

	resultArtifact := waitForSucceededReportFile(t, adminCtx, prefixed, reportID, cfg.MCPServerToken)
	downloadPath := stringField(t, resultArtifact, "downloadPath")
	if !strings.HasPrefix(downloadPath, "/api/v1/report-files/") || !strings.HasSuffix(downloadPath, "/content") {
		t.Fatalf("downloadPath = %q, want /api/v1/report-files/{reportFileId}/content", downloadPath)
	}
	verifyGatewayDownloadIfConfigured(t, ctx, downloadPath)

	forbiddenCtx := documentMCPContext(ctx, "qa-document-mcp-smoke-standard-user", "", "")
	forbidden := callDocumentTool(t, forbiddenCtx, prefixed, "document__get_report_result", map[string]any{"reportId": reportID})
	if !forbidden.IsError {
		t.Fatalf("standard user unexpectedly accessed another user's report")
	}
	assertNoSensitiveSmokeLeak(t, forbidden.Content, cfg.MCPServerToken)
	forbiddenSummary := qatools.GenerateResultSummary("document__get_report_result", forbidden.Content)
	if forbiddenSummary["error"] != "policy_denied" {
		t.Fatalf("forbidden summary error = %#v, want policy_denied", forbiddenSummary["error"])
	}
}

func documentMCPContext(parent context.Context, userID, roles, permissions string) context.Context {
	ctx := contextutil.WithRequestID(parent, "qa-document-mcp-smoke")
	ctx = contextutil.WithUserID(ctx, userID)
	ctx = contextutil.WithUserRoles(ctx, roles)
	ctx = contextutil.WithUserPermissions(ctx, permissions)
	return ctx
}

func callDocumentTool(t *testing.T, ctx context.Context, client *Prefixed, name string, payload map[string]any) agent.ToolResult {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s arguments: %v", name, err)
	}
	result, err := client.CallTool(ctx, name, data)
	if err != nil {
		t.Fatalf("%s failed; inspect QA and Document logs with request id qa-document-mcp-smoke", name)
	}
	return result
}

func waitForSucceededReportFile(t *testing.T, ctx context.Context, client *Prefixed, reportID string, secret string) map[string]any {
	t.Helper()
	timeout := durationEnv("QA_DOCUMENT_MCP_SMOKE_EXPORT_TIMEOUT", 30*time.Second)
	deadline := time.Now().Add(timeout)
	var latest map[string]any
	for {
		result := callDocumentTool(t, ctx, client, "document__get_report_result", map[string]any{"reportId": reportID})
		if result.IsError {
			t.Fatalf("document__get_report_result returned an error result")
		}
		assertNoSensitiveSmokeLeak(t, result.Content, secret)
		artifact := reportArtifact(t, "document__get_report_result", result.Content)
		latest = artifact
		if artifact["fileStatus"] == "succeeded" {
			return artifact
		}
		if time.Now().After(deadline) {
			t.Fatalf("report file did not reach succeeded before timeout; latest artifact=%#v", latest)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func reportArtifact(t *testing.T, toolName string, content string) map[string]any {
	t.Helper()
	summary := qatools.GenerateResultSummary(toolName, content)
	artifact, ok := summary["reportArtifact"].(map[string]any)
	if !ok {
		t.Fatalf("%s summary missing reportArtifact: %#v", toolName, summary)
	}
	if artifact["artifactType"] != "report_generation" {
		t.Fatalf("%s artifactType=%#v, want report_generation", toolName, artifact["artifactType"])
	}
	return artifact
}

func stringField(t *testing.T, artifact map[string]any, field string) string {
	t.Helper()
	value, _ := artifact[field].(string)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("artifact missing %s: %#v", field, artifact)
	}
	return value
}

func assertToolNames(t *testing.T, tools []agent.ToolDefinition, required []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		seen[tool.Function.Name] = struct{}{}
	}
	for _, name := range required {
		if _, ok := seen[name]; !ok {
			t.Fatalf("tools/list missing %q; seen=%v", name, seen)
		}
	}
}

func assertNoSensitiveSmokeLeak(t *testing.T, value string, secret string) {
	t.Helper()
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"api_key", "apikey", "service_token", "object_key", "objectkey",
		"bucket", "minio", "provider_error", "raw_prompt", "internal_url",
	} {
		if strings.Contains(lower, marker) {
			t.Fatalf("Document MCP result leaked sensitive marker %q", marker)
		}
	}
	if secret = strings.TrimSpace(secret); secret != "" && strings.Contains(value, secret) {
		t.Fatalf("Document MCP result leaked the service token")
	}
}

func verifyGatewayDownloadIfConfigured(t *testing.T, ctx context.Context, downloadPath string) {
	t.Helper()
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("QA_DOCUMENT_MCP_SMOKE_GATEWAY_BASE_URL")), "/")
	if baseURL == "" {
		t.Log("QA_DOCUMENT_MCP_SMOKE_GATEWAY_BASE_URL not set; skipping Gateway download probe")
		return
	}
	bearer := strings.TrimSpace(os.Getenv("QA_DOCUMENT_MCP_SMOKE_GATEWAY_BEARER"))
	if bearer == "" {
		t.Fatal("QA_DOCUMENT_MCP_SMOKE_GATEWAY_BEARER is required when Gateway download probe is enabled")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+downloadPath, nil)
	if err != nil {
		t.Fatalf("create Gateway download request")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Gateway report file download probe failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("Gateway report file download status = %d, want 2xx", resp.StatusCode)
	}
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envOr(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
