package site

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"slices"

	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/themes"
)

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
	LogoURL       string
	FaviconURL    string
	FaviconICOURL string
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

type buildPlan struct {
	config          Config
	basePath        string
	pages           []sourcePage
	routesBySource  map[string]string
	wikiTargets     map[string]string
	navigationPages []navigation.Page
	navigationTree  []navigation.Node
	themeData       template.JS
	templates       siteTemplates
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
	plan, err := b.planBuild(config)
	if err != nil {
		return Result{}, err
	}

	branding, err := b.prepareOutput(plan.config, plan.basePath)
	if err != nil {
		return Result{}, err
	}

	common := b.commonViewData(plan, branding)
	searchIndex, err := b.renderPages(ctx, plan, common)
	if err != nil {
		return Result{}, err
	}
	if err := b.writeSupportFiles(plan, common, searchIndex); err != nil {
		return Result{}, err
	}

	return Result{Pages: len(plan.pages), OutputDir: plan.config.OutputDir}, nil
}

func (b *Builder) planBuild(config Config) (buildPlan, error) {
	if err := config.validate(); err != nil {
		return buildPlan{}, err
	}

	themeData, err := loadThemeData(config.Theme)
	if err != nil {
		return buildPlan{}, err
	}

	pages, err := discoverPages(config.SourceDir)
	if err != nil {
		return buildPlan{}, err
	}
	if len(pages) == 0 {
		return buildPlan{}, fmt.Errorf("no Markdown files found in %s", config.SourceDir)
	}
	if !hasHomePage(pages) {
		return buildPlan{}, fmt.Errorf("%s must contain index.md for the site home page", config.SourceDir)
	}

	basePath, err := staticBasePath(config.SiteURL)
	if err != nil {
		return buildPlan{}, err
	}

	routesBySource, wikiTargets := indexPages(pages)
	navigationPages := buildNavigationPages(pages)
	templates, err := b.parseTemplates(basePath)
	if err != nil {
		return buildPlan{}, err
	}

	return buildPlan{
		config:          config,
		basePath:        basePath,
		pages:           pages,
		routesBySource:  routesBySource,
		wikiTargets:     wikiTargets,
		navigationPages: navigationPages,
		navigationTree:  navigation.Build(navigationPages, navigation.Options{}),
		themeData:       themeData,
		templates:       templates,
	}, nil
}

func loadThemeData(theme string) (template.JS, error) {
	availableThemes, err := themes.Load("")
	if err != nil {
		return "", err
	}
	if _, found := themes.Find(availableThemes, theme); !found {
		return "", fmt.Errorf("unknown theme %q", theme)
	}

	data, err := json.Marshal(availableThemes)
	if err != nil {
		return "", err
	}

	return template.JS(data), nil
}

func buildNavigationPages(pages []sourcePage) []navigation.Page {
	navigationPages := make([]navigation.Page, 0, len(pages))
	for _, page := range pages {
		if page.Route == "" {
			continue
		}
		navigationPages = append(navigationPages, navigation.Page{Slug: page.Route, Title: page.Title})
	}

	return navigationPages
}

func (b *Builder) commonViewData(plan buildPlan, branding brandingData) viewData {
	return viewData{
		LogoURL:       branding.LogoURL,
		FaviconURL:    branding.FaviconURL,
		FaviconICOURL: branding.FaviconICOURL,
		SiteName:      plan.config.SiteName,
		SiteURL:       plan.config.SiteURL,
		BasePath:      plan.basePath,
		Language:      plan.config.Language,
		ActiveTheme:   plan.config.Theme,
		ThemeData:     plan.themeData,
		Version:       b.version,
		Commit:        b.commit,
		RenderMermaid: plan.config.Mermaid,
	}
}

