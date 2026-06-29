package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

type LocalHasher struct {
	provider  string
	model     string
	dimension int
}

func NewLocalHasher(provider string, model string, dimension int) *LocalHasher {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "local_hashing"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "local_hashing"
	}
	if dimension <= 0 {
		dimension = 384
	}
	return &LocalHasher{provider: provider, model: model, dimension: dimension}
}

func (h *LocalHasher) Embed(ctx context.Context, request service.EmbeddingRequest) (service.EmbeddingResult, error) {
	if err := ctx.Err(); err != nil {
		return service.EmbeddingResult{}, err
	}
	if len(request.Texts) == 0 {
		return service.EmbeddingResult{}, fmt.Errorf("embedding input must not be empty")
	}
	vectors := make([][]float32, 0, len(request.Texts))
	for _, text := range request.Texts {
		vectors = append(vectors, hashTextVector(text, h.dimension))
	}
	return service.EmbeddingResult{
		Vectors:   vectors,
		Provider:  h.provider,
		Model:     h.model,
		Dimension: h.dimension,
	}, nil
}

func hashTextVector(text string, dimension int) []float32 {
	vector := make([]float32, dimension)
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		words = []string{text}
	}
	for _, word := range words {
		sum := sha256.Sum256([]byte(word))
		bucket := int(binary.BigEndian.Uint32(sum[:4]) % uint32(dimension))
		sign := float32(1)
		if sum[4]%2 == 1 {
			sign = -1
		}
		vector[bucket] += sign
	}
	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm == 0 {
		return vector
	}
	scale := float32(1 / math.Sqrt(norm))
	for i := range vector {
		vector[i] *= scale
	}
	return vector
}
