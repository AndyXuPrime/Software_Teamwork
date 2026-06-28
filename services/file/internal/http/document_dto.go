package httpapi

import (
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

type documentSummary struct {
	ID              string                 `json:"id"`
	KnowledgeBaseID string                 `json:"knowledgeBaseId"`
	Name            string                 `json:"name"`
	Status          service.DocumentStatus `json:"status"`
	Tags            []string               `json:"tags,omitempty"`
	ErrorMessage    *string                `json:"errorMessage,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
	ContentType     string                 `json:"contentType"`
	SizeBytes       int64                  `json:"sizeBytes"`
}

func documentSummaryFromDomain(doc service.Document) documentSummary {
	return documentSummary{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		Name:            doc.Name,
		Status:          doc.Status,
		Tags:            append([]string(nil), doc.Tags...),
		ErrorMessage:    doc.ErrorMessage,
		CreatedAt:       doc.CreatedAt.UTC().Format(time.RFC3339),
		ContentType:     doc.ContentType,
		SizeBytes:       doc.SizeBytes,
	}
}
