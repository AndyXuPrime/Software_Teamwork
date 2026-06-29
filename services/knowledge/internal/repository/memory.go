package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type MemoryRepository struct {
	mu             sync.RWMutex
	knowledgeBases map[string]service.KnowledgeBase
	documents      map[string]service.KnowledgeDocument
	jobs           map[string]service.ProcessingJob
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		knowledgeBases: map[string]service.KnowledgeBase{},
		documents:      map[string]service.KnowledgeDocument{},
		jobs:           map[string]service.ProcessingJob{},
	}
}

func (r *MemoryRepository) CreateKnowledgeBase(ctx context.Context, input service.CreateKnowledgeBaseRecord) (service.KnowledgeBase, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeBase{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.knowledgeBases[input.ID]; exists {
		return service.KnowledgeBase{}, service.ErrConflict
	}
	kb := service.KnowledgeBase{
		ID:                input.ID,
		Name:              input.Name,
		Description:       input.Description,
		DocType:           input.DocType,
		ChunkStrategy:     cloneRaw(input.ChunkStrategy),
		RetrievalStrategy: cloneRaw(input.RetrievalStrategy),
		CreatedBy:         input.CreatedBy,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	}
	r.knowledgeBases[kb.ID] = kb
	return r.hydrateKnowledgeBaseLocked(kb), nil
}

func (r *MemoryRepository) ListKnowledgeBases(ctx context.Context, scope service.AccessScope, page service.PageInput) (service.KnowledgeBaseList, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeBaseList{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]service.KnowledgeBase, 0, len(r.knowledgeBases))
	for _, kb := range r.knowledgeBases {
		if kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
			continue
		}
		items = append(items, r.hydrateKnowledgeBaseLocked(kb))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := int64(len(items))
	items = paginate(items, page)
	return service.KnowledgeBaseList{
		Items: cloneKnowledgeBases(items),
		Page: service.Page{
			Page:     page.Page,
			PageSize: page.PageSize,
			Total:    total,
		},
	}, nil
}

func (r *MemoryRepository) GetKnowledgeBase(ctx context.Context, id string, scope service.AccessScope) (service.KnowledgeBase, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeBase{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	kb, exists := r.knowledgeBases[id]
	if !exists || kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
		return service.KnowledgeBase{}, service.ErrNotFound
	}
	return cloneKnowledgeBase(r.hydrateKnowledgeBaseLocked(kb)), nil
}

func (r *MemoryRepository) UpdateKnowledgeBase(ctx context.Context, input service.UpdateKnowledgeBaseRecord, scope service.AccessScope) (service.KnowledgeBase, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeBase{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	kb, exists := r.knowledgeBases[input.ID]
	if !exists || kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
		return service.KnowledgeBase{}, service.ErrNotFound
	}
	if input.Name != nil {
		kb.Name = *input.Name
	}
	if input.Description != nil {
		kb.Description = *input.Description
	}
	if input.DocType != nil {
		kb.DocType = *input.DocType
	}
	if input.ChunkStrategy != nil {
		kb.ChunkStrategy = cloneRaw(*input.ChunkStrategy)
	}
	if input.RetrievalStrategy != nil {
		kb.RetrievalStrategy = cloneRaw(*input.RetrievalStrategy)
	}
	kb.UpdatedAt = input.UpdatedAt
	r.knowledgeBases[kb.ID] = kb
	return cloneKnowledgeBase(r.hydrateKnowledgeBaseLocked(kb)), nil
}

func (r *MemoryRepository) SoftDeleteKnowledgeBase(ctx context.Context, id string, deletedAt time.Time, scope service.AccessScope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	kb, exists := r.knowledgeBases[id]
	if !exists || kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
		return service.ErrNotFound
	}
	deleted := deletedAt.UTC()
	kb.DeletedAt = &deleted
	kb.UpdatedAt = deleted
	r.knowledgeBases[id] = kb

	for docID, doc := range r.documents {
		if doc.KnowledgeBaseID == id && doc.DeletedAt == nil {
			doc.DeletedAt = &deleted
			doc.UpdatedAt = deleted
			r.documents[docID] = doc
		}
	}
	return nil
}

