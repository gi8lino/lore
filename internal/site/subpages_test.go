package site

import (
	"testing"

	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/subpages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticSubpagesUseStaticPageURLs(t *testing.T) {
	t.Parallel()

	tree := navigation.Build([]navigation.Page{
		{Slug: "guide", Title: "Guide"},
		{Slug: "guide/install", Title: "Install"},
	}, navigation.Options{})
	render := subpages.NewRenderer(navigation.Children(tree, "guide"), func(slug string) string {
		return pageURL("/docs/", slug)
	})

	html, err := render(subpages.Options{Title: "Related pages", ShowTitle: true})

	require.NoError(t, err)
	assert.Contains(t, html, "Related pages")
	assert.Contains(t, html, `href="/docs/guide/install/"`)
	assert.NotContains(t, html, `href="/docs/guide/"`)
}
