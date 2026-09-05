package themes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbeddedThemeCatalog(t *testing.T) {
	t.Parallel()

	available, err := Load("")
	require.NoError(t, err)

	expected := map[string]string{
		"Catppuccin Frappe":    "dark",
		"Catppuccin Latte":     "light",
		"Catppuccin Macchiato": "dark",
		"Catppuccin Mocha":     "dark",
		"Dark":                 "dark",
		"Dracula":              "dark",
		"Gruvbox Dark":         "dark",
		"Gruvbox Light":        "light",
		"Light":                "light",
		"Nord":                 "dark",
		"One Dark":             "dark",
		"Rose Pine":            "dark",
		"Rose Pine Dawn":       "light",
		"Solarized Dark":       "dark",
		"Solarized Light":      "light",
		"Tokyo Night":          "dark",
	}

	assert.Len(t, available, len(expected))
	for title, colorScheme := range expected {
		theme, ok := Find(available, title)
		if !assert.True(t, ok, "embedded theme %q is missing", title) {
			continue
		}
		assert.Equal(t, colorScheme, theme.ColorScheme, title)
	}
}
