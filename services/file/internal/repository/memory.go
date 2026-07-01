package repository

import (
	"context"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

type MemoryRepository struct {
	mu    sync.RWMutex
	files map[string]service.FileObject
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		files: map[string]service.FileObject{},
	}
}

func (r *MemoryRepository) CreateFile(ctx context.Context, file service.FileObject) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.files[file.ID]; exists {
		return service.FileObject{}, service.ErrConflict
	}
	stored := cloneFileObject(file)
	r.files[stored.ID] = stored
	return cloneFileObject(stored), nil
}

func (r *MemoryRepository) FindFileByID(ctx context.Context, id string) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	file, exists := r.files[id]
	if !exists || file.DeletedAt != nil || file.Status != service.FileStatusAvailable {
		return service.FileObject{}, service.ErrNotFound
	}
	return cloneFileObject(file), nil
}

func (r *MemoryRepository) MarkFileDeleteRequested(ctx context.Context, id string, deletedAt time.Time) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	file, exists := r.files[id]
	if !exists {
		return service.FileObject{}, service.ErrNotFound
	}
	if file.Status == service.FileStatusPurged {
		return cloneFileObject(file), nil
	}
	deleted := deletedAt.UTC()
	if file.DeletedAt == nil {
		file.DeletedAt = &deleted
	}
	if file.DeleteRequestedAt == nil {
		file.DeleteRequestedAt = &deleted
	}
	file.UpdatedAt = deleted
	file.Status = service.FileStatusDeleteRequested
	r.files[id] = file
	return cloneFileObject(file), nil
}

func (r *MemoryRepository) MarkFilePurging(ctx context.Context, id string, purgingAt time.Time) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	file, exists := r.files[id]
	if !exists {
		return service.FileObject{}, service.ErrNotFound
	}
	if file.Status == service.FileStatusPurged {
		return cloneFileObject(file), nil
	}
	purging := purgingAt.UTC()
	file.UpdatedAt = purging
	file.Status = service.FileStatusPurging
	r.files[id] = file
	return cloneFileObject(file), nil
}

func (r *MemoryRepository) MarkFilePurged(ctx context.Context, id string, purgedAt time.Time) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	file, exists := r.files[id]
	if !exists {
		return service.FileObject{}, service.ErrNotFound
	}
	purged := purgedAt.UTC()
	file.PurgedAt = &purged
	file.UpdatedAt = purged
	file.Status = service.FileStatusPurged
	file.LastErrorCode = ""
	file.LastErrorMessage = ""
	r.files[id] = file
	return cloneFileObject(file), nil
}

func (r *MemoryRepository) MarkFilePurgeFailed(ctx context.Context, id string, code string, message string, failedAt time.Time) (service.FileObject, error) {
	if err := ctx.Err(); err != nil {
		return service.FileObject{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	file, exists := r.files[id]
	if !exists {
		return service.FileObject{}, service.ErrNotFound
	}
	failed := failedAt.UTC()
	file.UpdatedAt = failed
	file.Status = service.FileStatusFailed
	file.LastErrorCode = code
	file.LastErrorMessage = message
	r.files[id] = file
	return cloneFileObject(file), nil
}

func cloneFileObject(file service.FileObject) service.FileObject {
	if file.DeletedAt != nil {
		value := *file.DeletedAt
		file.DeletedAt = &value
	}
	if file.DeleteRequestedAt != nil {
		value := *file.DeleteRequestedAt
		file.DeleteRequestedAt = &value
	}
	if file.PurgedAt != nil {
		value := *file.PurgedAt
		file.PurgedAt = &value
	}
	return file
}
