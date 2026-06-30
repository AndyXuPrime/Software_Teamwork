package service

import (
	"context"
	"strings"
)

type requestIDContextKey struct{}
type userIDContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey{}).(string)
	return value
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, strings.TrimSpace(userID))
}

func UserIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userIDContextKey{}).(string)
	return value
}
