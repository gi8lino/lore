package site

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gi8lino/lore/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticBrowserAssetsIncludeModuleDependencies(t *testing.T) {
	t.Parallel()

	output := t.TempDir()
	require.NoError(t, NewBuilder(web.Assets, "test", "test").copyBrowserAssets(output))
	exported := os.DirFS(filepath.Join(output, "assets"))
	// Inspect the actual emitted modules, including side-effect and dynamic imports.
	imports := regexp.MustCompile(`(?:\bfrom\s*|\bimport\s*(?:\(\s*)?)["']([^"']+)["']`)
	for _, name := range staticBrowserAssets {
		if !strings.HasSuffix(name, ".js") {
			continue
		}
		data, err := fs.ReadFile(exported, name)
		require.NoError(t, err)
		for _, match := range imports.FindAllSubmatch(data, -1) {
			dependency := string(match[1])
			if !strings.HasPrefix(dependency, ".") {
				continue
			}
			_, err := fs.Stat(exported, path.Join(path.Dir(name), dependency))
			assert.NoError(t, err, "%s imports missing asset %s", name, dependency)
		}
	}
}

func TestMarkdownFileRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "root index", file: "index.md", want: ""},
		{name: "page", file: "getting-started.md", want: "getting-started"},
		{name: "section index", file: "installation/index.md", want: "installation"},
		{name: "nested page", file: "installation/docker.md", want: "installation/docker"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, markdownFileRoute(test.file))
		})
	}
}

func TestStaticBasePath(t *testing.T) {
	t.Parallel()

	root, err := staticBasePath("")

	require.NoError(t, err)
	assert.Equal(t, "/", root)

	project, err := staticBasePath("https://gi8lino.github.io/lore/")

	require.NoError(t, err)
	assert.Equal(t, "/lore/", project)
}

func TestRewriteLocalURL(t *testing.T) {
	t.Parallel()

	routes := map[string]string{
		"guide/other.md": "guide/other",
	}

	page, err := rewriteLocalURL("other.md#section", "guide/page.md", routes, "/lore/")

	require.NoError(t, err)
	assert.Equal(t, "/lore/guide/other/#section", page)

	asset, err := rewriteLocalURL("images/example.png", "guide/page.md", routes, "/lore/")

	require.NoError(t, err)
	assert.Equal(t, "/lore/guide/images/example.png", asset)

	_, err = rewriteLocalURL("missing.md", "guide/page.md", routes, "/lore/")

	require.Error(t, err)
}

func TestMarkdownTitle(t *testing.T) {
	t.Parallel()

	title, found := markdownTitle("Intro\n\n# Static sites\n", "static-sites")

	assert.True(t, found)
	assert.Equal(t, "Static sites", title)

	title, found = markdownTitle("No title\n", "getting-started")

	assert.False(t, found)
	assert.Equal(t, "Getting Started", title)
}

func TestDiscoverPagesRejectsDuplicateRoutes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "guide"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "index.md"), []byte("# Home\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "guide.md"), []byte("# Guide\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "guide", "index.md"), []byte("# Guide index\n"), 0o644))

	_, err := discoverPages(root)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "map to the same route")
}

func TestHasHomePage(t *testing.T) {
	t.Parallel()

	assert.True(t, hasHomePage([]sourcePage{{Route: ""}, {Route: "guide"}}))
	assert.False(t, hasHomePage([]sourcePage{{Route: "guide"}}))
}

func TestBuilderBuildsReadOnlyStaticSite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "docs")
	output := filepath.Join(root, "site")

	require.NoError(t, os.MkdirAll(filepath.Join(source, "images"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(source, "index.md"),
		[]byte("# Home\n\n[Guide](guide.md)\n\n[[Guide]]\n\n![Logo](images/logo.png)\n\n{{subpages}}\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(source, "guide.md"),
		[]byte("# Guide\n\n## Details\n\nStatic documentation.\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(filepath.Join(source, "images", "logo.png"), []byte("png"), 0o644))

	assets := fstest.MapFS{}

	for _, name := range staticBrowserAssets {
		assets[name] = &fstest.MapFile{Data: []byte("asset")}
	}

	assets["lore.svg"] = &fstest.MapFile{Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)}

	config := DefaultConfig()
	config.SiteURL = "https://example.com/docs/"
	config.SourceDir = source
	config.OutputDir = output

	result, err := NewBuilder(assets, "test", "deadbeef").Build(context.Background(), config)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Pages)

	home, err := os.ReadFile(filepath.Join(output, "index.html"))

	require.NoError(t, err)
	assert.Contains(t, string(home), `href="/docs/guide/"`)
	assert.Contains(t, string(home), `src="/docs/images/logo.png"`)
	assert.Contains(t, string(home), "Read-only static site")
	assert.NotContains(t, string(home), "/edit/")
	assert.NotContains(t, string(home), "/auth/")

	guide, err := os.ReadFile(filepath.Join(output, "guide", "index.html"))

	require.NoError(t, err)
	assert.Contains(t, string(guide), "Static documentation.")
	assert.NotContains(t, string(guide), ">Guide</h1></div><h1")

	for _, filename := range []string{
		"404.html",
		"search/index.html",
		"search-index.json",
		".nojekyll",
		"sitemap.xml",
		"images/logo.png",
		"assets/js/static.js",
	} {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash(filename)))
		require.NoError(t, err, filename)
	}
}

func TestValidateWikiLinksRejectsMissingTarget(t *testing.T) {
	t.Parallel()

	page := sourcePage{SourcePath: "index.md", Markdown: "See [[Missing page]]."}
	err := validateWikiLinks(page, map[string]string{"existing": "existing"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unresolved wiki link "missing-page"`)
}

func TestBuildConfiguredBranding(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	config := DefaultConfig()
	config.SourceDir = filepath.Join(root, "content")
	config.OutputDir = filepath.Join(root, "site")
	config.AssetsDir = filepath.Join(root, "images")
	config.SiteURL = "https://example.com/never/"
	require.NoError(t, os.MkdirAll(config.SourceDir, 0o755))
	require.NoError(t, os.MkdirAll(config.AssetsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(config.SourceDir, "index.md"), []byte("# Home"), 0o644))
	config.Logo = filepath.Join(config.AssetsDir, "logo.svg")
	config.Favicon = filepath.Join(config.AssetsDir, "icon.png")
	config.FaviconICO = filepath.Join(config.AssetsDir, "fallback.ico")
	for _, name := range []string{config.Logo, config.Favicon, config.FaviconICO, filepath.Join(config.AssetsDir, "extra.txt")} {
		require.NoError(t, os.WriteFile(name, []byte("custom image"), 0o644))
	}
	_, err := NewBuilder(web.Assets, "test", "test").Build(context.Background(), config)
	require.NoError(t, err)
	for _, route := range []string{"index.html", "search/index.html", "404.html"} {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, route))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/site-logo.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/site-favicon.png"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	}
	for _, name := range []string{"assets/site-logo.svg", "assets/site-favicon.png", "favicon.ico", "assets/extra.txt"} {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, name))
		require.NoError(t, err)
		assert.Equal(t, "custom image", string(data))
	}
	// Invalid branding must not erase the previously generated site.
	config.Logo = filepath.Join(root, "missing.svg")
	_, err = NewBuilder(web.Assets, "test", "test").Build(context.Background(), config)
	require.ErrorContains(t, err, "logo")
	_, err = os.Stat(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
}
