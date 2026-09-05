package handler

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/internal/auth"
	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/revision"
	"github.com/gi8lino/lore/internal/service"
)

// Home renders the wiki dashboard for the current user.
func Home(
	viewDataUseCases viewDataService,
	catalogUseCases homeCatalogService,
	draftUseCases draftListService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.User(r)
		favorites, err := catalogUseCases.Favorites(r.Context(), user.ID)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		recent, err := catalogUseCases.ListPages(r.Context(), 8)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		viewed, err := catalogUseCases.RecentViewed(r.Context(), user.ID, 8)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		popular, err := catalogUseCases.Popular(r.Context(), 8)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		recentEdits, err := catalogUseCases.RecentEdited(r.Context(), user.ID, 6)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		var drafts []service.PageDraft

		if user.Role == "admin" || user.Role == "editor" {
			drafts, err = draftUseCases.List(r.Context(), user.ID, 6)
			if err != nil {
				writePageProblem(views.logger, w, err)
				return
			}
		}

		data, err := viewData(r, viewDataUseCases, views, "Home")
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		data.Favorites, data.Recent, data.Pages, data.Popular = favorites, recent, viewed, popular
		data.RecentEdits = recentEdits
		data.Drafts = drafts

		render(views, w, "home", data)
	}
}

// ViewPage renders a wiki page with relations and revision history.
func ViewPage(
	viewDataUseCases viewDataService,
	catalogUseCases pageViewCatalogService,
	settingsUseCases settingsService,
	knowledgeUseCases knowledgeContentService,
	renderer *md.Renderer,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		page, err := catalogUseCases.GetPage(r.Context(), slug)

		if errors.Is(err, service.ErrNotFound) {
			if target, aliasErr := catalogUseCases.ResolvePageAlias(r.Context(), slug); aliasErr == nil {
				http.Redirect(w, r, "/pages/"+target, http.StatusPermanentRedirect)
				return
			}
		}

		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		user, _ := auth.User(r)
		_ = catalogUseCases.RecordView(r.Context(), slug, user.ID)
		pageFavorite, err := catalogUseCases.IsFavorite(r.Context(), slug, user.ID)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		options, _, err := renderingOptions(r.Context(), settingsUseCases)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		backlinks, err := catalogUseCases.Backlinks(r.Context(), slug)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		outgoingLinks, err := catalogUseCases.PageLinks(r.Context(), slug)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		latestRevision, revisionCount, err := catalogUseCases.LatestRevision(r.Context(), slug)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		var related []service.Page

		if len(page.Tags) > 0 {
			related, err = catalogUseCases.Search(r.Context(), "tag:"+page.Tags[0], 6)
			if err != nil {
				writePageProblem(views.logger, w, err)
				return
			}
		}

		data, err := viewData(r, viewDataUseCases, views, page.Title)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		var comments []service.PageComment

		if data.ApplicationSettings.DiscussionsEnabled {
			comments, err = catalogUseCases.PageComments(r.Context(), slug)
			if err != nil {
				writePageProblem(views.logger, w, err)
				return
			}
		}
		if page.Language != "" {
			data.PageContentLanguage = page.Language
		}

		data.Subpages = navigation.Children(data.Navigation, slug)
		subpages, err := renderTemplateHTML(views, "page", "subpage-toc", data)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		expandedMarkdown, err := expandKnowledgeMarkdown(
			r.Context(),
			knowledgeContentFrom(catalogUseCases, knowledgeUseCases),
			page.Markdown,
			nil,
			0,
		)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		rendered, err := renderer.RenderPageResolvedWithFunctions(
			expandedMarkdown,
			md.Slug,
			options,
			md.Functions{Subpages: string(subpages)},
		)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		renderedHTML := rendered.HTML
		var brokenLinks []service.PageLink

		for _, link := range outgoingLinks {
			if link.Exists {
				continue
			}

			brokenLinks = append(brokenLinks, link)
			renderedHTML = strings.ReplaceAll(
				renderedHTML,
				`<a href="/pages/`+link.TargetSlug+`"`,
				`<a class="wiki-link-broken" href="/pages/`+link.TargetSlug+`"`,
			)
		}

		data.Page, data.HTML, data.Backlinks = &page, template.HTML(renderedHTML), backlinks
		data.OutgoingLinks = outgoingLinks
		data.BrokenLinks = brokenLinks
		data.Comments = comments
		data.PageFavorite = pageFavorite
		data.RevisionCount = revisionCount

		if revisionCount > 0 {
			latestRevision = revision.Analyze(latestRevision)
			data.LatestRevision = &latestRevision
		}

		data.PageContents = rendered.Contents
		data.Related = withoutSlug(related, slug)

		render(views, w, "page", data)
	}
}

// EditPage renders the page creation or editing form.
func EditPage(
	viewDataUseCases viewDataService,
	catalogUseCases pageContentService,
	groupUseCases groupReader,
	knowledgeUseCases knowledgeSnippetReader,
	templateUseCases templateService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		data, err := viewData(r, viewDataUseCases, views, "New page")
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		groups, err := groupUseCases.AssignableGroups(r.Context(), user)
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		data.Groups = groups
		data.RenderingLanguages = renderingLanguageOptions
		data.PageStatuses = service.PageStatuses()
		snippets, err := knowledgeUseCases.KnowledgeSnippets(r.Context())
		if err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		data.KnowledgeSnippets = snippets

		if r.PathValue("slug") == "" {
			data.EditorInitialSlug = md.Slug(r.URL.Query().Get("slug"))
		}

		if r.PathValue("slug") == "" {
			templates, err := templateUseCases.PageTemplates(r.Context())
			if err != nil {
				writePageProblem(views.logger, w, err)
				return
			}

			data.PageTemplates = templates

			if value := r.URL.Query().Get("template"); value != "" {
				id, parseErr := strconv.ParseInt(value, 10, 64)
				if parseErr == nil && id > 0 {
					selected, templateErr := templateUseCases.PageTemplate(r.Context(), id)
					if templateErr == nil {
						data.EditorTemplate = &selected
					}
				}
			}
		}

		if slug := r.PathValue("slug"); slug != "" {
			page, err := catalogUseCases.GetPage(r.Context(), slug)
			if err != nil {
				writePageProblem(views.logger, w, err)
				return
			}

			data.Title, data.Page = "Edit "+page.Title, &page

			if page.Language != "" {
				data.PageContentLanguage = page.Language
			}
		}

		render(views, w, "edit", data)
	}
}

