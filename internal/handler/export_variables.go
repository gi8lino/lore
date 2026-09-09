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

// readExportVariables accepts variable overrides only from POST request bodies.
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

// renderExportHTML renders the shared self-contained page body used by print preview and PDF export.
func renderExportHTML(
	ctx context.Context,
	catalog pageContentService,
	knowledge knowledgeContentService,
	navigation navigationService,
	media imageContentService,
	renderer *md.Renderer,
	page domain.Page,
	settings domain.RenderingSettings,
	overrides map[string]string,
) (string, error) {
	expanded, err := expandPageKnowledge(ctx, knowledgeContentFrom(catalog, knowledge), page.Markdown, overrides, false)
	if err != nil {
		return "", err
	}
	renderSubpages, err := subpagesRenderer(ctx, navigation, page.Slug)
	if err != nil {
		return "", err
	}
	rendered, err := renderer.RenderPageResolvedWithFunctions(expanded.Markdown, md.Slug, renderingOptionsFromSettings(settings), md.Functions{Subpages: renderSubpages})
	if err != nil {
		return "", err
	}
	return inlineRenderedMedia(ctx, media, rendered.HTML)
}

// PreviewPageExport returns a self-contained script-free print document without calling the PDF service.
func PreviewPageExport(
	catalog pageContentService,
	settings settingsService,
	navigation navigationService,
	knowledge knowledgeContentService,
	media imageContentService,
	renderer *md.Renderer,
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
		rendered, err := renderExportHTML(r.Context(), catalog, knowledge, navigation, media, renderer, page, application.Rendering, overrides)
		if err != nil {
			writeRenderedExportProblem(logger, w, err)
			return
		}
		language := cmp.Or(page.Language, application.Rendering.ContentLanguage)
		httpresponse.Respond(w, http.StatusOK, exportPreviewResponse{Document: pdf.Document(page.Title, language, rendered)})
	}
}

// writeRenderedExportProblem translates expected rendered-export failures and hides infrastructure errors.
func writeRenderedExportProblem(logger *slog.Logger, w http.ResponseWriter, err error) {
	if tryWriteValidationProblem(w, err, "Export validation failed.") {
		return
	}
	if _, ok := errors.AsType[*exportMediaError](err); ok {
		writeExportProblem(logger, w, err)
		return
	}
	writeInternalServerError(logger, w, err)
}
