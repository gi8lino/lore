package wasm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/firstparty"
	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/pluginpackage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstalledBrowserPluginUsesSameRuntimeAndAssets(t *testing.T) {
	ctx := context.Background()
	archive, err := firstparty.Packages.ReadFile("mermaid.loreplugin")
	require.NoError(t, err)
	pkg, err := pluginpackage.Read(archive)
	require.NoError(t, err)
	denied, err := wasm.New(ctx, wasm.Limits{})
	require.NoError(t, err)
	_, err = denied.Load(ctx, pkg)
	require.ErrorContains(t, err, "browser:render")
	require.NoError(t, denied.Close(ctx))
	runtime, err := wasm.New(ctx, wasm.Limits{}, wasm.WithPermissions("browser:render"))
	require.NoError(t, err)
	registry := &plugin.Registry{}
	manager := plugin.NewManager(registry, runtime)
	defer func() { require.NoError(t, manager.Close(ctx)) }()
	_, err = manager.Install(ctx, archive)
	require.NoError(t, err)
	renderer := markdown.NewWithRegistry(registry)
	html, err := renderer.Render("```mermaid\ngraph LR; A --> B\n```")
	require.NoError(t, err)
	assert.Contains(t, html, `data-lore-plugin="io.lore.mermaid"`)
	modules := manager.BrowserModules()
	require.Len(t, modules, 1)
	old := modules[0]
	actual, err := manager.BrowserAsset(old.PluginID, old.Digest, "plugin.js")
	require.NoError(t, err)
	expected, err := pkg.Asset("plugin.js")
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
	replacement := changedManifest(t, archive, func(s string) string { return strings.ReplaceAll(s, "version: 1.0.0", "version: 1.1.0") })
	_, err = manager.Upgrade(ctx, old.PluginID, replacement)
	require.NoError(t, err)
	_, err = manager.BrowserAsset(old.PluginID, old.Digest, "plugin.js")
	require.Error(t, err)
	next := manager.BrowserModules()[0]
	assert.Equal(t, "1.1.0", next.Version)
	assert.NotEqual(t, old.Digest, next.Digest)
	_, err = manager.BrowserAsset(next.PluginID, next.Digest, "plugin.js")
	require.NoError(t, err)
	require.NoError(t, manager.Uninstall(ctx, next.PluginID))
	assert.Empty(t, manager.BrowserModules())
	_, err = manager.BrowserAsset(next.PluginID, next.Digest, "plugin.js")
	require.Error(t, err)
}

func TestTablesPackageOwnsSyntaxAndPresentation(t *testing.T) {
	ctx := context.Background()
	archive, err := firstparty.Packages.ReadFile("tables.loreplugin")
	require.NoError(t, err)
	runtime, err := wasm.New(ctx, wasm.Limits{}, wasm.WithPermissions("browser:render"))
	require.NoError(t, err)
	registry := &plugin.Registry{}
	manager := plugin.NewManager(registry, runtime)
	defer func() { require.NoError(t, manager.Close(ctx)) }()
	renderer := markdown.NewWithRegistry(registry)
	source := "| Service | Link |\n| --- | --- |\n| API | [[Runbook]] |\n\n{table header=blue sortable filterable}\n"
	_, err = manager.Install(ctx, archive)
	require.NoError(t, err)
	html, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, html, `data-lore-plugin="io.lore.tables"`)
	assert.Contains(t, html, `data-lore-fallback`)
	assert.Contains(t, html, `href="/pages/runbook"`)
	assert.Contains(t, html, `table-tone-blue`)
	assert.Contains(t, html, `lore-table-sortable`)
	assert.NotContains(t, html, "{table")
	require.Error(t, registry.Snapshot().ValidateFeatures(map[string]bool{"io.lore.tables.tables": false}))
	require.NoError(t, manager.Disable(ctx, "io.lore.tables"))
	html, err = renderer.Render(source)
	require.NoError(t, err)
	assert.NotContains(t, html, "<table")
	assert.Empty(t, manager.BrowserModules())
	require.NoError(t, manager.Enable(ctx, "io.lore.tables"))
	html, err = renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, html, "<table")
	require.NoError(t, manager.Uninstall(ctx, "io.lore.tables"))
	html, err = renderer.Render(source)
	require.NoError(t, err)
	assert.NotContains(t, html, "<table")
}