// SavePageForm creates or updates a wiki page from the browser form.
func SavePageForm(
	pageUseCases pageWriterService,
	draftUseCases draftDiscardService,
	views *Views,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid form.")
			return
		}

		originalSlug := strings.TrimSpace(r.FormValue("original_slug"))
		metadata, err := pageMetadataFromForm(r)
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, err.Error())
			return
		}

		properties := pagePropertiesFromForm(r)
		page, err := pageUseCases.Save(r.Context(), service.PageSaveInput{
			PreviousSlug:       originalSlug,
			Slug:               r.FormValue("slug"),
			Title:              r.FormValue("title"),
			Icon:               r.FormValue("icon"),
			Language:           r.FormValue("language"),
			Markdown:           r.FormValue("markdown"),
			Message:            r.FormValue("message"),
			Tags:               splitTags(r.FormValue("tags")),
			GroupIDs:           parseGroupIDs(r.Form["group_id"]),
			Status:             metadata.Status,
			OwnerGroupID:       metadata.OwnerGroupID,
			ReviewIntervalDays: metadata.ReviewIntervalDays,
			MarkReviewed:       metadata.MarkReviewed,
			DeprecatedTarget:   metadata.DeprecatedTarget,
			Properties:         properties,
			Actor:              user,
		})
		if err != nil {
			writePageSaveProblem(views.logger, w, err)
			return
		}

		draftKey := "new"

		if originalSlug != "" {
			draftKey = service.PageDraftKey(page.ID)
		}
		if err := draftUseCases.Delete(r.Context(), user.ID, draftKey); err != nil {
			views.logger.Warn(
				"discard saved page draft",
				"event", "page_draft_cleanup_failed",
				"draft_key", draftKey,
				"user_id", user.ID,
				"error", err,
			)
		}

		http.Redirect(w, r, "/pages/"+page.Slug, http.StatusSeeOther)
	}
}

// DeletePageForm deletes a page from the browser and returns to the wiki home page.
func DeletePageForm(pageUseCases pageWriterService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		slug := r.PathValue("slug")
		if err := pageUseCases.Delete(r.Context(), slug, user); err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// FavoritePage updates the current user's favorite status for a page.
func FavoritePage(catalogUseCases favoriteService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value := r.PathValue("slug")
		if !strings.HasSuffix(value, "/favorite") {
			httpresponse.Problem(w, http.StatusNotFound, "Not found.")
			return
		}

		slug := strings.TrimSuffix(value, "/favorite")
		user, _ := auth.User(r)
		if err := catalogUseCases.SetFavorite(r.Context(), slug, user.ID, r.FormValue("on") != "false"); err != nil {
			writePageProblem(views.logger, w, err)
			return
		}

		http.Redirect(w, r, "/pages/"+slug, http.StatusSeeOther)
	}
}

// splitTags normalizes a comma-separated tag list.
func splitTags(value string) []string {
	parts := strings.Split(value, ",")
	result := parts[:0]

	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}

	return result
}

// parseGroupIDs parses positive group identifiers from form values and leaves invalid values for store validation.
func parseGroupIDs(values []string) []int64 {
	groupIDs := make([]int64, 0, len(values))

	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			groupIDs = append(groupIDs, -1)
			continue
		}

		groupIDs = append(groupIDs, id)
	}

	return groupIDs
}

// pageMetadataFromForm parses page lifecycle and review metadata.
func pageMetadataFromForm(r *http.Request) (service.PageMetadata, error) {
	status := strings.TrimSpace(r.FormValue("status"))
	if !service.ValidPageStatus(status) {
		return service.PageMetadata{}, errors.New("invalid page status")
	}

	var ownerGroupID int64

	if value := strings.TrimSpace(r.FormValue("owner_group_id")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return service.PageMetadata{}, errors.New("invalid owner group")
		}

		ownerGroupID = parsed
	}

	interval := 0

	if value := strings.TrimSpace(r.FormValue("review_interval_days")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 || parsed > 3650 {
			return service.PageMetadata{}, errors.New("invalid review interval")
		}

		interval = parsed
	}

	return service.PageMetadata{
		Status:             status,
		OwnerGroupID:       ownerGroupID,
		ReviewIntervalDays: interval,
		MarkReviewed:       r.FormValue("mark_reviewed") == "on",
		DeprecatedTarget:   md.Slug(r.FormValue("deprecated_target")),
	}, nil
}

// pagePropertiesFromForm returns normalized structured page properties.
func pagePropertiesFromForm(r *http.Request) map[string]string {
	keys := r.Form["property_key"]
	values := r.Form["property_value"]
	properties := map[string]string{}

	for index, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || index >= len(values) {
			continue
		}

		value := strings.TrimSpace(values[index])

		if value != "" {
			properties[key] = value
		}
	}

	return properties
}

// withoutSlug returns pages excluding the supplied slug.
func withoutSlug(pages []service.Page, slug string) []service.Page {
	result := pages[:0]

	for _, page := range pages {
		if page.Slug != slug {
			result = append(result, page)
		}
	}

	return result
}