func (r *MemoryRepository) CreateDocumentWithJob(ctx context.Context, input service.CreateDocumentWithJobRecord, scope service.AccessScope) (service.KnowledgeDocument, service.ProcessingJob, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeDocument{}, service.ProcessingJob{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	kb, exists := r.knowledgeBases[input.KnowledgeBaseID]
	if !exists || kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
		return service.KnowledgeDocument{}, service.ProcessingJob{}, service.ErrNotFound
	}
	if _, exists := r.documents[input.DocumentID]; exists {
		return service.KnowledgeDocument{}, service.ProcessingJob{}, service.ErrConflict
	}
	if _, exists := r.jobs[input.JobID]; exists {
		return service.KnowledgeDocument{}, service.ProcessingJob{}, service.ErrConflict
	}

	fileRef := input.FileRef
	contentType := input.ContentType
	sizeBytes := input.SizeBytes
	jobID := input.CurrentJobID
	stage := input.JobStage
	message := input.JobMessage
	doc := service.KnowledgeDocument{
		ID:              input.DocumentID,
		KnowledgeBaseID: input.KnowledgeBaseID,
		FileRef:         &fileRef,
		Name:            input.Name,
		ContentType:     &contentType,
		SizeBytes:       &sizeBytes,
		Status:          input.Status,
		Tags:            append([]string(nil), input.Tags...),
		CurrentJobID:    &jobID,
		CreatedBy:       input.CreatedBy,
		CreatedAt:       input.CreatedAt,
		UpdatedAt:       input.UpdatedAt,
	}
	documentID := input.DocumentID
	job := service.ProcessingJob{
		ID:              input.JobID,
		KnowledgeBaseID: input.KnowledgeBaseID,
		DocumentID:      &documentID,
		JobType:         input.JobType,
		Status:          input.JobStatus,
		CurrentStage:    &stage,
		ProgressPercent: 0,
		Message:         &message,
		Attempts:        0,
		MaxAttempts:     input.MaxAttempts,
		CreatedAt:       input.CreatedAt,
		UpdatedAt:       input.UpdatedAt,
	}
	r.documents[doc.ID] = doc
	r.jobs[job.ID] = job
	return cloneDocument(r.hydrateDocumentLocked(doc)), cloneJob(job), nil
}

func (r *MemoryRepository) MarkDocumentJobFailed(ctx context.Context, documentID string, jobID string, code string, message string, failedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, docExists := r.documents[documentID]
	job, jobExists := r.jobs[jobID]
	if !docExists || !jobExists {
		return service.ErrNotFound
	}
	doc.Status = service.DocumentStatusFailed
	doc.ErrorCode = cloneStringPtr(&code)
	doc.ErrorMessage = cloneStringPtr(&message)
	doc.UpdatedAt = failedAt
	job.Status = service.JobStatusFailed
	job.ErrorCode = cloneStringPtr(&code)
	job.ErrorMessage = cloneStringPtr(&message)
	job.FinishedAt = &failedAt
	job.UpdatedAt = failedAt
	r.documents[documentID] = doc
	r.jobs[jobID] = job
	return nil
}

