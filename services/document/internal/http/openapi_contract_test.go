package httpapi

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDocumentOpenAPIReportBaseResourcesMatchImplementedEnvelope(t *testing.T) {
	spec := readDocumentDocsOpenAPI(t)
	document := parseOpenAPIDocument(t, spec)

	for _, route := range []struct {
		method string
		path   string
		status string
		ref    string
	}{
		{method: "get", path: "/report-types", status: "200", ref: "#/components/schemas/ReportTypeListResponse"},
		{method: "get", path: "/report-templates", status: "200", ref: "#/components/schemas/ReportTemplateListResponse"},
		{method: "post", path: "/report-templates", status: "201", ref: "#/components/schemas/ReportTemplateResponse"},
		{method: "get", path: "/report-templates/{reportTemplateId}", status: "200", ref: "#/components/schemas/ReportTemplateResponse"},
		{method: "patch", path: "/report-templates/{reportTemplateId}", status: "200", ref: "#/components/schemas/ReportTemplateResponse"},
		{method: "get", path: "/report-templates/{reportTemplateId}/structure", status: "200", ref: "#/components/schemas/ReportTemplateStructureResponse"},
		{method: "patch", path: "/report-templates/{reportTemplateId}/structure", status: "200", ref: "#/components/schemas/ReportTemplateStructureResponse"},
		{method: "get", path: "/report-materials", status: "200", ref: "#/components/schemas/ReportMaterialListResponse"},
		{method: "post", path: "/report-materials", status: "201", ref: "#/components/schemas/ReportMaterialResponse"},
		{method: "get", path: "/report-materials/{materialId}", status: "200", ref: "#/components/schemas/ReportMaterialResponse"},
		{method: "get", path: "/report-statistics/overview", status: "200", ref: "#/components/schemas/ReportStatisticsOverviewResponse"},
		{method: "get", path: "/report-statistics/daily", status: "200", ref: "#/components/schemas/ReportDailyStatisticsResponse"},
		{method: "get", path: "/report-operation-logs", status: "200", ref: "#/components/schemas/ReportOperationLogListResponse"},
	} {
		assertOpenAPISuccessResponseRef(t, document, route.method, route.path, route.status, route.ref)
	}

	assertSchemaHasFields(t, spec, "ReportTypeListResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateListResponse", "data:", "page:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportTemplateStructureResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportMaterialResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportMaterialListResponse", "data:", "page:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportStatisticsOverviewResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportDailyStatisticsResponse", "data:", "requestId:")
	assertSchemaHasFields(t, spec, "ReportOperationLogListResponse", "data:", "page:", "requestId:")

	assertOpenAPIQueryParameter(t, document, "get", "/report-templates", "enabled", "boolean")

	templateSchema := openAPISchemaBlock(t, spec, "ReportTemplate")
	for _, field := range []string{"templateName:", "version:", "enabled:"} {
		if !strings.Contains(templateSchema, field) {
			t.Fatalf("ReportTemplate schema missing %s in:\n%s", field, templateSchema)
		}
	}
	for _, staleField := range []string{"name:", "status:"} {
		if containsYAMLField(templateSchema, staleField) {
			t.Fatalf("ReportTemplate schema still contains stale field %s in:\n%s", staleField, templateSchema)
		}
	}

	materialSchema := openAPISchemaBlock(t, spec, "ReportMaterial")
	for _, field := range []string{"id:", "materialName:", "enabled:"} {
		if !strings.Contains(materialSchema, field) {
			t.Fatalf("ReportMaterial schema missing %s in:\n%s", field, materialSchema)
		}
	}
	for _, staleField := range []string{"materialId:", "name:"} {
		if containsYAMLField(materialSchema, staleField) {
			t.Fatalf("ReportMaterial schema still contains stale field %s in:\n%s", staleField, materialSchema)
		}
	}
}

