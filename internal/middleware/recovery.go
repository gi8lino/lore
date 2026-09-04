package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/httpresponse"
)

// RecoverPanics converts handler panics into logged internal server errors.
func RecoverPanics(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error(
						"request panic",
						"event",
						"request_panic",
						"method",
						r.Method,
						"path",
						r.URL.Path,
						"error",
						recovered,
					)
					httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
