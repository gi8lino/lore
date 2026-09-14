package markdown

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMermaidUsesRuntimeRegistryAndCentralSanitizer(t *testing.T) {
	ctx := context.Background()
	r := testRenderer(t, "mermaid")
	source := "```mermaid\ngraph LR; A --> B\n```"
	rendered, err := r.Render(source)
	require.NoError(t, err)
	assert.Contains(t, rendered, `data-lore-plugin="io.lore.mermaid"`)
	assert.Contains(t, rendered, `data-lore-module="diagrams"`)
	assert.Contains(t, rendered, "A --&gt; B")
	assert.NotContains(t, rendered, "<iframe")
	assert.NotContains(t, rendered, "<script")
	options := DefaultOptions()
	options.Mermaid = false
	rendered, err = r.RenderResolvedWithOptions(source, Slug, options)
	require.NoError(t, err)
	assert.NotContains(t, rendered, "data-lore-plugin")
	require.NoError(t, r.PluginManager().Disable(ctx, "io.lore.mermaid"))
	rendered, err = r.Render(source)
	require.NoError(t, err)
	assert.NotContains(t, rendered, "data-lore-plugin")
	require.NoError(t, r.PluginManager().Enable(ctx, "io.lore.mermaid"))
	rendered, err = r.Render(source)
	require.NoError(t, err)
	assert.Contains(t, rendered, "data-lore-plugin")
	rendered, err = r.Render("````\n" + source + "\n````")
	require.NoError(t, err)
	assert.NotContains(t, rendered, "data-lore-plugin")
}
