package subpages

import (
	"testing"

	"github.com/gi8lino/lore/internal/navigation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("uses default title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse("{{subpages}}")

		require.True(t, ok)
		assert.Equal(t, "Pages in this section", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("uses custom title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title="Related pages"}}`)

		require.True(t, ok)
		assert.Equal(t, "Related pages", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("hides empty title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title=""}}`)

		require.True(t, ok)
		assert.Empty(t, options.Title)
		assert.False(t, options.ShowTitle)
	})

	t.Run("allows escaped title characters", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title="A \"quoted\" title"}}`)

		require.True(t, ok)
		assert.Equal(t, `A "quoted" title`, options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("allows surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse("  {{subpages title = \"Related pages\"}}\t")

		require.True(t, ok)
		assert.Equal(t, "Related pages", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("allows equals sign in title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title="A = B"}}`)

		require.True(t, ok)
		assert.Equal(t, "A = B", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("rejects unsupported options", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse("{{subpages depth=2}}")

		assert.False(t, ok)
	})

	t.Run("rejects additional options", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse(`{{subpages title="Related pages" depth="2"}}`)

		assert.False(t, ok)
	})

	t.Run("rejects malformed title", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse(`{{subpages title=Related}}`)

		assert.False(t, ok)
	})

	t.Run("rejects invalid quoted title", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse(`{{subpages title="bad\qescape"}}`)

		assert.False(t, ok)
	})
}

func TestNewRenderer(t *testing.T) {
	t.Parallel()

	children := []navigation.Node{
		{
			Title: "Guide",
			Slug:  "guide",
			Icon:  "book-open-lucide",
			Page:  true,
			Children: []navigation.Node{
				{Title: "Install", Slug: "guide/install", Page: true},
			},
		},
	}
	render := NewRenderer(children, func(slug string) string {
		return "/docs/" + slug + "/"
	})

	t.Run("uses invocation title and resolved URLs", func(t *testing.T) {
		t.Parallel()

		html, err := render(Options{Title: "Related pages", ShowTitle: true})

		require.NoError(t, err)
		assert.Contains(t, html, "Related pages")
		assert.Contains(t, html, `href="/docs/guide/"`)
		assert.Contains(t, html, `href="/docs/guide/install/"`)
		assert.Contains(t, html, `subpage-toc-node-icon`)
	})

	t.Run("hides invocation title", func(t *testing.T) {
		t.Parallel()

		html, err := render(Options{ShowTitle: false})

		require.NoError(t, err)
		assert.NotContains(t, html, "subpage-toc-heading")
		assert.Contains(t, html, `href="/docs/guide/"`)
	})

	t.Run("escapes page labels", func(t *testing.T) {
		t.Parallel()

		render := NewRenderer([]navigation.Node{{Title: `<script>alert(1)</script>`, Slug: "safe", Page: true}}, func(slug string) string {
			return "/" + slug + "/"
		})
		html, err := render(Options{Title: `<unsafe>`, ShowTitle: true})

		require.NoError(t, err)
		assert.NotContains(t, html, "<script>")
		assert.Contains(t, html, "&lt;unsafe&gt;")
		assert.Contains(t, html, "&lt;script&gt;alert(1)&lt;/script&gt;")
	})

	t.Run("returns empty output without children", func(t *testing.T) {
		t.Parallel()

		resolved := false
		render := NewRenderer(nil, func(string) string {
			resolved = true
			return "/unexpected/"
		})

		html, err := render(Options{Title: "Related pages", ShowTitle: true})

		require.NoError(t, err)
		assert.Empty(t, html)
		assert.False(t, resolved)
	})

	t.Run("renders folders without resolving folder URLs", func(t *testing.T) {
		t.Parallel()

		resolved := 0
		resolvedSlug := ""
		render := NewRenderer([]navigation.Node{
			{
				Title: "Platform",
				Slug:  "platform",
				Page:  false,
				Children: []navigation.Node{
					{Title: "Kubernetes", Slug: "platform/kubernetes", Page: true},
				},
			},
		}, func(slug string) string {
			resolved++
			resolvedSlug = slug
			return "/docs/" + slug + "/"
		})

		html, err := render(Options{ShowTitle: false})

		require.NoError(t, err)
		assert.Equal(t, 1, resolved)
		assert.Equal(t, "platform/kubernetes", resolvedSlug)
		assert.Contains(t, html, `class="subpage-toc-label"`)
		assert.Contains(t, html, "Platform")
		assert.Contains(t, html, `href="/docs/platform/kubernetes/"`)
		assert.NotContains(t, html, `href="/docs/platform/"`)
	})

	t.Run("sanitizes unsafe page URLs", func(t *testing.T) {
		t.Parallel()

		render := NewRenderer([]navigation.Node{{Title: "Unsafe", Slug: "unsafe", Page: true}}, func(string) string {
			return "javascript:alert(1)"
		})

		html, err := render(Options{ShowTitle: false})

		require.NoError(t, err)
		assert.NotContains(t, html, `href="javascript:`)
		assert.Contains(t, html, `href="#ZgotmplZ"`)
	})
}
