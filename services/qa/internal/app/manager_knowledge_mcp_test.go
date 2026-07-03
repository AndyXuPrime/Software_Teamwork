package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/agent"
	toolspkg "github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHasRequiredKnowledgeMCPTools(t *testing.T) {
	definitions := []agent.ToolDefinition{
		{Function: agent.FunctionTool{Name: "knowledge__search"}},
		{Function: agent.FunctionTool{Name: "knowledge__list_documents"}},
		{Function: agent.FunctionTool{Name: "knowledge__get_document"}},
		{Function: agent.FunctionTool{Name: "knowledge__get_chunk"}},
	}
	if !hasRequiredKnowledgeMCPTools(definitions, "knowledge") {
		t.Fatal("expected the complete Knowledge MCP tool set to be accepted")
	}
	if hasRequiredKnowledgeMCPTools(definitions[:3], "knowledge") {
		t.Fatal("expected an incomplete Knowledge MCP tool set to be rejected for HTTP fallback")
	}
	if hasRequiredKnowledgeMCPTools(definitions, "other") {
		t.Fatal("expected alias mismatch to be rejected")
	}
}

func TestBuildKnowledgeProviderPrefersCompleteMCPDiscovery(t *testing.T) {
	server := newKnowledgeMCPTestServer(t, toolspkg.DefaultKnowledgeMCPToolNames)
	manager := &Manager{
		retriever: &fakeKnowledgeRetriever{},
		cfg: ManagerConfig{
			KnowledgeMCPURL: server.URL, KnowledgeMCPAlias: "knowledge",
			KnowledgeMCPTimeout: time.Second, DefaultToolTimeout: time.Second,
		},
	}
	provider, client, err := manager.buildKnowledgeProvider(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected a connected Knowledge MCP client")
	}
	defer client.Close()
	definitions, err := provider.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !hasRequiredKnowledgeMCPTools(definitions, "knowledge") || hasTool(definitions, toolspkg.ToolSearchKnowledge) {
		t.Fatalf("MCP definitions = %#v", definitions)
	}
}

func TestBuildKnowledgeProviderFallsBackWhenDiscoveryIsIncomplete(t *testing.T) {
	server := newKnowledgeMCPTestServer(t, toolspkg.DefaultKnowledgeMCPToolNames[:3])
	manager := &Manager{
		retriever: &fakeKnowledgeRetriever{},
		cfg: ManagerConfig{
			KnowledgeMCPURL: server.URL, KnowledgeMCPAlias: "knowledge",
			KnowledgeMCPTimeout: time.Second, DefaultToolTimeout: time.Second,
		},
	}
	provider, client, err := manager.buildKnowledgeProvider(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client != nil {
		defer client.Close()
		t.Fatal("incomplete discovery must close the MCP client")
	}
	definitions, err := provider.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !hasTool(definitions, toolspkg.ToolSearchKnowledge) || hasTool(definitions, "knowledge__search") {
		t.Fatalf("fallback definitions = %#v", definitions)
	}
}

type fakeKnowledgeRetriever struct{}

func (*fakeKnowledgeRetriever) Retrieve(context.Context, string, service.RetrievalTestInput) ([]service.RetrievalTestResult, error) {
	return nil, nil
}

func newKnowledgeMCPTestServer(t *testing.T, toolNames []string) *httptest.Server {
	t.Helper()
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		server := mcp.NewServer(&mcp.Implementation{Name: "knowledge-test", Version: "0.1.0"}, nil)
		for _, name := range toolNames {
			name := name
			server.AddTool(&mcp.Tool{
				Name: name,
				InputSchema: map[string]any{
					"type": "object", "properties": map[string]any{},
				},
			}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{}, nil
			})
		}
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func hasTool(definitions []agent.ToolDefinition, name string) bool {
	for _, definition := range definitions {
		if definition.Function.Name == name {
			return true
		}
	}
	return false
}
