package main

import (
	"html/template"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse verifies parse behavior.
func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("uses default title", func(t *testing.T) {
		t.Parallel()

		options, ok := parse("{{subpages}}")

		require.True(t, ok)
		assert.Equal(t, "Pages in this section", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("uses custom title", func(t *testing.T) {
		t.Parallel()

		options, ok := parse(`{{subpages title="Related pages"}}`)

		require.True(t, ok)
		assert.Equal(t, "Related pages", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("hides empty title", func(t *testing.T) {
		t.Parallel()

		options, ok := parse(`{{subpages title=""}}`)

		require.True(t, ok)
		assert.Empty(t, options.Title)
		assert.False(t, options.ShowTitle)
	})

	t.Run("allows escaped title characters", func(t *testing.T) {
		t.Parallel()

		options, ok := parse(`{{subpages title="A \"quoted\" title"}}`)

		require.True(t, ok)
		assert.Equal(t, `A "quoted" title`, options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("allows surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		options, ok := parse("  {{subpages title = \"Related pages\"}}\t")

		require.True(t, ok)
		assert.Equal(t, "Related pages", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("allows equals sign in title", func(t *testing.T) {
		t.Parallel()

		options, ok := parse(`{{subpages title="A = B"}}`)

		require.True(t, ok)
		assert.Equal(t, "A = B", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("rejects unsupported options", func(t *testing.T) {
		t.Parallel()

		_, ok := parse("{{subpages depth=2}}")

		assert.False(t, ok)
	})

	t.Run("rejects additional options", func(t *testing.T) {
		t.Parallel()

		_, ok := parse(`{{subpages title="Related pages" depth="2"}}`)

		assert.False(t, ok)
	})

	t.Run("rejects malformed title", func(t *testing.T) {
		t.Parallel()

		_, ok := parse(`{{subpages title=Related}}`)

		assert.False(t, ok)
	})

	t.Run("rejects invalid quoted title", func(t *testing.T) {
		t.Parallel()

		_, ok := parse(`{{subpages title="bad\qescape"}}`)

		assert.False(t, ok)
	})
}

// TestNewRenderer verifies new renderer behavior.
func TestNewRenderer(t *testing.T) {
	icon := func(string, int) template.HTML { return "" }
	nodes := []pluginapi.NavigationNode{{Title: "Guide", URL: "/docs/guide/", Icon: "book-open-lucide", Page: true, Children: []pluginapi.NavigationNode{{Title: "Install", URL: "/docs/guide/install/", Page: true}}}}
	render := newRenderer(nodes, icon)
	html, err := render(macroOptions{Title: "Related pages", ShowTitle: true})
	require.NoError(t, err)
	assert.Contains(t, html, "Related pages")
	assert.Contains(t, html, `href="/docs/guide/install/"`)
	assert.Contains(t, html, "subpage-toc-node-icon")
	html, err = render(macroOptions{ShowTitle: false})
	require.NoError(t, err)
	assert.NotContains(t, html, "subpage-toc-heading")
	html, err = newRenderer([]pluginapi.NavigationNode{{Title: `<script>bad()</script>`, URL: "javascript:bad()", Page: true}}, icon)(macroOptions{Title: "<unsafe>", ShowTitle: true})
	require.NoError(t, err)
	assert.NotContains(t, html, "<script>")
	assert.Contains(t, html, "&lt;unsafe&gt;")
	assert.Contains(t, html, `href="#ZgotmplZ"`)
	html, err = newRenderer(nil, func(string, int) template.HTML { t.Fatal("unexpected icon call"); return "" })(macroOptions{ShowTitle: true})
	require.NoError(t, err)
	assert.Empty(t, html)
	html, err = newRenderer([]pluginapi.NavigationNode{{Title: "Folder", Children: nodes}}, icon)(macroOptions{})
	require.NoError(t, err)
	assert.Contains(t, html, `class="subpage-toc-label"`)
	assert.Contains(t, html, "Folder")
	assert.Contains(t, html, `href="/docs/guide/"`)
}