func TestGatewayPublicOpenAPIContainsDocumentOwnerRoutes(t *testing.T) {
	spec := readGatewayPublicOpenAPI(t)
	document := parseOpenAPIDocument(t, spec)

	documentRoutes := []struct {
		method string
		path   string
	}{
		{"get", "/api/v1/report-types"},
		{"get", "/api/v1/report-templates"},
		{"post", "/api/v1/report-templates"},
		{"get", "/api/v1/report-templates/{reportTemplateId}"},
		{"patch", "/api/v1/report-templates/{reportTemplateId}"},
		{"delete", "/api/v1/report-templates/{reportTemplateId}"},
		{"get", "/api/v1/report-templates/{reportTemplateId}/structure"},
		{"patch", "/api/v1/report-templates/{reportTemplateId}/structure"},
		{"get", "/api/v1/report-materials"},
		{"post", "/api/v1/report-materials"},
		{"get", "/api/v1/report-materials/{materialId}"},
		{"delete", "/api/v1/report-materials/{materialId}"},
		{"get", "/api/v1/reports"},
		{"post", "/api/v1/reports"},
		{"get", "/api/v1/reports/{reportId}"},
		{"patch", "/api/v1/reports/{reportId}"},
		{"delete", "/api/v1/reports/{reportId}"},
		{"get", "/api/v1/reports/{reportId}/outlines"},
		{"post", "/api/v1/reports/{reportId}/outlines"},
		{"get", "/api/v1/reports/{reportId}/outlines/{outlineId}"},
		{"patch", "/api/v1/reports/{reportId}/outlines/{outlineId}"},
		{"delete", "/api/v1/reports/{reportId}/outlines/{outlineId}/sections/{sectionId}"},
		{"get", "/api/v1/reports/{reportId}/sections"},
		{"post", "/api/v1/reports/{reportId}/sections"},
		{"get", "/api/v1/reports/{reportId}/sections/{sectionId}"},
		{"patch", "/api/v1/reports/{reportId}/sections/{sectionId}"},
		{"get", "/api/v1/reports/{reportId}/sections/{sectionId}/versions"},
		{"post", "/api/v1/reports/{reportId}/sections/{sectionId}/versions"},
		{"get", "/api/v1/reports/{reportId}/jobs"},
		{"post", "/api/v1/reports/{reportId}/jobs"},
		{"get", "/api/v1/report-jobs/{jobId}"},
		{"get", "/api/v1/report-jobs/{jobId}/attempts"},
		{"post", "/api/v1/report-jobs/{jobId}/attempts"},
		{"get", "/api/v1/reports/{reportId}/events"},
		{"get", "/api/v1/report-files"},
		{"post", "/api/v1/report-files"},
		{"get", "/api/v1/report-files/{reportFileId}"},
		{"get", "/api/v1/report-files/{reportFileId}/content"},
		{"get", "/api/v1/report-statistics/overview"},
		{"get", "/api/v1/report-statistics/daily"},
		{"get", "/api/v1/report-operation-logs"},
		{"get", "/api/v1/report-settings"},
		{"patch", "/api/v1/report-settings"},
	}

	for _, route := range documentRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			operation, ok := digMap(document, "paths", route.path, route.method)
			if !ok {
				t.Fatalf("gateway OpenAPI missing route %s %s", strings.ToUpper(route.method), route.path)
			}
			owner, _ := operation["x-owner-service"].(string)
			if owner != "document" {
				t.Fatalf("%s %s x-owner-service = %q, want \"document\"", strings.ToUpper(route.method), route.path, owner)
			}
		})
	}

	for _, schemaName := range []string{
		"ReportTypeListResponse",
		"ReportTemplateListResponse",
		"ReportTemplateResponse",
		"ReportTemplateStructureResponse",
		"ReportMaterialListResponse",
		"ReportMaterialResponse",
		"ReportListResponse",
		"ReportResponse",
		"ReportOutlineListResponse",
		"ReportOutlineResponse",
		"ReportSectionListResponse",
		"ReportSectionResponse",
		"ReportSectionVersionListResponse",
		"ReportSectionVersionResponse",
		"ReportJobListResponse",
		"ReportJobResponse",
		"ReportJobAttemptListResponse",
		"ReportJobAttemptResponse",
		"ReportEventListResponse",
		"ReportFileListResponse",
		"ReportFileResponse",
		"ReportStatisticsOverviewResponse",
		"ReportDailyStatisticsResponse",
		"ReportOperationLogListResponse",
		"ReportSettingsResponse",
		"ReportSettingsUpdateResponse",
	} {
		t.Run("schema/"+schemaName, func(t *testing.T) {
			assertSchemaHasFields(t, spec, schemaName, "data:", "requestId:")
		})
	}
}

