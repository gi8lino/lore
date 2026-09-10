package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
)

// MovePageForm safely moves one page or subtree and optionally refactors direct wiki links.
func MovePageForm(pageUseCases pageMoveService, accessUseCases pageAccessReader, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid move form.")
			return
		}

		newSlug := md.Slug(r.FormValue("slug"))
		allowed, err := accessUseCases.CanEdit(r.Context(), user, newSlug)
		if err != nil {
			httpresponse.InternalServerError(logger, w, err)
			return
		}
		if !allowed {
			httpresponse.Problem(w, http.StatusForbidden, "You do not have permission to move a page to that path.")
			return
		}
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

// RequestPageReview opens a lightweight approval request for the current revision.
func RequestPageReview(pageUseCases pageApprovalService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid review request.")
			return
		}
		slug := strings.TrimSpace(r.PathValue("slug"))
		if _, err := pageUseCases.RequestReview(r.Context(), slug, r.FormValue("note"), user); err != nil {
			writePageProblem(logger, w, err)
			return
		}
		http.Redirect(w, r, "/pages/"+slug, http.StatusSeeOther)
	}
}

// DecidePageReview approves the requested revision or asks for changes.
func DecidePageReview(pageUseCases pageApprovalService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid review decision.")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid review request.")
			return
		}
		slug := strings.TrimSpace(r.FormValue("slug"))
		if err := pageUseCases.DecideReview(r.Context(), id, slug, r.FormValue("decision"), r.FormValue("note"), user); err != nil {
			if errors.Is(err, domain.ErrStaleReview) {
				httpresponse.Problem(w, http.StatusConflict, "The page changed after review was requested. Request a new review for the latest revision.")
				return
			}
			writePageProblem(logger, w, err)
			return
		}
		http.Redirect(w, r, "/pages/"+slug, http.StatusSeeOther)
	}
}
