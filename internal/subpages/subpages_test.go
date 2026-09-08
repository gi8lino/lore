package subpages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("uses default title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse("{{subpages}}")

		assert.True(t, ok)
		assert.Equal(t, "Pages in this section", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("uses custom title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title="Related pages"}}`)

		assert.True(t, ok)
		assert.Equal(t, "Related pages", options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("hides empty title", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title=""}}`)

		assert.True(t, ok)
		assert.Empty(t, options.Title)
		assert.False(t, options.ShowTitle)
	})

	t.Run("allows escaped title characters", func(t *testing.T) {
		t.Parallel()

		options, ok := Parse(`{{subpages title="A \"quoted\" title"}}`)

		assert.True(t, ok)
		assert.Equal(t, `A "quoted" title`, options.Title)
		assert.True(t, options.ShowTitle)
	})

	t.Run("rejects unsupported options", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse("{{subpages depth=2}}")

		assert.False(t, ok)
	})

	t.Run("rejects malformed title", func(t *testing.T) {
		t.Parallel()

		_, ok := Parse(`{{subpages title=Related}}`)

		assert.False(t, ok)
	})
}
