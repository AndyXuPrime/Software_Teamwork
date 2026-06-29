package httpapi

import "testing"

func TestActiveRouteMatrixCoversGatewayOwnerMap(t *testing.T) {
	if got, want := activeOperationCount(), 97; got != want {
		t.Fatalf("active operations = %d, want %d", got, want)
	}
	ownerCounts := map[string]int{}
	seen := map[string]bool{}
	for _, route := range activeProxyRoutes {
		ownerCounts[route.Owner]++
		key := route.Method + " " + route.Pattern
		if seen[key] {
			t.Fatalf("duplicate route %s", key)
		}
		seen[key] = true
	}
	expected := map[string]int{
		"knowledge":  18,
		"ai-gateway": 5,
		"document":   43,
		"qa":         25,
	}
	for owner, want := range expected {
		if got := ownerCounts[owner]; got != want {
			t.Fatalf("%s routes = %d, want %d", owner, got, want)
		}
	}
}
