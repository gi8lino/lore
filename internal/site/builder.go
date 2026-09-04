package site

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gi8lino/lore/internal/icons"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/themes"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//go:embed templates/*.gohtml
var templateFiles embed.FS

var staticBrowserAssets = []string{
	"css/app.css",
	"favicon.svg",
	"lore-mark.svg",
	"lore.svg",
	"js/static.js",
	"js/core/clipboard.js",
	"js/core/theme.js",
	"js/features/markdown.js",
	"js/features/static-layout.js",
	"js/features/static-page.js",
	"js/features/static-search.js",
}

// Builder converts Markdown files into a read-only Lore site.
type Builder struct {
	appFS    fs.FS
	version  string
	commit   string
	renderer *md.Renderer
}

// Result summarizes one completed static build.
type Result struct {
	Pages     int
	OutputDir string
}

type sourcePage struct {
	SourcePath      string
	Route           string
	Title           string
	Markdown        string
	HasTitleHeading bool
	HTML            template.HTML
	Contents        []md.Heading
	SearchText      string
}

type searchEntry struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
}

type viewData struct {
	SiteName      string
	SiteURL       string
	BasePath      string
	Language      string
	Title         string
	ActiveTheme   string
	ThemeData     template.JS
	Version       string
	Commit        string
	CurrentRoute  string
	Navigation    []navigation.Node
	HTML          template.HTML
	PageContents  []md.Heading
	RenderMermaid bool
}

// NewBuilder constructs a filesystem-backed site builder using embedded Lore assets.
func NewBuilder(appFS fs.FS, version, commit string) *Builder {
	return &Builder{
		appFS:    appFS,
		version:  version,
		commit:   commit,
		renderer: md.New(),
	}
}

