package site

import (
	"context"
	"fmt"
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

func TestStaticBrowserAssetsExcludeBranding(t *testing.T) {
	t.Parallel()

	assert.NotContains(t, staticBrowserAssets, "favicon.svg")
	assert.NotContains(t, staticBrowserAssets, "lore-mark.svg")
	assert.NotContains(t, staticBrowserAssets, "lore.svg")
}

func TestStaticBrowserAssetsIncludeModuleDependencies(t *testing.T) {
	t.Parallel()

	output := t.TempDir()
	require.NoError(t, newBuilder(web.Assets).copyBrowserAssets(output))
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

	t.Run("root index", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", markdownFileRoute("index.md"))
	})

	t.Run("page", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "getting-started", markdownFileRoute("getting-started.md"))
	})

	t.Run("section index", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "installation", markdownFileRoute("installation/index.md"))
	})

	t.Run("nested page", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "installation/docker", markdownFileRoute("installation/docker.md"))
	})
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
		[]byte("# Home\n\n[Guide](guide.md)\n\n[[Guide]]\n\n![Logo](images/logo.png)\n\n{{subpages title=\"Related pages\"}}\n"),
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

	config := defaultConfig()
	config.SiteURL = "https://example.com/docs/"
	config.SourceDir = source
	config.OutputDir = output

	result, err := newBuilder(assets).build(context.Background(), config)

	require.NoError(t, err)
	assert.Equal(t, 2, result.pages)

	home, err := os.ReadFile(filepath.Join(output, "index.html"))

	require.NoError(t, err)
	assert.Contains(t, string(home), `href="/docs/guide/"`)
	assert.Contains(t, string(home), `src="/docs/images/logo.png"`)
	assert.Contains(t, string(home), "Read-only static site")
	assert.NotContains(t, string(home), "/edit/")
	assert.NotContains(t, string(home), "/auth/")
	assert.Contains(t, string(home), "Related pages")
	assert.Contains(t, string(home), ">Documentation</span>")
	assert.NotContains(t, string(home), `<link rel="icon"`)
	assert.NotContains(t, string(home), "lore.svg")
	assert.NotContains(t, string(home), "lore-mark.svg")

	guide, err := os.ReadFile(filepath.Join(output, "guide", "index.html"))

	require.NoError(t, err)
	assert.Contains(t, string(guide), "Static documentation.")
	assert.NotContains(t, string(guide), ">Guide</h1></div><h1")

	t.Run("404.html", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("404.html")))
		assert.NoError(t, err)
	})

	t.Run("search/index.html", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("search/index.html")))
		assert.NoError(t, err)
	})

	t.Run("search-index.json", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("search-index.json")))
		assert.NoError(t, err)
	})

	t.Run(".nojekyll", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash(".nojekyll")))
		assert.NoError(t, err)
	})

	t.Run("sitemap.xml", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("sitemap.xml")))
		assert.NoError(t, err)
	})

	t.Run("images/logo.png", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("images/logo.png")))
		assert.NoError(t, err)
	})

	t.Run("does not publish bundled branding", func(t *testing.T) {
		for _, name := range []string{"favicon.svg", "lore-mark.svg", "lore.svg"} {
			_, err := os.Stat(filepath.Join(output, "assets", name))
			assert.ErrorIs(t, err, os.ErrNotExist)
		}
	})

	t.Run("assets/js/static.js", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(output, filepath.FromSlash("assets/js/static.js")))
		assert.NoError(t, err)
	})
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
	contentDir := filepath.Join(root, "content")
	assetsDir := filepath.Join(root, "assets")
	outputDir := filepath.Join(root, "site")
	require.NoError(t, os.MkdirAll(filepath.Join(contentDir, "assets"), 0o755))
	require.NoError(t, os.MkdirAll(assetsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(contentDir, "index.md"), []byte("# Home"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "never.svg"), []byte("custom logo"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(contentDir, "assets", "favicon.svg"), []byte("custom favicon"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(contentDir, "favicon.ico"), []byte("custom ico"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "extra.txt"), []byte("custom asset"), 0o644))

	configFile := filepath.Join(root, "lore-site.toml")
	configSource := fmt.Sprintf(`site_url = "https://example.com/never/"
source_dir = %q
output_dir = %q
logo = "assets/never.svg"
favicon = "content/assets/favicon.svg"
favicon_ico = "content/favicon.ico"
assets_dir = "assets"
`, contentDir, outputDir)
	require.NoError(t, os.WriteFile(configFile, []byte(configSource), 0o644))

	config, err := loadConfig(configFile, true)
	require.NoError(t, err)
	_, err = newBuilder(web.Assets).build(context.Background(), config)
	require.NoError(t, err)

	t.Run("home branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/never.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/favicon.svg"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})

	t.Run("search branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "search", "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/never.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/favicon.svg"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})

	t.Run("not found branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "404.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/never.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/favicon.svg"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})

	t.Run("logo copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets", "never.svg"))
		require.NoError(t, err)
		assert.Equal(t, "custom logo", string(data))
	})

	t.Run("favicon copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets", "favicon.svg"))
		require.NoError(t, err)
		assert.Equal(t, "custom favicon", string(data))
	})

	t.Run("ICO fallback copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "favicon.ico"))
		require.NoError(t, err)
		assert.Equal(t, "custom ico", string(data))
	})

	t.Run("assets directory copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets", "extra.txt"))
		require.NoError(t, err)
		assert.Equal(t, "custom asset", string(data))
	})

	// Invalid branding must not erase the previously generated site.
	config.Logo = filepath.Join(root, "missing.svg")
	_, err = newBuilder(web.Assets).build(context.Background(), config)
	require.ErrorContains(t, err, "logo")
	_, err = os.Stat(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
}
