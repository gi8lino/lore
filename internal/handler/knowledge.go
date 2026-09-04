package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/service"
)

// KnowledgeGraphPage renders the interactive page relationship explorer.
func KnowledgeGraphPage(viewDataUseCases viewDataService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := viewData(r, viewDataUseCases, views, "Knowledge graph")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		data.Query = strings.TrimSpace(r.URL.Query().Get("slug"))
		render(views, w, "graph", data)
	}
}

// KnowledgeGraphAPI returns pages and current wiki-link relationships.
func KnowledgeGraphAPI(knowledgeUseCases knowledgeGraphService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		graph, err := knowledgeUseCases.KnowledgeGraph(r.Context(), 300)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusOK, graph)
	}
}

// MovePageForm safely moves one page or subtree and optionally refactors direct wiki links.
func MovePageForm(pageUseCases pageMoveService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid move form.")
			return
		}
		newSlug := md.Slug(r.FormValue("slug"))
		options := service.MovePageOptions{
			MoveChildren:        r.FormValue("move_children") == "on",
			UpdateIncomingLinks: r.FormValue("update_links") == "on",
			KeepAliases:         r.FormValue("keep_aliases") == "on",
		}
		if err := pageUseCases.Move(r.Context(), r.PathValue("slug"), newSlug, options, user); err != nil {
			if validation, ok := errors.AsType[*service.ValidationError](err); ok {
				httpresponse.Problem(w, http.StatusBadRequest, validation.Error())
				return
			}
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

// CreateSavedSearch creates a personal smart collection.
func CreateSavedSearch(knowledgeUseCases savedSearchService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid saved search form.")
			return
		}
		if err := knowledgeUseCases.SaveSavedSearch(
			r.Context(),
			user.ID,
			0,
			r.FormValue("name"),
			r.FormValue("query"),
			r.FormValue("pinned") == "on",
		); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		next := strings.TrimSpace(r.FormValue("next"))
		if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
			next = "/settings#saved-searches"
		}
		http.Redirect(w, r, next, http.StatusSeeOther)
	}
}

// DeleteSavedSearch deletes one personal smart collection.
func DeleteSavedSearch(knowledgeUseCases savedSearchService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid saved search.")
			return
		}
		if err := knowledgeUseCases.DeleteSavedSearch(r.Context(), user.ID, id); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		http.Redirect(w, r, "/settings#saved-searches", http.StatusSeeOther)
	}
}

// NotificationsAPI returns the current user's notification inbox.
func NotificationsAPI(knowledgeUseCases notificationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		items, unread, err := knowledgeUseCases.Notifications(r.Context(), user.ID, 30)
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
	}
}

// MarkNotificationRead marks one notification or the complete inbox as read.
func MarkNotificationRead(knowledgeUseCases notificationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		var id int64
		if value := r.PathValue("id"); value != "" && value != "all" {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil || parsed <= 0 {
				httpresponse.Problem(w, http.StatusBadRequest, "Invalid notification.")
				return
			}
			id = parsed
		}
		if err := knowledgeUseCases.MarkNotificationRead(r.Context(), user.ID, id); err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// AddPageComment adds an anchored discussion item to one page.
func AddPageComment(pageUseCases pageDiscussionWriter, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.User(r)
		if !ok {
			httpresponse.Problem(w, http.StatusUnauthorized, "Unauthorized.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid comment form.")
			return
		}
		slug := strings.TrimSpace(r.PathValue("slug"))
		if err := pageUseCases.AddComment(r.Context(), slug, r.FormValue("anchor"), r.FormValue("body"), user); err != nil {
			writePageProblem(views.logger, w, err)
			return
		}
		http.Redirect(w, r, "/pages/"+slug+"#comments", http.StatusSeeOther)
	}
}

// ResolvePageComment resolves or reopens one discussion item.
func ResolvePageComment(pageUseCases pageDiscussionWriter, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid comment.")
			return
		}
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid comment form.")
			return
		}
		if err := pageUseCases.ResolveComment(r.Context(), id, r.FormValue("resolved") != "false"); err != nil {
			writePageProblem(views.logger, w, err)
			return
		}
		next := strings.TrimSpace(r.FormValue("next"))
		if next == "" || !strings.HasPrefix(next, "/pages/") {
			next = "/"
		}
		http.Redirect(w, r, next+"#comments", http.StatusSeeOther)
	}
}

// EditorCatalog returns page and reusable-snippet metadata used by editor intelligence.
func EditorCatalog(
	navigationUseCases navigationService,
	knowledgeUseCases knowledgeSnippetReader,
	catalogUseCases pageAliasService,
	logger *slog.Logger,
) http.HandlerFunc {
	type pageItem struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		items := make([]pageItem, 0, len(pages))
		for _, page := range pages {
			items = append(items, pageItem{Slug: page.Slug, Title: page.Title})
		}
		snippets, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		aliases, err := catalogUseCases.PageAliases(r.Context())
		if err != nil {
			writeUnexpectedProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusOK, map[string]any{"pages": items, "snippets": snippets, "aliases": aliases})
	}
}

// AdminSnippets renders reusable variable and Markdown-snippet management.
func AdminSnippets(
	viewDataUseCases viewDataService,
	knowledgeUseCases knowledgeSnippetReader,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Snippets & variables", "snippets")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		items, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
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

// AdminPages renders bulk page management.
func AdminPages(
	viewDataUseCases viewDataService,
	catalogUseCases pageInventoryService,
	groupUseCases groupReader,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Pages", "pages")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		pages, err := catalogUseCases.PageInventory(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		groups, err := groupUseCases.Groups(r.Context())
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}
		data.AdminPages = pages
		data.Groups = groups
		render(views, w, "admin_pages", data)
	}
}

// BulkAdminPages applies one action to selected pages.
func BulkAdminPages(
	pageUseCases pageBulkService,
	catalogUseCases pageContentService,
	mediaUseCases imageContentService,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid bulk page form.")
			return
		}
		slugs := uniqueNonEmpty(r.Form["slug"])
		if len(slugs) == 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Select at least one page.")
			return
		}
		action := r.FormValue("action")
		if action == "export" {
			file, modTime, cleanup, exportErr := createExportArchive(
				r.Context(),
				catalogUseCases,
				mediaUseCases,
				slugs,
			)
			if exportErr != nil {
				writePageProblem(logger, w, exportErr)
				return
			}
			defer cleanup()
			filename := "lore-pages-" + time.Now().UTC().Format("20060102-150405") + ".zip"
			serveExportArchive(w, r, filename, file, modTime)
			return
		}

		groupID, _ := strconv.ParseInt(r.FormValue("group_id"), 10, 64)
		err := pageUseCases.Bulk(r.Context(), service.BulkPageInput{
			Action:  action,
			Slugs:   slugs,
			Status:  r.FormValue("status"),
			Tag:     r.FormValue("tag"),
			GroupID: groupID,
			Target:  r.FormValue("target"),
			Actor:   user,
		})
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
	}
}

// uniqueNonEmpty trims, deduplicates, and removes empty strings while preserving order.
func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
