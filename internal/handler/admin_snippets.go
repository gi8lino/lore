package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/lore/internal/httpresponse"
)

// AdminSnippets renders reusable variable and Markdown-snippet management.
func AdminSnippets(
	viewDataUseCases viewDataService,
	knowledgeUseCases knowledgeSnippetReader,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Snippets & variables", "snippets")
		if err != nil {
			writeInternalServerError(views.logger, w, err)
			return
		}

		items, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			writeInternalServerError(views.logger, w, err)
			return
		}

		data.KnowledgeSnippets = items
		render(views, w, "admin_snippets", data)
	}
}

// SaveAdminSnippet creates or updates one reusable variable or Markdown snippet.
func SaveAdminSnippet(knowledgeUseCases knowledgeSnippetService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid snippet form.")
			return
		}

		var id int64
		if value := r.PathValue("id"); value != "" {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed <= 0 {
				httpresponse.Problem(w, http.StatusBadRequest, "Invalid snippet.")
				return
			}
			id = parsed
		}

		_, err := knowledgeUseCases.SaveKnowledgeSnippet(
			r.Context(),
			id,
			user.ID,
			r.FormValue("kind"),
			r.FormValue("name"),
			r.FormValue("description"),
			r.FormValue("content"),
		)
		if err != nil {
			writeAdminProblem(logger, w, err, "Snippet")
			return
		}

		http.Redirect(w, r, "/admin/snippets", http.StatusSeeOther)
	}
}

// DeleteAdminSnippet deletes one reusable variable or Markdown snippet.
func DeleteAdminSnippet(knowledgeUseCases knowledgeSnippetService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid snippet.")
			return
		}
		if err := knowledgeUseCases.DeleteKnowledgeSnippet(r.Context(), id, user.ID); err != nil {
			writeAdminProblem(logger, w, err, "Snippet")
			return
		}

		http.Redirect(w, r, "/admin/snippets", http.StatusSeeOther)
	}
}
