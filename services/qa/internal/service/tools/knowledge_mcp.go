package tools

const (
	KnowledgeMCPToolSearch        = "search"
	KnowledgeMCPToolListDocuments = "list_documents"
	KnowledgeMCPToolGetDocument   = "get_document"
	KnowledgeMCPToolGetChunk      = "get_chunk"
)

var DefaultKnowledgeMCPToolNames = []string{
	KnowledgeMCPToolSearch,
	KnowledgeMCPToolListDocuments,
	KnowledgeMCPToolGetDocument,
	KnowledgeMCPToolGetChunk,
}

func ModelFacingKnowledgeMCPToolNames(alias string) []string {
	names := make([]string, 0, len(DefaultKnowledgeMCPToolNames))
	for _, name := range DefaultKnowledgeMCPToolNames {
		names = append(names, alias+"__"+name)
	}
	return names
}
