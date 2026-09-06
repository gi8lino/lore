package store

import (
	"testing"

	"github.com/gi8lino/lore/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestDefaultUserPreferences(t *testing.T) {
	t.Parallel()

	preferences := model.DefaultUserPreferences()

	assert.True(t, preferences.ShowPageContents)
	assert.Equal(t, model.NavigationDensityComfortable, preferences.NavigationDensity)
	assert.Equal(t, model.DefaultSidebarWidth, preferences.SidebarWidth)
	assert.True(t, preferences.ShowNavigationGuides)
	assert.True(t, preferences.RememberNavigationState)
	assert.True(t, preferences.ShowPinnedPages)
	assert.False(t, preferences.ShowRecentlyViewed)
	assert.False(t, preferences.ShowNavigationPageCounts)
	assert.Empty(t, preferences.ExpandedNavigation)
}

func TestNormalizeNavigationPaths(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"applications", "applications/identity"}, normalizeNavigationPaths([]string{
		" /applications/ ",
		"applications",
		"",
		"applications/identity",
	}))
}
