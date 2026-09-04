package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
)

// PagePermalink redirects an authenticated stable page identifier to its current path.
func PagePermalink(catalogUseCases pagePermalinkService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := permalinkPageID(r.PathValue("id"))
		if err != nil {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}

		slug, err := catalogUseCases.PageSlugByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				httpresponse.Problem(w, http.StatusNotFound, "Not found.")
				return
			}
			writeUnexpectedProblem(logger, w, err)
			return
		}

		// The target may change when a page is moved, so this redirect must not be cached permanently.
		http.Redirect(w, r, "/pages/"+slug, http.StatusFound)
	}
}

// permalinkPageID validates the stable numeric page identifier used in permalink routes.
func permalinkPageID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}
