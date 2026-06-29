package service

import (
	"context"
	"slices"
	"sync"
	"time"
)

type memoryRepository struct {
	mu          sync.Mutex
	profiles    map[string]ModelProfile
	credentials map[string]ProviderCredential
	revisions   []ModelProfileRevision
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		profiles:    map[string]ModelProfile{},
		credentials: map[string]ProviderCredential{},
	}
}

func (r *memoryRepository) CheckReady(context.Context) error { return nil }

func (r *memoryRepository) ListModelProfiles(ctx context.Context, filter ListModelProfilesFilter) ([]ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	items := []ModelProfile{}
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
		items = append(items, cloneProfile(profile))
	}
	return items, nil
}

func (r *memoryRepository) GetModelProfile(ctx context.Context, id string) (ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, ok := r.profiles[id]
	if !ok || profile.DeletedAt != nil {
		return ModelProfile{}, ErrNotFound
	}
	return cloneProfile(profile), nil
}

func (r *memoryRepository) CreateModelProfile(ctx context.Context, profile ModelProfile, credential ProviderCredential, revision ModelProfileRevision) (ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, current := range r.profiles {
		if profile.Enabled && profile.IsDefault && current.DeletedAt == nil && current.Purpose == profile.Purpose {
			current.IsDefault = false
			r.profiles[id] = current
		}
	}
	r.credentials[credential.ID] = credential
	r.profiles[profile.ID] = cloneProfile(profile)
	r.revisions = append(r.revisions, revision)
	return cloneProfile(profile), nil
}

func (r *memoryRepository) UpdateModelProfile(ctx context.Context, profile ModelProfile, credential *ProviderCredential, revision ModelProfileRevision) (ModelProfile, error) {
	if err := ctx.Err(); err != nil {
		return ModelProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.profiles[profile.ID]; !ok {
		return ModelProfile{}, ErrNotFound
	}
	if profile.Enabled && profile.IsDefault {
		for id, current := range r.profiles {
			if id != profile.ID && current.DeletedAt == nil && current.Purpose == profile.Purpose {
				current.IsDefault = false
				r.profiles[id] = current
			}
		}
	}
	if credential != nil {
		for id, current := range r.credentials {
			if current.ProfileID == profile.ID && current.Status == CredentialActive {
				now := profile.UpdatedAt
				current.Status = CredentialRotated
				current.RotatedAt = &now
				r.credentials[id] = current
			}
		}
		r.credentials[credential.ID] = *credential
	}
	r.profiles[profile.ID] = cloneProfile(profile)
	r.revisions = append(r.revisions, revision)
	return cloneProfile(profile), nil
}

func (r *memoryRepository) SoftDeleteModelProfile(ctx context.Context, id string, deletedAt time.Time, revision ModelProfileRevision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	profile, ok := r.profiles[id]
	if !ok || profile.DeletedAt != nil {
		return ErrNotFound
	}
	profile.DeletedAt = &deletedAt
	profile.Enabled = false
	profile.UpdatedAt = deletedAt
	r.profiles[id] = profile
	r.revisions = append(r.revisions, revision)
	return nil
}

func cloneProfile(profile ModelProfile) ModelProfile {
	profile.DefaultParameters = slices.Clone(profile.DefaultParameters)
	if profile.Dimensions != nil {
		value := *profile.Dimensions
		profile.Dimensions = &value
	}
	if profile.TopN != nil {
		value := *profile.TopN
		profile.TopN = &value
	}
	if profile.DeletedAt != nil {
		value := *profile.DeletedAt
		profile.DeletedAt = &value
	}
	return profile
}
