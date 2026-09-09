package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
)

// MovePageForm safely moves one page or subtree and optionally refactors direct wiki links.
func MovePageForm(pageUseCases pageMoveService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid move form.")
			return
		}

		newSlug := md.Slug(r.FormValue("slug"))
		options := domain.MovePageOptions{
			MoveChildren:        r.FormValue("move_children") == "on",
			UpdateIncomingLinks: r.FormValue("update_links") == "on",
			KeepAliases:         r.FormValue("keep_aliases") == "on",
		}
		if err := pageUseCases.Move(r.Context(), r.PathValue("slug"), newSlug, options, user); err != nil {
			writePageProblem(logger, w, err)
			return
		}

		http.Redirect(w, r, "/pages/"+newSlug, http.StatusSeeOther)
	}
}

// ReviewPageForm records an explicit documentation review.
func ReviewPageForm(pageUseCases pageReviewService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		slug := strings.TrimSpace(r.PathValue("slug"))
		if err := pageUseCases.Review(r.Context(), slug, user); err != nil {
			writePageProblem(logger, w, err)
			return
		}

		http.Redirect(w, r, "/pages/"+slug, http.StatusSeeOther)
	}
}
