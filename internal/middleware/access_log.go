package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// AccessLog records request method, path, and elapsed time after handling.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()

			next.ServeHTTP(w, r)
			logger.Info(
				"request",
				"event",
				"request_complete",
				"method",
				r.Method,
				"path",
				r.URL.Path,
				"duration",
				time.Since(started),
			)
		})
	}
}
