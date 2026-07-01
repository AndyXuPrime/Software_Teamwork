package httpapi_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestInternalOpenAPIDoesNotDriftFromDocs(t *testing.T) {
	docsContract := readNormalizedYAML(t, "../../../../docs/services/file/api/internal.openapi.yaml")
	serviceContract := readNormalizedYAML(t, "../../api/openapi.yaml")

	if !reflect.DeepEqual(docsContract, serviceContract) {
		t.Fatalf("services/file/api/openapi.yaml drifted from docs/services/file/api/internal.openapi.yaml\n--- docs\n%s\n--- service\n%s",
			prettyJSON(t, docsContract),
			prettyJSON(t, serviceContract),
		)
	}
}

func readNormalizedYAML(t *testing.T, path string) any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var parsed any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return normalizeYAML(parsed)
}

func normalizeYAML(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, value := range typed {
			normalized[key] = normalizeYAML(value)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, value := range typed {
			normalized[toString(key)] = normalizeYAML(value)
		}
		return normalized
	case []any:
		for i, item := range typed {
			typed[i] = normalizeYAML(item)
		}
		return typed
	default:
		return typed
	}
}

func toString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func prettyJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal normalized contract: %v", err)
	}
	return string(data)
}