// Build renders all Markdown files from SourceDir into OutputDir.
func (b *Builder) Build(ctx context.Context, config Config) (Result, error) {
	if err := config.validate(); err != nil {
		return Result{}, err
	}

	availableThemes, err := themes.Load("")
	if err != nil {
		return Result{}, err
	}
	if _, found := themes.Find(availableThemes, config.Theme); !found {
		return Result{}, fmt.Errorf("unknown theme %q", config.Theme)
	}
	themeJSON, err := json.Marshal(availableThemes)
	if err != nil {
		return Result{}, err
	}

	pages, err := discoverPages(config.SourceDir)
	if err != nil {
		return Result{}, err
	}
	if len(pages) == 0 {
		return Result{}, fmt.Errorf("no Markdown files found in %s", config.SourceDir)
	}
	if !hasHomePage(pages) {
		return Result{}, fmt.Errorf("%s must contain index.md for the site home page", config.SourceDir)
	}

	basePath, err := staticBasePath(config.SiteURL)
	if err != nil {
		return Result{}, err
	}
	routesBySource := make(map[string]string, len(pages))
	wikiTargets := make(map[string]string, len(pages)*2)
	ambiguousWikiTargets := make(map[string]bool)
	for index := range pages {
		page := &pages[index]
		routesBySource[page.SourcePath] = page.Route
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Route), page.Route)
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Title), page.Route)
	}
	for target := range ambiguousWikiTargets {
		delete(wikiTargets, target)
	}

	if err := os.RemoveAll(config.OutputDir); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(config.OutputDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := b.copyBrowserAssets(config.OutputDir); err != nil {
		return Result{}, err
	}
	if err := copySourceAssets(config.SourceDir, config.OutputDir); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(filepath.Join(config.OutputDir, ".nojekyll"), nil, 0o644); err != nil {
		return Result{}, err
	}

	baseNavigationPages := make([]navigation.Page, 0, len(pages))
	for _, page := range pages {
		if page.Route == "" {
			continue
		}
		baseNavigationPages = append(baseNavigationPages, navigation.Page{Slug: page.Route, Title: page.Title})
	}
	baseTree := navigation.Build(baseNavigationPages, navigation.Options{})

	pageTemplate, err := b.parseTemplate("page.gohtml", basePath)
	if err != nil {
		return Result{}, err
	}
	searchTemplate, err := b.parseTemplate("search.gohtml", basePath)
	if err != nil {
		return Result{}, err
	}
	notFoundTemplate, err := b.parseTemplate("not_found.gohtml", basePath)
	if err != nil {
		return Result{}, err
	}

	common := viewData{
		SiteName:      config.SiteName,
		SiteURL:       config.SiteURL,
		BasePath:      basePath,
		Language:      config.Language,
		ActiveTheme:   config.Theme,
		ThemeData:     template.JS(string(themeJSON)),
		Version:       b.version,
		Commit:        b.commit,
		RenderMermaid: config.Mermaid,
	}

	searchIndex := make([]searchEntry, 0, len(pages))
	options := md.DefaultOptions()
	options.WikiLinkPrefix = basePath
	for index := range pages {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		page := &pages[index]
		if err := validateWikiLinks(*page, wikiTargets); err != nil {
			return Result{}, err
		}
		resolveWiki := func(target string) string {
			normalized := md.Slug(target)
			if route, found := wikiTargets[normalized]; found {
				return routeSuffix(route)
			}
			return routeSuffix(normalized)
		}
		subpages := renderSubpages(baseTree, page.Route, basePath)
		rendered, err := b.renderer.RenderPageResolvedWithFunctions(
			page.Markdown,
			resolveWiki,
			options,
			md.Functions{Subpages: subpages},
		)
		if err != nil {
			return Result{}, fmt.Errorf("render %s: %w", page.SourcePath, err)
		}
		processedHTML, searchText, err := processRenderedHTML(
			rendered.HTML,
			page.SourcePath,
			page.HasTitleHeading,
			routesBySource,
			basePath,
		)
		if err != nil {
			return Result{}, fmt.Errorf("rewrite %s: %w", page.SourcePath, err)
		}
		page.HTML = template.HTML(processedHTML)
		page.SearchText = searchText
		page.Contents = rendered.Contents
		if page.HasTitleHeading && len(page.Contents) > 0 && page.Contents[0].Level == 1 {
			page.Contents = page.Contents[1:]
		}

		data := common
		data.Title = page.Title
		data.CurrentRoute = page.Route
		data.Navigation = navigation.Build(baseNavigationPages, navigation.Options{
			ActiveSlug: page.Route,
			Expanded:   expandedPrefixes(page.Route),
		})
		data.HTML = page.HTML
		data.PageContents = page.Contents
		if err := writeTemplate(
			pageTemplate,
			filepath.Join(config.OutputDir, outputPath(page.Route)),
			data,
		); err != nil {
			return Result{}, err
		}

		searchIndex = append(searchIndex, searchEntry{
			Title: page.Title,
			URL:   pageURL(basePath, page.Route),
			Text:  page.SearchText,
		})
	}

	sort.Slice(searchIndex, func(i, j int) bool {
		return strings.ToLower(searchIndex[i].Title) < strings.ToLower(searchIndex[j].Title)
	})
	if err := writeJSON(filepath.Join(config.OutputDir, "search-index.json"), searchIndex); err != nil {
		return Result{}, err
	}

	searchData := common
	searchData.Title = "Search"
	searchData.Navigation = navigation.Build(baseNavigationPages, navigation.Options{})
	if err := writeTemplate(
		searchTemplate,
		filepath.Join(config.OutputDir, "search", "index.html"),
		searchData,
	); err != nil {
		return Result{}, err
	}

	notFoundData := common
	notFoundData.Title = "Page not found"
	notFoundData.Navigation = navigation.Build(baseNavigationPages, navigation.Options{})
	if err := writeTemplate(
		notFoundTemplate,
		filepath.Join(config.OutputDir, "404.html"),
		notFoundData,
	); err != nil {
		return Result{}, err
	}

	if err := writeSitemap(config, pages); err != nil {
		return Result{}, err
	}
	return Result{Pages: len(pages), OutputDir: config.OutputDir}, nil
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

func discoverPages(sourceDir string) ([]sourcePage, error) {
	var pages []sourcePage
	routes := make(map[string]string)
	err := filepath.WalkDir(sourceDir, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == sourceDir {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}

		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		route := markdownFileRoute(relative)
		if existing, found := routes[route]; found {
			return fmt.Errorf("markdown files %s and %s map to the same route %q", existing, relative, route)
		}
		routes[route] = relative
		title, hasTitle := markdownTitle(string(data), route)
		pages = append(pages, sourcePage{
			SourcePath:      relative,
			Route:           route,
			Title:           title,
			Markdown:        string(data),
			HasTitleHeading: hasTitle,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].SourcePath < pages[j].SourcePath })
	return pages, nil
}

func hasHomePage(pages []sourcePage) bool {
	for _, page := range pages {
		if page.Route == "" {
			return true
		}
	}
	return false
}

func markdownFileRoute(filename string) string {
	clean := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(filename)), "/")
	clean = strings.TrimSuffix(clean, path.Ext(clean))
	if path.Base(clean) == "index" {
		clean = path.Dir(clean)
		if clean == "." {
			clean = ""
		}
	}
	return strings.Trim(clean, "/")
}

