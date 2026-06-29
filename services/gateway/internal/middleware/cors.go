package middleware

import (
	"net/http"
	"slices"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

func CORS(cfg CORSConfig) Middleware {
	allowedOrigins := normalizeList(cfg.AllowedOrigins)
	allowedMethods := normalizeMethods(cfg.AllowedMethods)
	allowedHeaders := normalizeList(cfg.AllowedHeaders)
	if len(allowedMethods) == 0 {
		allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions}
	}
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Authorization", "Content-Type", "X-Request-Id"}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				applyCORSHeaders(w, origin, allowedOrigins, allowedMethods, allowedHeaders, cfg.AllowCredentials)
			}
			if r.Method == http.MethodOptions && origin != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func applyCORSHeaders(w http.ResponseWriter, origin string, allowedOrigins []string, allowedMethods []string, allowedHeaders []string, allowCredentials bool) {
	allowOrigin := ""
	switch {
	case slices.Contains(allowedOrigins, "*") && allowCredentials:
		allowOrigin = origin
	case slices.Contains(allowedOrigins, "*"):
		allowOrigin = "*"
	case slices.Contains(allowedOrigins, origin):
		allowOrigin = origin
	}
	if allowOrigin == "" {
		return
	}

	header := w.Header()
	header.Set("Access-Control-Allow-Origin", allowOrigin)
	header.Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
	header.Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
	if allowCredentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}
	header.Add("Vary", "Origin")
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func normalizeMethods(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
