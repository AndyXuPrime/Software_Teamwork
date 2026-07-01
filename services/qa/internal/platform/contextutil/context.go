package contextutil

import (
	"context"
	"strings"
	"sync"
)

type userIDContextKey struct{}

type requestIDContextKey struct{}

type userRolesContextKey struct{}

type userPermissionsContextKey struct{}

type citationNoContextKey struct{}

type knowledgeBaseIDsContextKey struct{}

type defaultKnowledgeBaseIDsContextKey struct{}

type retrievalSettingsContextKey struct{}

type atomicCitationNo struct {
	mu      sync.Mutex
	counter int
}

type RetrievalSettings struct {
	TopK                     int     `json:"topK"`
	ScoreThreshold           float64 `json:"scoreThreshold"`
	ScoreThresholdConfigured bool    `json:"-"`
	EnableRerank             bool    `json:"enableRerank"`
	RerankThreshold          float64 `json:"rerankThreshold"`
	RerankTopN               int     `json:"rerankTopN"`
}

func newAtomicCitationNo(initial int) *atomicCitationNo {
	return &atomicCitationNo{counter: initial}
}

func (a *atomicCitationNo) get() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.counter
}

func (a *atomicCitationNo) add(delta int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counter += delta
}

func (a *atomicCitationNo) set(value int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counter = value
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, strings.TrimSpace(userID))
}

func UserIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userIDContextKey{}).(string)
	return value
}

func WithUserRoles(ctx context.Context, roles string) context.Context {
	return context.WithValue(ctx, userRolesContextKey{}, strings.TrimSpace(roles))
}

func UserRolesFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userRolesContextKey{}).(string)
	return value
}

func WithUserPermissions(ctx context.Context, permissions string) context.Context {
	return context.WithValue(ctx, userPermissionsContextKey{}, strings.TrimSpace(permissions))
}

func UserPermissionsFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userPermissionsContextKey{}).(string)
	return value
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}

func WithCitationNo(ctx context.Context, citationNo int) context.Context {
	return context.WithValue(ctx, citationNoContextKey{}, newAtomicCitationNo(citationNo))
}

func CitationNoFromContext(ctx context.Context) int {
	value, _ := ctx.Value(citationNoContextKey{}).(*atomicCitationNo)
	if value == nil {
		return 0
	}
	return value.get()
}

func AddCitationNo(ctx context.Context, delta int) {
	value, _ := ctx.Value(citationNoContextKey{}).(*atomicCitationNo)
	if value != nil {
		value.add(delta)
	}
}

func WithKnowledgeBaseIDs(ctx context.Context, ids []string) context.Context {
	return context.WithValue(ctx, knowledgeBaseIDsContextKey{}, ids)
}

func KnowledgeBaseIDsFromContext(ctx context.Context) []string {
	value, _ := ctx.Value(knowledgeBaseIDsContextKey{}).([]string)
	return value
}

func WithDefaultKnowledgeBaseIDs(ctx context.Context, ids []string) context.Context {
	return context.WithValue(ctx, defaultKnowledgeBaseIDsContextKey{}, ids)
}

func DefaultKnowledgeBaseIDsFromContext(ctx context.Context) []string {
	value, _ := ctx.Value(defaultKnowledgeBaseIDsContextKey{}).([]string)
	return value
}

func WithRetrievalSettings(ctx context.Context, settings RetrievalSettings) context.Context {
	return context.WithValue(ctx, retrievalSettingsContextKey{}, settings)
}

func RetrievalSettingsFromContext(ctx context.Context) RetrievalSettings {
	value, _ := ctx.Value(retrievalSettingsContextKey{}).(RetrievalSettings)
	return value
}
