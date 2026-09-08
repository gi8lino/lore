package site

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticHTMLPreservesImageWidthsWhenRewritingURLs(t *testing.T) {
	t.Parallel()

	for _, width := range []string{"640px", "50%"} {
		t.Run(width, func(t *testing.T) {
			t.Parallel()
			rendered, err := markdown.New().Render("![Diagram](images/diagram.png){width=" + width + "}")
			require.NoError(t, err)

			got, searchText, err := processRenderedHTML(rendered, "guide/page.md", false, nil, "/lore/")

			require.NoError(t, err)
			assert.Contains(t, got, `src="/lore/guide/images/diagram.png"`)
			assert.Contains(t, strings.ReplaceAll(got, " ", ""), `style="width:`+width+`"`)
			assert.NotContains(t, got, "{width=")
			assert.NotContains(t, searchText, "{width=")
		})
	}
}
