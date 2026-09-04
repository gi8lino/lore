package icons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigationOptionsAreUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, option := range NavigationOptions() {
		assert.NotEmpty(t, option.Name)
		assert.Falsef(t, seen[option.Name], "duplicate navigation icon %q", option.Name)
		seen[option.Name] = true
		assert.Truef(t, IsNavigationIcon(option.Name), "navigation icon %q is not accepted", option.Name)
	}
}

func TestNavigationIconAllowsEmptyValue(t *testing.T) {
	t.Parallel()
	assert.True(t, IsNavigationIcon(""))
	assert.False(t, IsNavigationIcon("not-a-real-navigation-icon"))
}

func TestSearchGeneratedCatalog(t *testing.T) {
	t.Parallel()

	options := Search("database", 10)
	require.NotEmpty(t, options)
	found := false
	for _, option := range options {
		if option.Name == "database" {
			found = true
		}
	}
	assert.Truef(t, found, "database icon missing from results: %#v", options)
	assert.Len(t, Search("", 3), 3)
}

func TestSearchGeneratedCatalogPagination(t *testing.T) {
	t.Parallel()

	first, hasMore := SearchPage("", 0, 3)
	require.Len(t, first, 3)
	assert.True(t, hasMore)

	second, _ := SearchPage("", len(first), 3)
	require.Len(t, second, 3)
	assert.NotEqual(t, first[len(first)-1].Name, second[0].Name)

	all := NavigationOptions()
	last, hasMore := SearchPage("", len(all)-1, 3)
	assert.Len(t, last, 1)
	assert.False(t, hasMore)
}
