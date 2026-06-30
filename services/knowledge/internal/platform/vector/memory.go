package vector

import (
	"context"
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

func (i *MemoryIndex) Points() []service.VectorPoint {
	i.mu.RLock()
	defer i.mu.RUnlock()
	points := make([]service.VectorPoint, 0, len(i.points))
	for _, point := range i.points {
		points = append(points, clonePoint(point))
	}
	return points
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
