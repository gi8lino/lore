package handler

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/service"
)

type createPageShareLinkResponse struct {
	URL string `json:"url"`
}

// CreatePageShareLink creates a reusable public permalink for the selected page.
func CreatePageShareLink(sharingUseCases sharingService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		issued, err := sharingUseCases.CreatePageShareLink(
			r.Context(),
			r.PathValue("slug"),
			currentUser(r),
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		httpresponse.Respond(w, http.StatusCreated, createPageShareLinkResponse{
			URL: "/share/" + issued.Token,
		})
	}
}

// SharedPage serves a reusable public permalink without authentication.
func SharedPage(
	sharingUseCases sharingService,
	catalogUseCases pageContentService,
	settingsUseCases settingsService,
	knowledgeUseCases knowledgeContentService,
	mediaUseCases imageContentService,
	renderer *md.Renderer,
	views *Views,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		publicShareHeaders(w)
		link, err := sharingUseCases.PageShareLink(r.Context(), strings.TrimSpace(r.PathValue("token")))
		if err != nil {
			writePublicShareError(logger, w, err)
			return
		}
		renderSharedPage(
			w,
			r,
			catalogUseCases,
			settingsUseCases,
			knowledgeUseCases,
			mediaUseCases,
			renderer,
			views,
			logger,
			link.Slug,
		)
	}
}

// renderSharedPage renders a standalone page without authenticated navigation or controls.
func renderSharedPage(
	w http.ResponseWriter,
	r *http.Request,
	catalogUseCases pageContentService,
	settingsUseCases settingsService,
	knowledgeUseCases knowledgeContentService,
	mediaUseCases imageContentService,
	renderer *md.Renderer,
	views *Views,
	logger *slog.Logger,
	slug string,
) {
	page, err := catalogUseCases.GetPage(r.Context(), slug)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}
	options, settings, err := renderingOptions(r.Context(), settingsUseCases)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}
	expanded, err := expandKnowledgeMarkdown(
		r.Context(),
		knowledgeContentFrom(catalogUseCases, knowledgeUseCases),
		page.Markdown,
		nil,
		0,
	)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}
	rendered, err := renderer.RenderPageResolvedWithFunctions(
		expanded,
		md.Slug,
		options,
		md.Functions{},
	)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}
	standaloneHTML, err := inlineRenderedMedia(r.Context(), mediaUseCases, rendered.HTML)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}

	data, err := publicViewData(views, page.Title)
	if err != nil {
		writePublicShareError(logger, w, err)
		return
	}
	data.Page = &page
	data.HTML = template.HTML(standaloneHTML)
	data.ApplicationSettings.Rendering = settings
	data.RenderMermaid = settings.Mermaid
	data.PageContentLanguage = settings.ContentLanguage
	if page.Language != "" {
		data.PageContentLanguage = page.Language
	}
	renderTemplate(views, w, "shared_page", "shared-layout", data)
}

// publicShareHeaders prevent public bearer URLs from leaking through caches, referrers, or indexing.
func publicShareHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
}

// writePublicShareError hides whether an invalid, revoked, or deleted share link ever existed.
func writePublicShareError(logger *slog.Logger, w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrNotFound) {
		httpresponse.Problem(w, http.StatusNotFound, "Share link not found or no longer available.")
		return
	}
	logger.Error("public share request failed", "event", "public_share_failed", "error", err)
	httpresponse.Problem(w, http.StatusInternalServerError, "The request could not be processed.")
}
