package storage

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/file/internal/service"
)

type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	data        []byte
	contentType string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{objects: map[string]memoryObject{}}
}

func (s *MemoryStore) Put(ctx context.Context, key string, body io.Reader, contentType string, sizeBytes int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if body == nil {
		return service.ErrNotFound
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if sizeBytes >= 0 && int64(len(data)) != sizeBytes {
		return service.ErrConflict
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = memoryObject{data: append([]byte(nil), data...), contentType: contentType}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, key string) (service.StoredObject, error) {
	select {
	case <-ctx.Done():
		return service.StoredObject{}, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	object, exists := s.objects[key]
	if !exists {
		return service.StoredObject{}, service.ErrNotFound
	}
	data := append([]byte(nil), object.data...)
	return service.StoredObject{
		Body:        io.NopCloser(bytes.NewReader(data)),
		ContentType: object.contentType,
		SizeBytes:   int64(len(data)),
	}, nil
}

func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.objects[key]; !exists {
		return service.ErrNotFound
	}
	delete(s.objects, key)
	return nil
}
