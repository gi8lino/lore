package handler

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/pdf"
)

type exportVariablesRequest struct {
	Variables map[string]string `json:"variables"`
}

type exportPreviewResponse struct {
	Document string `json:"document"`
}

// readExportVariables accepts overrides only in a POST body, never in a URL.
// Existing GET PDF links continue to render the currently saved values.
func readExportVariables(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	if r.Method != http.MethodPost {
		return nil, nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512<<10)
	request, err := decode[exportVariablesRequest](w, r)
	if err != nil {
		return nil, err
	}
	return request.Variables, nil
}

// renderExportHTML is shared by print preview and PDF export. It has no mutation
// dependencies and never emits the reading page's variable inspection markers.
func renderExportHTML(
	ctx context.Context,
	catalog pageContentService,
	knowledge knowledgeContentService,
	navigation navigationService,
	media imageContentService,
	renderer *md.Renderer,
	views *Views,
	page domain.Page,
	settings domain.RenderingSettings,
	overrides map[string]string,
) (string, error) {
	expanded, err := expandPageKnowledge(ctx, knowledgeContentFrom(catalog, knowledge), page.Markdown, overrides, false)
	if err != nil {
		return "", err
	}
	subpages, err := subpagesHTML(ctx, navigation, views, page.Slug)
	if err != nil {
		return "", err
	}
	rendered, err := renderer.RenderPageResolvedWithFunctions(expanded.Markdown, md.Slug, renderingOptionsFromSettings(settings), md.Functions{Subpages: string(subpages)})
	if err != nil {
		return "", err
	}
	return inlineRenderedMedia(ctx, media, rendered.HTML)
}

// PreviewPageExport returns a self-contained, script-free print document. The
// same stylesheet and resolver are used by PDF export; no PDF service is needed.
func PreviewPageExport(
	catalog pageContentService,
	settings settingsService,
	navigation navigationService,
	knowledge knowledgeContentService,
	media imageContentService,
	renderer *md.Renderer,
	views *Views,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		overrides, err := readExportVariables(w, r)
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid export request.")
			return
		}
		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			httpresponse.Problem(w, http.StatusBadRequest, "A page path is required.")
			return
		}
		page, err := catalog.GetPage(r.Context(), slug)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		application, err := settings.ApplicationSettings(r.Context())
		if err != nil {
			writeInternalServerError(logger, w, err)
			return
		}
		rendered, err := renderExportHTML(r.Context(), catalog, knowledge, navigation, media, renderer, views, page, application.Rendering, overrides)
		if err != nil {
			writeRenderedExportProblem(logger, w, err)
			return
		}
		language := cmp.Or(page.Language, application.Rendering.ContentLanguage)
		httpresponse.Respond(w, http.StatusOK, exportPreviewResponse{Document: pdf.Document(page.Title, language, rendered)})
	}
}

// writeRenderedExportProblem distinguishes invalid overrides and missing media
// from failures loading the dependencies of an otherwise existing page.
func writeRenderedExportProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	if writeValidationProblem(w, err, "Export validation failed.") {
		return
	}
	if _, ok := errors.AsType[*exportMediaError](err); ok {
		writeExportProblem(logger, w, err)
		return
	}
	writeInternalServerError(logger, w, err)
}
