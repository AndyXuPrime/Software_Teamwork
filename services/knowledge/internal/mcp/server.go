package mcp

import (
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/adapter"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/aigateway"
)

const serverName = "knowledge-mcp"

// NewStreamableHTTPHandler returns an HTTP handler for MCP Streamable HTTP transport.
func NewStreamableHTTPHandler(adapterServer *adapter.Server, chatClient *aigateway.ChatClient) http.Handler {
	return sdkmcp.NewStreamableHTTPHandler(func(r *http.Request) *sdkmcp.Server {
		return newMCPServer(adapterServer, callerFromHTTP(r), chatClient)
	}, nil)
}

func newMCPServer(adapterServer *adapter.Server, caller CallerContext, chatClient *aigateway.ChatClient) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: serverName, Version: "0.1.0"}, nil)
	h := &toolHandlers{
		bridge: NewBridge(adapterServer),
		caller: caller,
		chat:   chatClient,
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolSearchKnowledge,
		Description: "Pure retrieval over knowledge bases; returns ranked chunks with citation fields and no LLM synthesis.",
	}, h.searchKnowledge)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolAnswerFromKnowledge,
		Description: "Retrieve relevant chunks and synthesize an answer via AI Gateway.",
	}, h.answerFromKnowledge)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolListKnowledgeBases,
		Description: "List knowledge bases visible to the caller.",
	}, h.listKnowledgeBases)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolGetKnowledgeBase,
		Description: "Get a knowledge base by ID.",
	}, h.getKnowledgeBase)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolCreateKnowledgeBase,
		Description: "Create a new knowledge base.",
	}, h.createKnowledgeBase)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolUpdateKnowledgeBase,
		Description: "Update knowledge base metadata.",
	}, h.updateKnowledgeBase)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolDeleteKnowledgeBase,
		Description: "Soft-delete a knowledge base.",
	}, h.deleteKnowledgeBase)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolListDocuments,
		Description: "List documents in a knowledge base.",
	}, h.listDocuments)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolGetDocument,
		Description: "Get document processing details by ID.",
	}, h.getDocument)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolCreateDocument,
		Description: "Upload a document for async parse and indexing.",
	}, h.createDocument)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolUpdateDocument,
		Description: "Update document metadata such as tags.",
	}, h.updateDocument)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolDeleteDocument,
		Description: "Soft-delete a document and queue cleanup.",
	}, h.deleteDocument)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolListDocumentChunks,
		Description: "List indexed chunks for a document.",
	}, h.listDocumentChunks)

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        toolGetDocumentContent,
		Description: "Download original document content.",
	}, h.getDocumentContent)

	return server
}

// NewInMemoryServer creates an MCP server for unit tests without HTTP transport.
func NewInMemoryServer(adapterServer *adapter.Server, caller CallerContext, chatClient *aigateway.ChatClient) *sdkmcp.Server {
	return newMCPServer(adapterServer, caller, chatClient)
}
