package handler

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gi8lino/lore/internal/httpresponse"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/pdf"
	"github.com/gi8lino/lore/internal/service"
)

var (
	mediaReference         = regexp.MustCompile(`/media/([0-9]+)/([^\s)"']+)`)
	renderedMediaReference = regexp.MustCompile(`/media/([0-9]+)/[^"'\s>]+`)
)

// ExportPageMarkdown exports one page as Markdown or as a ZIP when referenced images must be included.
func ExportPageMarkdown(
	catalogUseCases pageContentService,
	mediaUseCases imageContentService,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Export validation failed.",
				httpresponse.NewFieldProblem("slug", "A page path is required."),
			)
			return
		}

		pageData, err := catalogUseCases.GetPage(r.Context(), slug)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		if len(referencedImageIDs(pageData.Markdown)) == 0 {
			serveMarkdown(w, pageData)
			return
		}

		file, modTime, cleanup, err := createExportArchive(
			r.Context(),
			catalogUseCases,
			mediaUseCases,
			[]string{slug},
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		defer cleanup()
		serveExportArchive(w, r, path.Base(slug)+".zip", file, modTime)
	}
}

// ExportPagePDF renders one page into a downloadable PDF using the configured PDF service.
func ExportPagePDF(
	catalogUseCases pageContentService,
	settingsUseCases settingsService,
	navigationUseCases navigationService,
	knowledgeUseCases knowledgeContentService,
	mediaUseCases imageContentService,
	renderer *md.Renderer,
	views *Views,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applicationSettings, err := settingsUseCases.ApplicationSettings(r.Context())
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		pdfURL := effectivePDFURL(views.runtime.PDFURL, applicationSettings.PDFURL)
		if pdfURL == "" {
			httpresponse.Problem(w, http.StatusServiceUnavailable, "PDF export is not configured. Configure a PDF service in Administration or set LORE__PDF_URL.")
			return
		}
		slug := strings.TrimSpace(r.PathValue("slug"))
		if slug == "" {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Export validation failed.",
				httpresponse.NewFieldProblem("slug", "A page path is required."),
			)
			return
		}

		pageData, err := catalogUseCases.GetPage(r.Context(), slug)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		settings := applicationSettings.Rendering
		options := renderingOptionsFromSettings(settings)
		subpages, err := subpagesHTML(r.Context(), navigationUseCases, views, slug)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		expandedMarkdown, err := expandKnowledgeMarkdown(
			r.Context(),
			knowledgeContentFrom(
				catalogUseCases,
				knowledgeUseCases,
			),
			pageData.Markdown,
			nil,
			0,
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		page, err := renderer.RenderPageResolvedWithFunctions(
			expandedMarkdown,
			md.Slug,
			options,
			md.Functions{Subpages: string(subpages)},
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		rendered, err := inlineRenderedMedia(r.Context(), mediaUseCases, page.HTML)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		language := settings.ContentLanguage
		if pageData.Language != "" {
			language = pageData.Language
		}
		pdfFile, cleanup, err := pdf.Render(r.Context(), pdfURL, pageData.Title, language, rendered)
		if err != nil {
			logger.Error("generate PDF", "event", "pdf_generation_failed", "slug", slug, "error", err)
			httpresponse.Problem(w,
				http.StatusBadGateway,
				"PDF could not be generated. Check the configured PDF service and try again.",
			)
			return
		}
		defer cleanup()

		filename := path.Base(slug) + ".pdf"
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
		http.ServeContent(w, r, filename, time.Now(), pdfFile)
	}
}

// ExportPages creates an archive containing selected Markdown pages and referenced images.
func ExportPages(
	catalogUseCases pageContentService,
	navigationUseCases navigationService,
	mediaUseCases imageContentService,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Invalid export request.")
			return
		}

		slugs, err := exportSlugs(r, navigationUseCases)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		if len(slugs) == 0 {
			httpresponse.Problem(w,
				http.StatusBadRequest,
				"Export validation failed.",
				httpresponse.NewFieldProblem("slug", "Select at least one page to export."),
			)
			return
		}
		if len(slugs) == 1 {
			pageData, err := catalogUseCases.GetPage(r.Context(), slugs[0])
			if err != nil {
				writePageProblem(logger, w, err)
				return
			}
			if len(referencedImageIDs(pageData.Markdown)) == 0 {
				serveMarkdown(w, pageData)
				return
			}
		}

		file, modTime, cleanup, err := createExportArchive(
			r.Context(),
			catalogUseCases,
			mediaUseCases,
			slugs,
		)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}
		defer cleanup()

		filename := "lore-export-" + time.Now().UTC().Format("20060102-150405") + ".zip"
		serveExportArchive(w, r, filename, file, modTime)
	}
}

// serveMarkdown writes one page as a plain Markdown attachment.
func serveMarkdown(w http.ResponseWriter, pageData service.Page) {
	filename := path.Base(pageData.Slug) + ".md"
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	_, _ = io.WriteString(w, pageData.Markdown)
}

