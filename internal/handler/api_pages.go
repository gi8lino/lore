package handler

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/service"
)

// previewRequest contains Markdown submitted for server-side editor preview.
type previewRequest struct {
	// Markdown is the unsaved Markdown source to render.
	Markdown string `json:"markdown"`
	// Slug is the current unsaved page path used by dynamic page functions.
	Slug string `json:"slug"`
}

// pageRequest contains the mutable page fields accepted by the JSON API.
type pageRequest struct {
	// Slug supplies the desired page path for create requests.
	Slug string `json:"slug"`
	// Title is the required page title.
	Title string `json:"title"`
	// Icon is the optional Lucide icon displayed with the page title.
	Icon string `json:"icon"`
	// Language optionally overrides the wiki-wide content language.
	Language string `json:"language"`
	// Markdown is the page Markdown body.
	Markdown string `json:"markdown_content"`
	// Tags contains the requested page tags.
	Tags []string `json:"tags"`
	// GroupIDs contains collaboration groups assigned to the page.
	GroupIDs []int64 `json:"group_ids"`
	// Message describes the revision being created.
	Message string `json:"message"`
	// Status is the page lifecycle state.
	Status string `json:"status"`
	// OwnerGroupID optionally assigns documentation ownership to a group.
	OwnerGroupID int64 `json:"owner_group_id"`
	// ReviewIntervalDays configures documentation review cadence.
	ReviewIntervalDays int `json:"review_interval_days"`
	// MarkReviewed records a review at save time.
	MarkReviewed bool `json:"mark_reviewed"`
	// DeprecatedTarget points deprecated content at its replacement.
	DeprecatedTarget string `json:"deprecated_target"`
	// Properties contains structured page metadata.
	Properties map[string]string `json:"properties"`
}

// PreviewMarkdown renders unsaved Markdown with the same resolver used by persisted pages.
func PreviewMarkdown(
	settingsUseCases settingsService,
	navigationUseCases navigationService,
	catalogUseCases pageContentService,
	knowledgeUseCases knowledgeContentService,
	renderer *md.Renderer,
	views *Views,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := decode[previewRequest](w, r)
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid JSON request.",
				httpresponse.NewFieldProblem("request", err.Error()),
			)
			return
		}

		options, _, err := renderingOptions(r.Context(), settingsUseCases)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		slug := md.Slug(request.Slug)
		subpages, err := subpagesHTML(r.Context(), navigationUseCases, views, slug)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		expandedMarkdown, err := expandKnowledgeMarkdown(
			r.Context(),
			knowledgeContentFrom(catalogUseCases, knowledgeUseCases),
			request.Markdown,
			nil,
			0,
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		rendered, err := renderer.RenderPageResolvedWithFunctions(
			expandedMarkdown,
			md.Slug,
			options,
			md.Functions{Subpages: string(subpages)},
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, map[string]string{"html": rendered.HTML})
	}
}

// subpagesHTML builds the current subtree used by dynamic Markdown rendering.
func subpagesHTML(
	ctx context.Context,
	navigationUseCases navigationService,
	views *Views,
	slug string,
) (template.HTML, error) {
	pages, err := navigationUseCases.NavigationPages(ctx)
	if err != nil {
		return "", err
	}

	icons, err := navigationUseCases.NavigationIcons(ctx)
	if err != nil {
		return "", err
	}

	items := make([]navigation.Page, 0, len(pages))

	for _, page := range pages {
		items = append(items, navigation.Page{Slug: page.Slug, Title: page.Title, Icon: page.Icon})
	}

	data := ViewData{Subpages: navigation.Children(navigation.Build(items, navigation.Options{Icons: icons}), slug)}

	return renderTemplateHTML(views, "page", "subpage-toc", data)
}

// ListPages returns recently updated wiki pages up to the requested limit.
func ListPages(catalogUseCases pageListService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pages, err := catalogUseCases.ListPages(r.Context(), 100)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		stripMarkdown(pages)
		httpresponse.Respond(w, http.StatusOK, pages)
	}
}

// GetPage returns a wiki page by slug.
func GetPage(catalogUseCases pageLookupService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if strings.HasSuffix(slug, "/raw") {
			page, err := catalogUseCases.GetPage(r.Context(), strings.TrimSuffix(slug, "/raw"))
			if err != nil {
				writePageProblem(logger, w, err)
				return
			}

			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")

			_, _ = w.Write([]byte(page.Markdown))

			return
		}

		page, err := catalogUseCases.GetPage(r.Context(), slug)

		if errors.Is(err, service.ErrNotFound) {
			if target, aliasErr := catalogUseCases.ResolvePageAlias(r.Context(), slug); aliasErr == nil {
				w.Header().Set("Content-Location", "/api/pages/"+target)
				page, err = catalogUseCases.GetPage(r.Context(), target)
			}
		}

		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		httpresponse.Respond(w, http.StatusOK, page)
	}
}

// SavePage persists page content, revision history, tags, and links transactionally.
func SavePage(pageUseCases pageWriterService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		request, err := decode[pageRequest](w, r)
		if err != nil {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Invalid JSON request.",
				httpresponse.NewFieldProblem("request", err.Error()),
			)
			return
		}

		slug := r.PathValue("slug")

		if slug == "" {
			slug = request.Slug
		}

		page, err := pageUseCases.Save(r.Context(), service.PageSaveInput{
			PreviousSlug:       r.PathValue("slug"),
			Slug:               slug,
			Title:              request.Title,
			Icon:               request.Icon,
			Language:           request.Language,
			Markdown:           request.Markdown,
			Message:            request.Message,
			Tags:               request.Tags,
			GroupIDs:           request.GroupIDs,
			Status:             request.Status,
			OwnerGroupID:       request.OwnerGroupID,
			ReviewIntervalDays: request.ReviewIntervalDays,
			MarkReviewed:       request.MarkReviewed,
			DeprecatedTarget:   request.DeprecatedTarget,
			Properties:         request.Properties,
			Actor:              user,
		})
		if err != nil {
			writePageSaveProblem(logger, w, err)
			return
		}

		status := http.StatusOK

		if r.Method == http.MethodPost {
			status = http.StatusCreated
			w.Header().Set("Location", "/api/pages/"+page.Slug)
		}

		httpresponse.Respond(w, status, page)
	}
}

// DeletePage removes a wiki page by slug.
func DeletePage(pageUseCases pageWriterService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if err := pageUseCases.Delete(r.Context(), r.PathValue("slug"), user); err != nil {
			writePageProblem(logger, w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// PermanentlyDeletePage removes a page already held in the recycle bin.
func PermanentlyDeletePage(recycleBinUseCases recycleBinService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := recycleBinUseCases.PermanentlyDeletePage(r.Context(), r.PathValue("slug")); err != nil {
			writePageProblem(logger, w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// stripMarkdown removes Markdown bodies from page summaries.
func stripMarkdown(pages []service.Page) {
	for index := range pages {
		pages[index].Markdown = ""
	}
}
