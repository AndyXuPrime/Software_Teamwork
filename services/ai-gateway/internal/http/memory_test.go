package httpapi

import (
	"context"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/ai-gateway/internal/service"
)

type memoryRepository struct {
	mu          sync.Mutex
	profiles    map[string]service.ModelProfile
	credentials map[string]service.ProviderCredential
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{profiles: map[string]service.ModelProfile{}, credentials: map[string]service.ProviderCredential{}}
}

func (r *memoryRepository) CheckReady(context.Context) error { return nil }

func (r *memoryRepository) ListModelProfiles(ctx context.Context, filter service.ListModelProfilesFilter) ([]service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	items := []service.ModelProfile{}
	for _, profile := range r.profiles {
		if profile.DeletedAt != nil {
			continue
		}
		if filter.Purpose != nil && profile.Purpose != *filter.Purpose {
			continue
		}
		if filter.Enabled != nil && profile.Enabled != *filter.Enabled {
			continue
		}
		items = append(items, profile)
	}
	return items, nil
}

func (r *memoryRepository) GetModelProfile(ctx context.Context, id string) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, ok := r.profiles[id]
	if !ok || profile.DeletedAt != nil {
		return service.ModelProfile{}, service.ErrNotFound
	}
	return profile, nil
}

func (r *memoryRepository) CreateModelProfile(ctx context.Context, profile service.ModelProfile, credential service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, current := range r.profiles {
		if profile.Enabled && profile.IsDefault && current.Purpose == profile.Purpose && current.DeletedAt == nil {
			current.IsDefault = false
			r.profiles[id] = current
		}
	}
	r.profiles[profile.ID] = profile
	r.credentials[credential.ID] = credential
	return profile, nil
}

func (r *memoryRepository) UpdateModelProfile(ctx context.Context, profile service.ModelProfile, credential *service.ProviderCredential, revision service.ModelProfileRevision) (service.ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return service.ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.profiles[profile.ID]; !ok {
		return service.ModelProfile{}, service.ErrNotFound
	}
	if credential != nil {
		r.credentials[credential.ID] = *credential
	}
	r.profiles[profile.ID] = profile
	return profile, nil
}

func (r *memoryRepository) SoftDeleteModelProfile(ctx context.Context, id string, deletedAt time.Time, revision service.ModelProfileRevision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, ok := r.profiles[id]
	if !ok {
		return service.ErrNotFound
	}
	profile.DeletedAt = &deletedAt
	profile.Enabled = false
	r.profiles[id] = profile
	return nil
}
