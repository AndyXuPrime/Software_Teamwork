package adapter

import (
	"net/http"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func (s *Server) gatewayContext(w http.ResponseWriter, r *http.Request) (service.RequestContext, bool) {
	reqCtx := service.RequestContext{
		RequestID:      requestIDFromContext(r.Context()),
		UserID:         strings.TrimSpace(r.Header.Get("X-User-Id")),
		CallerService:  strings.TrimSpace(r.Header.Get("X-Caller-Service")),
		ServiceToken:   strings.TrimSpace(r.Header.Get("X-Service-Token")),
		Roles:          splitCSV(r.Header.Get("X-User-Roles")),
		Permissions:    splitCSV(r.Header.Get("X-User-Permissions")),
		ForwardedFor:   strings.TrimSpace(r.Header.Get("X-Forwarded-For")),
		ForwardedProto: strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")),
	}
	if reqCtx.UserID == "" {
		writeAppError(w, r, service.UnauthorizedError())
		return service.RequestContext{}, false
	}
	return reqCtx, true
}

func readScope(reqCtx service.RequestContext) (service.AccessScope, error) {
	if strings.TrimSpace(reqCtx.UserID) == "" {
		return service.AccessScope{}, service.UnauthorizedError()
	}
	return service.AccessScope{
		UserID:     strings.TrimSpace(reqCtx.UserID),
		CanReadAll: hasAdminRole(reqCtx.Roles) || hasPermission(reqCtx.Permissions, service.PermissionKnowledgeRead) || hasPermission(reqCtx.Permissions, service.PermissionKnowledgeWrite),
		CanWrite:   hasAdminRole(reqCtx.Roles) || hasPermission(reqCtx.Permissions, service.PermissionKnowledgeWrite),
	}, nil
}

func mutationScope(reqCtx service.RequestContext) (service.AccessScope, error) {
	scope, err := readScope(reqCtx)
	if err != nil {
		return service.AccessScope{}, err
	}
	if !scope.CanWrite {
		return service.AccessScope{}, service.ForbiddenError("knowledge write permission is required")
	}
	return scope, nil
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "admin", "super_admin", "superadmin":
			return true
		}
	}
	return false
}

func hasPermission(permissions []string, target string) bool {
	for _, permission := range permissions {
		if strings.TrimSpace(permission) == target {
			return true
		}
	}
	return false
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
