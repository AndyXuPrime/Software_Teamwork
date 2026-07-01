package middleware

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

func BodyLimit(maxBytes int64) Middleware {
	return BodyLimitForRequest(maxBytes, nil)
}

func BodyLimitForRequest(maxBytes int64, limitForRequest func(*http.Request) int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := maxBytes
			if limitForRequest != nil {
				if override := limitForRequest(r); override > 0 {
					limit = override
				}
			}
			if limit <= 0 || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > limit {
				response.WriteError(w, http.StatusRequestEntityTooLarge, response.ErrorDetail{
					Code:      response.CodeValidation,
					Message:   "request body is too large",
					RequestID: RequestIDFromContext(r.Context()),
				})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
