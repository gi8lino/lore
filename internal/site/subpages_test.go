package site

import (
	"testing"

	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubpagesRenderer(t *testing.T) {
	t.Parallel()

	tree := []navigation.Node{
		{
			Title: "Guide",
			Slug:  "guide",
			Page:  true,
			Children: []navigation.Node{
				{Title: "Install", Slug: "guide/install", Page: true},
			},
		},
	}

	t.Run("uses default title", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "", "/docs/")
		html, err := render(md.SubpagesOptions{Title: "Pages in this section", ShowTitle: true})

		require.NoError(t, err)
		assert.Contains(t, html, `<h2>Pages in this section</h2>`)
		assert.Contains(t, html, `href="/docs/guide/"`)
	})

	t.Run("uses custom title", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "", "/docs/")
		html, err := render(md.SubpagesOptions{Title: "Related pages", ShowTitle: true})

		require.NoError(t, err)
		assert.Contains(t, html, `<h2>Related pages</h2>`)
		assert.Contains(t, html, `aria-label="Related pages"`)
		assert.NotContains(t, html, `<h2>Pages in this section</h2>`)
	})

	t.Run("hides empty title", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "", "/docs/")
		html, err := render(md.SubpagesOptions{ShowTitle: false})

		require.NoError(t, err)
		assert.NotContains(t, html, `subpage-toc-heading`)
		assert.Contains(t, html, `aria-label="Pages in this section"`)
		assert.Contains(t, html, `href="/docs/guide/"`)
	})

	t.Run("renders current route children", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "guide", "/docs/")
		html, err := render(md.SubpagesOptions{Title: "Pages in this section", ShowTitle: true})

		require.NoError(t, err)
		assert.Contains(t, html, `href="/docs/guide/install/"`)
		assert.NotContains(t, html, `href="/docs/guide/"`)
	})

	t.Run("returns empty HTML without children", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "guide/install", "/docs/")
		html, err := render(md.SubpagesOptions{Title: "Pages in this section", ShowTitle: true})

		require.NoError(t, err)
		assert.Empty(t, html)
	})

	t.Run("escapes custom title", func(t *testing.T) {
		t.Parallel()

		render := subpagesRenderer(tree, "", "/docs/")
		html, err := render(md.SubpagesOptions{Title: `<script>alert("x")</script>`, ShowTitle: true})

		require.NoError(t, err)
		assert.NotContains(t, html, `<script>`)
		assert.Contains(t, html, `&lt;script&gt;`)
	})
}
