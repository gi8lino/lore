package store

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugin/wasm"
	"github.com/gi8lino/lore/internal/plugins/bundled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginInstallationSurvivesDatabaseAndRuntimeRestart(t *testing.T) {
	ctx := context.Background()
	dsn := integrationDatabase(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	database, err := Open(ctx, dsn, logger)
	require.NoError(t, err)
	data, err := bundled.Packages.ReadFile("callouts.loreplugin")
	require.NoError(t, err)
	runtime, err := wasm.New(ctx, wasm.Limits{})
	require.NoError(t, err)
	registry := &plugin.Registry{}
	manager := plugin.NewManager(registry, runtime, plugin.WithStore(database))
	_, err = manager.Install(ctx, data)
	require.NoError(t, err)
	require.NoError(t, manager.Disable(ctx, "io.lore.callouts"))
	require.NoError(t, manager.Close(ctx))
	database.Close()
	database, err = Open(ctx, dsn, logger)
	require.NoError(t, err)
	defer database.Close()
	runtime, err = wasm.New(ctx, wasm.Limits{})
	require.NoError(t, err)
	registry = &plugin.Registry{}
	manager = plugin.NewManager(registry, runtime, plugin.WithStore(database))
	defer func() { require.NoError(t, manager.Close(ctx)) }()
	require.NoError(t, manager.Bootstrap(ctx, nil))
	assert.False(t, manager.Plugins()[0].Enabled)
	renderer := markdown.NewWithRegistry(registry)
	require.NoError(t, manager.Enable(ctx, "io.lore.callouts"))
	html, err := renderer.Render("!!! note\nRestored")
	require.NoError(t, err)
	assert.Contains(t, html, `class="callout note"`)
	require.NoError(t, manager.Uninstall(ctx, "io.lore.callouts"))
	records, err := database.ListPlugins(ctx)
	require.NoError(t, err)
	assert.Empty(t, records)
}