func readGatewayPublicOpenAPI(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller failed; skipping gateway OpenAPI contract test")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		candidate := filepath.Join(dir, "docs", "services", "gateway", "api", "public.openapi.yaml")
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("docs/services/gateway/api/public.openapi.yaml not found; skipping gateway OpenAPI contract test")
	return ""
}

func readDocumentDocsOpenAPI(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller failed; skipping OpenAPI contract test")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		candidate := filepath.Join(dir, "docs", "services", "document", "api", "public.openapi.yaml")
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("docs/services/document/api/public.openapi.yaml not found; skipping OpenAPI contract test")
	return ""
}

func assertSchemaHasFields(t *testing.T, spec string, schema string, fields ...string) {
	t.Helper()
	block := openAPISchemaBlock(t, spec, schema)
	for _, field := range fields {
		if !strings.Contains(block, field) {
			t.Fatalf("%s schema missing %s in:\n%s", schema, field, block)
		}
	}
}

func openAPISchemaBlock(t *testing.T, spec string, schema string) string {
	t.Helper()
	lines := strings.Split(spec, "\n")
	start := -1
	startIndent := 0
	for i, line := range lines {
		if strings.TrimSpace(line) != schema+":" {
			continue
		}
		start = i
		startIndent = leadingSpaces(line)
		break
	}
	if start == -1 {
		t.Fatalf("schema %s not found in document OpenAPI", schema)
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if leadingSpaces(lines[i]) <= startIndent {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func leadingSpaces(value string) int {
	return len(value) - len(strings.TrimLeft(value, " "))
}

func containsYAMLField(block string, field string) bool {
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == field {
			return true
		}
	}
	return false
}

func assertOpenAPISuccessResponseRef(t *testing.T, document map[string]any, method string, path string, status string, wantRef string) {
	t.Helper()
	operation := openAPIOperationMap(t, document, method, path)
	gotRef, ok := digString(operation, "responses", status, "content", "application/json", "schema", "$ref")
	if !ok {
		t.Fatalf("%s %s missing %s application/json response schema $ref", strings.ToUpper(method), path, status)
	}
	if gotRef != wantRef {
		t.Fatalf("%s %s %s response schema ref = %q, want %q", strings.ToUpper(method), path, status, gotRef, wantRef)
	}
}

func assertOpenAPIQueryParameter(t *testing.T, document map[string]any, method string, path string, name string, wantType string) {
	t.Helper()
	operation := openAPIOperationMap(t, document, method, path)
	parameters, ok := operation["parameters"].([]any)
	if !ok {
		t.Fatalf("%s %s missing parameters array", strings.ToUpper(method), path)
	}
	for _, value := range parameters {
		parameter, ok := value.(map[string]any)
		if !ok || parameter["name"] != name || parameter["in"] != "query" {
			continue
		}
		gotType, ok := digString(parameter, "schema", "type")
		if !ok {
			t.Fatalf("%s %s query parameter %s missing schema type", strings.ToUpper(method), path, name)
		}
		if gotType != wantType {
			t.Fatalf("%s %s query parameter %s type = %q, want %q", strings.ToUpper(method), path, name, gotType, wantType)
		}
		return
	}
	t.Fatalf("%s %s missing query parameter %s", strings.ToUpper(method), path, name)
}

func parseOpenAPIDocument(t *testing.T, spec string) map[string]any {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal([]byte(spec), &document); err != nil {
		t.Fatalf("parse document OpenAPI YAML: %v", err)
	}
	return document
}

func openAPIOperationMap(t *testing.T, document map[string]any, method string, path string) map[string]any {
	t.Helper()
	operation, ok := digMap(document, "paths", path, method)
	if !ok {
		t.Fatalf("OpenAPI operation %s %s not found", strings.ToUpper(method), path)
	}
	return operation
}

func digMap(current map[string]any, path ...string) (map[string]any, bool) {
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func digString(current map[string]any, path ...string) (string, bool) {
	for i, key := range path {
		value, ok := current[key]
		if !ok {
			return "", false
		}
		if i == len(path)-1 {
			result, ok := value.(string)
			return result, ok
		}
		next, ok := value.(map[string]any)
		if !ok {
			return "", false
		}
		current = next
	}
	return "", false
}
