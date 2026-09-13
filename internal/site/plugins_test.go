package site

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticBuildUsesLiveRendererRegistry(t *testing.T) {
	ctx := context.Background()
	renderer := testMarkdownRenderer(t)
	root := t.TempDir()
	source := filepath.Join(root, "docs")
	output := filepath.Join(root, "site")
	require.NoError(t, os.MkdirAll(source, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "index.md"), []byte("# Home\n\n!!! note\nDynamic plugin\n"), 0644))
	assets := fstest.MapFS{}
	for _, name := range staticBrowserAssets {
		assets[name] = &fstest.MapFile{Data: []byte("asset")}
	}
	config := defaultConfig()
	config.SourceDir = source
	config.OutputDir = output
	config.SiteURL = "https://example.com/"
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	html, err := os.ReadFile(filepath.Join(output, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(html), `class="callout note"`)
	require.NoError(t, renderer.PluginManager().Disable(ctx, "io.lore.callouts"))
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	html, err = os.ReadFile(filepath.Join(output, "index.html"))
	require.NoError(t, err)
	assert.NotContains(t, string(html), `class="callout note"`)
	require.NoError(t, renderer.PluginManager().Enable(ctx, "io.lore.callouts"))
}

func TestStaticBrowserPackagesFollowLiveRegistry(t *testing.T) {
	ctx := context.Background()
	renderer := testMarkdownRenderer(t)
	root := t.TempDir()
	source := filepath.Join(root, "docs")
	require.NoError(t, os.MkdirAll(source, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "index.md"), []byte("```mermaid\ngraph LR; A --> B\n```"), 0644))
	assets := fstest.MapFS{}
	for _, name := range staticBrowserAssets {
		assets[name] = &fstest.MapFile{Data: []byte("asset")}
	}
	config := defaultConfig()
	config.SourceDir = source
	config.OutputDir = filepath.Join(root, "site")
	config.SiteURL = "https://example.com/docs/"
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	catalog, err := os.ReadFile(filepath.Join(config.OutputDir, "plugins", "modules.json"))
	require.NoError(t, err)
	assert.Contains(t, string(catalog), "/docs/plugins/io.lore.mermaid/")
	module := renderer.PluginManager().BrowserModules()[0]
	frame, err := os.ReadFile(filepath.Join(config.OutputDir, "plugins", module.PluginID, module.Digest, "frames", "diagrams.html"))
	require.NoError(t, err)
	assert.Contains(t, string(frame), "/docs/assets/js/plugins/frame.js")
	assert.Contains(t, string(frame), "https://example.com/docs/plugins/")
	html, err := os.ReadFile(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(html), `data-plugin-modules="/docs/plugins/modules.json"`)
	assert.Contains(t, string(html), `data-plugin-live="false"`)
	require.NoError(t, renderer.PluginManager().Disable(ctx, module.PluginID))
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	catalog, err = os.ReadFile(filepath.Join(config.OutputDir, "plugins", "modules.json"))
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(catalog))
	_, err = os.Stat(filepath.Join(config.OutputDir, "plugins", module.PluginID))
	assert.True(t, os.IsNotExist(err))
	require.NoError(t, renderer.PluginManager().Enable(ctx, module.PluginID))
}
