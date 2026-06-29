package middleware

import (
	"context"
	"net/http"
	"time"
)

type TimeoutSkipFunc func(*http.Request) bool

func Timeout(timeout time.Duration) Middleware {
	return TimeoutWithSkip(timeout, nil)
}

func TimeoutWithSkip(timeout time.Duration, skip TimeoutSkipFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if timeout <= 0 || (skip != nil && skip(r)) {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
