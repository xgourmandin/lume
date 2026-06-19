package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/mrshabel/mach"
)

// statusRecorder captures the response status code and size so the request
// logger can report them after the handler chain has run.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// RequestLogger emits one structured log entry per HTTP request (method, path,
// status, duration). It is the single source of route/status context, so
// individual handlers log only business context and the error itself.
func RequestLogger(logger *slog.Logger) mach.MiddlewareFunc {
	logger = logger.With("component", "http")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"bytes", rec.size,
			)
		})
	}
}
