package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/internal/httpresponse"
	"github.com/gi8lino/lore/internal/service"
	xhtml "golang.org/x/net/html"
)

const maxImportBytes = 100 << 20

type importCandidate struct {
	Slug     string
	Title    string
	Markdown string
	Source   string
}

type importFormat string

const (
	markdownImport   importFormat = "markdown"
	wikiJSImport     importFormat = "wikijs"
	confluenceImport importFormat = "confluence"
)

// parseImportFormat validates an explicitly selected import source format.
func parseImportFormat(value string) (importFormat, error) {
	format := importFormat(strings.TrimSpace(value))
	switch format {
	case markdownImport, wikiJSImport, confluenceImport:
		return format, nil
	default:
		return "", fmt.Errorf("choose a source format")
	}
}

// AdminImport renders the migration/import workspace.
func AdminImport(viewDataUseCases viewDataService, views *Views) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := administrationData(r, viewDataUseCases, views, "Import", "import")
		if err != nil {
			writeUnexpectedProblem(views.logger, w, err)
			return
		}

		data.Query = r.URL.Query().Get("result")

		render(views, w, "admin_import", data)
	}
}

// ImportPages imports files using the explicitly selected source format.
func ImportPages(pageUseCases pageImportService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
		if err := r.ParseMultipartForm(maxImportBytes); err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, "Import is too large or invalid.")
			return
		}

		format, err := parseImportFormat(r.FormValue("format"))
		if err != nil {
			httpresponse.Problem(w, http.StatusBadRequest, err.Error())
			return
		}

		var candidates []importCandidate

		for _, header := range r.MultipartForm.File["files"] {
			items, err := importCandidatesFromFile(header, format)
			if err != nil {
				httpresponse.Problem(w, http.StatusBadRequest, "Import "+header.Filename+": "+err.Error())
				return
			}

			candidates = append(candidates, items...)
		}

		if len(candidates) == 0 {
			httpresponse.Problem(w, http.StatusBadRequest, "Choose at least one supported import file.")
			return
		}

		importPages := make([]service.ImportedPage, 0, len(candidates))

		for _, candidate := range candidates {
			importPages = append(importPages, service.ImportedPage{
				Slug:     candidate.Slug,
				Title:    candidate.Title,
				Markdown: candidate.Markdown,
				Source:   candidate.Source,
			})
		}

		imported, err := pageUseCases.Import(r.Context(), importPages, string(format), user)
		if err != nil {
			writePageProblem(logger, w, err)
			return
		}

		http.Redirect(w, r, "/admin/import?result="+strconv.Itoa(imported), http.StatusSeeOther)
	}
}

// importCandidatesFromFile extracts import candidates from one uploaded file.
func importCandidatesFromFile(header *multipart.FileHeader, format importFormat) ([]importCandidate, error) {
	file, err := header.Open()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(file, maxImportBytes+1))
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(data) > maxImportBytes {
		return nil, fmt.Errorf("file exceeds 100 MiB")
	}

	name := strings.TrimPrefix(strings.ReplaceAll(header.Filename, "\\", "/"), "./")
	ext := strings.ToLower(path.Ext(name))

	switch format {
	case markdownImport:
		if ext != ".md" && ext != ".markdown" {
			if ext == ".zip" {
				return importZIP(data, format)
			}
			return nil, fmt.Errorf("markdown imports require .md, .markdown, or .zip files")
		}

		title, err := markdownTitle(string(data))
		if err != nil {
			return nil, err
		}

		slug := strings.TrimSuffix(name, ext)

		return []importCandidate{{Slug: slug, Title: title, Markdown: string(data), Source: "Markdown"}}, nil
	case wikiJSImport:
		if ext == ".zip" {
			return importZIP(data, format)
		}
		if ext != ".json" {
			return nil, fmt.Errorf("imports from Wiki.js require .json or .zip files")
		}

		return importWikiJSON(data)
	case confluenceImport:
		if ext == ".zip" {
			return importZIP(data, format)
		}
		if ext != ".html" && ext != ".htm" {
			return nil, fmt.Errorf("imports from Confluence require .html, .htm, or .zip files")
		}

		markdown, err := htmlToMarkdown(data)
		if err != nil {
			return nil, err
		}

		title, err := markdownTitle(markdown)
		if err != nil {
			return nil, err
		}

		slug := strings.TrimSuffix(name, ext)

		return []importCandidate{{
			Slug: slug, Title: title, Markdown: markdown, Source: "Confluence HTML",
		}}, nil
	default:
		return nil, fmt.Errorf("unsupported source format %q", format)
	}
}

