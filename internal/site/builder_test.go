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
	t.Run("home branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/site-logo.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/site-favicon.png"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})

	t.Run("search branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "search/index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/site-logo.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/site-favicon.png"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})

	t.Run("not found branding", func(t *testing.T) {
		html, err := os.ReadFile(filepath.Join(config.OutputDir, "404.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `src="/never/assets/site-logo.svg"`)
		assert.Contains(t, string(html), `href="/never/assets/site-favicon.png"`)
		assert.Contains(t, string(html), `href="/never/favicon.ico"`)
	})
	t.Run("logo copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets/site-logo.svg"))
		require.NoError(t, err)
		assert.Equal(t, "custom image", string(data))
	})

	t.Run("favicon copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets/site-favicon.png"))
		require.NoError(t, err)
		assert.Equal(t, "custom image", string(data))
	})

	t.Run("ICO fallback copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "favicon.ico"))
		require.NoError(t, err)
		assert.Equal(t, "custom image", string(data))
	})

	t.Run("extra asset copied", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(config.OutputDir, "assets/extra.txt"))
		require.NoError(t, err)
		assert.Equal(t, "custom image", string(data))
	})
	// Invalid branding must not erase the previously generated site.
	config.Logo = filepath.Join(root, "missing.svg")
	_, err = NewBuilder(web.Assets, "test", "test").Build(context.Background(), config)
	require.ErrorContains(t, err, "logo")
	_, err = os.Stat(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
}
