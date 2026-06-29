package middleware

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

func BodyLimit(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes <= 0 || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > maxBytes {
				response.WriteError(w, http.StatusRequestEntityTooLarge, response.ErrorDetail{
					Code:      response.CodeValidation,
					Message:   "request body is too large",
					RequestID: RequestIDFromContext(r.Context()),
				})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
