package middleware

import (
	"log/slog"
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

func Recover(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w}
			defer func() {
				if recovered := recover(); recovered != nil {
					requestID := RequestIDFromContext(r.Context())
					logger.ErrorContext(r.Context(), "http panic recovered",
						"service", "gateway",
						"request_id", requestID,
						"operation", "http_request",
					)
					if recorder.status == 0 {
						response.WriteError(recorder, http.StatusInternalServerError, response.ErrorDetail{
							Code:      response.CodeInternal,
							Message:   "internal server error",
							RequestID: requestID,
						})
					}
				}
			}()
			next.ServeHTTP(recorder, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(body)
}