func (r *MemoryRepository) ListDocumentsByKnowledgeBase(ctx context.Context, knowledgeBaseID string, status *service.DocumentStatus, scope service.AccessScope, page service.PageInput) (service.DocumentList, error) {
	if err := ctx.Err(); err != nil {
		return service.DocumentList{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	kb, exists := r.knowledgeBases[knowledgeBaseID]
	if !exists || kb.DeletedAt != nil || !canRead(kb.CreatedBy, scope) {
		return service.DocumentList{}, service.ErrNotFound
	}

	items := make([]service.KnowledgeDocument, 0)
	for _, doc := range r.documents {
		if doc.KnowledgeBaseID != knowledgeBaseID || doc.DeletedAt != nil {
			continue
		}
		if status != nil && doc.Status != *status {
			continue
		}
		if !canRead(doc.CreatedBy, scope) && !canRead(kb.CreatedBy, scope) {
			continue
		}
		items = append(items, r.hydrateDocumentLocked(doc))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := int64(len(items))
	items = paginate(items, page)
	return service.DocumentList{
		Items: cloneDocuments(items),
		Page: service.Page{
			Page:     page.Page,
			PageSize: page.PageSize,
			Total:    total,
		},
	}, nil
}

func (r *MemoryRepository) GetDocument(ctx context.Context, id string, scope service.AccessScope) (service.KnowledgeDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.KnowledgeDocument{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc, exists := r.documents[id]
	if !exists || doc.DeletedAt != nil {
		return service.KnowledgeDocument{}, service.ErrNotFound
	}
	kb, exists := r.knowledgeBases[doc.KnowledgeBaseID]
	if !exists || kb.DeletedAt != nil {
		return service.KnowledgeDocument{}, service.ErrNotFound
	}
	if !canRead(doc.CreatedBy, scope) && !canRead(kb.CreatedBy, scope) {
		return service.KnowledgeDocument{}, service.ErrNotFound
	}
	return cloneDocument(r.hydrateDocumentLocked(doc)), nil
}

func (r *MemoryRepository) SeedKnowledgeBase(kb service.KnowledgeBase) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.knowledgeBases[kb.ID] = cloneKnowledgeBase(kb)
}

func (r *MemoryRepository) SeedDocument(doc service.KnowledgeDocument) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.documents[doc.ID] = cloneDocument(doc)
}

func (r *MemoryRepository) hydrateKnowledgeBaseLocked(kb service.KnowledgeBase) service.KnowledgeBase {
	kb.DocumentCount = 0
	kb.ChunkCount = 0
	for _, doc := range r.documents {
		if doc.KnowledgeBaseID == kb.ID && doc.DeletedAt == nil {
			kb.DocumentCount++
			kb.ChunkCount += doc.ChunkCount
		}
	}
	return cloneKnowledgeBase(kb)
}

func (r *MemoryRepository) hydrateDocumentLocked(doc service.KnowledgeDocument) service.KnowledgeDocument {
	return cloneDocument(doc)
}

func canRead(createdBy string, scope service.AccessScope) bool {
	return scope.CanReadAll || createdBy == scope.UserID
}

func paginate[T any](items []T, page service.PageInput) []T {
	start := (page.Page - 1) * page.PageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + page.PageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func cloneKnowledgeBases(items []service.KnowledgeBase) []service.KnowledgeBase {
	out := make([]service.KnowledgeBase, len(items))
	for i, item := range items {
		out[i] = cloneKnowledgeBase(item)
	}
	return out
}

func cloneKnowledgeBase(kb service.KnowledgeBase) service.KnowledgeBase {
	kb.ChunkStrategy = cloneRaw(kb.ChunkStrategy)
	kb.RetrievalStrategy = cloneRaw(kb.RetrievalStrategy)
	if kb.DeletedAt != nil {
		value := *kb.DeletedAt
		kb.DeletedAt = &value
	}
	return kb
}

func cloneDocuments(items []service.KnowledgeDocument) []service.KnowledgeDocument {
	out := make([]service.KnowledgeDocument, len(items))
	for i, item := range items {
		out[i] = cloneDocument(item)
	}
	return out
}

func cloneDocument(doc service.KnowledgeDocument) service.KnowledgeDocument {
	doc.Tags = append([]string(nil), doc.Tags...)
	doc.FileRef = cloneStringPtr(doc.FileRef)
	doc.ContentType = cloneStringPtr(doc.ContentType)
	doc.SizeBytes = cloneInt64Ptr(doc.SizeBytes)
	doc.ErrorCode = cloneStringPtr(doc.ErrorCode)
	doc.ErrorMessage = cloneStringPtr(doc.ErrorMessage)
	doc.ParserBackend = cloneStringPtr(doc.ParserBackend)
	doc.CurrentJobID = cloneStringPtr(doc.CurrentJobID)
	if doc.DeletedAt != nil {
		value := *doc.DeletedAt
		doc.DeletedAt = &value
	}
	return doc
}

func cloneJob(job service.ProcessingJob) service.ProcessingJob {
	job.DocumentID = cloneStringPtr(job.DocumentID)
	job.CurrentStage = cloneStringPtr(job.CurrentStage)
	job.Message = cloneStringPtr(job.Message)
	job.ErrorCode = cloneStringPtr(job.ErrorCode)
	job.ErrorMessage = cloneStringPtr(job.ErrorMessage)
	if job.StartedAt != nil {
		value := *job.StartedAt
		job.StartedAt = &value
	}
	if job.FinishedAt != nil {
		value := *job.FinishedAt
		job.FinishedAt = &value
	}
	return job
}

func cloneRaw(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
