package mcp

import (
	"net/http"
	"strings"
)

// CallerContext carries auth and tracing headers forwarded to the adapter layer.
type CallerContext struct {
	UserID      string
	RequestID   string
	Roles       string
	Permissions string
}

func callerFromHTTP(r *http.Request) CallerContext {
	if r == nil {
		return CallerContext{}
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	}
	return CallerContext{
		UserID:      strings.TrimSpace(r.Header.Get("X-User-Id")),
		RequestID:   requestID,
		Roles:       strings.TrimSpace(r.Header.Get("X-User-Roles")),
		Permissions: strings.TrimSpace(r.Header.Get("X-User-Permissions")),
	}
}

func (c CallerContext) applyHeaders(r *http.Request) {
	if c.RequestID != "" {
		r.Header.Set("X-Request-Id", c.RequestID)
	}
	if c.UserID != "" {
		r.Header.Set("X-User-Id", c.UserID)
	}
	if c.Roles != "" {
		r.Header.Set("X-User-Roles", c.Roles)
	}
	if c.Permissions != "" {
		r.Header.Set("X-User-Permissions", c.Permissions)
	}
}