// createExportArchive builds a temporary ZIP file for selected pages and returns its cleanup function.
func createExportArchive(
	ctx context.Context,
	catalogUseCases pageContentService,
	mediaUseCases imageContentService,
	slugs []string,
) (*os.File, time.Time, func(), error) {
	file, err := os.CreateTemp("", "lore-export-*.zip")
	if err != nil {
		return nil, time.Time{}, nil, err
	}
	name := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(name)
	}

	if err := writeExportArchive(ctx, catalogUseCases, mediaUseCases, file, slugs); err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		cleanup()
		return nil, time.Time{}, nil, err
	}
	return file, info.ModTime(), cleanup, nil
}

// serveExportArchive writes a prepared ZIP archive to the client.
func serveExportArchive(w http.ResponseWriter, r *http.Request, filename string, file *os.File, modTime time.Time) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	http.ServeContent(w, r, filename, modTime, file)
}

// exportSlugs resolves either the explicit page selection or all page slugs.
func exportSlugs(r *http.Request, navigationUseCases navigationService) ([]string, error) {
	if r.FormValue("all") == "true" {
		pages, err := navigationUseCases.NavigationPages(r.Context())
		if err != nil {
			return nil, err
		}
		slugs := make([]string, 0, len(pages))
		for _, page := range pages {
			slugs = append(slugs, page.Slug)
		}
		return slugs, nil
	}

	seen := map[string]bool{}
	var slugs []string
	for _, slug := range r.Form["slug"] {
		slug = strings.TrimSpace(slug)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		slugs = append(slugs, slug)
	}
	return slugs, nil
}

// writeExportArchive writes selected Markdown pages and each referenced image once.
func writeExportArchive(
	ctx context.Context,
	catalogUseCases pageContentService,
	mediaUseCases imageContentService,
	output io.Writer,
	slugs []string,
) error {
	archive := zip.NewWriter(output)

	exportedImages := map[int64]bool{}
	imageCache := map[int64]service.ImageData{}
	for _, slug := range slugs {
		pageData, err := catalogUseCases.GetPage(ctx, slug)
		if err != nil {
			return err
		}

		cleanSlug := strings.Trim(path.Clean("/"+pageData.Slug), "/")
		if cleanSlug == "" || cleanSlug == "." {
			return fmt.Errorf("invalid page slug %q", pageData.Slug)
		}
		markdownPath := path.Join("pages", cleanSlug+".md")
		markdown, imageIDs, err := exportedMarkdown(
			ctx,
			mediaUseCases,
			markdownPath,
			pageData.Markdown,
			imageCache,
		)
		if err != nil {
			return err
		}
		entry, err := archive.Create(markdownPath)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(entry, markdown); err != nil {
			return err
		}

		for _, imageID := range imageIDs {
			if exportedImages[imageID] {
				continue
			}
			image := imageCache[imageID]
			imageEntry, err := archive.Create(
				path.Join("media", strconv.FormatInt(imageID, 10), path.Base(image.Filename)),
			)
			if err != nil {
				return err
			}
			if _, err := imageEntry.Write(image.Data); err != nil {
				return err
			}
			exportedImages[imageID] = true
		}
	}

	return archive.Close()
}

// exportedMarkdown rewrites stored media URLs to relative archive paths and returns referenced image identifiers.
func exportedMarkdown(
	ctx context.Context,
	mediaUseCases imageContentService,
	markdownPath, source string,
	imageCache map[int64]service.ImageData,
) (string, []int64, error) {
	seen := map[int64]bool{}
	var ids []int64
	var exportErr error

	result := mediaReference.ReplaceAllStringFunc(source, func(reference string) string {
		if exportErr != nil {
			return reference
		}
		match := mediaReference.FindStringSubmatch(reference)
		if len(match) != 3 {
			return reference
		}
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return reference
		}
		image, ok := imageCache[id]
		if !ok {
			image, err = mediaUseCases.ImageContent(ctx, id)
			if err != nil {
				exportErr = err
				return reference
			}
			imageCache[id] = image
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}

		target := filepath.FromSlash(path.Join("media", strconv.FormatInt(id, 10), path.Base(image.Filename)))
		from := filepath.FromSlash(path.Dir(markdownPath))
		relative, err := filepath.Rel(from, target)
		if err != nil {
			exportErr = err
			return reference
		}
		return filepath.ToSlash(relative)
	})
	if exportErr != nil {
		return "", nil, exportErr
	}
	return result, ids, nil
}

// referencedImageIDs returns unique image identifiers referenced from Markdown source.
func referencedImageIDs(source string) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	for _, match := range mediaReference.FindAllStringSubmatch(source, -1) {
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// inlineRenderedMedia replaces authenticated media URLs with data URLs for standalone PDF rendering.
func inlineRenderedMedia(ctx context.Context, mediaUseCases imageContentService, rendered string) (string, error) {
	cache := map[int64]string{}
	var inlineErr error
	result := renderedMediaReference.ReplaceAllStringFunc(rendered, func(reference string) string {
		if inlineErr != nil {
			return reference
		}
		match := renderedMediaReference.FindStringSubmatch(reference)
		if len(match) != 2 {
			return reference
		}
		id, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return reference
		}
		if dataURL, ok := cache[id]; ok {
			return dataURL
		}
		image, err := mediaUseCases.ImageContent(ctx, id)
		if err != nil {
			inlineErr = err
			return reference
		}
		dataURL := "data:" + image.ContentType + ";base64," + base64.StdEncoding.EncodeToString(image.Data)
		cache[id] = dataURL
		return dataURL
	})
	return result, inlineErr
}
