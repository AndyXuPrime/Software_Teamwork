package repository

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	parserConfigs map[string]service.ParserConfig
	parserAudits  []service.ParserConfigAudit
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		parserConfigs: map[string]service.ParserConfig{},
	}
}

func (r *MemoryRepository) ListParserConfigs(ctx context.Context, enabled *bool) ([]service.ParserConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]service.ParserConfig, 0, len(r.parserConfigs))
	for _, config := range r.parserConfigs {
		if config.DeletedAt != nil || enabled != nil && config.Enabled != *enabled {
			continue
		}
		items = append(items, cloneParserConfig(config))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) GetParserConfig(ctx context.Context, id string) (service.ParserConfig, error) {
	if err := ctx.Err(); err != nil {
		return service.ParserConfig{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	config, ok := r.parserConfigs[id]
	if !ok || config.DeletedAt != nil {
		return service.ParserConfig{}, service.ErrNotFound
	}
	return cloneParserConfig(config), nil
}

func (r *MemoryRepository) CreateParserConfig(ctx context.Context, config service.ParserConfig, audit service.ParserConfigAudit) (service.ParserConfig, error) {
	if err := ctx.Err(); err != nil {
		return service.ParserConfig{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.parserConfigs {
		if existing.DeletedAt == nil && strings.EqualFold(existing.Name, config.Name) {
			return service.ParserConfig{}, service.ErrConflict
		}
	}
	if config.IsDefault {
		r.clearDefaultLocked(config.ID, config.UpdatedAt)
	}
	r.parserConfigs[config.ID] = cloneParserConfig(config)
	r.parserAudits = append(r.parserAudits, cloneParserAudit(audit))
	return cloneParserConfig(config), nil
}

func (r *MemoryRepository) UpdateParserConfig(ctx context.Context, config service.ParserConfig, audit service.ParserConfigAudit) (service.ParserConfig, error) {
	if err := ctx.Err(); err != nil {
		return service.ParserConfig{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.parserConfigs[config.ID]; !ok || existing.DeletedAt != nil {
		return service.ParserConfig{}, service.ErrNotFound
	}
	for id, existing := range r.parserConfigs {
		if id != config.ID && existing.DeletedAt == nil && strings.EqualFold(existing.Name, config.Name) {
			return service.ParserConfig{}, service.ErrConflict
		}
	}
	if config.IsDefault {
		r.clearDefaultLocked(config.ID, config.UpdatedAt)
	}
	r.parserConfigs[config.ID] = cloneParserConfig(config)
	r.parserAudits = append(r.parserAudits, cloneParserAudit(audit))
	return cloneParserConfig(config), nil
}

func (r *MemoryRepository) SoftDeleteParserConfig(ctx context.Context, id string, deletedAt time.Time, audit service.ParserConfigAudit) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	config, ok := r.parserConfigs[id]
	if !ok || config.DeletedAt != nil {
		return service.ErrNotFound
	}
	if config.IsDefault {
		return service.ErrConflict
	}
	config.Enabled = false
	config.UpdatedAt = deletedAt
	config.DeletedAt = &deletedAt
	r.parserConfigs[id] = config
	r.parserAudits = append(r.parserAudits, cloneParserAudit(audit))
	return nil
}

func (r *MemoryRepository) GetEffectiveParserConfig(ctx context.Context, contentType string) (service.ParserConfig, error) {
	if err := ctx.Err(); err != nil {
		return service.ParserConfig{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	type candidate struct {
		config service.ParserConfig
		rank   int
	}
	candidates := []candidate{}
	for _, config := range r.parserConfigs {
		if config.DeletedAt != nil || !config.Enabled {
			continue
		}
		rank, ok := parserContentTypeMatchRank(config, contentType)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{config: config, rank: rank})
	}
	if len(candidates) == 0 {
		return service.ParserConfig{}, service.ErrNotFound
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.rank != right.rank {
			return left.rank < right.rank
		}
		if left.config.IsDefault != right.config.IsDefault {
			return left.config.IsDefault
		}
		if !left.config.CreatedAt.Equal(right.config.CreatedAt) {
			return left.config.CreatedAt.Before(right.config.CreatedAt)
		}
		return left.config.ID < right.config.ID
	})
	return cloneParserConfig(candidates[0].config), nil
}

func (r *MemoryRepository) SeedParserConfig(config service.ParserConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parserConfigs[config.ID] = cloneParserConfig(config)
}

func (r *MemoryRepository) ParserAudits() []service.ParserConfigAudit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]service.ParserConfigAudit, len(r.parserAudits))
	for i, v := range r.parserAudits {
		out[i] = cloneParserAudit(v)
	}
	return out
}

func (r *MemoryRepository) clearDefaultLocked(except string, updatedAt time.Time) {
	for id, config := range r.parserConfigs {
		if id != except && config.DeletedAt == nil && config.IsDefault {
			config.IsDefault = false
			config.UpdatedAt = updatedAt
			r.parserConfigs[id] = config
		}
	}
}

func parserContentTypeMatchRank(config service.ParserConfig, contentType string) (int, bool) {
	if contentType == "" {
		return 0, true
	}
	if len(config.SupportedContentTypes) == 0 {
		return 2, true
	}
	wildcardMatch := false
	for _, candidate := range config.SupportedContentTypes {
		if candidate == contentType {
			return 0, true
		}
		if strings.HasSuffix(candidate, "/*") && strings.HasPrefix(contentType, strings.TrimSuffix(candidate, "*")) {
			wildcardMatch = true
		}
	}
	if wildcardMatch {
		return 1, true
	}
	return 0, false
}

func cloneParserConfig(config service.ParserConfig) service.ParserConfig {
	config.SupportedContentTypes = append([]string(nil), config.SupportedContentTypes...)
	config.DefaultParameters = cloneRawJSON(config.DefaultParameters)
	config.EndpointURL = cloneStringPtr(config.EndpointURL)
	if config.DeletedAt != nil {
		v := *config.DeletedAt
		config.DeletedAt = &v
	}
	return config
}

func cloneParserAudit(audit service.ParserConfigAudit) service.ParserConfigAudit {
	audit.Summary = cloneRawJSON(audit.Summary)
	return audit
}

func cloneRawJSON(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
