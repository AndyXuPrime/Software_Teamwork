package vector

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type MemoryIndex struct {
	mu     sync.RWMutex
	points map[string]service.VectorPoint
}

func NewMemoryIndex() *MemoryIndex {
	return &MemoryIndex{points: map[string]service.VectorPoint{}}
}

func (i *MemoryIndex) Upsert(ctx context.Context, points []service.VectorPoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, point := range points {
		i.points[point.ID] = clonePoint(point)
	}
	return nil
}

func (i *MemoryIndex) DeleteByDocumentIngestionAttempt(ctx context.Context, documentID string, ingestionAttempt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for id, point := range i.points {
		if point.Payload[service.VectorPayloadDocumentID] == documentID &&
			point.Payload[service.VectorPayloadIngestionAttempt] == ingestionAttempt {
			delete(i.points, id)
		}
	}
	return nil
}

func (i *MemoryIndex) DeleteStaleDocumentPoints(ctx context.Context, documentID string, activeIngestionAttempt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for id, point := range i.points {
		if point.Payload[service.VectorPayloadDocumentID] == documentID &&
			point.Payload[service.VectorPayloadIngestionAttempt] != activeIngestionAttempt {
			delete(i.points, id)
		}
	}
	return nil
}

func (i *MemoryIndex) Search(ctx context.Context, request service.VectorSearchRequest) ([]service.VectorSearchHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	hits := make([]service.VectorSearchHit, 0, len(i.points))
	for _, point := range i.points {
		if !matchesSearchRequest(point.Payload, request) {
			continue
		}
		score := dotProduct(request.Vector, point.Vector)
		if score < request.ScoreThreshold {
			continue
		}
		hits = append(hits, service.VectorSearchHit{
			ID:      point.ID,
			Score:   score,
			Payload: clonePayload(point.Payload),
		})
	}
	sort.SliceStable(hits, func(a, b int) bool {
		return hits[a].Score > hits[b].Score
	})
	if request.Limit > 0 && len(hits) > request.Limit {
		hits = hits[:request.Limit]
	}
	return hits, nil
}

func (i *MemoryIndex) Points() []service.VectorPoint {
	i.mu.RLock()
	defer i.mu.RUnlock()
	points := make([]service.VectorPoint, 0, len(i.points))
	for _, point := range i.points {
		points = append(points, clonePoint(point))
	}
	return points
}

func matchesSearchRequest(payload map[string]any, request service.VectorSearchRequest) bool {
	if len(request.KnowledgeBaseIDs) > 0 {
		kbID := strings.TrimSpace(fmt.Sprint(payload["knowledge_base_id"]))
		matched := false
		for _, allowed := range request.KnowledgeBaseIDs {
			if kbID == strings.TrimSpace(allowed) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, tag := range request.Tags {
		if !payloadContainsString(payload["tags"], strings.TrimSpace(tag)) {
			return false
		}
	}
	metadata, _ := payload["metadata"].(map[string]any)
	for key, expected := range request.MetadataFilter {
		if strings.TrimSpace(fmt.Sprint(metadata[strings.TrimSpace(key)])) != strings.TrimSpace(expected) {
			return false
		}
	}
	return true
}

func payloadContainsString(value any, target string) bool {
	switch tags := value.(type) {
	case []string:
		for _, tag := range tags {
			if strings.TrimSpace(tag) == target {
				return true
			}
		}
	case []any:
		for _, tag := range tags {
			if strings.TrimSpace(fmt.Sprint(tag)) == target {
				return true
			}
		}
	}
	return false
}

func dotProduct(left []float32, right []float32) float64 {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	var score float64
	for i := 0; i < limit; i++ {
		score += float64(left[i] * right[i])
	}
	return score
}

func clonePayload(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func clonePoint(point service.VectorPoint) service.VectorPoint {
	payload := make(map[string]any, len(point.Payload))
	for key, value := range point.Payload {
		payload[key] = value
	}
	return service.VectorPoint{
		ID:      point.ID,
		Vector:  append([]float32(nil), point.Vector...),
		Payload: payload,
	}
}
