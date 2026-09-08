package site

import (
	"bytes"
	"cmp"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gi8lino/lore/internal/icons"
)

//go:embed templates/*.gohtml
var templateFiles embed.FS

type siteTemplates struct {
	page     *template.Template
	search   *template.Template
	notFound *template.Template
}

func (b *Builder) parseTemplates(basePath string) (siteTemplates, error) {
	page, err := b.parseTemplate("page.gohtml", basePath)
	if err != nil {
		return siteTemplates{}, err
	}

	search, err := b.parseTemplate("search.gohtml", basePath)
	if err != nil {
		return siteTemplates{}, err
	}

	notFound, err := b.parseTemplate("not_found.gohtml", basePath)
	if err != nil {
		return siteTemplates{}, err
	}

	return siteTemplates{page: page, search: search, notFound: notFound}, nil
}

func (b *Builder) parseTemplate(pageTemplate, basePath string) (*template.Template, error) {
	logoSVG, err := fs.ReadFile(b.appFS, "lore.svg")
	if err != nil {
		return nil, err
	}

	funcs := template.FuncMap{
		"icon": icons.SVG,
		"logo": func() template.HTML {
			return template.HTML(logoSVG)
		},
		"pageurl": func(route string) string {
			return pageURL(basePath, route)
		},
		"asseturl": func(name string) string {
			return basePath + "assets/" + strings.TrimPrefix(name, "/")
		},
		"searchurl": func() string {
			return basePath + "search/"
		},
	}

	return template.New("static").Funcs(funcs).ParseFS(
		templateFiles,
		"templates/layout.gohtml",
		"templates/navigation.gohtml",
		"templates/"+pageTemplate,
	)
}

func staticBasePath(siteURL string) (string, error) {
	if strings.TrimSpace(siteURL) == "" {
		return "/", nil
	}

	parsed, err := url.Parse(strings.TrimSpace(siteURL))
	if err != nil {
		return "", fmt.Errorf("parse site_url: %w", err)
	}

	base := parsed.Path
	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	cleaned := path.Clean(base)
	if cleaned == "/" {
		return "/", nil
	}

	return strings.TrimSuffix(cleaned, "/") + "/", nil
}

func pageURL(basePath, route string) string {
	basePath = ensureBasePath(basePath)
	route = strings.Trim(route, "/")
	if route == "" {
		return basePath
	}

	return basePath + route + "/"
}

func ensureBasePath(basePath string) string {
	if basePath == "" || basePath == "." {
		return "/"
	}

	basePath = "/" + strings.Trim(basePath, "/")
	if basePath == "/" {
		return basePath
	}

	return basePath + "/"
}

func routeSuffix(route string) string {
	route = strings.Trim(route, "/")
	if route == "" {
		return ""
	}

	return route + "/"
}

func outputPath(route string) string {
	if strings.Trim(route, "/") == "" {
		return "index.html"
	}

	return filepath.Join(filepath.FromSlash(strings.Trim(route, "/")), "index.html")
}

func outputFilename(outputDir, route string) string {
	return outputFile(outputDir, outputPath(route))
}

func outputFile(outputDir string, parts ...string) string {
	all := make([]string, 0, len(parts)+1)
	all = append(all, outputDir)
	all = append(all, parts...)
	return filepath.Join(all...)
}

func writeTemplate(tmpl *template.Template, filename string, data viewData) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.ExecuteTemplate(&output, "layout", data); err != nil {
		return err
	}

	return os.WriteFile(filename, output.Bytes(), 0o644)
}

func writeJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(filename, data, 0o644)
}

func writeSitemap(config Config, pages []sourcePage) error {
	if strings.TrimSpace(config.SiteURL) == "" {
		return nil
	}

	parsed, err := url.Parse(config.SiteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil
	}

	basePath, err := staticBasePath(config.SiteURL)
	if err != nil {
		return err
	}

	origin := parsed.Scheme + "://" + parsed.Host
	var output strings.Builder
	output.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	output.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, page := range pages {
		output.WriteString("  <url><loc>")
		output.WriteString(template.HTMLEscapeString(origin + pageURL(basePath, page.Route)))
		output.WriteString("</loc></url>\n")
	}
	output.WriteString("</urlset>\n")

	return os.WriteFile(outputFile(config.OutputDir, "sitemap.xml"), []byte(output.String()), 0o644)
}

// compareSearchEntries orders search results by case-insensitive page title.
func compareSearchEntries(left, right searchEntry) int {
	return cmp.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
}
