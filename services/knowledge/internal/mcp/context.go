package mcp

import (
	"net/http"
	"strings"
)

// CallerContext carries trusted auth and tracing headers forwarded to the adapter layer.
type CallerContext struct {
	UserID       string
	RequestID    string
	ServiceToken string
	Roles        string
	Permissions  string
}

func callerFromHTTP(r *http.Request, trusted CallerContext) CallerContext {
	caller := trusted
	if r == nil {
		return caller
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	}
	if requestID != "" {
		caller.RequestID = requestID
	}
	return caller
}

func (c CallerContext) applyHeaders(r *http.Request) {
	if c.RequestID != "" {
		r.Header.Set("X-Request-Id", c.RequestID)
	}
	if c.UserID != "" {
		r.Header.Set("X-User-Id", c.UserID)
	}
	if c.ServiceToken != "" {
		r.Header.Set("X-Service-Token", c.ServiceToken)
	}
	if c.Roles != "" {
		r.Header.Set("X-User-Roles", c.Roles)
	}
	if c.Permissions != "" {
		r.Header.Set("X-User-Permissions", c.Permissions)
	}
}