func markdownTitle(source, route string) (string, bool) {
	lines := strings.Split(strings.TrimPrefix(source, "\ufeff"), "\n")
	fence := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if fence == "" {
				fence = marker
			} else if marker == fence {
				fence = ""
			}
			continue
		}
		if fence == "" && strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title, true
			}
		}
	}
	if route == "" {
		return "Home", false
	}
	segment := path.Base(route)
	segment = strings.NewReplacer("-", " ", "_", " ").Replace(segment)
	words := strings.Fields(segment)
	for index, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " "), false
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

func registerWikiTarget(targets map[string]string, ambiguous map[string]bool, target, route string) {
	target = strings.Trim(target, "/")
	if target == "" && route != "" {
		return
	}
	if existing, found := targets[target]; found && existing != route {
		ambiguous[target] = true
		return
	}
	targets[target] = route
}

func validateWikiLinks(page sourcePage, targets map[string]string) error {
	for _, target := range md.Links(page.Markdown) {
		if _, found := targets[target]; !found {
			return fmt.Errorf("%s contains unresolved wiki link %q", page.SourcePath, target)
		}
	}
	return nil
}

func expandedPrefixes(route string) []string {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) <= 1 {
		return nil
	}
	expanded := make([]string, 0, len(parts)-1)
	for index := 1; index < len(parts); index++ {
		expanded = append(expanded, strings.Join(parts[:index], "/"))
	}
	return expanded
}

func renderSubpages(tree []navigation.Node, route, basePath string) string {
	children := tree
	if strings.Trim(route, "/") != "" {
		children = navigation.Children(tree, route)
	}
	if len(children) == 0 {
		return ""
	}
	var output strings.Builder
	output.WriteString(`<nav class="subpage-toc" aria-label="Pages in this section"><div class="subpage-toc-heading"><h2>Pages in this section</h2></div><ul class="subpage-toc-list subpage-toc-root">`)
	for _, child := range children {
		renderSubpageNode(&output, child, basePath)
	}
	output.WriteString(`</ul></nav>`)
	return output.String()
}

func renderSubpageNode(output *strings.Builder, node navigation.Node, basePath string) {
	output.WriteString(`<li class="subpage-toc-item">`)
	if node.Page {
		output.WriteString(`<a class="subpage-toc-link" href="`)
		output.WriteString(template.HTMLEscapeString(pageURL(basePath, node.Slug)))
		output.WriteString(`"><span>`)
		output.WriteString(template.HTMLEscapeString(node.Title))
		output.WriteString(`</span></a>`)
	} else {
		output.WriteString(`<span class="subpage-toc-label"><span>`)
		output.WriteString(template.HTMLEscapeString(node.Title))
		output.WriteString(`</span></span>`)
	}
	if len(node.Children) > 0 {
		output.WriteString(`<ul class="subpage-toc-list">`)
		for _, child := range node.Children {
			renderSubpageNode(output, child, basePath)
		}
		output.WriteString(`</ul>`)
	}
	output.WriteString(`</li>`)
}

