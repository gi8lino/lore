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
