package tools

const (
	KnowledgeMCPToolSearch        = "search_knowledge"
	KnowledgeMCPToolListDocuments = "list_documents"
	KnowledgeMCPToolGetDocument   = "get_document"
	KnowledgeMCPToolListChunks    = "list_document_chunks"
)

var DefaultKnowledgeMCPToolNames = []string{
	KnowledgeMCPToolSearch,
	KnowledgeMCPToolListDocuments,
	KnowledgeMCPToolGetDocument,
	KnowledgeMCPToolListChunks,
}

func ModelFacingKnowledgeMCPToolNames(alias string) []string {
	names := make([]string, 0, len(DefaultKnowledgeMCPToolNames))
	for _, name := range DefaultKnowledgeMCPToolNames {
		names = append(names, alias+"__"+name)
	}
	return names
}
