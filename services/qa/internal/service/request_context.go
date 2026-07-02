package service

import (
	"context"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return contextutil.WithRequestID(ctx, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	return contextutil.RequestIDFromContext(ctx)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return contextutil.WithUserID(ctx, userID)
}

func UserIDFromContext(ctx context.Context) string {
	return contextutil.UserIDFromContext(ctx)
}

func WithUserRoles(ctx context.Context, roles string) context.Context {
	return contextutil.WithUserRoles(ctx, roles)
}

func UserRolesFromContext(ctx context.Context) string {
	return contextutil.UserRolesFromContext(ctx)
}

func WithUserPermissions(ctx context.Context, permissions string) context.Context {
	return contextutil.WithUserPermissions(ctx, permissions)
}

func UserPermissionsFromContext(ctx context.Context) string {
	return contextutil.UserPermissionsFromContext(ctx)
}