func (b *Builder) renderPages(ctx context.Context, plan buildPlan, common viewData) ([]searchEntry, error) {
	searchIndex := make([]searchEntry, 0, len(plan.pages))
	for index := range plan.pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		page := &plan.pages[index]
		if err := validateWikiLinks(*page, plan.wikiTargets); err != nil {
			return nil, err
		}

		rendered, err := b.renderPage(*page, plan)
		if err != nil {
			return nil, err
		}
		page.HTML = template.HTML(rendered.html)
		page.SearchText = rendered.searchText
		page.Contents = rendered.contents

		data := pageViewData(common, *page, plan.navigationPages)
		if err := writeTemplate(
			plan.templates.page,
			outputFilename(plan.config.OutputDir, page.Route),
			data,
		); err != nil {
			return nil, err
		}

		searchIndex = append(searchIndex, searchEntry{
			Title: page.Title,
			URL:   pageURL(plan.basePath, page.Route),
			Text:  page.SearchText,
		})
	}

	return searchIndex, nil
}

type renderedPage struct {
	html       string
	searchText string
	contents   []md.Heading
}

func (b *Builder) renderPage(page sourcePage, plan buildPlan) (renderedPage, error) {
	options := md.DefaultOptions()
	options.WikiLinkPrefix = plan.basePath
	resolveWiki := func(target string) string {
		normalized := md.Slug(target)
		if route, found := plan.wikiTargets[normalized]; found {
			return routeSuffix(route)
		}

		return routeSuffix(normalized)
	}

	rendered, err := b.renderer.RenderPageResolvedWithFunctions(
		page.Markdown,
		resolveWiki,
		options,
		md.Functions{Subpages: subpagesRenderer(plan.navigationTree, page.Route, plan.basePath)},
	)
	if err != nil {
		return renderedPage{}, fmt.Errorf("render %s: %w", page.SourcePath, err)
	}

	html, searchText, err := processRenderedHTML(
		rendered.HTML,
		page.SourcePath,
		page.HasTitleHeading,
		plan.routesBySource,
		plan.basePath,
	)
	if err != nil {
		return renderedPage{}, fmt.Errorf("rewrite %s: %w", page.SourcePath, err)
	}

	contents := rendered.Contents
	if page.HasTitleHeading && len(contents) > 0 && contents[0].Level == 1 {
		contents = contents[1:]
	}

	return renderedPage{html: html, searchText: searchText, contents: contents}, nil
}

func pageViewData(common viewData, page sourcePage, navigationPages []navigation.Page) viewData {
	data := common
	data.Title = page.Title
	data.CurrentRoute = page.Route
	data.Navigation = navigation.Build(navigationPages, navigation.Options{
		ActiveSlug: page.Route,
		Expanded:   expandedPrefixes(page.Route),
	})
	data.HTML = page.HTML
	data.PageContents = page.Contents

	return data
}

func (b *Builder) writeSupportFiles(plan buildPlan, common viewData, searchIndex []searchEntry) error {
	slices.SortFunc(searchIndex, compareSearchEntries)
	if err := writeJSON(outputFile(plan.config.OutputDir, "search-index.json"), searchIndex); err != nil {
		return err
	}

	if err := writeSearchPage(plan, common); err != nil {
		return err
	}
	if err := writeNotFoundPage(plan, common); err != nil {
		return err
	}

	return writeSitemap(plan.config, plan.pages)
}

func writeSearchPage(plan buildPlan, common viewData) error {
	data := common
	data.Title = "Search"
	data.Navigation = navigation.Build(plan.navigationPages, navigation.Options{})

	return writeTemplate(plan.templates.search, outputFile(plan.config.OutputDir, "search", "index.html"), data)
}

func writeNotFoundPage(plan buildPlan, common viewData) error {
	data := common
	data.Title = "Page not found"
	data.Navigation = navigation.Build(plan.navigationPages, navigation.Options{})

	return writeTemplate(plan.templates.notFound, outputFile(plan.config.OutputDir, "404.html"), data)
}
