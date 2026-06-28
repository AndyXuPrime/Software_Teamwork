package repository

import (
	"context"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	documents map[string]service.Document
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{documents: map[string]service.Document{}}
}

func (r *MemoryRepository) Create(ctx context.Context, doc service.Document) (service.Document, error) {
	select {
	case <-ctx.Done():
		return service.Document{}, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.documents[doc.ID]; exists {
		return service.Document{}, service.ErrConflict
	}
	stored := cloneDocument(doc)
	r.documents[stored.ID] = stored
	return cloneDocument(stored), nil
}

func (r *MemoryRepository) FindByID(ctx context.Context, id string) (service.Document, error) {
	select {
	case <-ctx.Done():
		return service.Document{}, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, exists := r.documents[id]
	if !exists || doc.DeletedAt != nil {
		return service.Document{}, service.ErrNotFound
	}
	return cloneDocument(doc), nil
}

func (r *MemoryRepository) ReplaceTags(ctx context.Context, id string, tags []string) (service.Document, error) {
	select {
	case <-ctx.Done():
		return service.Document{}, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	doc, exists := r.documents[id]
	if !exists || doc.DeletedAt != nil {
		return service.Document{}, service.ErrNotFound
	}
	doc.Tags = append([]string(nil), tags...)
	r.documents[id] = doc
	return cloneDocument(doc), nil
}

func (r *MemoryRepository) MarkDeleted(ctx context.Context, id string, deletedAt time.Time) (service.Document, error) {
	select {
	case <-ctx.Done():
		return service.Document{}, ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	doc, exists := r.documents[id]
	if !exists || doc.DeletedAt != nil {
		return service.Document{}, service.ErrNotFound
	}
	deleted := deletedAt.UTC()
	doc.DeletedAt = &deleted
	r.documents[id] = doc
	return cloneDocument(doc), nil
}

func cloneDocument(doc service.Document) service.Document {
	doc.Tags = append([]string(nil), doc.Tags...)
	if doc.ErrorMessage != nil {
		message := *doc.ErrorMessage
		doc.ErrorMessage = &message
	}
	if doc.DeletedAt != nil {
		deletedAt := *doc.DeletedAt
		doc.DeletedAt = &deletedAt
	}
	return doc
}