// importZIP extracts supported import candidates from a ZIP archive.
func importZIP(data []byte, format importFormat) ([]importCandidate, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var result []importCandidate
	remaining := int64(maxImportBytes)

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}

		name := strings.TrimPrefix(strings.ReplaceAll(entry.Name, "\\", "/"), "./")
		ext := strings.ToLower(path.Ext(name))
		if !supportedImportExtension(format, ext) {
			continue
		}
		if entry.UncompressedSize64 > uint64(remaining) {
			return nil, fmt.Errorf("archive contents exceed 100 MiB")
		}

		file, err := entry.Open()
		if err != nil {
			return nil, err
		}

		content, err := readImportArchiveEntry(file, remaining)
		if err != nil {
			return nil, err
		}

		remaining -= int64(len(content))

		switch format {
		case markdownImport:
			title, err := markdownTitle(string(content))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			slug := strings.TrimSuffix(name, ext)
			result = append(result, importCandidate{
				Slug:     slug,
				Title:    title,
				Markdown: string(content),
				Source:   "ZIP/Markdown",
			})
		case wikiJSImport:
			items, err := importWikiJSON(content)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			result = append(result, items...)
		case confluenceImport:
			markdown, err := htmlToMarkdown(content)
			if err != nil {
				return nil, err
			}

			title, err := markdownTitle(markdown)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}

			slug := strings.TrimSuffix(name, ext)
			result = append(result, importCandidate{
				Slug:     slug,
				Title:    title,
				Markdown: markdown,
				Source:   "Confluence ZIP",
			})
		}
	}

	return result, nil
}

// supportedImportExtension reports whether a file extension belongs to a selected import format.
func supportedImportExtension(format importFormat, extension string) bool {
	switch format {
	case markdownImport:
		return extension == ".md" || extension == ".markdown"
	case wikiJSImport:
		return extension == ".json"
	case confluenceImport:
		return extension == ".html" || extension == ".htm"
	default:
		return false
	}
}

// readImportArchiveEntry reads and closes one entry without exceeding the archive budget.
func readImportArchiveEntry(file io.ReadCloser, remaining int64) ([]byte, error) {
	content, readErr := io.ReadAll(io.LimitReader(file, remaining+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(content)) > remaining {
		return nil, fmt.Errorf("archive contents exceed 100 MiB")
	}

	return content, nil
}

// importWikiJSON decodes pages from a Wiki.js JSON export.
func importWikiJSON(data []byte) ([]importCandidate, error) {
	var pages []struct {
		Path    string `json:"path"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("export from Wiki.js must contain at least one page")
	}

	result := make([]importCandidate, 0, len(pages))

	for index, page := range pages {
		page.Path = strings.TrimSpace(page.Path)
		page.Title = strings.TrimSpace(page.Title)
		if page.Path == "" {
			return nil, fmt.Errorf("page %d from Wiki.js has no path", index+1)
		}
		if page.Title == "" {
			return nil, fmt.Errorf("page %d from Wiki.js has no title", index+1)
		}
		if page.Content == "" {
			return nil, fmt.Errorf("page %d from Wiki.js has no content", index+1)
		}

		result = append(result, importCandidate{
			Slug: page.Path, Title: page.Title, Markdown: page.Content, Source: "Wiki.js JSON",
		})
	}

	return result, nil
}

// markdownTitle returns the first level-one heading in a Markdown document.
func markdownTitle(markdown string) (string, error) {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			if title != "" {
				return title, nil
			}
		}
	}
	return "", fmt.Errorf("document requires a level-one Markdown heading for its title")
}

// htmlToMarkdown converts the supported Confluence HTML subset to Markdown.
func htmlToMarkdown(data []byte) (string, error) {
	root, err := xhtml.Parse(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	var output strings.Builder
	var walk func(*xhtml.Node, int)
	walk = func(node *xhtml.Node, listDepth int) {
		if node.Type == xhtml.TextNode {
			text := strings.Join(strings.Fields(node.Data), " ")
			if text == "" {
				return
			}

			leadingSpace := strings.TrimLeft(node.Data, " \t\r\n") != node.Data
			trailingSpace := strings.TrimRight(node.Data, " \t\r\n") != node.Data

			if leadingSpace && output.Len() > 0 {
				current := output.String()
				last := current[len(current)-1]

				if last != ' ' && last != '\n' {
					output.WriteByte(' ')
				}
			}

			output.WriteString(text)

			if trailingSpace {
				output.WriteByte(' ')
			}

			return
		}
		if node.Type != xhtml.ElementNode && node.Type != xhtml.DocumentNode {
			return
		}

		tag := strings.ToLower(node.Data)

		switch tag {
		case "script", "style", "nav":
			return
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(tag[1] - '0')
			output.WriteString("\n\n" + strings.Repeat("#", level) + " ")
		case "p", "div", "section", "article":
			output.WriteString("\n\n")
		case "br":
			output.WriteByte('\n')
		case "strong", "b":
			output.WriteString("**")
		case "em", "i":
			output.WriteString("*")
		case "code":
			if node.Parent == nil || strings.ToLower(node.Parent.Data) != "pre" {
				output.WriteString("`")
			}
		case "pre":
			output.WriteString("\n\n```\n")
		case "li":
			output.WriteString("\n" + strings.Repeat("  ", listDepth) + "- ")
		case "ul", "ol":
			listDepth++
		case "a":
			output.WriteString("[")
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, listDepth)
		}
		switch tag {
		case "strong", "b":
			output.WriteString("**")
		case "em", "i":
			output.WriteString("*")
		case "code":
			if node.Parent == nil || strings.ToLower(node.Parent.Data) != "pre" {
				output.WriteString("`")
			}
		case "pre":
			output.WriteString("\n```\n")
		case "a":
			href := ""

			for _, attr := range node.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}

			output.WriteString("](" + href + ")")
		case "p", "div", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6":
			output.WriteString("\n")
		}
	}

	walk(root, 0)

	lines := strings.Split(output.String(), "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			if blank {
				continue
			}

			blank = true
			cleaned = append(cleaned, "")
			continue
		}

		blank = false
		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n")) + "\n", nil
}
