package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
)

// pageDraftRequest is the private editor state accepted by the draft API.
type pageDraftRequest struct {
	PageID int64               `json:"page_id"`
	Title  string              `json:"title"`
	Slug   string              `json:"slug"`
	Values map[string][]string `json:"values"`
}

// GetPageDraft returns one private server-side editor draft owned by the current user.
func GetPageDraft(draftUseCases editorDraftService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		draft, err := draftUseCases.Draft(r.Context(), user.ID, strings.TrimSpace(r.PathValue("key")))
		if err != nil {
			writeDraftProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, draft)
	}
}

// SavePageDraft creates or updates one private server-side editor draft without creating a page revision.
func SavePageDraft(draftUseCases editorDraftService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := decode[pageDraftRequest](w, r)
		if err != nil {
			httpresponse.Problem(
				w,
				http.StatusBadRequest,
				"Invalid draft request.",
				httpresponse.NewFieldProblem("request", err.Error()),
			)
			return
		}

		user := currentUser(r)
		draft, err := draftUseCases.Save(r.Context(), service.PageDraftSaveInput{
			Key:    strings.TrimSpace(r.PathValue("key")),
			PageID: request.PageID,
			Title:  request.Title,
			Slug:   request.Slug,
			Values: request.Values,
			Actor:  user,
		})
		if err != nil {
			writeDraftProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, draft)
	}
}

// DeletePageDraft discards one private editor draft owned by the current user.
func DeletePageDraft(draftUseCases editorDraftService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := draftUseCases.Delete(r.Context(), user.ID, strings.TrimSpace(r.PathValue("key"))); err != nil {
			writeDraftProblem(logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// writeDraftProblem translates private-draft failures into structured HTTP problems.
func writeDraftProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	if writeValidationProblem(w, err, "Draft validation failed.") {
		return
	}
	switch {
	case errors.Is(err, service.ErrNotFound):
		httpresponse.Problem(w, http.StatusNotFound, "Draft not found.")
	case errors.Is(err, service.ErrForbidden):
		httpresponse.Problem(w, http.StatusForbidden, "The draft operation is not permitted.")
	default:
		writeUnexpectedProblem(logger, w, err)
	}
}
