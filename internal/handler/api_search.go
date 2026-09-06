package handler

import (
	"log/slog"
	"net/http"

	"github.com/gi8lino/lore/internal/httpresponse"

	"github.com/gi8lino/lore/internal/auth"
)

// SearchAPI executes a wiki search and returns page summaries as JSON.
func SearchAPI(catalogUseCases pageSearchService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := catalogUseCases.Search(r.Context(), r.URL.Query().Get("q"), 50)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		stripMarkdown(pages)
		httpresponse.Respond(w, http.StatusOK, jsonSlice(pages))
	}
}

// Tags returns all known tags as JSON.
func Tags(catalogUseCases pageTagService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tags, err := catalogUseCases.Tags(r.Context())
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, jsonSlice(tags))
	}
}

// Recent returns recently updated page summaries as JSON.
func Recent(catalogUseCases pageListService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := catalogUseCases.ListPages(r.Context(), 50)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		stripMarkdown(pages)
		httpresponse.Respond(w, http.StatusOK, jsonSlice(pages))
	}
}

// GroupsAPI returns collaboration groups the current user may assign to pages.
func GroupsAPI(groupUseCases groupReader, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w,
				http.StatusUnauthorized,
				"Unauthorized.",
				httpresponse.NewFieldProblem("authorization", "An authenticated user is required."),
			)
			return
		}

		groups, err := groupUseCases.AssignableGroups(r.Context(), user)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, jsonSlice(groups))
	}
}
