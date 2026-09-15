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
	renderer, manager := testPluginMarkdownRenderer(t, "callouts")
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
	require.NoError(t, manager.Disable(ctx, "io.lore.callouts"))
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	html, err = os.ReadFile(filepath.Join(output, "index.html"))
	require.NoError(t, err)
	assert.NotContains(t, string(html), `class="callout note"`)
	require.NoError(t, manager.Enable(ctx, "io.lore.callouts"))
}

func TestStaticBrowserPackagesFollowLiveRegistry(t *testing.T) {
	ctx := context.Background()
	renderer, manager := testPluginMarkdownRenderer(t, "mermaid", "tables")
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
	html, err := os.ReadFile(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(html), `id="lore-plugin-modules"`)
	assert.Contains(t, string(html), "/docs/plugins/io.lore.mermaid/")
	module := manager.BrowserModules()[0]
	frame, err := os.ReadFile(filepath.Join(config.OutputDir, "plugins", module.PluginID, module.Digest, "frames", "diagrams.html"))
	require.NoError(t, err)
	assert.Contains(t, string(frame), "/docs/assets/js/plugins/frame.js")
	assert.Contains(t, string(frame), "https://example.com/docs/plugins/")
	assert.NotContains(t, string(html), "modules.json")
	require.NoError(t, manager.Disable(ctx, module.PluginID))
	require.NoError(t, BuildWithRenderer(ctx, assets, config, renderer))
	html, err = os.ReadFile(filepath.Join(config.OutputDir, "index.html"))
	require.NoError(t, err)
	assert.NotContains(t, string(html), "io.lore.mermaid")
	assert.Contains(t, string(html), "io.lore.tables")
	_, err = os.Stat(filepath.Join(config.OutputDir, "plugins", module.PluginID))
	assert.True(t, os.IsNotExist(err))
	require.NoError(t, manager.Enable(ctx, module.PluginID))
}
