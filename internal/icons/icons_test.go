package icons

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)

	for _, option := range Options() {
		assert.NotEmpty(t, option.Name)
		assert.Falsef(t, seen[option.Name], "duplicate icon %q", option.Name)
		assert.Truef(t, IsIcon(option.Name), "icon %q is not accepted", option.Name)

		seen[option.Name] = true
	}
}

func TestIsIcon(t *testing.T) {
	t.Parallel()

	t.Run("allows empty value", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsIcon(""))
	})

	t.Run("accepts Lucide icon", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsIcon("search-lucide"))
	})

	t.Run("accepts Simple Icon", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsIcon("github-simple"))
	})

	t.Run("rejects unsuffixed icon", func(t *testing.T) {
		t.Parallel()

		assert.False(t, IsIcon("search"))
	})

	t.Run("rejects unknown icon", func(t *testing.T) {
		t.Parallel()

		assert.False(t, IsIcon("not-a-real-icon-lucide"))
	})
}

func TestSVG(t *testing.T) {
	t.Parallel()

	t.Run("renders Lucide icon", func(t *testing.T) {
		t.Parallel()

		svg := string(SVG("search-lucide", 18))

		assert.Contains(t, svg, `class="lucide-icon"`)
		assert.Contains(t, svg, `width="18"`)
		assert.Contains(t, svg, `height="18"`)
	})

	t.Run("renders Simple Icon", func(t *testing.T) {
		t.Parallel()

		svg := string(SVG("github-simple", 18))

		assert.Contains(t, svg, `class="simple-icon"`)
		assert.Contains(t, svg, `width="18"`)
		assert.Contains(t, svg, `height="18"`)
		assert.Contains(t, svg, `fill="currentColor"`)
	})

	t.Run("rejects unsuffixed icon", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, SVG("search", 18))
	})
}

func TestIconSources(t *testing.T) {
	t.Parallel()

	t.Run("Lucide", func(t *testing.T) {
		t.Parallel()

		options := Search("search-lucide", 20)
		require.NotEmpty(t, options)

		option := findOption(options, "search-lucide")
		require.NotNil(t, option)
		assert.Equal(t, "Lucide", option.Source)
	})

	t.Run("Simple Icons", func(t *testing.T) {
		t.Parallel()

		options := Search("github-simple", 20)
		require.NotEmpty(t, options)

		option := findOption(options, "github-simple")
		require.NotNil(t, option)
		assert.Equal(t, "Simple Icons", option.Source)
	})
}

func TestSimpleLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "github", simpleLabel("github"))
}

func TestSearchCatalog(t *testing.T) {
	t.Parallel()

	t.Run("finds Lucide icon", func(t *testing.T) {
		t.Parallel()

		options := Search("database", 20)

		require.NotEmpty(t, options)
		assert.True(t, containsOption(options, "database-lucide"))
	})

	t.Run("finds Simple Icon", func(t *testing.T) {
		t.Parallel()

		options := Search("github", 20)

		require.NotEmpty(t, options)
		assert.True(t, containsOption(options, "github-simple"))
	})

	t.Run("matches source suffix", func(t *testing.T) {
		t.Parallel()

		options := Search("-simple", 10)

		require.NotEmpty(t, options)
		for _, option := range options {
			assert.True(t, strings.HasSuffix(option.Name, simpleSuffix))
		}
	})
}

func TestSearchCatalogPagination(t *testing.T) {
	t.Parallel()

	first, hasMore := SearchPage("", 0, 3)

	require.Len(t, first, 3)
	assert.True(t, hasMore)

	second, _ := SearchPage("", len(first), 3)

	require.Len(t, second, 3)
	assert.NotEqual(t, first[len(first)-1].Name, second[0].Name)

	all := Options()
	last, hasMore := SearchPage("", len(all)-1, 3)

	assert.Len(t, last, 1)
	assert.False(t, hasMore)
}

func containsOption(options []Option, name string) bool {
	return findOption(options, name) != nil
}

func findOption(options []Option, name string) *Option {
	for index := range options {
		if options[index].Name == name {
			return &options[index]
		}
	}

	return nil
}