func processRenderedHTML(
	rendered, sourcePath string,
	removeTitle bool,
	routesBySource map[string]string,
	basePath string,
) (string, string, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(rendered), contextNode)
	if err != nil {
		return "", "", err
	}
	if removeTitle {
		for index, node := range nodes {
			if node.Type == xhtml.ElementNode && node.Data == "h1" {
				nodes = append(nodes[:index], nodes[index+1:]...)
				break
			}
		}
	}
	for _, node := range nodes {
		if err := rewriteHTMLURLs(node, sourcePath, routesBySource, basePath); err != nil {
			return "", "", err
		}
	}
	var htmlOutput bytes.Buffer
	for _, node := range nodes {
		if err := xhtml.Render(&htmlOutput, node); err != nil {
			return "", "", err
		}
	}
	return htmlOutput.String(), normalizeSearchText(textFromNodes(nodes)), nil
}

func rewriteHTMLURLs(node *xhtml.Node, sourcePath string, routesBySource map[string]string, basePath string) error {
	if node.Type == xhtml.ElementNode {
		for index := range node.Attr {
			attribute := &node.Attr[index]
			if (node.Data == "a" && attribute.Key == "href") ||
				(node.Data == "img" && attribute.Key == "src") {
				rewritten, err := rewriteLocalURL(attribute.Val, sourcePath, routesBySource, basePath)
				if err != nil {
					return err
				}
				attribute.Val = rewritten
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := rewriteHTMLURLs(child, sourcePath, routesBySource, basePath); err != nil {
			return err
		}
	}
	return nil
}

func rewriteLocalURL(value, sourcePath string, routesBySource map[string]string, basePath string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(value, "//") || parsed.Path == "" {
		return value, nil
	}
	basePath = ensureBasePath(basePath)
	if basePath != "/" && strings.HasPrefix(parsed.Path, basePath) {
		return value, nil
	}

	trailingSlash := strings.HasSuffix(parsed.Path, "/")
	var resolved string
	if strings.HasPrefix(parsed.Path, "/") {
		resolved = strings.TrimPrefix(path.Clean(parsed.Path), "/")
	} else {
		resolved = path.Clean(path.Join(path.Dir(sourcePath), parsed.Path))
	}
	if resolved == "." {
		resolved = ""
	}
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("link %q escapes the documentation source", value)
	}

	if strings.EqualFold(path.Ext(resolved), ".md") {
		route, found := routesBySource[resolved]
		if !found {
			return "", fmt.Errorf("markdown link %q points to missing file %s", value, resolved)
		}
		parsed.Path = pageURL(basePath, route)
	} else {
		parsed.Path = basePath + strings.TrimPrefix(resolved, "/")
		if trailingSlash && !strings.HasSuffix(parsed.Path, "/") {
			parsed.Path += "/"
		}
	}
	return parsed.String(), nil
}

func textFromNodes(nodes []*xhtml.Node) string {
	var output strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			output.WriteString(node.Data)
			output.WriteByte(' ')
		}
		if node.Type == xhtml.ElementNode && (node.Data == "script" || node.Data == "style") {
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	return output.String()
}

func normalizeSearchText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func (b *Builder) copyBrowserAssets(outputDir string) error {
	for _, name := range staticBrowserAssets {
		data, err := fs.ReadFile(b.appFS, name)
		if err != nil {
			return fmt.Errorf("read browser asset %s: %w", name, err)
		}
		destination := filepath.Join(outputDir, "assets", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func copySourceAssets(sourceDir, outputDir string) error {
	return filepath.WalkDir(sourceDir, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filename == sourceDir {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil {
			return err
		}
		destination := filepath.Join(outputDir, relative)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		input, err := os.Open(filename)
		if err != nil {
			return err
		}
		output, err := os.Create(destination)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		return outputCloseErr
	})
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
	return os.WriteFile(filepath.Join(config.OutputDir, "sitemap.xml"), []byte(output.String()), 0o644)
}
